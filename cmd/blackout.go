package cmd

import (
	"fmt"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/spf13/cobra"
)

var blackoutCmd = &cobra.Command{
	Use:   "blackout",
	Short: "Manage blackout dates (days you can't take off)",
}

var blackoutListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all blackout dates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.BlackoutDates) == 0 {
			fmt.Println("No blackout dates. Use 'ptopt blackout add' to add some.")
			return nil
		}

		fmt.Println("Blackout Dates:")
		for _, b := range cfg.BlackoutDates {
			if b.Reason != "" {
				fmt.Printf("  %s to %s  %s\n", b.StartDate, b.EndDate, b.Reason)
			} else {
				fmt.Printf("  %s to %s\n", b.StartDate, b.EndDate)
			}
		}
		return nil
	},
}

var blackoutAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a blackout date range",
	RunE: func(cmd *cobra.Command, args []string) error {
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		reason, _ := cmd.Flags().GetString("reason")

		if startStr == "" {
			return fmt.Errorf("--start is required")
		}

		if _, err := time.Parse("2006-01-02", startStr); err != nil {
			return fmt.Errorf("invalid start date, use YYYY-MM-DD")
		}

		if endStr == "" {
			endStr = startStr
		}

		if _, err := time.Parse("2006-01-02", endStr); err != nil {
			return fmt.Errorf("invalid end date, use YYYY-MM-DD")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cfg.BlackoutDates = append(cfg.BlackoutDates, config.BlackoutDate{
			StartDate: startStr,
			EndDate:   endStr,
			Reason:    reason,
		})

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		if reason != "" {
			fmt.Printf("Added blackout: %s to %s (%s)\n", startStr, endStr, reason)
		} else {
			fmt.Printf("Added blackout: %s to %s\n", startStr, endStr)
		}
		return nil
	},
}

var blackoutRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a blackout date range by start date",
	RunE: func(cmd *cobra.Command, args []string) error {
		startStr, _ := cmd.Flags().GetString("start")
		if startStr == "" {
			return fmt.Errorf("--start is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		filtered := cfg.BlackoutDates[:0]
		found := false
		for _, b := range cfg.BlackoutDates {
			if b.StartDate == startStr {
				found = true
				fmt.Printf("Removed blackout: %s to %s\n", b.StartDate, b.EndDate)
			} else {
				filtered = append(filtered, b)
			}
		}
		cfg.BlackoutDates = filtered

		if !found {
			return fmt.Errorf("no blackout found starting on %s", startStr)
		}

		return config.Save(cfg)
	},
}

func init() {
	blackoutAddCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	blackoutAddCmd.Flags().String("end", "", "End date (YYYY-MM-DD, defaults to start)")
	blackoutAddCmd.Flags().String("reason", "", "Reason for blackout (e.g. \"Q4 freeze\")")

	blackoutRemoveCmd.Flags().String("start", "", "Start date of blackout to remove")

	blackoutCmd.AddCommand(blackoutListCmd)
	blackoutCmd.AddCommand(blackoutAddCmd)
	blackoutCmd.AddCommand(blackoutRemoveCmd)
	rootCmd.AddCommand(blackoutCmd)
}
