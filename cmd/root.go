package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ptopt",
	Short: "Optimize your paid time off",
	Long:  "ptopt helps you maximize your PTO by finding the best days to take off around holidays and weekends.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
