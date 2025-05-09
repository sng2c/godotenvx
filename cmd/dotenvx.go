package main

import (
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"go-dotenvx"
	"log"
	"os"
	"syscall"
)

var args []string
var envFile string
var override bool
var verbose bool

func main() {
	kingpin.Version("0.1.0")
	kingpin.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Load environment variables from .env file"
	kingpin.Flag("file", "Path to .env file").Short('f').Default(".env").StringVar(&envFile)
	kingpin.Flag("override", "Override environment variables").Short('o').BoolVar(&override)
	kingpin.Flag("verbose", "Verbose mode").Short('v').BoolVar(&verbose)
	kingpin.Arg("args", "Arguments to be passed to the command").StringsVar(&args)
	kingpin.Parse()

	log.Printf("Loading environment variables from %s", envFile)
	log.Printf("Override: %v", override)
	log.Printf("Verbose: %v", verbose)
	log.Printf("Args: %v", args)

	envMap, err := godotenvx.LoadEnvFile(envFile, override, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	} else {
		return
	}
	cmdargs := []string{}
	if len(args) > 1 {
		cmdargs = args[1:]
	}
	err = syscall.Exec(cmd, cmdargs, envMap.GetEnviron())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
