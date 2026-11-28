// rights-impact reports registry-derived model rights decisions under the
// canonical business context or a temporary hypothetical revenue category.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/impact"
	"github.com/vrooli/ai-provider-go/policy"
)

func main() {
	fs := flag.NewFlagSet("rights-impact", flag.ExitOnError)
	rootFlag := fs.String("root", "", "Vrooli repository root (detected from the current directory by default)")
	contextFlag := fs.String("context", "", "recorded business context JSON (defaults to the canonical ai-gateway context)")
	status := fs.String("revenue-status", "", "hypothetical status: pre_revenue or at_or_above_threshold; does not change the context file")
	compareFrom := fs.String("compare-from-revenue-status", "", "compare from this hypothetical status instead of the recorded context")
	annualRevenue := fs.String("annual-revenue-usd", "", "hypothetical exact annual revenue in USD; does not change the context file")
	compareFromAnnualRevenue := fs.String("compare-from-annual-revenue-usd", "", "compare from this hypothetical exact annual revenue in USD")
	var targetMAU, baseMAU []string
	fs.Func("monthly-active-users", "hypothetical historical MAU fact PERIOD=COUNT|at_or_below_threshold|above_threshold (repeatable)", func(value string) error {
		targetMAU = append(targetMAU, value)
		return nil
	})
	fs.Func("compare-from-monthly-active-users", "compare from historical MAU fact PERIOD=COUNT|at_or_below_threshold|above_threshold (repeatable)", func(value string) error {
		baseMAU = append(baseMAU, value)
		return nil
	})
	asJSON := fs.Bool("json", false, "emit JSON")
	_ = fs.Parse(os.Args[1:])
	root := strings.TrimSpace(*rootFlag)
	if root == "" {
		root = findRepoRoot()
	}
	contextPath := strings.TrimSpace(*contextFlag)
	if contextPath == "" {
		contextPath = filepath.Join(root, "scenarios/ai-gateway/config/model-rights-context.json")
	}
	ctx, err := policy.LoadRightsContextFile(contextPath)
	fatalIf(err)
	if *status != "" && *annualRevenue != "" {
		fatalIf(fmt.Errorf("choose either --revenue-status or --annual-revenue-usd"))
	}
	if *compareFrom != "" && *compareFromAnnualRevenue != "" {
		fatalIf(fmt.Errorf("choose either --compare-from-revenue-status or --compare-from-annual-revenue-usd"))
	}
	base := ctx
	compare := false
	if strings.TrimSpace(*compareFrom) != "" {
		base.RevenueStatus = normalizeRevenueStatus(*compareFrom)
		compare = true
	}
	if strings.TrimSpace(*compareFromAnnualRevenue) != "" {
		base, err = withAnnualRevenue(base, *compareFromAnnualRevenue)
		fatalIf(err)
		compare = true
	}
	base.MonthlyActiveUsers = cloneMonthlyActiveUsers(ctx.MonthlyActiveUsers)
	for _, fact := range baseMAU {
		base, err = withMonthlyActiveUsers(base, fact)
		fatalIf(err)
		compare = true
	}
	for _, fact := range targetMAU {
		before := ctx.MonthlyActiveUsers
		ctx, err = withMonthlyActiveUsers(ctx, fact)
		fatalIf(err)
		period, _, _ := strings.Cut(fact, "=")
		if before[strings.TrimSpace(period)] != ctx.MonthlyActiveUsers[strings.TrimSpace(period)] {
			compare = true
		}
	}
	if *status != "" {
		target := normalizeRevenueStatus(*status)
		compare = compare || ctx.RevenueStatus != target
		ctx.RevenueStatus = target
	}
	if strings.TrimSpace(*annualRevenue) != "" {
		amount, parseErr := strconv.ParseFloat(strings.TrimSpace(*annualRevenue), 64)
		fatalIf(parseErr)
		if ctx.AnnualRevenueUSD == nil || *ctx.AnnualRevenueUSD != amount {
			compare = true
		}
		ctx, err = withAnnualRevenue(ctx, *annualRevenue)
		fatalIf(err)
	}
	registries := impact.DefaultRegistries(root)
	var report impact.Report
	if compare {
		report, err = impact.Compare(registries, base, ctx, time.Now().UTC())
	} else {
		report, err = impact.Load(registries, ctx, time.Now().UTC())
	}
	fatalIf(err)
	if compare {
		report = onlyChanged(report)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fatalIf(enc.Encode(report))
		return
	}
	fmt.Printf("business context %s (%s), revenue_status=%s", ctx.ID, ctx.Version, ctx.RevenueStatus)
	if ctx.AnnualRevenueUSD != nil {
		fmt.Printf(", annual_revenue_usd=%.2f", *ctx.AnnualRevenueUSD)
	}
	fmt.Println()
	for _, model := range report.Models {
		for _, p := range model.Paths {
			if p.Changed {
				fmt.Printf("%s/%s | %s | %s -> %s | %s (%s)\n", model.Provider, model.ID, p.Use, p.PreviousStatus, p.Decision.Status, p.Decision.Reason, p.Path)
			} else {
				fmt.Printf("%s/%s | %s | %s | %s (%s)\n", model.Provider, model.ID, p.Use, p.Decision.Status, p.Decision.Reason, p.Path)
			}
		}
	}
}

