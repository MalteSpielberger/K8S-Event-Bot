package configuration

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Configuration struct {
	configType          ConfigType
	MattermostHost      string   `json:"mattermost_host"`
	ClientToken         string   `json:"client_token"`
	BotUsername         string   `json:"bot_username"`
	BotPassword         string   `json:"bot_password"`
	BotWantedUsername   string   `json:"bot_wanted_username"`
	MaintainerUsernames []string `json:"maintainer_usernames"`
	DevOpsChannel       string   `json:"dev_ops_channel"`
	TeamID              string   `json:"team_id"`
	WarnOnEventReasons  []string `json:"warn_on_event_reasons"`
	WarnOnReachCount    int      `json:"warn_on_reach_count"`
}

// NewConfiguration is used, to create a new configuration
// for the server. The configuration will check if the user
// has set the config-type flag to indicate if the config
// should be loaded from a file or from environment variables.
// When the flag isn't represented, the server-server will panic
func NewConfiguration() *Configuration {
	typeValFlag := flag.String("config-type", "", "From where the config should be loaded.")
	filePathFlag := flag.String("config-path", "", "The path to the config, when a file should be used.")

	flag.Parse()

	typeVal := *typeValFlag

	if typeVal == "" {
		panic("You need to add the config-type flag! Use -help for more information.")
	}

	configType := ParseType(typeVal)

	if configType == FromEnvVars {
		return newConfigurationFromEnv()
	} else if configType == FromFile {
		if *filePathFlag == "" {
			panic("When you want to use a file, you need to add the path! Use -help for more information.")
		}

		return newConfigurationFromFile(*filePathFlag)
	}

	return nil
}

// newConfigurationFromFile is used to create a new config
// from a file.
// When something went wrong, the function will panic.
func newConfigurationFromFile(path string) *Configuration {
	file, err := os.ReadFile(path)

	if err != nil {
		panic(fmt.Errorf("cannot read file: %w", err))
	}

	config := &Configuration{}

	if err := json.Unmarshal(file, &config); err != nil {
		panic(fmt.Errorf("cannot parse json-file to config: %w", err))
	}

	return config
}

// TODO: Implement me please
func newConfigurationFromEnv() *Configuration {
	return nil
}
