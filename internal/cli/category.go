package cli

import (
	"context"
	"sort"

	"github.com/spf13/cobra"

	"lunchmoney-cli/internal/lunchmoney"
)

func newCategoryCmd() *cobra.Command {
	categoryCmd := &cobra.Command{
		Use:   "category",
		Short: "Category operations",
	}

	var jsonOutput bool

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := lunchmoney.NewFromEnv()
			if err != nil {
				return err
			}

			categories, err := client.ListCategories(context.Background())
			if err != nil {
				return err
			}

			views := toCategoryViews(categories)
			sort.Slice(views, func(i, j int) bool {
				if views[i].Group != views[j].Group {
					return views[i].Group < views[j].Group
				}
				return views[i].Name < views[j].Name
			})

			if jsonOutput {
				return printJSON(views)
			}

			printCategoriesTable(views)
			return nil
		},
	}
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")

	categoryCmd.AddCommand(listCmd)
	return categoryCmd
}

func toCategoryViews(categories []lunchmoney.Category) []categoryView {
	views := make([]categoryView, 0, len(categories))

	groupNames := make(map[int64]string, len(categories))
	for _, c := range categories {
		if c.IsGroup {
			groupNames[c.ID] = c.Name
		}
	}

	for _, c := range categories {
		if c.Archived {
			continue
		}
		group := ""
		if c.GroupID != nil {
			group = groupNames[*c.GroupID]
		}
		views = append(views, categoryView{
			ID:                c.ID,
			Name:              c.Name,
			Group:             group,
			IsIncome:          c.IsIncome,
			ExcludeFromTotals: c.ExcludeFromTotals,
		})
	}

	return views
}
