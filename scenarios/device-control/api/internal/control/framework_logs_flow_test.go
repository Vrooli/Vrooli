package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type frameworkLogsFlowStrategy struct {
	*fakes.Strategy
}

func (s *frameworkLogsFlowStrategy) Actuate(ctx context.Context, actuation strategy.Actuation) error {
	if actuation.Action == "framework-logs" && actuation.Output != nil {
		summary, err := json.Marshal(map[string]any{
			"component":           actuation.Component,
			"tag":                 actuation.LogTag,
			"priority":            actuation.LogPriority,
			"matching_line_count": 1,
			"diagnostic_classes":  map[string]int{"other_matching_component_line": 1},
		})
		if err != nil {
			return err
		}
		*actuation.Output = summary
	}
	return s.Strategy.Actuate(ctx, actuation)
}

func TestFrameworkLogFlowCarriesWarningAndInfoPriorityIntoSanitizedEvidence(t *testing.T) {
	for _, priority := range []string{"W", "I"} {
		t.Run(priority, func(t *testing.T) {
			db, err := sql.Open("sqlite", fmt.Sprintf("file:framework-logs-flow-%s-%s?mode=memory&cache=shared", priority, filepath.Base(t.Name())))
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			base := fakes.New("fake-logs", strategy.StatusAvailable, strategy.CapDeviceLogs)
			adapter := &frameworkLogsFlowStrategy{Strategy: base}
			svc, err := NewWithDB(strategyregistry.New(adapter), db)
			require.NoError(t, err)

			result, err := svc.Run(context.Background(), Flow{ID: "framework-logs", Steps: []Step{{
				ID: "read-framework-logs", Kind: "framework-logs",
				RequiredCapabilities: []string{strategy.CapDeviceLogs},
				Arguments:            map[string]any{"component": "com.example/.service.Home", "tag": "ActivityManager", "priority": priority},
			}}}, "fake-logs", "operator")
			require.NoError(t, err)
			require.Equal(t, "passed", result.Disposition)
			require.Len(t, result.Evidence, 1)

			calls := base.Calls()
			require.Len(t, calls, 1)
			require.Equal(t, priority, calls[0].LogPriority)

			artifact, _, err := svc.Artifact(result.Evidence[0].ID)
			require.NoError(t, err)
			require.Contains(t, string(artifact), `"priority":"`+priority+`"`)
			require.NotContains(t, string(artifact), "private-secret")
		})
	}
}
