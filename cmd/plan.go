package cmd

import (
	"fmt"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage planned PTO",
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all planned PTO",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.PlannedTimeOff) == 0 {
			fmt.Println("No planned PTO. Use 'ptopt plan add' to add some.")
			return nil
		}

		fmt.Println("Planned PTO:")
		for _, p := range cfg.PlannedTimeOff {
			hours := p.Hours
			if hours == 0 {
				start, err1 := time.Parse("2006-01-02", p.StartDate)
				end, err2 := time.Parse("2006-01-02", p.EndDate)
				if err1 == nil && err2 == nil {
					weekdays := 0
					for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
						if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
							weekdays++
						}
					}
					hours = float64(weekdays) * 8
				}
			}
			fmt.Printf("  %s to %s  (%.0f hours)\n", p.StartDate, p.EndDate, hours)
		}
		return nil
	},
}

var planAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add planned PTO",
	RunE: func(cmd *cobra.Command, args []string) error {
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		hours, _ := cmd.Flags().GetFloat64("hours")

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

		cfg.PlannedTimeOff = append(cfg.PlannedTimeOff, config.PlannedPTO{
			StartDate: startStr,
			EndDate:   endStr,
			Hours:     hours,
		})

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Added PTO: %s to %s\n", startStr, endStr)
		return nil
	},
}

var planRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove planned PTO by start date",
	RunE: func(cmd *cobra.Command, args []string) error {
		startStr, _ := cmd.Flags().GetString("start")
		if startStr == "" {
			return fmt.Errorf("--start is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		filtered := cfg.PlannedTimeOff[:0]
		found := false
		for _, p := range cfg.PlannedTimeOff {
			if p.StartDate == startStr {
				found = true
				fmt.Printf("Removed PTO: %s to %s\n", p.StartDate, p.EndDate)
			} else {
				filtered = append(filtered, p)
			}
		}
		cfg.PlannedTimeOff = filtered

		if !found {
			return fmt.Errorf("no PTO found starting on %s", startStr)
		}

		return config.Save(cfg)
	},
}

func init() {
	planAddCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	planAddCmd.Flags().String("end", "", "End date (YYYY-MM-DD, defaults to start)")
	planAddCmd.Flags().Float64("hours", 0, "Hours to use (default: 8 per weekday)")

	planRemoveCmd.Flags().String("start", "", "Start date of PTO to remove")

	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planAddCmd)
	planCmd.AddCommand(planRemoveCmd)
	rootCmd.AddCommand(planCmd)
}
