package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"cloud.google.com/go/pubsub"
	"darlinggo.co/version"
	"github.com/mitchellh/cli"
	"google.golang.org/api/option"
	"tangl.es/code/blobs"
	"tangl.es/code/images"
	yall "yall.in"
	"yall.in/colour"
)

func main() {
	c := cli.NewCLI("tanglesd", fmt.Sprintf("%s (%s)", version.Tag, version.Hash))
	c.Args = os.Args[1:]
	ui := &cli.ColoredUi{
		InfoColor:  cli.UiColorCyan,
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}
	log := yall.New(colour.New(os.Stderr, yall.Debug))
	blobStorer := blobs.Filestore{Root: "/usr/local/var/www/static.carvers.house/tangles-process-test"}
	sqip := images.SQIP{
		WorkSize:   256,
		Count:      8,
		Mode:       1,
		Alpha:      128,
		NumWorkers: runtime.NumCPU(),
	}
	pubClient, err := pubsub.NewClient(context.Background(), "tangles-testing", option.WithCredentialsFile(os.ExpandEnv("$HOME/tangles-testing.json")))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	gcp := images.NewGCPPubSub(pubClient, "paddy-test", "paddy-test-sub")
	c.Commands = map[string]cli.CommandFactory{
		// start image processing
		"images process": imagesProcessCommandFactory(ui, sqip, gcp, blobStorer, log),

		// start api server
		//"api serve": apiServeCommandFactory(ui),

		// start html server
		//"html serve": htmlServeCommandFactory(ui),

		// check for updates
		"version": versionCommandFactory(ui),
	}

	c.HiddenCommands = []string{}

	c.Autocomplete = true

	status, err := c.Run()
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(status)
}
