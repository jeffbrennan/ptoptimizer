package cmd

import (
	"fmt"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage accrual settings",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current accrual settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		a := cfg.Accrual
		fmt.Println("Accrual Settings:")
		fmt.Printf("  Balance:        %.2f hours\n", a.CurrentBalanceHours)
		fmt.Printf("  As of:          %s\n", a.BalanceAsOfDate)
		fmt.Printf("  Accrual rate:   %.2f hours per pay period\n", a.AccrualRateHours)
		fmt.Printf("  Pay period:     %d days\n", a.PayPeriodDays)
		if a.MaxBalanceHours > 0 {
			fmt.Printf("  Max balance:    %.2f hours\n", a.MaxBalanceHours)
		} else {
			fmt.Println("  Max balance:    unlimited")
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set accrual configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if cmd.Flags().Changed("balance") {
			v, _ := cmd.Flags().GetFloat64("balance")
			cfg.Accrual.CurrentBalanceHours = v
		}
		if cmd.Flags().Changed("rate") {
			v, _ := cmd.Flags().GetFloat64("rate")
			cfg.Accrual.AccrualRateHours = v
		}
		if cmd.Flags().Changed("period") {
			v, _ := cmd.Flags().GetInt("period")
			cfg.Accrual.PayPeriodDays = v
		}
		if cmd.Flags().Changed("max") {
			v, _ := cmd.Flags().GetFloat64("max")
			cfg.Accrual.MaxBalanceHours = v
		}
		if cmd.Flags().Changed("as-of") {
			v, _ := cmd.Flags().GetString("as-of")
			cfg.Accrual.BalanceAsOfDate = v
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("Configuration saved.")
		return nil
	},
}

func init() {
	configSetCmd.Flags().Float64("balance", 0, "Current PTO balance in hours")
	configSetCmd.Flags().Float64("rate", 0, "Accrual rate in hours per pay period")
	configSetCmd.Flags().Int("period", 14, "Pay period length in days")
	configSetCmd.Flags().Float64("max", 0, "Maximum balance cap in hours (0 = unlimited)")
	configSetCmd.Flags().String("as-of", "", "Date the balance is as of (YYYY-MM-DD)")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
