package main

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/itiky/collaborate-storage/service/server"
)

const (
	FlagPort         = "port"
	FlagBatchChSize  = "batch-ch-size"
	FlagHandlePeriod = "handle-period"
)

// GetServerCmd returns RPC-server start command.
func GetServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start RPC server",
		Run: func(cmd *cobra.Command, args []string) {
			// Parse inputs
			port, err := cmd.Flags().GetInt(FlagPort)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagPort, err)
			}
			chSize, err := cmd.Flags().GetInt(FlagBatchChSize)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagBatchChSize, err)
			}
			handleDur, err := cmd.Flags().GetDuration(FlagHandlePeriod)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagHandlePeriod, err)
			}
			filePath, err := cmd.Flags().GetString(FlagFilePath)
			if err != nil {
				log.Fatalf("%s flag: %v", FlagFilePath, err)
			}

			// Init service
			svc, err := server.NewSortedListService(chSize, handleDur, filePath)
			if err != nil {
				log.Fatalf("service init: %v", err)
			}

			// Start server
			if err := rpc.Register(svc); err != nil {
				log.Fatalf("RPC server: register: %v", err)
			}
			svc.Start()

			listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
			if err != nil {
				log.Fatalf("RPC server: listen: %v", err)
			}
			defer listener.Close()

			go rpc.Accept(listener)

			log.Printf("RPC server started: :%d", port)

			// Wait for signal
			signalCh := make(chan os.Signal, 1)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			<-signalCh

			svc.Stop()
		},
	}
	cmd.Flags().Int(FlagPort, 2412, "(optional) server port")
	cmd.Flags().Int(FlagBatchChSize, 50, "(optional) input operation channel limit")
	cmd.Flags().Duration(FlagHandlePeriod, 500*time.Millisecond, "(optional) input operations handling period")
	cmd.Flags().String(FlagFilePath, "./resources/doc_v0_10M.dat", "(optional) path to generated storage file")

	return cmd
}

func init() {
	rootCmd.AddCommand(GetServerCmd())
}
