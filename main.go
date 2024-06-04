package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

func run(ctx *cli.Context) error {
	supplyFilePath := ctx.String("supply.file")
	stateFilePath := ctx.String("state.file")

	// Clean up state file if fresh flag is set
	if ctx.Bool("fresh") {
		log.Println("Removing existing state file...")
		os.Remove(stateFilePath)
	}

	state := NewState()

	// Load state from file if it exists
	lastParsedFile, err := state.LoadState(stateFilePath)
	if err != nil {
		log.Println(err)
	}

	// Handle fatal errors from goroutines that will exit the program
	errCh := make(chan error, 16)
	defer close(errCh)

	linesCh, err := readFileStream(supplyFilePath, lastParsedFile, errCh)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for line := range linesCh {
			if supply, ok := line.(supplyInfo); ok {
				state.handleEntry(supply, errCh)
			} else if lastParsedFilename, ok := line.(SaveLastParsedFile); ok {
				state.SaveState(stateFilePath, string(lastParsedFilename))
			}
		}
	}()

	// Setup signal handling for a graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case err := <-errCh:
				log.Fatal("Exiting due to error: ", err)
				os.Exit(1)
			case sig := <-sigs:
				log.Printf("Received signal \"%v\", exiting...", sig)
				os.Exit(0)
			}
		}
	}()

	if err := startAPI(ctx.Int("api.port"), state); err == nil {
		return fmt.Errorf("failed to start the API: %s", err)
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:  "supply-tracer-parser",
		Usage: "Parse and sum supply data from a JSONL file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "supply.file",
				Value: "supply.jsonl",
				Usage: "File to read supply data from. Supports reading log rotated files.",
			},
			&cli.StringFlag{
				Name:  "state.file",
				Value: "state.json",
				Usage: "File to store latest state for subsequent runs",
			},
			&cli.IntFlag{
				Name:  "api.port",
				Usage: "API port to expose the latest state",
				Value: 8080,
			},
			&cli.BoolFlag{
				Name:  "fresh",
				Usage: "nuke the state and start fresh",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
