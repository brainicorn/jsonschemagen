package generator

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/types"
	"time"

	"github.com/brainicorn/ganno"
	"github.com/brainicorn/jsonschemagen/schema"
	"golang.org/x/tools/go/loader"
)

// Options holds the configuration options for the schema generator instance
type Options struct {
	// SpecVersion is the url of the spec version
	SpecVersion schema.SpecVersion
	// IncludeTests will parse test files during generation when set to true or ignore them when false.
	IncludeTests bool
	// AutoCreateDefs will automatically create definitions when set to true. If false, schemas will be
	// included inline unless the 'definition' attribute is set within an @jsonSchema annotation
	AutoCreateDefs bool
	// LogLevel sets the verbosity of log statements
	LogLevel LogLevel
	// DefinitionPrefix is an optional prefix that will get pre-pended to all definition refs after the definition root.
	DefinitionPrefix string
	// SupressXAttrs is a flag ti supress non-standard schema properties like x-*
	SupressXAttrs bool
}

// JSONSchemaGenerator is the thing that generates schemas.
// This should not be created manually, instead use NewJSONSchemaGenerator(...)
type JSONSchemaGenerator struct {
	basePackage      string
	rootPackage      string
	rootType         string
	options          Options
	program          *loader.Program
	annoParser       ganno.AnnotationParser
	globalDefCache   map[string]*definition
	embeddedDefCache map[string]*definition
	simpleTypeCache  map[string]*definition
	fieldAnnoCache   map[*ast.Field]*schemaAnno
}

type declInfo struct {
	pkg              *loader.PackageInfo
	file             *ast.File
	decl             *ast.GenDecl
	typeSpec         *ast.TypeSpec
	defKey           string
	schemaAnnotation *schemaAnno
	parsedAnnos      bool
	isRoot           bool
}

type definition struct {
	decl   *declInfo
	schema schema.JSONSchema
}

// NewOptions creates a new options instance with sane defaults.
func NewOptions() Options {
	return Options{
		SpecVersion:    schema.SpecVersionDraftV4,
		IncludeTests:   false,
		AutoCreateDefs: true,
		LogLevel:       InfoLevel,
		SupressXAttrs:  false,
	}
}

func (g *JSONSchemaGenerator) newDeclInfo(pkg *loader.PackageInfo, file *ast.File, decl *ast.GenDecl, spec *ast.TypeSpec) *declInfo {
	di := &declInfo{
		pkg:      pkg,
		file:     file,
		decl:     decl,
		typeSpec: spec,
	}

	di.defKey = g.getDefinitionKey(di)

	return di
}

func (g *JSONSchemaGenerator) loadProgram(basePackage string, options Options) (*loader.Program, error) {
	if g.program != nil {
		return g.program, nil
	}

	var scan loader.Config

	scan.ParserMode = goparser.ParseComments
	scan.TypeCheckFuncBodies = func(path string) bool { return false }

	if options.IncludeTests {
		scan.ImportWithTests(basePackage)
	} else {
		scan.Import(basePackage)
	}

	return scan.Load()
}

// NewJSONSchemaGenerator creates an instance of the generator.
// basePackage is the package where the root schema object lives.
// rootType is the name of the type/struct ib the basePackage that represents the schema root.
func NewJSONSchemaGenerator(basePackage, rootType string, options Options) *JSONSchemaGenerator {
	aparser := ganno.NewAnnotationParser()
	aparser.RegisterFactory("jsonSchema", &schemaAnnoFactory{})

	return &JSONSchemaGenerator{
		basePackage:      basePackage,
		rootType:         rootType,
		options:          options,
		annoParser:       aparser,
		globalDefCache:   make(map[string]*definition),
		embeddedDefCache: make(map[string]*definition),
		simpleTypeCache:  make(map[string]*definition),
		fieldAnnoCache:   make(map[*ast.Field]*schemaAnno),
	}
}

// Generate is the main function that is used to generate a JSONSchema.
func (g *JSONSchemaGenerator) Generate() (schema.JSONSchema, error) {
	var program *loader.Program
	var err error
	var rootSchema schema.JSONSchema

	start := time.Now()
	program, err = g.loadProgram(g.basePackage, g.options)

	if err == nil {
		g.program = program

		rootSchema, err = g.doGenerate()
	}

	g.LogInfoF("generation completed in %s for %s\n", time.Since(start), g.basePackage+"/"+g.rootType)
	return rootSchema, err
}

