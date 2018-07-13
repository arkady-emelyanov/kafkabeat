package main

import (
	"os"

	"github.com/arkady-emelyanov/kafkabeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
