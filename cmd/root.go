package main

import (
	"log"

	"github.com/spf13/cobra"
)

// rootCmd is a base command.
var rootCmd = &cobra.Command{
	Use:   "collaborate-storage",
	Short: "Collaborate storage client/server",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("rootCmd.Execute: %v", err)
	}
}
