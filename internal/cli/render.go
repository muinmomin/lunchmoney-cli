package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

type transactionView struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Account     string  `json:"account"`
	Institution string  `json:"institution"`
	Group       string  `json:"group"`
	Type        string  `json:"type"`
	Notes       string  `json:"notes"`
	Tags        string  `json:"tags"`
	Status      string  `json:"status"`
	IsPending   bool    `json:"is_pending"`
}

type categoryView struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Group             string `json:"group"`
	IsIncome          bool   `json:"is_income"`
	ExcludeFromTotals bool   `json:"exclude_from_totals"`
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printTransactionsTable(transactions []transactionView) {
	if len(transactions) == 0 {
		fmt.Println("No transactions found.")
		return
	}

	w := newTabWriter(os.Stdout)
	fmt.Fprintln(w, "DATE\tID\tDESCRIPTION\tCATEGORY\tNOTE\tAMOUNT\tACCOUNT\tSTATUS\tPENDING")
	for _, tx := range transactions {
		fmt.Fprintf(
			w,
			"%s\t%d\t%s\t%s\t%s\t%.2f\t%s\t%s\t%t\n",
			tx.Date,
			tx.ID,
			tx.Description,
			tx.Category,
			tx.Notes,
			tx.Amount,
			tx.Account,
			tx.Status,
			tx.IsPending,
		)
	}
	_ = w.Flush()
}

func printCategoriesTable(categories []categoryView) {
	w := newTabWriter(os.Stdout)
	fmt.Fprintln(w, "ID\tNAME\tGROUP\tINCOME\tEXCLUDE_FROM_TOTALS")
	for _, c := range categories {
		fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%t\t%t\n",
			c.ID,
			c.Name,
			c.Group,
			c.IsIncome,
			c.ExcludeFromTotals,
		)
	}
	_ = w.Flush()
}

func newTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
}
