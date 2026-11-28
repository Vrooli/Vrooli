package componenttests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestConsumerContractProviderDescribesNamedPhase(t *testing.T) {
	handler := &sharedHandler{}
	response, err := handler.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{Phase: consumerContractPhase}))
	require.NoError(t, err)
	require.Equal(t, "react-component-library", response.Msg.GetProvider())
	require.Equal(t, consumerContractPhase, response.Msg.GetPhase())
	require.True(t, response.Msg.GetCapabilities().GetSupportsExecution())
}

func TestConsumerContractPhaseRunsFleetCommandAndFailsClosed(t *testing.T) {
	tests := []struct {
		name             string
		stdout           string
		exitCode         int
		wantStatus       scenariovalidationv1.ValidationStatus
		wantFailureClass string
	}{
		{
			name:       "fleet ratchet passes",
			stdout:     `{"fleet":{"scenarios":114,"files":8801,"findings":25,"byContract":{"Overlay":25}},"ratchet":{"status":"passed","baseline":"scenarios/react-component-library/ui/usage-contract-ratchet.json","comparisons":{"Overlay":{"baseline":32,"current":25,"delta":-7}},"failures":[]},"catalogCoverage":{"status":"passed","eligible":102,"contracted":102,"errors":[]},"findings":[]}`,
			exitCode:   0,
			wantStatus: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		},
		{
			name:             "ratchet increase is a test failure",
			stdout:           `{"fleet":{"scenarios":114,"files":8801,"findings":33,"byContract":{"Overlay":33}},"ratchet":{"status":"failed","baseline":"scenarios/react-component-library/ui/usage-contract-ratchet.json","comparisons":{"Overlay":{"baseline":32,"current":33,"delta":1}},"failures":["Overlay increased from 32 to 33"]},"catalogCoverage":{"status":"passed","eligible":102,"contracted":102,"errors":[]},"findings":[]}`,
			exitCode:         1,
			wantStatus:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			wantFailureClass: "test_failure",
		},
		{
			name:             "missing latest contract is a test failure",
			stdout:           `{"fleet":{"scenarios":114,"files":8801,"findings":3,"byContract":{"Card":1,"SettingsList":2}},"ratchet":{"status":"passed","baseline":"ratchet.json","comparisons":{},"failures":[]},"catalogCoverage":{"status":"failed","eligible":105,"contracted":99,"errors":["AppNotificationSettings: missing usage-contract.json for latest 1.0.6"]},"findings":[]}`,
			exitCode:         1,
			wantStatus:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			wantFailureClass: "test_failure",
		},
		{
			name:             "missing report is a system error",
			stdout:           "not-json",
			exitCode:         0,
			wantStatus:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR,
			wantFailureClass: "system",
		},
		{
			name:             "missing counts are an error",
			stdout:           `{"fleet":{"scenarios":114,"files":8801},"ratchet":{"status":"passed","baseline":"ratchet.json"}}`,
			exitCode:         0,
			wantStatus:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR,
			wantFailureClass: "system",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			uiDir := filepath.Join(root, "ui")
			require.NoError(t, os.MkdirAll(uiDir, 0o755))
			binDir := t.TempDir()
			commandPath := filepath.Join(binDir, "pnpm")
			argvPath := filepath.Join(root, "pnpm-argv.txt")
			t.Setenv("RCL_TEST_ARGV_FILE", argvPath)
			command := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$RCL_TEST_ARGV_FILE\"\nprintf '%s\\n' '" + test.stdout + "'\nexit " + string(rune('0'+test.exitCode)) + "\n"
			require.NoError(t, os.WriteFile(commandPath, []byte(command), 0o700))
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			handler := &sharedHandler{sourceRoot: filepath.Join(root, "library")}
			response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
				Scenario: "react-component-library", Phase: consumerContractPhase, IncludeExecution: true,
			}))
			require.NoError(t, err)
			argv, readErr := os.ReadFile(argvPath)
			require.NoError(t, readErr)
			require.Contains(t, string(argv), "usage-contracts:fleet")
			require.Contains(t, string(argv), "--json")
			require.Equal(t, test.wantStatus, response.Msg.GetStatus())
			require.Equal(t, test.wantFailureClass, response.Msg.GetFailureClassification())
			require.NotNil(t, response.Msg.GetAssessment())
			if test.wantStatus == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
				require.NotNil(t, response.Msg.GetNativeDetail())
				require.True(t, response.Msg.GetAssessment().GetLocal().GetClean())
			} else {
				require.False(t, response.Msg.GetAssessment().GetLocal().GetClean())
				if test.name == "missing latest contract is a test failure" {
					require.Contains(t, response.Msg.GetAssessment().GetFindings()[0].GetMessage(), "AppNotificationSettings")
				}
			}
		})
	}
}