func withAnnualRevenue(context policy.RightsContext, raw string) (policy.RightsContext, error) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return context, fmt.Errorf("annual revenue must be a finite, nonnegative USD amount")
	}
	context.AnnualRevenueUSD = &amount
	context.RevenueStatus = "exact_value"
	return context, nil
}

func withMonthlyActiveUsers(context policy.RightsContext, value string) (policy.RightsContext, error) {
	period, status, ok := strings.Cut(strings.TrimSpace(value), "=")
	period, status = strings.TrimSpace(period), strings.TrimSpace(status)
	if !ok || timeFormatYearMonth(period) != nil {
		return context, fmt.Errorf("historical MAU fact must use YYYY-MM=STATUS")
	}
	switch status {
	case "at_or_below_threshold", "above_threshold":
	default:
		if _, err := strconv.ParseUint(status, 10, 64); err != nil {
			return context, fmt.Errorf("historical MAU fact must be a nonnegative integer count or explicit threshold status")
		}
	}
	context.MonthlyActiveUsers = cloneMonthlyActiveUsers(context.MonthlyActiveUsers)
	if context.MonthlyActiveUsers == nil {
		context.MonthlyActiveUsers = make(map[string]string)
	}
	context.MonthlyActiveUsers[period] = status
	if _, err := strconv.ParseUint(status, 10, 64); err == nil {
		context.MonthlyActiveUsersEvidence = cloneMonthlyActiveUsers(context.MonthlyActiveUsersEvidence)
		if context.MonthlyActiveUsersEvidence == nil {
			context.MonthlyActiveUsersEvidence = make(map[string]string)
		}
		context.MonthlyActiveUsersEvidence[period] = "hypothetical rights-impact command-line input"
	}
	return context, nil
}

func timeFormatYearMonth(value string) error {
	_, err := time.Parse("2006-01", value)
	return err
}

func cloneMonthlyActiveUsers(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for period, status := range in {
		out[period] = status
	}
	return out
}

func onlyChanged(report impact.Report) impact.Report {
	models := make([]impact.Model, 0, len(report.Models))
	for _, model := range report.Models {
		paths := model.Paths[:0]
		for _, path := range model.Paths {
			if path.Changed {
				paths = append(paths, path)
			}
		}
		if len(paths) > 0 {
			model.Paths = paths
			models = append(models, model)
		}
	}
	report.Models = models
	return report
}

func normalizeRevenueStatus(value string) string {
	status := strings.TrimSpace(value)
	if status == "over_threshold" {
		return "at_or_above_threshold"
	}
	return status
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func findRepoRoot() string {
	wd, err := os.Getwd()
	fatalIf(err)
	for current := wd; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "resources/ollama/model-policy.json")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			fatalIf(fmt.Errorf("could not find Vrooli repository root above %s", wd))
		}
	}
}
