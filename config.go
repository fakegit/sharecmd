package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"

	"github.com/manifoldco/promptui"
	"golang.org/x/oauth2"
)

var providers = []string{"dropbox"}

// Config File Structure
type Config struct {
	Provider         string            `json:"provider"`
	ProviderSettings map[string]string `json:"providersettings"`
	Path             string
}

func userHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func (c Config) Write() error {
	err := os.MkdirAll(path.Dir(c.Path), 0700)
	if err != nil {
		return err
	}
	fmt.Printf("Saving config to %s \n", c.Path)
	p := c.Path
	c.Path = ""
	output, err := os.Create(p)
	if err != nil {
		return err
	}
	defer output.Close()

	return json.NewEncoder(output).Encode(c)
}

// LoadConfig from disk
func LoadConfig(path string) (Config, error) {
	config := Config{
		Path:             path,
		ProviderSettings: make(map[string]string),
	}

	content, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	} else if err != nil {
		return config, err
	}

	err = json.Unmarshal(content, &config)
	config.Path = path

	return config, err
}

func lookupConfig() (Config, error) {
	path := *configFile
	if path == "" {
		path = userHomeDir() + "/.config/sharecmd/config.json"
	}

	config, err := LoadConfig(path)
	if err != nil {
		return config, err
	}
	return config, nil
}

/*
 ██████  ██████  ███    ██ ███████ ██  ██████  ███████ ███████ ████████ ██    ██ ██████
██      ██    ██ ████   ██ ██      ██ ██       ██      ██         ██    ██    ██ ██   ██
██      ██    ██ ██ ██  ██ █████   ██ ██   ███ ███████ █████      ██    ██    ██ ██████
██      ██    ██ ██  ██ ██ ██      ██ ██    ██      ██ ██         ██    ██    ██ ██
 ██████  ██████  ██   ████ ██      ██  ██████  ███████ ███████    ██     ██████  ██
*/

func configSetup() error {
	config, err := lookupConfig()
	if err != nil {
		return err
	}

	prompt := promptui.Select{
		Label: "Select Provider",
		Items: providers,
	}

	_, provider, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return err
	}

	switch provider {
	case "dropbox":
		config.Provider = provider

		fmt.Printf("You choose %q\n", provider)

		conf := oauth2DropboxConfig()
		fmt.Printf("1. Go to %v\n", conf.AuthCodeURL("state"))
		fmt.Printf("2. Click \"Allow\" (you might have to log in first).\n")
		fmt.Printf("3. Copy the authorization code.\n")

		authorizationprompt := promptui.Prompt{
			Label:   "Authorization Code",
			Default: config.ProviderSettings["token"],
		}
		authcode, err := authorizationprompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return err
		}

		var token *oauth2.Token
		ctx := context.Background()
		token, err = conf.Exchange(ctx, authcode)
		if err != nil {
			return err
		}
		config.ProviderSettings["token"] = token.AccessToken
	}

	return config.Write()
}