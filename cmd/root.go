package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vyuldashev/releaser/internal/config"
	"log"
)

var (
	cfgFile string
)

func Execute() {
	c := config.Load("config.yml")

	rootCmd := &cobra.Command{
		Use: "releaser",
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")

	rootCmd.AddCommand(
		NewRelease(c),
	)

	err := rootCmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
