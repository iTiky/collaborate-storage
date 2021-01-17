package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/itiky/collaborate-storage/model"
	"github.com/itiky/collaborate-storage/service/client"
)

const (
	FlagServerUrl     = "server-url"
	FlagClientId      = "client-id"
	FlagOpsSendPeriod = "updates-period"
	FlagOpsSendMax    = "updates-max"
	FlagPollPeriod    = "poll-period"
)

// GetClientCmd returns RPC-client start command.
func GetClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Start RPC client",
		Run: func(cmd *cobra.Command, args []string) {
			// Parse inputs
			serverUrl, err := cmd.Flags().GetString(FlagServerUrl)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagServerUrl, err)
			}
			clientId, err := cmd.Flags().GetUint(FlagClientId)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagClientId, err)
			}
			opsSendMax, err := cmd.Flags().GetInt(FlagOpsSendMax)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagOpsSendMax, err)
			}
			opsSendDur, err := cmd.Flags().GetDuration(FlagOpsSendPeriod)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagOpsSendPeriod, err)
			}
			pollDur, err := cmd.Flags().GetDuration(FlagPollPeriod)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagPollPeriod, err)
			}

			if clientId == 0 {
				clientId = uint(rand.Uint32())
			}

			// Init service
			svc, err := client.NewClient(
				model.ClientId(clientId),
				opsSendDur,
				pollDur,
				opsSendMax,
				serverUrl,
			)
			if err != nil {
				log.Fatalf("service init: %v", err)
			}

			svc.Start()

			// Wait for signal
			signalCh := make(chan os.Signal, 1)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			<-signalCh

			svc.Stop()
		},
	}
	cmd.Flags().Uint(FlagClientId, 1, "unique clientID")
	cmd.Flags().Int(FlagOpsSendMax, 5, "max number of snapshot updates per period")
	cmd.Flags().String(FlagServerUrl, "127.0.0.1:2412", "(optional) server url")
	cmd.Flags().Duration(FlagOpsSendPeriod, 1*time.Second, "(optional) snapshot updates send period")
	cmd.Flags().Duration(FlagPollPeriod, 2*time.Second, "(optional) snapshot updates poll period")

	return cmd
}

func init() {
	rootCmd.AddCommand(GetClientCmd())
}
