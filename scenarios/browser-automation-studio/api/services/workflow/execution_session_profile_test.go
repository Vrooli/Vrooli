package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/services/session-profile"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/proto"
)

func TestSavedAndAdhocExecutionShareSessionProfileResolution(t *testing.T) {
	for _, mode := range []string{"saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			captured := make(chan executor.Request, 1)
			service, repo, _ := newCompletionFixture(t, "saved", func(_ context.Context, req executor.Request) error {
				captured <- req
				return nil
			})

			profiles := sessionprofile.NewService(persistence.NewFileRepository(t.TempDir(), nil), nil)
			profile, err := profiles.CreateProfile("shared execution profile")
			require.NoError(t, err)
			profileSettings := &persistence.BrowserProfile{
				Preset:       "stealth",
				ExtraHeaders: map[string]string{"X-Profile": "persisted"},
			}
			_, err = profiles.UpdateProfileSettings(profile.ID, nil, profileSettings)
			require.NoError(t, err)
			_, err = profiles.UpdateProfile(profile.ID, func(p *persistence.SessionProfile) error {
				p.StorageState = json.RawMessage(`{"cookies":[{"name":"session","value":"fixture"}]}`)
				return nil
			})
			require.NoError(t, err)
			openTabs := []persistence.TabState{{URL: "https://fixture.invalid/account", IsActive: true}}
			_, err = profiles.SaveOpenTabs(profile.ID, openTabs)
			require.NoError(t, err)
			profile, err = profiles.GetProfile(profile.ID)
			require.NoError(t, err)
			service.sessionProfileService = profiles

			profileID := string(profile.ID)
			restoreTabs := true
			parameters := &basexecution.ExecutionParameters{
				SessionProfileId: &profileID,
				RestoreTabs:      &restoreTabs,
				BrowserProfile:   &basbase.BrowserProfile{Preset: proto.String("fast")},
			}
			if mode == "saved" {
				_, err = service.ExecuteWorkflowAPI(context.Background(), &basapi.ExecuteWorkflowRequest{
					WorkflowId: repo.workflow.ID.String(), Parameters: parameters, WaitForCompletion: true,
				})
			} else {
				workflow, getErr := service.GetWorkflowAPI(context.Background(), &basapi.GetWorkflowRequest{WorkflowId: repo.workflow.ID.String()})
				require.NoError(t, getErr)
				_, err = service.ExecuteAdhocWorkflowAPI(context.Background(), &basexecution.ExecuteAdhocRequest{
					FlowDefinition: workflow.Workflow.FlowDefinition, Parameters: parameters, WaitForCompletion: true,
				})
			}
			require.NoError(t, err)
			runRequest := <-captured
			require.JSONEq(t, string(profile.StorageState), string(runRequest.StorageState))
			require.Equal(t, profileVersionForReuse(profile), runRequest.SessionProfileVersion)
			require.True(t, runRequest.RestoreTabs)
			require.Equal(t, openTabs, runRequest.OpenTabs)
			require.Equal(t, "fast", runRequest.BrowserProfile.Preset)
			require.Equal(t, map[string]string{"X-Profile": "persisted"}, runRequest.BrowserProfile.ExtraHeaders)
		})
	}
}
