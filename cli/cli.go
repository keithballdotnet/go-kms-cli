package cli

import (
	//"encoding/json"
	//"io/ioutil"
	"os"

	"github.com/keithballdotnet/go-kms/kms"
	"github.com/crackcomm/go-clitable"
	"github.com/jawher/mow.cli"
	"log"
)

// exit will return an error code and the reason to the os
func Exit(messages string, errorCode int) {
	// Exit code and messages based on Nagios plugin return codes (https://nagios-plugins.org/doc/guidelines.html#AEN78)
	var prefix = map[int]string{0: "OK", 1: "Warning", 2: "Critical", 3: "Unknown"}

	// Catch all unknown errorCode and convert them to Unknown
	if errorCode < 0 || errorCode > 3 {
		errorCode = 3
	}

	log.Printf("%s %s\n", prefix[errorCode], messages)
	os.Exit(errorCode)
}

// CreateCommands for the kms
func CreateCommands(app *cli.Cli) {

	/* Functions to implement...
	/api/v1/go-kms/listkeys
	/api/v1/go-kms/createkey
	/api/v1/go-kms/generatedatakey
	/api/v1/go-kms/enablekey
	/api/v1/go-kms/disablekey
	/api/v1/go-kms/decrypt
	/api/v1/go-kms/encrypt*/

	app.Command("keys", "Key functions", func(commandCmd *cli.Cmd) {
		commandCmd.Command("list", "List all available keys", func(listKeysCmd *cli.Cmd) {
			listKeysCmd.Action = func() {

				// List the key available...
				listKeyRequest := kms.ListKeysRequest{}

				listKeyResponse := &kms.ListKeysResponse{}
				err := Client.Do("POST", "/api/v1/go-kms/listkeys", &listKeyRequest, listKeyResponse)
				if err != nil {
					Exit(err.Error(), 1)
				}

				OutputMetadata(listKeyResponse.KeyMetadata)
			}
		})
		commandCmd.Command("create", "Create a new key", func(createKeyCmd *cli.Cmd) {

			description := createKeyCmd.StringOpt("d description", "", "Description for the new key")

			createKeyCmd.Action = func() {
				createKeyRequest := kms.CreateKeyRequest{Description: *(description)}

				createKeyResponse := &kms.CreateKeyResponse{}
				err := Client.Do("POST", "/api/v1/go-kms/createkey", &createKeyRequest, createKeyResponse)
				if err != nil {
					Exit(err.Error(), 1)
				}

				OutputMetadata([]kms.KeyMetadata{createKeyResponse.KeyMetadata})
			}
		})
		commandCmd.Command("disable", "Disable a key", func(disableKeyCmd *cli.Cmd) {

			keyID := disableKeyCmd.StringArg("KEYID", "", "The KeyID of the key to be disabled")

			disableKeyCmd.Action = func() {
				disableKeyRequest := kms.DisableKeyRequest{KeyID: *(keyID)}

				disableKeyResponse := &kms.DisableKeyResponse{}
				err := Client.Do("POST", "/api/v1/go-kms/disablekey", &disableKeyRequest, disableKeyResponse)
				if err != nil {
					Exit(err.Error(), 1)
				}

				OutputMetadata([]kms.KeyMetadata{disableKeyResponse.KeyMetadata})

			}
		})
		commandCmd.Command("enable", "Enable a key", func(enableKeyCmd *cli.Cmd) {

			keyID := enableKeyCmd.StringArg("KEYID", "", "The KeyID of the key to be enabled")

			enableKeyCmd.Action = func() {
				enableKeyRequest := kms.EnableKeyRequest{KeyID: *(keyID)}

				enableKeyResponse := &kms.EnableKeyResponse{}
				err := Client.Do("POST", "/api/v1/go-kms/enablekey", &enableKeyRequest, enableKeyResponse)
				if err != nil {
					Exit(err.Error(), 1)
				}

				OutputMetadata([]kms.KeyMetadata{enableKeyResponse.KeyMetadata})
			}
		})

	})
}

// OutputMetadata prints a table of metadata to the console
func OutputMetadata(metadata []kms.KeyMetadata) {
	cliTable := clitable.New([]string{"KeyID", "Created", "Enabled", "Description"})

	for _, key := range metadata {
		cliTable.AddRow(map[string]interface{}{
			"KeyID":       key.KeyID,
			"Created":     key.CreationDate,
			"Enabled":     key.Enabled,
			"Description": key.Description})
	}

	cliTable.Print()
}
