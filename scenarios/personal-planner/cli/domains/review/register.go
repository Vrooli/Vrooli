package review

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "review"

func Register(c *cliapp.ScenarioApp, m []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(c)
	g, e := cliapp.LoadFromManifestPrimitives(m, GroupName, map[string]cliapp.PrimitiveHandler{
		"ReviewService.GetDailySummary":  cliapp.ProtoList(h.dailyCall, h.dailyReport),
		"ReviewService.GetWeeklySummary": cliapp.ProtoList(h.weeklyCall, h.weeklyReport),
		"ReviewService.GetReflection":    cliapp.ProtoList(h.reflectionCall, h.reflectionReport),
		"ReviewService.SaveReflection":   cliapp.ProtoMutation(h.saveReflectionCall, h.saveReflectionReport),
	})
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("review: load from manifest: %w", e)
	}
	return g, nil
}
