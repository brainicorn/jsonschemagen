package schema

import "encoding/json"

const (
	// DefinitionRoot is the root of the definition ref
	DefinitionRoot = "#/definitions/"
)

// JSONSchema is the base interface that represents common json-schema attributes.
type JSONSchema interface {
	Clone() JSONSchema
	GetSchemaURI() string
	GetID() string
	GetType() string
	GetRef() string
	GetTitle() string
	GetDescription() string
	GetAllOf() []JSONSchema
	GetAnyOf() []JSONSchema
	GetOneOf() []JSONSchema
	GetNot() JSONSchema
	GetDefinitions() map[string]JSONSchema
	GetDefault() interface{}

	AddDefinition(key string, def JSONSchema)
	SetSchemaURI(uri string)
	SetID(id string)
	SetRef(ref string)
	SetTitle(title string)
	SetDescription(description string)
	SetAllOf(items []JSONSchema)
	SetAnyOf(items []JSONSchema)
	SetOneOf(items []JSONSchema)
	SetNot(not JSONSchema)
	SetDefault(def interface{})
}

// BasicSchema is the base implementation of the JsonSchema interface.
type basicSchema struct {
	SchemaURI    string                `json:"$schema,omitempty"`
	ID           string                `json:"id,omitempty"`
	Ref          string                `json:"$ref,omitempty"`
	JSONType     string                `json:"type,omitempty"`
	Title        string                `json:"title,omitempty"`
	Description  string                `json:"description,omitempty"`
	AllOf        []JSONSchema          `json:"allOf,omitempty"`
	AnyOf        []JSONSchema          `json:"anyOf,omitempty"`
	OneOf        []JSONSchema          `json:"oneOf,omitempty"`
	Not          JSONSchema            `json:"not,omitempty"`
	Definitions  map[string]JSONSchema `json:"definitions,omitempty"`
	DefaultValue interface{}           `json:"default,omitempty"`
}

// FromJSON returns a JSONSchema object from the given json bytes.
func FromJSON(js []byte) (JSONSchema, error) {
	var err error
	var obj JSONSchema

	var stuff map[string]interface{}

	err = json.Unmarshal(js, &stuff)

	if err != nil {
		return nil, err
	}

	if jsontype, ok := stuff["type"]; ok {
		switch jsontype {
		case SchemaTypeObject:
			obj = &defaultObjectSchema{}
			err = json.Unmarshal(js, obj)
			return obj, err
		case SchemaTypeString:
			obj = &defaultStringSchema{}
			err = json.Unmarshal(js, obj)
		case SchemaTypeArray:
			obj = &defaultArraySchema{}
			err = json.Unmarshal(js, obj)
		case SchemaTypeInteger, SchemaTypeNumber:
			obj = &defaultNumericSchema{}
			err = json.Unmarshal(js, obj)
		case SchemaTypeBoolean:
			obj = &defaultSimpleSchema{}
			err = json.Unmarshal(js, obj)
		}
	}

	if ref, ok := stuff["$ref"]; ok {
		obj = &basicSchema{}
		obj.(*basicSchema).Ref = ref.(string)
	}

	return obj, err

}

// NewBasicSchema creates a new BasicSchema
func NewBasicSchema(jsonType string) JSONSchema {
	return &basicSchema{
		JSONType:    jsonType,
		AllOf:       make([]JSONSchema, 0),
		AnyOf:       make([]JSONSchema, 0),
		OneOf:       make([]JSONSchema, 0),
		Definitions: make(map[string]JSONSchema),
	}
}

