package configuration

import "fmt"

type ConfigType string

const (
	FromEnvVars ConfigType = "env"
	FromFile ConfigType = "file"
)

// ParseType will try to parse the given value
// to a ConfigType. When the value cannot be parsed,
// the function will panic automatically.
func ParseType(val string) ConfigType {
	switch val {
	case "environment":
		return FromEnvVars
	case "file":
		return FromFile
	default:
		panic(fmt.Sprintf("Invalid config type %v!", val))
	}
}
