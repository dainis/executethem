package cmd

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/dainis/executethem/execute"
	"github.com/spf13/cobra"
	"os"
)

var (
	cfgFile   string
	RootCmd   *cobra.Command
	GitHash   string
	BuildDate string
	Version   string
)

func init() {
	RootCmd = &cobra.Command{
		Use:   "executethem [flags] <folder>",
		Short: "Execute executables located in folder",
		Long: fmt.Sprintf("Execute executables in parallel that all are located in a folder and keeps them alive\n"+
			"Version: %s\nBuilt: %s\nGit Hash : %s",
			Version, BuildDate, GitHash),

		Run: func(cmd *cobra.Command, args []string) {
			timeout, _ := cmd.Flags().GetInt("timeout")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if verbose {
				log.SetLevel(log.DebugLevel)
			}

			if len(args) != 1 {
				log.Error("No folder with executables specified")
				os.Exit(-1)
			}

			folder := args[0]

			e, err := execute.New(timeout, folder)

			if err != nil {
				log.WithError(err).Fatal("Failed to read executable list")
			}

			log.WithField("Executables", e.GetExecutableList()).Debug("Will execute these files")

			e.ExecuteExecutables()
		},
	}
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.WithError(err).Print("Failed to execute command")
		os.Exit(-1)
	}
}

func init() {
	RootCmd.Flags().IntP("timeout", "t", 1000, "Miliseconds between restarts if one of the tasks exits")
	RootCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}
