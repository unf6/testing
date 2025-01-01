package utils

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"
)

func GetConfigDir() string {
    var configDir string

    switch runtime.GOOS{
    case "windows":
        configDir = filepath.Join(os.Getenv("APPDATA"), "tasks-cli")
         default: 
         configDir = filepath.Join(os.Getenv("HOME"), ".config", "tasks-cli")
    }

    if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			fmt.Printf("Failed to create config directory: %v\n", err)
			os.Exit(1)
		}
	}

	return configDir
}


