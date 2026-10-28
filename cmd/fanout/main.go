package main

import (
	"os"

	"github.com/taylortyler-1/fanout/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
