# jsonschemagen Annotation Guide #

The jsonschemagen tool uses the [ganno annotation library](https://github.com/brainicorn/ganno) to provide a custom annotation named @jsonSchema that can be used to decorate GO code within comments.

These annotations enable generating json-schemas with support for the keywords contained within the [json-schema spec](http://json-schema.org/latest/json-schema-validation.html)

## Annotating In General ##

This library provides a single annotation: @jsonSchema

It can be written in any case: @jsonschema, @JsonSchema, @jsonSchema, @JsOnScHeMa all work.

The annotation can take many different attributes (name/value pairs).

The general rules for writing annotations are:

* Annotations must be written within code comments above the thing they are annotating
* They can be written within single line or multi-line comments
* The annotation itself can span multiple lines
* There can only be a single @jsonSchema annotation for any given type/field
* The type of thing being annotated determines which attributes are valid. _see the tables below_
* GoDoc comments can appear above the annotation with an empty comment line separating the annotation from the GoDoc comments.
* GoDoc comments _can_ be used as the title/description entries in the schema. _see the section about this below_

## Annotation Specifics ##

If you just want to see some code, you can refer to [A Complex, Annotated Type Structure We Use To Test This Tool](https://github.com/brainicorn/schematestobjects)

### Titles and Descriptions ###
JSON Schema nodes can contain "title" and "description" keywords and by default the jsonschemagen tool will auto-fill in the values for these based on GoDoc comments.

That is, the first comment line up until a "." or a newline will be used as the title. The rest of the comment block will be used as the description up until a blank line followed by a @jsonSchema annotation or the end of the comment.

In most cases this is fine, but in some cases you may want the title and/or the description to differ from the GoDoc comments. In these cases you can override the values for these fields using a @jsonSchema annotation.

For example:
```go
// SomeType is going to have this as the title.
// This line and the next will become the description.
// Yes, this is part of the description.
type SomeType struct {}

// SomeOtherType will have this displayed in GoDoc but we will override it in the schema.
// This line will also show up in GoDoc but we'll set it to blank in the schema.
//
// @jsonSchema(title="I like this", description="")
type SomeOtherType struct {}

```

### Using Attributes ###
The "attributes" of the @jsonSchema annotation directly relate to the keywords found in the [json-schema validation spec](http://json-schema.org/latest/json-schema-validation.html).

Just like in the spec itself, which attributes are valid depends on the type of thing you're annotating. For example, strings can have a "maxLength" whereas numbers can have a "maximum"

The tables below describe which attributes are valid in which contexts as well as how to use / apply them.

#### A Note About Fully-Qualified Types ####
Some of the attributes in the tables below refer to their value as a "fully-qualified type". This is to facillitate referring to another go type.

A fully-qualified type is simply a string that combines the full package/import path of the type as well as the type name separated by /.

For example:

```go
// this package lives under the github.com/example repo
package pets

type Dog struct{}
```
In the above code snippet, the fully-qualified type string for Dog would be: "github.com/example/pets/Dog"

Using this format allows us to refer to types in any package including 3rd party libraries and avoids collisions when 2 libraries use the same tyoe name.

For the rest of this document "fully-qualified" will refer to the format above.

#### Common Attributes ####
These attributes are common to all types.

| Attribute   | Type                                     | Description                                                                                                                                                               | Example                                                                                      |
| ----------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| title       | string                                   | A title. _see title and descriptions section_                                                                                                                             | @jsonSchema(title="a title")                                                                 |
| description | string                                   | A description. _see title and descriptions section_                                                                                                                       | @jsonSchema(description="a description")                                                     |
| allOf       | array of fully-qualified go type strings | The input must validate against **all** of the listed types. see [combining schemas](https://spacetelescope.github.io/understanding-json-schema/reference/combining.html) | @jsonSchema(allOf=["github.com/example/SomeType", "github.com/example/SomeOtherType"]) |
| anyOf       | array of fully-qualified go type strings | The input must validate against **any** of the listed types. see [combining schemas](https://spacetelescope.github.io/understanding-json-schema/reference/combining.html) | @jsonSchema(anyOf=["github.com/example/SomeType", "github.com/example/SomeOtherType"]) |
| oneOf       | array of fully-qualified go type strings | The input must validate against **one** of the listed types. see [combining schemas](https://spacetelescope.github.io/understanding-json-schema/reference/combining.html) | @jsonSchema(oneOf=["github.com/example/SomeType", "github.com/example/SomeOtherType"]) |
| not         | fully-qualified go type string           | The input must **not**validate against the listed type. see [combining schemas](https://spacetelescope.github.io/understanding-json-schema/reference/combining.html)      | @jsonSchema(not="github.com/example/SomeType")                                            |
| default     | string                                   | A default value                                                                                                                                                           | @jsonSchema(default="default value")                                                         |

**NOTE:** The allOf, anyOf, and oneOf attributes can be combined with GO interface types to refer to implementations of the interface. For example:
```go
// @jsonSchema(
// 	oneOf=["github.com/example/music/Vinyl"
// 		,"github.com/example/music/Cassette"
// 		,"github.com/example/music/CD"
// 		,"github.com/example/music/MP3"
// 	]
// )
type MusicFormat interface {}

type MusicPlayer struct {

	Format MusicFormat `json:"format,omitempty"`
}

```

#### Object Attributes ####
The following attributes can be applied to a struct type in GO. These attributes are valid when they appear in an annotation **above** the type declaration for a struct.

| Attribute            | Type    | Description                                                                                                          | Example                                |
| -------------------- | ------- | -------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| maxProperties        | int     | The maximum number of properties the object is allowed to have                                                       | @jsonSchema(maxProperties=100)         |
| minProperties        | int     | The minimum number of properties the object must have                                                                | @jsonSchema(minProperties=1)           |
| additionalProperties | boolean or fully-qualified go type | If set to true the object can contain properties in addition to the explicitly defined properties. If a fully-qualified type string is provided, the object can contain any of the properties defined by the type listed. | @jsonSchema(additionalProperties=true)  @jsonSchema(additionalProperties="github.com/example/SomeType") |

#### A Note About Maps ####
When jsonschemagen encounters a GO map as the type for a field, it generates a basic object schema with "additionalProperties" automatically set to true.

#### String Attributes ####
The following attributes can be applied to a string type in GO, either as a field type or a top-level type definition.

| Attribute | Type   | Description                                                                                                                            | Example                             |
| --------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| pattern   | string | An ECMA 262 regular expression that the value must match                                                                               | @jsonSchema(pattern="\^info[0-9]$") |
| maxLength | int    | The maximum length the string can be                                                                                                   | @jsonSchema(maxLength=100)          |
| minLength | int    | The minimum length the string must be                                                                                                  | @jsonSchema(minLength=1)            |
| format    | string | a valid json-schema format identifier. see [defined formats](http://json-schema.org/latest/json-schema-validation.html#rfc.section.7)  | @jsonSchema(format="uuid")          |

#### A Note About Time ####
When jsonschemagen encounters a time.Time as the type for a field, it generates a string schema with "format" automatically set to "datetime".

see [more about the datetime format](http://json-schema.org/latest/json-schema-validation.html#rfc.section.7.3.1)

#### Numeric Attributes ####
The following attributes can be applied to a numeric type in GO, either as a field type or a top-level type definition.

When generating schemas for numeric types, all int and uint "flavors" will have the json type "integer" and all floats will have the json type "number"

| Attribute        | Type    | Description                                                      | Example                            |
| ---------------- | ------- | ---------------------------------------------------------------- | ---------------------------------- |
| maximum          | float   | The maximum value                                                | @jsonSchema(maximum=100)           |
| minimum          | float   | The minimum value                                                | @jsonSchema(minimum=1)             |
| multipleOf       | float   | The value must be a multiple of this                             | @jsonSchema(multipleOf=5)          |
| exclusiveMaximum | boolean | If true, the value must not equal the value specified in maximum | @jsonSchema(exclusiveMaximum=true) |
| exclusiveMinimum | bollean | If true, the value must not equal the value specified in minimum | @jsonSchema(exclusiveMinimum=true) |

#### Slice Attributes ####
The following attributes can be applied to a slice type in GO, either as a field type or a top-level type definition.

| Attribute       | Type    | Description                                                          | Example                           |
| --------------- | ------- | -------------------------------------------------------------------- | --------------------------------- |
| maxItems        | int     | The maximum number of items                                          | @jsonSchema(maxItems=100)         |
| minItems        | int     | The minimum number of items                                          | @jsonSchema(minItems=1)           |
| additionalItems | boolean | If true, validation will always pass regardless of the type of items | @jsonSchema(additionalItems=true) |
| uniqueItems     | bollean | If true, all items in the slice must be unique                       | @jsonSchema(uniqueItems=true)     |

## Known Limitations ##
Although we've tried to be as complete as possible when adhering to the json-schema spec, there are a few things that are currently unsupported.

Most of these will be addressed in a future release where possible.

### The "id" Keyword ###
JSON-Schema supports adding an "id" keyword to any object/sub-schema which defines the base uri for that schema and the base uri that all references below that schema are validated against.

The jsonschemagen tool allows you to include the id keyword, however including it on any type other than the root will cause unexpected results.

Currently the tool places **all** encountered sub-types, regardless of how deep they are nested, under a root level #/definitions node. This greatly reduces the amount of complexity the parser has to deal with when creating references to nested types and promotes the maximum level of type re-use.

In a future release we may add full support for id/schema uris. For now, either don't set it or only set it on the root type.

For more information, see [the id keyword json-schema spec](http://json-schema.org/latest/json-schema-core.html#rfc.section.8.2)

### The "patternProperties" Keyword ###
The "patternProperties" keyword allows you to specify a JSON object whose property _names_ are ECMA 262 regular expressions that the input object's property names must match.
The value of each on the patternProperty fields must be a valid json-schema / type.

Due to the complexity of representing this in a straight-forward manner, this keyword is currently not supported.

### The "dependencies" Keyword ###
The "dependency" keyword can be used to change the schema of the object in question based on rules around the presence of other properties and/or schemas.

This is not currently supported by the jsonschemagen tool.

For more information, see [this page about schema dependencies](https://spacetelescope.github.io/understanding-json-schema/reference/object.html#dependencies)

## Useful Links ##

[A Complex, Annotated Type Structure We Use To Test This Tool](https://github.com/brainicorn/schematestobjects/src)

[API Documentation](https://godoc.org/github.com/brainicorn/jsonschemagen)

[Issue Tracker](https://github.com/brainicorn/jsonschemagen/issues)

## Contributors ##

Pull requests, issues and comments welcome. For pull requests:

* Add tests for new features and bug fixes
* Follow the existing style
* Separate unrelated changes into multiple pull requests

See the existing issues for things to start contributing.

For bigger changes, make sure you start a discussion first by creating
an issue and explaining the intended change.

## License ##

Apache 2.0 licensed, see [LICENSE.txt](LICENSE.txt) file.
