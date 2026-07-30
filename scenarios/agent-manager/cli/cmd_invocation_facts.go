package main

import (
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	"net/url"
	"strings"
)

func (a *App) runInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationFacts(args[0])
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

func (a *App) runRefreshInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run refresh-invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run refresh-invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.RefreshInvocationFacts(args[0])
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

func (a *App) runReplayInvocationCorpus(args []string) error {
	fs := flag.NewFlagSet("run replay-invocation-corpus", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	from := fs.String("from", "", "RFC3339 inclusive start")
	to := fs.String("to", "", "RFC3339 exclusive end")
	status := fs.String("status", "", "run status")
	profileID := fs.String("profile-id", "", "agent profile UUID")
	tagPrefix := fs.String("tag-prefix", "", "tag prefix")
	limit := fs.String("limit", "", "maximum runs")
	refresh := fs.Bool("refresh", false, "refresh only runs with events newer than watermark")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := url.Values{}
	for key, value := range map[string]string{"from": *from, "to": *to, "status": *status, "profile_id": *profileID, "tag_prefix": *tagPrefix, "limit": *limit} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if *refresh {
		values.Set("mode", "refresh")
	}
	body, err := a.services.Runs.ReplayInvocationCorpus(values)
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

func invocationQueryFlags(fs *flag.FlagSet) (dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus *string, limit *int) {
	dimension = fs.String("dimension", "", "aggregate dimension")
	ownership = fs.String("ownership", "", "ownership filter")
	outcome = fs.String("outcome", "", "outcome filter")
	executable = fs.String("executable", "", "executable filter")
	fingerprint = fs.String("fingerprint", "", "fingerprint filter")
	from = fs.String("from", "", "RFC3339 start")
	to = fs.String("to", "", "RFC3339 end")
	profileID = fs.String("profile-id", "", "profile filter")
	runnerType = fs.String("runner-type", "", "runner filter")
	model = fs.String("model", "", "model filter")
	tagPrefix = fs.String("tag-prefix", "", "tag prefix")
	runStatus = fs.String("run-status", "", "status filter")
	limit = fs.Int("limit", 100, "maximum results")
	return
}
func invocationQueryValues(dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus string, limit int) url.Values {
	values := url.Values{"limit": []string{fmt.Sprint(limit)}}
	for key, value := range map[string]string{"dimension": dimension, "ownership": ownership, "outcome": outcome, "executable": executable, "fingerprint": fingerprint, "from": from, "to": to, "profile_id": profileID, "runner_type": runnerType, "model": model, "tag_prefix": tagPrefix, "run_status": runStatus} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}
func (a *App) runAggregateInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-aggregate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *d == "" {
		return fmt.Errorf("--dimension is required")
	}
	body, err := a.services.Runs.AggregateInvocationFacts(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}
func (a *App) runSelectInvocationCohort(args []string) error {
	fs := flag.NewFlagSet("run invocation-cohort", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.SelectInvocationCohort(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}
func (a *App) runInvocationMetrics(args []string) error {
	fs := flag.NewFlagSet("run invocation-metrics", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationMetrics(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runReplayInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run replay-invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run replay-invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.ReplayInvocationFacts(args[0])
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
