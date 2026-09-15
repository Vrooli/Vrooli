package eval

import (
	"fmt"
	"sort"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// RouterSuiteID is a virtual suite owned by Search Hub's router. It is never
// accepted through RegisterSuite; the service projects it from live corpora.
const RouterSuiteID = "router.routing"

// ComposeRoutingSuite derives the router corpus from provider-owned suites.
// registered contains provider ids currently present in the registry. The
// builder intentionally operates on declared suite content only: it knows no
// scenario names, provider implementations, or authored routing labels.
func ComposeRoutingSuite(suites []*evalv1.EvalSuite, registered map[string]struct{}) *evalv1.EvalSuite {
	providers := make(map[string]int)
	excluded := map[string]int{"candidate": 0, "generated": 0, "negative": 0, "degenerate": 0, "unregistered": 0}
	cases := make([]*evalv1.EvalCase, 0)
	for _, suite := range suites {
		if suite == nil || suite.GetSuiteId() == RouterSuiteID {
			continue
		}
		if _, ok := registered[suite.GetProviderId()]; !ok {
			for _, c := range suite.GetCases() {
				if c != nil && c.GetStatus() != "candidate" && !hasTag(c.GetTags(), "generated") && !c.GetExpectNoStrongHit() {
					excluded["unregistered"]++
				}
			}
			continue
		}
		for _, c := range suite.GetCases() {
			if c == nil {
				continue
			}
			switch {
			case c.GetStatus() == "candidate":
				excluded["candidate"]++
			case composeHasTag(c.GetTags(), "generated"):
				excluded["generated"]++
			case c.GetExpectNoStrongHit() || len(c.GetExpectIds()) == 0:
				excluded["negative"]++
			case composeDegenerate(c):
				excluded["degenerate"]++
			default:
				copyCase := &evalv1.EvalCase{
					CaseId:             fmt.Sprintf("%s.%s", suite.GetProviderId(), c.GetCaseId()),
					Query:              c.GetQuery(),
					Tags:               append([]string(nil), c.GetTags()...),
					ExpectIds:          append([]string(nil), c.GetExpectIds()...),
					ExpectWithinTopK:   c.GetExpectWithinTopK(),
					ExpectMinScore:     c.GetExpectMinScore(),
					ExpectMaxScore:     c.GetExpectMaxScore(),
					Note:               c.GetNote(),
					ExpectedProviderId: suite.GetProviderId(),
				}
				cases = append(cases, copyCase)
				providers[suite.GetProviderId()]++
			}
		}
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].GetCaseId() < cases[j].GetCaseId() })
	providerNames := make([]string, 0, len(providers))
	for provider := range providers {
		providerNames = append(providerNames, provider)
	}
	sort.Strings(providerNames)
	breakdown := make([]string, 0, len(providerNames))
	for _, provider := range providerNames {
		breakdown = append(breakdown, fmt.Sprintf("%s=%d", provider, providers[provider]))
	}
	description := fmt.Sprintf("Composed from registered reviewed positive corpora; cases=%d; providers=%s; excluded candidate=%d generated=%d negative=%d degenerate=%d unregistered=%d.", len(cases), strings.Join(breakdown, ","), excluded["candidate"], excluded["generated"], excluded["negative"], excluded["degenerate"], excluded["unregistered"])
	return &evalv1.EvalSuite{
		SuiteId:     RouterSuiteID,
		ProviderId:  RouterSuiteID,
		Name:        "Composed router routing",
		Description: description,
		Cases:       cases,
		State:       "active",
	}
}

func composeHasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func composeDegenerate(c *evalv1.EvalCase) bool {
	query := strings.ToLower(strings.Join(strings.Fields(c.GetQuery()), " "))
	if query == "" || len(strings.Fields(query)) > 3 {
		return false
	}
	for _, expected := range c.GetExpectIds() {
		if strings.Contains(strings.ToLower(expected), query) {
			return true
		}
	}
	return false
}