// SubGenerate can be used to generate sub schemas after the main root has been generated.
func (g *JSONSchemaGenerator) SubGenerate(basePackage, rootType string) (schema.JSONSchema, error) {
	var err error
	var rootSchema schema.JSONSchema

	g.basePackage = basePackage
	g.rootType = rootType
	g.globalDefCache = make(map[string]*definition)
	g.embeddedDefCache = make(map[string]*definition)

	start := time.Now()
	rootSchema, err = g.doGenerate()

	g.LogInfoF("generation completed in %s for %s\n", time.Since(start), g.basePackage+"/"+g.rootType)

	return rootSchema, err
}

func (g *JSONSchemaGenerator) doGenerate() (schema.JSONSchema, error) {
	var err error
	var rootDeclInfo *declInfo
	var rootSchema schema.JSONSchema

	rootDeclInfo, err = g.findRootDecl(g.program)

	if err == nil {
		g.LogVerbose("root decl: ", rootDeclInfo.typeSpec.Name.Name)
		rootSchema, err = g.generateSchemaForExpr(rootDeclInfo, rootDeclInfo.typeSpec.Type, nil)
	}

	if err == nil {
		rootSchema.SetSchemaURI(string(g.options.SpecVersion))

		for _, def := range g.globalDefCache {
			if g.shouldReturnRef(def.decl) {
				rootSchema.AddDefinition(def.decl.defKey, def.schema)
			}
		}
	}

	if err == nil && rootSchema.GetID() == "" {
		rootSchema.SetID("http://" + rootDeclInfo.pkg.Pkg.Path() + "/" + rootDeclInfo.typeSpec.Name.Name)
	}

	return rootSchema, err
}

