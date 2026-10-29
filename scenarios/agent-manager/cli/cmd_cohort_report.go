package main

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) runEpisodeTrend(args []string) error {
	fs := flag.NewFlagSet("measures episode-trend", flag.ContinueOnError)
	from := fs.String("from", "", "RFC3339 inclusive time")
	to := fs.String("to", "", "RFC3339 exclusive time")
	bucket := fs.String("bucket", "24h", "positive Go duration bucket")
	limit := fs.Int("limit", 100, "maximum buckets")
	cohort := fs.String("cohort", "", "durable cohort name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := url.Values{"bucket": []string{*bucket}, "limit": []string{strconv.Itoa(*limit)}}
	if *from != "" {
		values.Set("from", *from)
	}
	if *to != "" {
		values.Set("to", *to)
	}
	if *cohort != "" {
		values.Set("cohort", *cohort)
	}
	body, err := a.services.Runs.EpisodeTrend(values)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}
