package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lunchmoney-cli/internal/lunchmoney"
)

type categoryMeta struct {
	Name              string
	Group             string
	IsIncome          bool
	ExcludeFromTotals bool
}

type accountMeta struct {
	DisplayName string
	Institution string
}

func newTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:   "tx",
		Short: "Transaction operations",
	}

	txCmd.AddCommand(newTxListCmd())
	txCmd.AddCommand(newTxUpdateCmd())
	txCmd.AddCommand(newTxMarkReviewedCmd())

	return txCmd
}

func newTxListCmd() *cobra.Command {
	var (
		startDate      string
		endDate        string
		unreviewed     bool
		includePending bool
		jsonOutput     bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if endDate == "" {
				endDate = time.Now().Format("2006-01-02")
			}
			if err := validateDateRange(startDate, endDate); err != nil {
				return err
			}

			status := "reviewed"
			if unreviewed {
				status = "unreviewed"
			}
			if includePending && !unreviewed {
				return errors.New("--include-pending requires --unreviewed (pending transactions are always unreviewed)")
			}

			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}

			params := lunchmoney.ListTransactionsParams{
				StartDate:      startDate,
				EndDate:        endDate,
				Limit:          1000,
			}
			if includePending {
				pendingOnly := true
				params.IsPending = &pendingOnly
			} else {
				params.Status = status
			}

			transactions, err := client.ListTransactions(context.Background(), params)
			if err != nil {
				return err
			}

			categories, err := client.ListCategories(context.Background())
			if err != nil {
				return err
			}
			categoryLookup := buildCategoryLookup(categories)

			tags, err := client.ListTags(context.Background())
			if err != nil {
				return err
			}
			tagLookup := make(map[int64]string, len(tags))
			for _, t := range tags {
				tagLookup[t.ID] = t.Name
			}

			manualAccounts, err := client.ListManualAccounts(context.Background())
			if err != nil {
				return err
			}
			manualLookup := buildManualAccountLookup(manualAccounts)

			plaidAccounts, err := client.ListPlaidAccounts(context.Background())
			if err != nil {
				return err
			}
			plaidLookup := buildPlaidAccountLookup(plaidAccounts)

			views := make([]transactionView, 0, len(transactions))
			for _, tx := range transactions {
				if !unreviewed && shouldExcludeFromTotalsFilter(tx, categoryLookup) {
					continue
				}
				views = append(views, toTransactionView(tx, categoryLookup, tagLookup, manualLookup, plaidLookup))
			}

			sortTransactionsNewestFirst(views)

			if jsonOutput {
				return printJSON(views)
			}

			printTransactionsTable(views)
			return nil
		},
	}

	cmd.Flags().StringVar(&startDate, "start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end", "", "End date (YYYY-MM-DD), defaults to today")
	cmd.Flags().BoolVar(&unreviewed, "unreviewed", false, "List unreviewed transactions (default is reviewed)")
	cmd.Flags().BoolVar(&includePending, "include-pending", false, "List pending transactions only (requires --unreviewed)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	_ = cmd.MarkFlagRequired("start")

	return cmd
}

func newTxUpdateCmd() *cobra.Command {
	var (
		categoryID int64
		note       string
	)

	cmd := &cobra.Command{
		Use:   "update <tx-id>",
		Short: "Update a transaction category and/or note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			txID, err := parseTxID(args[0])
			if err != nil {
				return err
			}

			categorySet := cmd.Flags().Changed("category-id")
			noteSet := cmd.Flags().Changed("note")
			if !categorySet && !noteSet {
				return errors.New("must provide at least one of --category-id or --note")
			}
			if categorySet && categoryID <= 0 {
				return errors.New("--category-id must be a positive integer")
			}
			if noteSet && strings.TrimSpace(note) == "" {
				return errors.New("--note cannot be empty")
			}

			var categoryPtr *int64
			if categorySet {
				categoryPtr = &categoryID
			}
			var notePtr *string
			if noteSet {
				noteValue := note
				notePtr = &noteValue
			}

			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}
			if _, err := client.UpdateTransaction(context.Background(), txID, categoryPtr, notePtr); err != nil {
				return err
			}

			updatedFields := make([]string, 0, 2)
			if categorySet {
				updatedFields = append(updatedFields, "category")
			}
			if noteSet {
				updatedFields = append(updatedFields, "note")
			}
			fmt.Printf("Updated transaction %d (%s).\n", txID, strings.Join(updatedFields, ", "))
			return nil
		},
	}

	cmd.Flags().Int64Var(&categoryID, "category-id", 0, "Category ID")
	cmd.Flags().StringVar(&note, "note", "", "Transaction note")

	return cmd
}

func newTxMarkReviewedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-reviewed <tx-id> [<tx-id>...]",
		Short: "Mark one or more transactions as reviewed",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]int64, 0, len(args))
			for _, raw := range args {
				id, err := parseTxID(raw)
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}

			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}
			updated, err := client.MarkReviewed(context.Background(), ids)
			if err != nil {
				return err
			}

			fmt.Printf("Marked %d transaction(s) as reviewed.\n", len(updated))
			return nil
		},
	}

	return cmd
}

