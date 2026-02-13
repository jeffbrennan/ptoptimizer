package cmd

import (
	"fmt"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/jeffb/ptoptimizer/internal/display"
	"github.com/jeffb/ptoptimizer/internal/engine"
	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest optimal PTO days",
	RunE: func(cmd *cobra.Command, args []string) error {
		year, _ := cmd.Flags().GetInt("year")
		days, _ := cmd.Flags().GetInt("days")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.Holidays) == 0 {
			fmt.Println("No holidays configured. Run 'ptopt holidays reset' first.")
			return nil
		}

		suggestions := engine.SuggestDays(cfg, year, days)

		// Show balance bar
		balance := engine.ProjectBalance(cfg, time.Date(year, 12, 31, 0, 0, 0, 0, time.Local))
		fmt.Println()
		fmt.Println(display.RenderBalanceBar(balance, cfg.Accrual.MaxBalanceHours))
		fmt.Println()

		// Build calendar data
		holidaySet := make(map[string]bool)
		for _, h := range cfg.Holidays {
			holidaySet[h.Date] = true
		}

		ptoSet := make(map[string]bool)
		for _, p := range cfg.PlannedTimeOff {
			start, err1 := time.Parse("2006-01-02", p.StartDate)
			end, err2 := time.Parse("2006-01-02", p.EndDate)
			if err1 != nil || err2 != nil {
				continue
			}
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				ptoSet[d.Format("2006-01-02")] = true
			}
		}

		suggestedSet := engine.GetSuggestedDates(suggestions)

		data := display.CalendarData{
			Holidays:  holidaySet,
			PTO:       ptoSet,
			Suggested: suggestedSet,
		}

		fmt.Println(display.RenderYearCalendar(year, data))
		fmt.Println(display.RenderLegend())
		fmt.Println()

		// Show suggestions list
		fmt.Printf("  Top %d suggested PTO days for %d:\n\n", len(suggestions), year)
		for i, s := range suggestions {
			fmt.Printf("  %d. %s — %s\n", i+1, s.Date.Format("Mon Jan 2"), s.Explanation)
		}

		totalHours := float64(len(suggestions)) * 8
		fmt.Printf("\n  Total: %d days (%.0f hours)\n\n", len(suggestions), totalHours)

		return nil
	},
}

func init() {
	suggestCmd.Flags().Int("year", time.Now().Year(), "Year to suggest PTO for")
	suggestCmd.Flags().Int("days", 5, "Number of days to suggest")
	rootCmd.AddCommand(suggestCmd)
}
