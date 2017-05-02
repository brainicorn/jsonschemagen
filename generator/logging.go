package generator

import "fmt"

// LogLevel is an enum specifying log verbosity
type LogLevel uint8

const (
	// QuietLevel level. Makes everything silent.
	QuietLevel LogLevel = iota
	// InfoLevel level.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. chatty logging.
	DebugLevel
	// VerboseLevel level. Usually only enabled when debugging. Very verbose logging.
	VerboseLevel
)

// LogInfo writes an info message to std out
func (g *JSONSchemaGenerator) LogInfo(args ...interface{}) {
	if g.options.LogLevel >= InfoLevel {
		fmt.Println(args...)
	}
}

// LogInfoF writes an info formatted message to std out
func (g *JSONSchemaGenerator) LogInfoF(format string, args ...interface{}) {
	if g.options.LogLevel >= InfoLevel {
		fmt.Printf(format, args...)
	}
}

// LogDebug writes a debug message to std out
func (g *JSONSchemaGenerator) LogDebug(args ...interface{}) {
	if g.options.LogLevel >= DebugLevel {
		fmt.Println(args...)
	}
}

// LogDebugF writes a debug formatted message to std out
func (g *JSONSchemaGenerator) LogDebugF(format string, args ...interface{}) {
	if g.options.LogLevel >= DebugLevel {
		fmt.Printf(format, args...)
	}
}

// LogVerbose writes a verbose message to std out
func (g *JSONSchemaGenerator) LogVerbose(args ...interface{}) {
	if g.options.LogLevel >= VerboseLevel {
		fmt.Println(args...)
	}
}

// LogVerboseF writes a verbose formatted message to std out
func (g *JSONSchemaGenerator) LogVerboseF(format string, args ...interface{}) {
	if g.options.LogLevel >= VerboseLevel {
		fmt.Printf(format, args...)
	}
}