func validateDateRange(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid --start date %q (expected YYYY-MM-DD)", startDate)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid --end date %q (expected YYYY-MM-DD)", endDate)
	}
	if end.Before(start) {
		return errors.New("--end cannot be earlier than --start")
	}
	return nil
}

func parseTxID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid transaction id %q", raw)
	}
	return id, nil
}

func buildCategoryLookup(categories []lunchmoney.Category) map[int64]categoryMeta {
	groups := make(map[int64]string, len(categories))
	for _, c := range categories {
		if c.IsGroup {
			groups[c.ID] = c.Name
		}
	}

	lookup := make(map[int64]categoryMeta, len(categories))
	for _, c := range categories {
		groupName := ""
		if c.GroupID != nil {
			groupName = groups[*c.GroupID]
		}

		lookup[c.ID] = categoryMeta{
			Name:              c.Name,
			Group:             groupName,
			IsIncome:          c.IsIncome,
			ExcludeFromTotals: c.ExcludeFromTotals,
		}
	}

	return lookup
}

func buildManualAccountLookup(accounts []lunchmoney.ManualAccount) map[int64]accountMeta {
	lookup := make(map[int64]accountMeta, len(accounts))
	for _, a := range accounts {
		display := strings.TrimSpace(stringOrDefault(a.DisplayName, ""))
		if display == "" {
			display = a.Name
		}
		inst := strings.TrimSpace(stringOrDefault(a.InstitutionName, ""))
		lookup[a.ID] = accountMeta{DisplayName: display, Institution: inst}
	}
	return lookup
}

func buildPlaidAccountLookup(accounts []lunchmoney.PlaidAccount) map[int64]accountMeta {
	lookup := make(map[int64]accountMeta, len(accounts))
	for _, a := range accounts {
		display := strings.TrimSpace(stringOrDefault(a.DisplayName, ""))
		if display == "" {
			display = strings.TrimSpace(strings.Join([]string{a.InstitutionName, a.Name}, " "))
		}
		lookup[a.ID] = accountMeta{DisplayName: display, Institution: a.InstitutionName}
	}
	return lookup
}

func shouldExcludeFromTotalsFilter(tx lunchmoney.Transaction, categories map[int64]categoryMeta) bool {
	if tx.CategoryID == nil {
		return false
	}
	c, ok := categories[*tx.CategoryID]
	if !ok {
		return false
	}
	return c.ExcludeFromTotals
}

func toTransactionView(
	tx lunchmoney.Transaction,
	categories map[int64]categoryMeta,
	tags map[int64]string,
	manual map[int64]accountMeta,
	plaid map[int64]accountMeta,
) transactionView {
	normalizedAmount := -tx.ToBase

	categoryName := ""
	categoryGroup := ""
	txType := "expense"
	if normalizedAmount > 0 {
		txType = "income"
	}

	if tx.CategoryID != nil {
		if c, ok := categories[*tx.CategoryID]; ok {
			categoryName = c.Name
			categoryGroup = c.Group
			if c.IsIncome {
				txType = "income"
			} else if c.Name == "Payment, Transfer" {
				txType = "transfer"
			} else {
				txType = "expense"
			}
		}
	}

	notes := ""
	if tx.Notes != nil {
		notes = *tx.Notes
	}

	tagNames := make([]string, 0, len(tx.TagIDs))
	for _, tagID := range tx.TagIDs {
		if name, ok := tags[tagID]; ok {
			tagNames = append(tagNames, name)
		}
	}
	sort.Strings(tagNames)

	account := "Cash Transaction"
	institution := ""
	if tx.ManualAccountID != nil {
		if info, ok := manual[*tx.ManualAccountID]; ok {
			account = info.DisplayName
			institution = info.Institution
		} else {
			account = fmt.Sprintf("manual:%d", *tx.ManualAccountID)
		}
	} else if tx.PlaidAccountID != nil {
		if info, ok := plaid[*tx.PlaidAccountID]; ok {
			account = info.DisplayName
			institution = info.Institution
		} else {
			account = fmt.Sprintf("plaid:%d", *tx.PlaidAccountID)
		}
	}

	return transactionView{
		ID:          tx.ID,
		Date:        tx.Date,
		Description: tx.Payee,
		Category:    categoryName,
		Amount:      normalizedAmount,
		Account:     account,
		Institution: institution,
		Group:       categoryGroup,
		Type:        txType,
		Notes:       notes,
		Tags:        strings.Join(tagNames, ", "),
		Status:      tx.Status,
		IsPending:   tx.IsPending,
	}
}

func sortTransactionsNewestFirst(transactions []transactionView) {
	sort.SliceStable(transactions, func(i, j int) bool {
		if transactions[i].Date != transactions[j].Date {
			// Date format is YYYY-MM-DD, so lexical comparison is chronological.
			return transactions[i].Date > transactions[j].Date
		}
		return transactions[i].ID > transactions[j].ID
	})
}

func stringOrDefault(v *string, fallback string) string {
	if v == nil {
		return fallback
	}
	return *v
}
