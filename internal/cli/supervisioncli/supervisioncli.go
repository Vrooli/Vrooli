// Package supervisioncli owns parsing and rendering for `vrooli supervision-set`.
package supervisioncli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type Request struct {
	Kind string
	JSON bool
}

func ParseRequest(globalJSON bool, args []string) (Request, error) {
	fs := flag.NewFlagSet("supervision-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	request := Request{JSON: globalJSON}
	fs.StringVar(&request.Kind, "kind", "", "filter by scenario or resource")
	fs.BoolVar(&request.JSON, "json", request.JSON, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return Request{}, err
	}
	if fs.NArg() != 0 {
		return Request{}, fmt.Errorf("usage: vrooli supervision-set [--kind scenario|resource] [--json]")
	}
	request.Kind = strings.ToLower(strings.TrimSpace(request.Kind))
	switch request.Kind {
	case "", apicoreset.MemberKindScenario, apicoreset.MemberKindResource:
		return request, nil
	default:
		return Request{}, fmt.Errorf("--kind must be %q or %q", apicoreset.MemberKindScenario, apicoreset.MemberKindResource)
	}
}

func Filter(report apicoreset.Report, kind string) apicoreset.Report {
	if kind == "" {
		return report
	}
	filtered := report
	filtered.Members = make([]apicoreset.Member, 0)
	for _, member := range report.Members {
		if member.Kind == kind {
			filtered.Members = append(filtered.Members, member)
		}
	}
	filtered.MemberCounts = map[string]int{kind: len(filtered.Members)}
	return filtered
}

func Render(w io.Writer, format cliout.Format, report apicoreset.Report) error {
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, responseMessage(report)) },
		func(w io.Writer) error {
			rows := make([][]string, 0, len(report.Members))
			for _, member := range report.Members {
				rows = append(rows, []string{
					member.Name,
					member.Kind,
					member.SupervisionIntent,
					formatAttribution(member.AttributionChain),
				})
			}
			return cliout.RenderTable(w, []string{"Name", "Kind", "Intent", "Attribution"}, rows)
		})
}

func responseMessage(report apicoreset.Report) *cliv1.SupervisionSetResponse {
	message := &cliv1.SupervisionSetResponse{
		Source:         report.Source,
		CoreSet:        append([]string(nil), report.CoreSet...),
		Seed:           append([]string(nil), report.Seed...),
		AddedByClosure: append([]string(nil), report.AddedByClosure...),
		TrustedBase:    append([]string(nil), report.TrustedBase...),
		MemberCounts:   make(map[string]int32, len(report.MemberCounts)),
		LoadErrors:     report.LoadErrors,
		Members:        make([]*cliv1.SupervisionMember, 0, len(report.Members)),
	}
	for kind, count := range report.MemberCounts {
		message.MemberCounts[kind] = int32(count) // #nosec G115 -- repository member counts are bounded by manifest inventory.
	}
	for _, member := range report.Members {
		item := &cliv1.SupervisionMember{
			Name:              member.Name,
			Kind:              member.Kind,
			SupervisionIntent: member.SupervisionIntent,
			AttributionChain:  make([]*cliv1.SupervisionAttributionStep, 0, len(member.AttributionChain)),
		}
		for _, step := range member.AttributionChain {
			item.AttributionChain = append(item.AttributionChain, &cliv1.SupervisionAttributionStep{
				Name:              step.Name,
				Kind:              step.Kind,
				DeclaredBy:        step.DeclaredBy,
				SupervisionIntent: step.SupervisionIntent,
				Source:            step.Source,
			})
		}
		message.Members = append(message.Members, item)
	}
	return message
}

func formatAttribution(chain []apicoreset.AttributionStep) string {
	parts := make([]string, 0, len(chain))
	for _, step := range chain {
		parts = append(parts, step.Kind+":"+step.Name)
	}
	return strings.Join(parts, " <- ")
}