func (s *basicSchema) UnmarshalJSON(b []byte) error {
	var err error
	var stuff map[string]interface{}

	s.AllOf = make([]JSONSchema, 0)
	s.AnyOf = make([]JSONSchema, 0)
	s.OneOf = make([]JSONSchema, 0)
	s.Definitions = make(map[string]JSONSchema)

	err = json.Unmarshal(b, &stuff)

	if err == nil {
		for k, v := range stuff {
			switch k {
			case "$schema":
				s.SchemaURI = v.(string)
			case "id":
				s.ID = v.(string)
			case "$ref":
				s.Ref = v.(string)
			case "type":
				s.JSONType = v.(string)
			case "title":
				s.Title = v.(string)
			case "description":
				s.Description = v.(string)
			case "allOf":
				for _, xo := range v.([]interface{}) {
					xb, xerr := json.Marshal(xo)
					if xerr != nil {
						return xerr
					}
					xs, xerr := FromJSON(xb)
					if xerr != nil {
						return xerr
					}
					s.AllOf = append(s.AllOf, xs)
				}

			case "anyOf":
				for _, xo := range v.([]interface{}) {
					xb, xerr := json.Marshal(xo)
					if xerr != nil {
						return xerr
					}
					xs, xerr := FromJSON(xb)
					if xerr != nil {
						return xerr
					}
					s.AnyOf = append(s.AnyOf, xs)
				}

			case "oneOf":
				for _, xo := range v.([]interface{}) {
					xb, xerr := json.Marshal(xo)
					if xerr != nil {
						return xerr
					}
					xs, xerr := FromJSON(xb)
					if xerr != nil {
						return xerr
					}
					s.OneOf = append(s.OneOf, xs)
				}

			case "not":
				xb, xerr := json.Marshal(v)
				if xerr != nil {
					return err
				}
				xs, xerr := FromJSON(xb)
				if xerr != nil {
					return xerr
				}
				s.Not = xs
			case "definitions":
				for mk, mv := range v.(map[string]interface{}) {
					mb, xerr := json.Marshal(mv)
					if xerr != nil {
						return xerr
					}
					ms, xerr := FromJSON(mb)
					if xerr != nil {
						return xerr
					}
					s.Definitions[mk] = ms
				}
				//			case "default":
				//				s.Description = v.(string)
			}
		}
	}

	return err
}

func (s *basicSchema) Clone() JSONSchema {
	s2 := &basicSchema{}
	*s2 = *s

	return s2
}

func (s *basicSchema) GetSchemaURI() string {
	return s.SchemaURI
}

func (s *basicSchema) GetID() string {
	return s.ID
}

func (s *basicSchema) GetType() string {
	return s.JSONType
}

func (s *basicSchema) GetRef() string {
	return s.Ref
}

func (s *basicSchema) GetTitle() string {
	return s.Title
}

func (s *basicSchema) GetDescription() string {
	return s.Description
}

func (s *basicSchema) GetAllOf() []JSONSchema {
	return s.AllOf
}

func (s *basicSchema) GetAnyOf() []JSONSchema {
	return s.AnyOf
}

func (s *basicSchema) GetOneOf() []JSONSchema {
	return s.OneOf
}

func (s *basicSchema) GetNot() JSONSchema {
	return s.Not
}

func (s *basicSchema) GetDefinitions() map[string]JSONSchema {
	return s.Definitions
}

func (s *basicSchema) GetDefault() interface{} {
	return s.DefaultValue
}

func (s *basicSchema) AddDefinition(key string, def JSONSchema) {
	s.Definitions[key] = def
}
func (s *basicSchema) SetSchemaURI(uri string) {
	s.SchemaURI = uri
}

func (s *basicSchema) SetID(id string) {
	s.ID = id
}

func (s *basicSchema) SetRef(ref string) {
	s.Ref = ref
}

func (s *basicSchema) SetTitle(title string) {
	s.Title = title
}

func (s *basicSchema) SetDescription(description string) {
	s.Description = description
}

func (s *basicSchema) SetAllOf(items []JSONSchema) {
	s.AllOf = items
}

func (s *basicSchema) SetAnyOf(items []JSONSchema) {
	s.AnyOf = items
}

func (s *basicSchema) SetOneOf(items []JSONSchema) {
	s.OneOf = items
}

func (s *basicSchema) SetNot(not JSONSchema) {
	s.Not = not
}

func (s *basicSchema) SetDefault(def interface{}) {
	s.DefaultValue = def
}
