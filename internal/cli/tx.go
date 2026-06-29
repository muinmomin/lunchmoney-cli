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
	txCmd.AddCommand(newTxSplitCmd())

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
				StartDate: startDate,
				EndDate:   endDate,
				Limit:     1000,
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

func newTxSplitCmd() *cobra.Command {
	var (
		amounts    []string
		parts      int
		dryRun     bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "split <tx-id>",
		Short: "Split a transaction into child transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			txID, err := parseTxID(args[0])
			if err != nil {
				return err
			}
			if parts > 0 && len(amounts) > 0 {
				return errors.New("use either --parts or repeated --amount values, not both")
			}
			if parts == 0 && len(amounts) == 0 {
				return errors.New("must provide --parts or at least two --amount values")
			}
			if parts < 0 {
				return errors.New("--parts must be at least 2")
			}
			if parts > 0 && parts < 2 {
				return errors.New("--parts must be at least 2")
			}
			if parts > 500 {
				return errors.New("--parts cannot exceed 500")
			}
			if len(amounts) == 1 {
				return errors.New("at least two --amount values are required")
			}
			if len(amounts) > 500 {
				return errors.New("cannot split into more than 500 child transactions")
			}

			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}

			parent, err := client.GetTransaction(context.Background(), txID)
			if err != nil {
				return err
			}
			if err := validateSplittableTransaction(parent); err != nil {
				return err
			}

			var splitAmounts []string
			if parts > 0 {
				splitAmounts, err = splitAmountIntoParts(parent.Amount, parts)
			} else {
				splitAmounts, err = validateSplitAmounts(parent.Amount, amounts)
			}
			if err != nil {
				return err
			}

			children := make([]lunchmoney.SplitTransactionChild, 0, len(splitAmounts))
			for _, amount := range splitAmounts {
				children = append(children, lunchmoney.SplitTransactionChild{Amount: amount})
			}

			plan := toSplitPlanView(txID, parent.Amount, splitAmounts, dryRun)
			if dryRun {
				if jsonOutput {
					return printJSON(plan)
				}
				fmt.Printf("Split transaction %d into %d child transaction(s) (dry run).\n", txID, len(splitAmounts))
				printSplitChildrenTable(plan.ChildTransactions)
				return nil
			}

			result, err := client.SplitTransaction(context.Background(), txID, children)
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(result)
			}

			fmt.Printf("Split transaction %d into %d child transaction(s).\n", txID, len(splitAmounts))
			resultChildren := toSplitResultChildViews(result.Children, splitAmounts)
			printSplitChildrenTable(resultChildren)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&amounts, "amount", nil, "Child amount; repeat for each split child")
	cmd.Flags().IntVar(&parts, "parts", 0, "Split into this many equal parts")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate and print the split without making the split API call")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")

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

func validateSplittableTransaction(tx lunchmoney.Transaction) error {
	if strings.TrimSpace(tx.Amount) == "" {
		return fmt.Errorf("transaction %d has no amount", tx.ID)
	}
	if tx.IsSplitParent || tx.SplitParentID != nil {
		return fmt.Errorf("transaction %d is already split", tx.ID)
	}
	if tx.IsGroupParent || tx.GroupParentID != nil {
		return fmt.Errorf("transaction %d is grouped", tx.ID)
	}
	return nil
}

func splitAmountIntoParts(parentAmount string, parts int) ([]string, error) {
	if parts < 2 {
		return nil, errors.New("--parts must be at least 2")
	}
	parentUnits, err := parseMoneyUnits(parentAmount)
	if err != nil {
		return nil, fmt.Errorf("invalid parent amount %q: %w", parentAmount, err)
	}
	if parentUnits == 0 {
		return nil, errors.New("cannot split a zero-amount transaction")
	}
	if parentUnits%centsScale != 0 {
		return nil, errors.New("--parts requires a parent amount with no fractional cents; use repeated --amount values instead")
	}

	totalCents := parentUnits / centsScale
	sign := int64(1)
	if totalCents < 0 {
		sign = -1
		totalCents = -totalCents
	}
	if totalCents < int64(parts) {
		return nil, fmt.Errorf("cannot split %s into %d non-zero cent amounts", formatMoneyUnits(parentUnits), parts)
	}

	base := totalCents / int64(parts)
	remainder := totalCents % int64(parts)
	amounts := make([]string, 0, parts)
	for i := 0; i < parts; i++ {
		cents := base
		if int64(i) < remainder {
			cents++
		}
		amounts = append(amounts, formatMoneyUnits(sign*cents*centsScale))
	}
	return amounts, nil
}

