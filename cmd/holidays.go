package cmd

import (
	"fmt"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/jeffb/ptoptimizer/internal/holidays"
	"github.com/spf13/cobra"
)

var holidaysCmd = &cobra.Command{
	Use:   "holidays",
	Short: "Manage holidays",
}

var holidaysListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configured holidays",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.Holidays) == 0 {
			fmt.Println("No holidays configured. Run 'ptopt holidays reset' to load US federal holidays.")
			return nil
		}

		fmt.Println("Holidays:")
		for _, h := range cfg.Holidays {
			d, err := time.Parse("2006-01-02", h.Date)
			dayName := ""
			if err == nil {
				dayName = d.Format(" (Mon)")
			}
			fmt.Printf("  %s  %s%s\n", h.Date, h.Name, dayName)
		}
		return nil
	},
}

var holidaysAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a holiday",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		date, _ := cmd.Flags().GetString("date")

		if name == "" || date == "" {
			return fmt.Errorf("--name and --date are required")
		}

		if _, err := time.Parse("2006-01-02", date); err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cfg.Holidays = append(cfg.Holidays, config.Holiday{Name: name, Date: date})

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Added holiday: %s on %s\n", name, date)
		return nil
	},
}

var holidaysRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a holiday by date",
	RunE: func(cmd *cobra.Command, args []string) error {
		date, _ := cmd.Flags().GetString("date")
		if date == "" {
			return fmt.Errorf("--date is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		filtered := cfg.Holidays[:0]
		found := false
		for _, h := range cfg.Holidays {
			if h.Date == date {
				found = true
				fmt.Printf("Removed: %s (%s)\n", h.Name, h.Date)
			} else {
				filtered = append(filtered, h)
			}
		}
		cfg.Holidays = filtered

		if !found {
			return fmt.Errorf("no holiday found on %s", date)
		}

		return config.Save(cfg)
	},
}

var holidaysResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset to US federal holiday defaults for the current year",
	RunE: func(cmd *cobra.Command, args []string) error {
		year := time.Now().Year()
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cfg.Holidays = holidays.USFederalHolidays(year)

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Holidays reset to US federal holidays for %d.\n", year)
		return nil
	},
}

func init() {
	holidaysAddCmd.Flags().String("name", "", "Holiday name")
	holidaysAddCmd.Flags().String("date", "", "Holiday date (YYYY-MM-DD)")

	holidaysRemoveCmd.Flags().String("date", "", "Holiday date to remove (YYYY-MM-DD)")

	holidaysCmd.AddCommand(holidaysListCmd)
	holidaysCmd.AddCommand(holidaysAddCmd)
	holidaysCmd.AddCommand(holidaysRemoveCmd)
	holidaysCmd.AddCommand(holidaysResetCmd)
	rootCmd.AddCommand(holidaysCmd)
}
