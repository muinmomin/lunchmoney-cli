package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lunchmoney-cli/internal/lunchmoney"
)

type balanceAccountView struct {
	ID          string  `json:"id"`
	Source      string  `json:"source"`
	Name        string  `json:"name"`
	Institution string  `json:"institution"`
	Kind        string  `json:"kind"`
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	Status      string  `json:"status"`
	Balance     string  `json:"balance"`
	Currency    string  `json:"currency"`
	ToBase      float64 `json:"to_base"`
	BalanceAsOf string  `json:"balance_as_of"`
}

type balanceTotalsView struct {
	ActiveAccounts     int     `json:"active_accounts"`
	InactiveAccounts   int     `json:"inactive_accounts"`
	Cash               float64 `json:"cash"`
	CreditBalances     float64 `json:"credit_balances"`
	NetCashAfterCredit float64 `json:"net_cash_after_credit"`
}

type balancesView struct {
	GeneratedAt  string               `json:"generated_at"`
	BalancesAsOf string               `json:"balances_as_of"`
	Totals       balanceTotalsView    `json:"totals"`
	Cash         []balanceAccountView `json:"cash"`
	Credit       []balanceAccountView `json:"credit"`
	Inactive     []balanceAccountView `json:"inactive,omitempty"`
}

func newBalancesCmd() *cobra.Command {
	var (
		jsonOutput      bool
		includeInactive bool
	)

	cmd := &cobra.Command{
		Use:   "balances",
		Short: "Show account balances",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}

			manualAccounts, err := client.ListManualAccounts(context.Background())
			if err != nil {
				return err
			}

			plaidAccounts, err := client.ListPlaidAccounts(context.Background())
			if err != nil {
				return err
			}

			view := toBalancesView(manualAccounts, plaidAccounts, includeInactive)
			if jsonOutput {
				return printJSON(view)
			}

			printBalances(view, includeInactive)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&includeInactive, "include-inactive", false, "Include closed or inactive accounts")

	return cmd
}

func toBalancesView(
	manualAccounts []lunchmoney.ManualAccount,
	plaidAccounts []lunchmoney.PlaidAccount,
	includeInactive bool,
) balancesView {
	accounts := make([]balanceAccountView, 0, len(manualAccounts)+len(plaidAccounts))
	for _, account := range manualAccounts {
		accounts = append(accounts, manualBalanceAccountView(account))
	}
	for _, account := range plaidAccounts {
		accounts = append(accounts, plaidBalanceAccountView(account))
	}

	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Status == "active" && accounts[j].Status != "active" {
			return true
		}
		if accounts[i].Status != "active" && accounts[j].Status == "active" {
			return false
		}
		if isCreditBalance(accounts[i]) != isCreditBalance(accounts[j]) {
			return !isCreditBalance(accounts[i])
		}
		return accounts[i].Name < accounts[j].Name
	})

	view := balancesView{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, account := range accounts {
		if account.Status != "active" {
			view.Totals.InactiveAccounts++
			if includeInactive {
				view.Inactive = append(view.Inactive, account)
			}
			continue
		}

		view.Totals.ActiveAccounts++
		view.BalancesAsOf = newerTimestamp(view.BalancesAsOf, account.BalanceAsOf)
		if isCreditBalance(account) {
			view.Credit = append(view.Credit, account)
			view.Totals.CreditBalances += account.ToBase
			continue
		}

		view.Cash = append(view.Cash, account)
		view.Totals.Cash += account.ToBase
	}

	view.Totals.NetCashAfterCredit = view.Totals.Cash - view.Totals.CreditBalances
	return view
}

func manualBalanceAccountView(account lunchmoney.ManualAccount) balanceAccountView {
	subtype := stringOrDefault(account.Subtype, "")
	return balanceAccountView{
		ID:          fmt.Sprintf("manual:%d", account.ID),
		Source:      "manual",
		Name:        balanceAccountName(account.Name, account.DisplayName, stringOrDefault(account.InstitutionName, "")),
		Institution: stringOrDefault(account.InstitutionName, ""),
		Kind:        balanceAccountKind(account.Type, subtype),
		Type:        account.Type,
		Subtype:     subtype,
		Status:      account.Status,
		Balance:     account.Balance,
		Currency:    account.Currency,
		ToBase:      account.ToBase,
		BalanceAsOf: account.BalanceAsOf,
	}
}

func plaidBalanceAccountView(account lunchmoney.PlaidAccount) balanceAccountView {
	subtype := stringOrDefault(account.Subtype, "")
	return balanceAccountView{
		ID:          fmt.Sprintf("plaid:%d", account.ID),
		Source:      "plaid",
		Name:        balanceAccountName(account.Name, account.DisplayName, account.InstitutionName),
		Institution: account.InstitutionName,
		Kind:        balanceAccountKind(account.Type, subtype),
		Type:        account.Type,
		Subtype:     subtype,
		Status:      account.Status,
		Balance:     account.Balance,
		Currency:    account.Currency,
		ToBase:      account.ToBase,
		BalanceAsOf: account.BalanceAsOf,
	}
}

func balanceAccountName(name string, displayName *string, institutionName string) string {
	display := strings.TrimSpace(stringOrDefault(displayName, ""))
	if display != "" {
		return display
	}

	parts := []string{strings.TrimSpace(institutionName), strings.TrimSpace(name)}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func balanceAccountKind(accountType, subtype string) string {
	if subtype == "" {
		return accountType
	}
	if accountType == "" {
		return subtype
	}
	return accountType + "/" + subtype
}

func isCreditBalance(account balanceAccountView) bool {
	return account.Type == "credit" || strings.Contains(strings.ToLower(account.Kind), "credit")
}

func newerTimestamp(current, candidate string) string {
	if candidate == "" {
		return current
	}
	if current == "" || candidate > current {
		return candidate
	}
	return current
}
