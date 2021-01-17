package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/itiky/collaborate-storage/storage"
)

const (
	FlagFilePath    = "file-path"
	FlagStorageSize = "storage-size"
)

// GetGenerateCmd returns generate mock data command.
func GetGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate mock data",
		Run: func(cmd *cobra.Command, args []string) {
			// Parse inputs
			filePath, err := cmd.Flags().GetString(FlagFilePath)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagFilePath, err)
			}
			storageSize, err := cmd.Flags().GetInt(FlagStorageSize)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagStorageSize, err)
			}

			// Work
			if err := storage.GenAndSaveInitialStorage(filePath, storageSize); err != nil {
				log.Fatalf("gen failed: %v", err)
			}
		},
	}
	cmd.Flags().String(FlagFilePath, "./doc_v0.json", "(optional) output file path")
	cmd.Flags().Int(FlagStorageSize, 10e6, "(optional) storage size")

	return cmd
}

func init() {
	rootCmd.AddCommand(GetGenerateCmd())
}
