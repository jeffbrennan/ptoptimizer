package cmd

import (
	"fmt"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/jeffb/ptoptimizer/internal/display"
	"github.com/jeffb/ptoptimizer/internal/engine"
	"github.com/spf13/cobra"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Project PTO balance on a date",
	RunE: func(cmd *cobra.Command, args []string) error {
		dateStr, _ := cmd.Flags().GetString("date")

		var targetDate time.Time
		if dateStr == "" {
			targetDate = time.Now()
		} else {
			var err error
			targetDate, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD")
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		balance := engine.ProjectBalance(cfg, targetDate)

		fmt.Println()
		fmt.Println(display.RenderBalanceBar(balance, cfg.Accrual.MaxBalanceHours))
		fmt.Println()
		fmt.Printf("  Projected balance on %s: %.1f hours (%.1f days)\n",
			targetDate.Format("Jan 2, 2006"), balance, balance/8)
		fmt.Println()

		// Render calendar for the target year
		year := targetDate.Year()
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

		data := display.CalendarData{
			Holidays: holidaySet,
			PTO:      ptoSet,
		}

		fmt.Println(display.RenderYearCalendar(year, data))

		return nil
	},
}

func init() {
	balanceCmd.Flags().String("date", "", "Target date (YYYY-MM-DD, default: today)")
	rootCmd.AddCommand(balanceCmd)
}
