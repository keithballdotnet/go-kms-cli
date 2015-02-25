package main

import (
	"log"
	"os"

	gokmscli "github.com/Inflatablewoman/go-kms-cli/cli"
	"github.com/jawher/mow.cli"
)

// main will start up the application
func main() {

	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetPrefix("GO-KMS-CLI:")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Attempt to create the json client
	gokmscli.CreateClient()

	commandLineApp := cli.App("go-kms-cli", "Command line interface for GO-KMS")
	gokmscli.CreateCommands(commandLineApp)
	commandLineApp.Run(os.Args)
}