func (g *JSONSchemaGenerator) findRootDecl(program *loader.Program) (*declInfo, error) {
	var searchPackages map[*types.Package]*loader.PackageInfo

	g.LogDebug("looking for root object")

	if baseInfo, ok := program.Imported[g.basePackage]; ok {
		searchPackages = make(map[*types.Package]*loader.PackageInfo)
		searchPackages[baseInfo.Pkg] = baseInfo
	}

	if searchPackages == nil {
		for _, pi := range program.AllPackages {
			if pi.Pkg.Path() == g.basePackage {
				searchPackages = make(map[*types.Package]*loader.PackageInfo)
				searchPackages[pi.Pkg] = pi
			}
		}
	}

	//let's find the file with the root object in it
	for pkg, pkgInfo := range searchPackages {
		g.LogVerbose("analyzing package: ", pkg.Path())

		for _, file := range pkgInfo.Files {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spc := range gd.Specs {
					if ts, ok := spc.(*ast.TypeSpec); ok {
						if ts.Name.Name == g.rootType {
							g.LogVerboseF("found root decl %s: %#v\n", ts.Name.Name, gd)
							rd := g.newDeclInfo(pkgInfo, file, gd, ts)
							rd.isRoot = true
							return rd, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("root not found")

}

func (g *JSONSchemaGenerator) generateObjectSchema(declInfo *declInfo, field *ast.Field, embedded bool) (schema.JSONSchema, error) {
	g.LogDebugF("processing object schema for struct %s\n", declInfo.defKey)

	var err error
	var defCache map[string]*definition

	objectSchema := schema.NewObjectSchema(g.options.SupressXAttrs)

	if !embedded {
		defCache = g.globalDefCache
	} else {
		defCache = g.embeddedDefCache
	}

	// if we already have the schema...
	if objDef, found := defCache[declInfo.defKey]; found {

		g.LogDebug("returning cached object schema for ", declInfo.defKey)
		if !embedded && g.shouldReturnRef(declInfo) {
			refSchema := schema.NewBasicSchema("")
			refSchema.SetRef(schema.DefinitionRoot + declInfo.defKey)
			return refSchema, nil
		}

		return objDef.schema, nil

	}

	g.LogDebug("creating new object schema for struct ", declInfo.defKey)

	objectSchema.SetGoPath(declInfo.pkg.Pkg.Path() + "/" + declInfo.typeSpec.Name.Name)
	if err != nil {
		return nil, err
	}

	err = g.addObjectAttrsForDecl(objectSchema, declInfo)

	if err != nil {
		return nil, err
	}

	props := make(map[string]schema.JSONSchema)

	for _, propField := range declInfo.typeSpec.Type.(*ast.StructType).Fields.List {

		if len(propField.Names) == 0 {
			g.LogVerbose("processing field without a name, must be embedded...")
			embeddedSchema, e := g.generateEmbeddedSchema(declInfo, propField.Type)

			if e != nil {
				err = e
				break
			}
			for k, v := range embeddedSchema.(schema.ObjectSchema).GetProperties() {
				props[k] = v
			}

			for _, r := range embeddedSchema.(schema.ObjectSchema).GetRequired() {
				objectSchema.AddRequiredField(r)
			}

		} else if propField.Names[0] != nil && propField.Names[0].IsExported() {
			g.LogVerboseF("processing field '%s' on struct %s\n", propField.Names[0].Name, declInfo.typeSpec.Name.Name)

			propName, propIgnore := jsonTagInfo(propField)

			if propIgnore {
				continue
			}

			fschema, e := g.generateSchemaForExpr(declInfo, propField.Type, propField)

			if e != nil {
				err = e
				break
			}

			props[propName] = fschema

			if g.fieldIsRequired(propField) {
				objectSchema.AddRequiredField(propName)
			}
		}
	}

	if err == nil {
		objectSchema.SetProperties(props)

		g.LogDebug("adding def to cache: ", declInfo.defKey)
		def := &definition{
			decl:   declInfo,
			schema: objectSchema,
		}

		defCache[declInfo.defKey] = def

	}

	if g.shouldReturnRef(declInfo) && !embedded {
		refSchema := schema.NewBasicSchema("")
		refSchema.SetRef(schema.DefinitionRoot + declInfo.defKey)
		return refSchema, err
	}
	return objectSchema, err
}

func (g *JSONSchemaGenerator) generateSchemaForExpr(ownerDecl *declInfo, fieldExpr ast.Expr, field *ast.Field) (schema.JSONSchema, error) {
	var foundDecl *declInfo
	var err error
	var ok bool
	var simpleSchema schema.JSONSchema
	var generatedSchema schema.JSONSchema
	var fieldSchema schema.JSONSchema

	// if we already have the schema...
	if simpleDef, found := g.simpleTypeCache[ownerDecl.defKey]; found {
		g.LogDebug("returning cached simple schema for ", ownerDecl.defKey)
		generatedSchema = simpleDef.schema
	}

	if generatedSchema == nil {

		switch fieldType := fieldExpr.(type) {
		case *ast.StructType:
			g.LogVerbose("field type is struct: ")
			generatedSchema, err = g.generateObjectSchema(ownerDecl, field, false)

		case *ast.Ident:
			g.LogVerbose("field type is ident: ", fieldType.Name)

			if simpleSchema, ok, err = g.generateSchemaForBuiltIn(fieldType.Name, field); ok {
				generatedSchema = simpleSchema
				break
			}

			if err == nil {
				foundDecl, err = g.findDeclInfoForPackage(ownerDecl.pkg, ownerDecl.file, fieldType.Name)
			}

			if err == nil {
				g.LogDebug("found decl ", foundDecl.typeSpec.Name.Name)
				//if we already have the schema...
				if simpleDef, found := g.simpleTypeCache[foundDecl.defKey]; found {
					g.LogDebug("returning cached simple schema for ", foundDecl.defKey)
					generatedSchema = simpleDef.schema
					break
				}

				g.LogVerboseF("checking if %s type %s is a simple type\n", fieldType.Name, types.ExprString(foundDecl.typeSpec.Type))
				// if the declared type is a built-in, we need to fill in any schema attrs on the base type
				if jsonType, found := builtinTypes[types.ExprString(foundDecl.typeSpec.Type)]; found {
					g.LogVerbose("found decl is a simple type: ", jsonType)

					if simpleSchema, ok, err = g.generateSchemaForBuiltIn(types.ExprString(foundDecl.typeSpec.Type), field); ok {

						g.LogVerbose("got a simpleSchema for ", fieldType.Name)
						anno, simpleErr := g.findJSONSchemaAnnotationForDecl(foundDecl)
						if simpleErr != nil {
							err = simpleErr
							break
						}

						switch jsonType {
						case "string":
							g.populateStringAttrs(simpleSchema.(schema.StringSchema), anno)

						case "number", "integer":
							g.populateNumericAttrs(simpleSchema.(schema.NumericSchema), anno)
						}

						g.simpleTypeCache[foundDecl.defKey] = &definition{
							decl:   foundDecl,
							schema: simpleSchema,
						}

						generatedSchema = simpleSchema
						break
					}

				}
				generatedSchema, err = g.generateSchemaForExpr(foundDecl, foundDecl.typeSpec.Type, field)
			}

		case *ast.StarExpr:
			g.LogVerbose("got star expression type ", fieldType.X)
			g.LogVerbose("selector is ", fieldType.X)
			generatedSchema, err = g.generateSchemaForExpr(ownerDecl, fieldType.X, field)

		case *ast.SelectorExpr:
			g.LogVerboseF("got selector expression type %s.%s\n", fieldType.X, fieldType.Sel.Name)
			fullSelectorName := fmt.Sprintf("%s.%s", fieldType.X, fieldType.Sel.Name)

			if "json.RawMessage" == fullSelectorName {
				generatedSchema = schema.NewMapSchema(g.options.SupressXAttrs)
				break
			}

			if simpleSchema, ok, err = g.generateSchemaForBuiltIn(fullSelectorName, field); ok {
				generatedSchema = simpleSchema
				break
			}

			if err == nil {
				foundDecl, err = g.findDeclInfoForSelector(ownerDecl, fieldType)
			}

			if err == nil {
				generatedSchema, err = g.generateSchemaForExpr(foundDecl, foundDecl.typeSpec.Type, field)
			}

		case *ast.ArrayType:
			g.LogVerbose("got array type ")
			generatedSchema, err = g.generateArraySchema(ownerDecl, fieldType.Elt, field)

		case *ast.InterfaceType:
			g.LogVerbose("got interface type ")

			if field != nil {
				generatedSchema, err = g.generateInterfaceSchemaForField(ownerDecl, field)
				break
			}

			generatedSchema, err = g.generateInterfaceSchemaForDecl(ownerDecl)

		case *ast.MapType:
			g.LogVerbose("got map type ")
			if field != nil {
				generatedSchema, err = g.generateInterfaceSchemaForField(ownerDecl, field)
				break
			}

			generatedSchema, err = g.generateInterfaceSchemaForDecl(ownerDecl)
		}
	}

	if err == nil {
		fieldSchema = generatedSchema.Clone()
		if field != nil {
			g.addCommonAttrsForField(fieldSchema, field)
		}

		g.addDocsForField(fieldSchema, foundDecl, field)
	}

	return fieldSchema, err
}

func (g *JSONSchemaGenerator) generateEmbeddedSchema(ownerDecl *declInfo, expr ast.Expr) (schema.JSONSchema, error) {
	var embeddedDecl *declInfo

	switch embeddedType := expr.(type) {
	case *ast.Ident:
		g.LogVerbose("embedded type is ident")

		embeddedDecl, _ = g.findDeclInfoForPackage(ownerDecl.pkg, ownerDecl.file, embeddedType.Name)

	case *ast.SelectorExpr:
		g.LogVerbose("embedded type is SelectorExpr")

		embeddedDecl, _ = g.findDeclInfoForSelector(ownerDecl, embeddedType)

	case *ast.StarExpr:
		return g.generateEmbeddedSchema(ownerDecl, embeddedType.X)
	}

	switch embeddedDecl.typeSpec.Type.(type) {
	case *ast.StructType:
		return g.generateObjectSchema(embeddedDecl, nil, true)

	}
	return nil, fmt.Errorf("unable to resolve embedded struct for: %#v", expr)
}

func (g *JSONSchemaGenerator) generateSchemaForBuiltIn(name string, field *ast.Field) (schema.JSONSchema, bool, error) {
	var err error
	var simpleSchema schema.JSONSchema
	var jsonType string
	var found bool

	g.LogVerbose("looking for built-in type ", name)

	if _, found = jsonTypes[name]; found {
		jsonType = name
	} else {
		jsonType, found = builtinTypes[name]
	}

	if found {
		g.LogVerbose("found built-in type ", jsonType)
		simpleSchema, err = g.generateSimpleSchema(jsonType, field)
		if err == nil {
			g.LogVerbose("returning simple schema ", jsonType)
			return simpleSchema, true, err
		}
	}

	return nil, false, err
}

func (g *JSONSchemaGenerator) generateSimpleSchema(simpleType string, field *ast.Field) (schema.JSONSchema, error) {
	var err error

	switch simpleType {
	case "string":
		ss := schema.NewStringSchema()
		err = g.addStringAttrsForField(ss, field)
		if nil != field {
			g.LogDebug("got string, checking if it's a time: ", types.ExprString(field.Type))
			if types.ExprString(field.Type) == "time.Time" {
				g.LogDebug("it's a time, adding date-time format")
				ss.SetFormat("date-time")
			}
		}
		return ss, err
	case "number", "integer":
		ss := schema.NewNumericSchema(simpleType)
		err = g.addNumericAttrsForField(ss, field)

		return ss, err
	}

	ss := schema.NewSimpleSchema(simpleType)

	return ss, err
}

func (g *JSONSchemaGenerator) generateArraySchema(ownerDecl *declInfo, elemExpr ast.Expr, field *ast.Field) (schema.JSONSchema, error) {
	var err error
	var elemSchema schema.JSONSchema

	// first we need to check if we need to convert []byte to string instead of an array
	//	switch elemType := elemExpr.(type) {
	//	case *ast.Ident:
	//		g.LogVerbose("field type is ident: ", elemType.Name)
	//		var ok bool
	//		if elemSchema, ok, err = g.generateSchemaForBuiltIn("[]"+elemType.Name, field); ok {
	//			return elemSchema, err
	//		}
	//	}

	arraySchema := schema.NewArraySchema()

	err = g.addArrayAttrsForField(arraySchema, field)
	g.LogDebug("generating schema for array elem expr: ", elemExpr)
	if err == nil {
		elemSchema, err = g.generateSchemaForExpr(ownerDecl, elemExpr, nil)
	}

	if err == nil {
		arraySchema.SetItems(elemSchema)
	}

	return arraySchema, err

}

func (g *JSONSchemaGenerator) generateInterfaceSchemaForField(decl *declInfo, field *ast.Field) (schema.JSONSchema, error) {
	var err error
	var hasAnno, fhasXof, dhasAnno bool
	var iSchema schema.JSONSchema

	fhasXof, err = g.fieldHasXofAnnotation(field)

	if err == nil {
		dhasAnno, err = g.declHasSchemaAnnotation(decl)
	}

	hasAnno = fhasXof || dhasAnno

	if err == nil {
		g.LogVerbose("hasAnno?: ", hasAnno)
		if !hasAnno {
			iSchema = schema.NewMapSchema(g.options.SupressXAttrs)
			err = g.addCommonAttrsForDecl(iSchema, decl)
		} else {
			iSchema = schema.NewObjectSchema(g.options.SupressXAttrs)
			err = g.addObjectAttrsForDecl(iSchema.(schema.ObjectSchema), decl)
		}
	}
	return iSchema, err
}

func (g *JSONSchemaGenerator) generateInterfaceSchemaForDecl(decl *declInfo) (schema.JSONSchema, error) {
	var err error
	//var hasXof bool
	var iSchema schema.JSONSchema

	//	hasXof, err = g.declHasXofAnnotation(decl)

	//	if err == nil {
	//		if !hasXof {
	//			iSchema = schema.NewMapSchema(g.options.SupressXAttrs)
	//			err = g.addCommonAttrsForDecl(iSchema, decl)
	//		} else {
	iSchema = schema.NewObjectSchema(g.options.SupressXAttrs)
	err = g.addObjectAttrsForDecl(iSchema.(schema.ObjectSchema), decl)
	//		}
	//	}

	return iSchema, err
}

func (g *JSONSchemaGenerator) generateMapSchema(field *ast.Field) (schema.JSONSchema, error) {
	mSchema := schema.NewMapSchema(g.options.SupressXAttrs)

	return mSchema, nil
}