func validateSplitAmounts(parentAmount string, rawAmounts []string) ([]string, error) {
	if len(rawAmounts) < 2 {
		return nil, errors.New("at least two --amount values are required")
	}

	parentUnits, err := parseMoneyUnits(parentAmount)
	if err != nil {
		return nil, fmt.Errorf("invalid parent amount %q: %w", parentAmount, err)
	}

	amounts := make([]string, 0, len(rawAmounts))
	var sum int64
	for i, raw := range rawAmounts {
		units, err := parseMoneyUnits(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid --amount value %q: %w", raw, err)
		}
		if units == 0 {
			return nil, fmt.Errorf("--amount value %d cannot be zero", i+1)
		}
		sum += units
		amounts = append(amounts, formatMoneyUnits(units))
	}

	if sum != parentUnits {
		return nil, fmt.Errorf("split amounts sum to %s, but parent transaction amount is %s", formatMoneyUnits(sum), formatMoneyUnits(parentUnits))
	}
	return amounts, nil
}

const (
	moneyScale = int64(10000)
	centsScale = int64(100)
)

func parseMoneyUnits(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, errors.New("amount cannot be empty")
	}

	sign := int64(1)
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	}
	if strings.HasPrefix(s, "+") {
		return 0, errors.New("amount must not include a plus sign")
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" || !isDigits(whole) {
		return 0, errors.New("amount must include whole-number digits")
	}
	if hasFrac {
		if frac == "" {
			return 0, errors.New("amount must include fractional digits after decimal point")
		}
		if len(frac) > 4 {
			return 0, errors.New("amount cannot have more than 4 decimal places")
		}
		if !isDigits(frac) {
			return 0, errors.New("amount contains invalid characters")
		}
	} else {
		frac = ""
	}

	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid whole amount: %w", err)
	}

	frac = frac + strings.Repeat("0", 4-len(frac))
	fracUnits, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid fractional amount: %w", err)
	}

	return sign * ((wholeUnits * moneyScale) + fracUnits), nil
}

func formatMoneyUnits(units int64) string {
	sign := ""
	if units < 0 {
		sign = "-"
		units = -units
	}

	whole := units / moneyScale
	frac := units % moneyScale
	fracText := fmt.Sprintf("%04d", frac)
	for len(fracText) > 2 && strings.HasSuffix(fracText, "0") {
		fracText = strings.TrimSuffix(fracText, "0")
	}

	return fmt.Sprintf("%s%d.%s", sign, whole, fracText)
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func toSplitPlanView(txID int64, parentAmount string, amounts []string, dryRun bool) splitPlanView {
	children := make([]splitChildView, 0, len(amounts))
	for i, amount := range amounts {
		children = append(children, splitChildView{
			Index:  i + 1,
			Amount: amount,
		})
	}
	return splitPlanView{
		TransactionID:     txID,
		ParentAmount:      formatParentAmount(parentAmount),
		ChildTransactions: children,
		DryRun:            dryRun,
	}
}

func toSplitResultChildViews(children []lunchmoney.Transaction, fallbackAmounts []string) []splitChildView {
	if len(children) == 0 {
		views := make([]splitChildView, 0, len(fallbackAmounts))
		for i, amount := range fallbackAmounts {
			views = append(views, splitChildView{Index: i + 1, Amount: amount})
		}
		return views
	}

	views := make([]splitChildView, 0, len(children))
	for i, child := range children {
		views = append(views, splitChildView{
			Index:  i + 1,
			ID:     child.ID,
			Amount: formatParentAmount(child.Amount),
			Payee:  child.Payee,
		})
	}
	return views
}

func formatParentAmount(amount string) string {
	units, err := parseMoneyUnits(amount)
	if err != nil {
		return amount
	}
	return formatMoneyUnits(units)
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
