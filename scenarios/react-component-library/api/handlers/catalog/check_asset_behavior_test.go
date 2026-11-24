package catalog

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogbuild"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/componenttests"
)

func checkAssetHandler() *handler {
	return &handler{
		checkBuild: func(string, catalogbuild.Options) (catalogbuild.Report, error) {
			return catalogbuild.Report{}, nil
		},
		checkAsset: func(context.Context, string) (components.Component, error) {
			return components.Component{ID: "component-1", LibraryID: "react-component-library:Card", LatestVersion: "2.0.0"}, nil
		},
		checkGate: func(_ context.Context, gate, assetID string) (*connect.Response[catalogv1.RunGateResponse], error) {
			return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
		},
	}
}

func TestCheckAssetReturnsStaleTestsWithoutRunningCapture(t *testing.T) {
	h := checkAssetHandler()
	var requested componenttests.Request
	h.checkTest = func(_ context.Context, got componenttests.Request) (componenttests.Report, bool, error) {
		requested = got
		return componenttests.Report{}, false, nil
	}
	var gates []string
	h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
		gates = append(gates, gate)
		return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
	}

	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: " card ", Version: "1.4.0", RunTests: true}))
	require.NoError(t, err)
	require.Equal(t, "STALE_TESTS", response.Msg.GetVerdict())
	require.Equal(t, componenttests.Request{ComponentID: "react-component-library:Card", Version: "1.4.0", IncludeClosure: false}, requested)
	require.Equal(t, []string{"all", "dist-resolution"}, gates)
	require.Len(t, response.Msg.Stages, 4)
	require.Equal(t, "stale", response.Msg.Stages[2].Status)
}

func TestCheckAssetPassedReportRemovesOnlyHistoricalEvidenceFreshnessFinding(t *testing.T) {
	h := checkAssetHandler()
	h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
		findings := []*catalogv1.GateFinding{}
		if gate == "all" {
			findings = []*catalogv1.GateFinding{
				{Code: "catalog.evidence_freshness", Severity: "error", Blocking: true},
				{Code: "catalog.naming", Severity: "warning"},
			}
		}
		return connect.NewResponse(&catalogv1.RunGateResponse{Findings: findings}), nil
	}
	h.checkTest = func(ctx context.Context, request componenttests.Request) (componenttests.Report, bool, error) {
		require.Equal(t, assetCheckProfilePublish, ctx.Value(assetCheckProfileContextKey{}))
		require.Equal(t, "2.0.0", request.Version)
		return componenttests.Report{ID: "report-1", SourceRevision: "rev-1", Verdict: componenttests.VerdictPassed}, true, nil
	}

	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "PUBLISHABLE", response.Msg.GetVerdict())
	require.Equal(t, "report-1", response.Msg.GetReusedReportId())
	require.Equal(t, "rev-1", response.Msg.GetSourceRevision())
	require.Len(t, response.Msg.GetFindings(), 1)
	require.Equal(t, "catalog.naming", response.Msg.GetFindings()[0].GetCode())
	require.Equal(t, "passed", response.Msg.GetStages()[1].GetStatus())
}

func TestCheckAssetSurfacesComponentTestFailureAndPackageGateFailure(t *testing.T) {
	t.Run("component test failure blocks", func(t *testing.T) {
		h := checkAssetHandler()
		h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
			return componenttests.Report{ID: "failed", Verdict: componenttests.VerdictFailed}, true, nil
		}
		response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
		require.NoError(t, err)
		require.Equal(t, "BLOCKED", response.Msg.GetVerdict())
		require.Equal(t, "blocked", response.Msg.GetStages()[2].GetStatus())
	})

	t.Run("package gate failure is a fault response", func(t *testing.T) {
		h := checkAssetHandler()
		h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
			return componenttests.Report{ID: "passed", Verdict: componenttests.VerdictPassed}, true, nil
		}
		h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
			if gate == "dist-resolution" {
				return nil, errors.New("package metadata unavailable")
			}
			return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
		}
		response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
		// Gate execution failures are represented as FAULT verdicts so callers
		// receive stage diagnostics along with the successful RPC response.
		require.NoError(t, err)
		require.Equal(t, "FAULT", response.Msg.GetVerdict())
		require.Equal(t, "fault", response.Msg.GetStages()[3].GetStatus())
	})
}

func TestCheckAssetRejectsMissingIdentityAndUnknownProfile(t *testing.T) {
	h := checkAssetHandler()
	_, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card", Profile: "preview"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCheckAssetReportsCatalogBuildFailureWithoutRunningGates(t *testing.T) {
	h := checkAssetHandler()
	h.checkBuild = func(string, catalogbuild.Options) (catalogbuild.Report, error) {
		return catalogbuild.Report{}, errors.New("generated catalog is stale")
	}
	gateCalls := 0
	h.checkGate = func(context.Context, string, string) (*connect.Response[catalogv1.RunGateResponse], error) {
		gateCalls++
		return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
	}

	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "STALE_CATALOG", response.Msg.GetVerdict())
	require.Equal(t, "stale", response.Msg.GetStages()[0].GetStatus())
	require.Zero(t, gateCalls, "catalog gates must not run against stale generated output")
}

func TestCheckAssetMissingComponentFaultsUnlessGateFindingsAlreadyBlock(t *testing.T) {
	t.Run("unblocked asset lookup failure faults", func(t *testing.T) {
		h := checkAssetHandler()
		h.checkAsset = func(context.Context, string) (components.Component, error) {
			return components.Component{}, errors.New("asset is not indexed")
		}
		response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "missing"}))
		require.NoError(t, err)
		require.Equal(t, "FAULT", response.Msg.GetVerdict())
		require.Equal(t, "fault", response.Msg.GetStages()[2].GetStatus())
	})

	t.Run("blocking gate finding skips unnecessary component lookup", func(t *testing.T) {
		h := checkAssetHandler()
		h.checkAsset = func(context.Context, string) (components.Component, error) {
			return components.Component{}, errors.New("asset is not indexed")
		}
		h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
			return componenttests.Report{}, false, nil
		}
		var gates []string
		h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
			gates = append(gates, gate)
			findings := []*catalogv1.GateFinding(nil)
			if gate == "all" {
				findings = append(findings, &catalogv1.GateFinding{Code: "catalog.invalid_asset", Severity: "error", Blocking: true})
			}
			return connect.NewResponse(&catalogv1.RunGateResponse{Findings: findings}), nil
		}
		response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "missing"}))
		require.NoError(t, err)
		require.Equal(t, "BLOCKED", response.Msg.GetVerdict())
		require.Equal(t, "skipped", response.Msg.GetStages()[2].GetStatus())
		require.Equal(t, []string{"all", "dist-resolution"}, gates)
	})
}

func TestCheckAssetHandlesHookPackageVerdictWithoutBrowserReport(t *testing.T) {
	for _, tc := range []struct {
		name            string
		allFindings     []*catalogv1.GateFinding
		packageFindings []*catalogv1.GateFinding
		packageErr      error
		wantVerdict     string
		wantPackage     string
	}{
		{name: "warning findings remain publishable", allFindings: []*catalogv1.GateFinding{{Code: "catalog.naming", Severity: "warning"}}, packageFindings: []*catalogv1.GateFinding{{Code: "catalog.package_notice", Severity: "warning"}}, wantVerdict: "PUBLISHABLE", wantPackage: "passed"},
		{name: "package runner failure is explicit", packageErr: errors.New("package gate unavailable"), wantVerdict: "FAULT", wantPackage: "fault"},
		{name: "package error keeps hook blocked", allFindings: []*catalogv1.GateFinding{{Code: "catalog.hook_invalid", Severity: "error"}}, packageFindings: []*catalogv1.GateFinding{{Code: "catalog.package_invalid", Severity: "error"}}, wantVerdict: "BLOCKED", wantPackage: "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := checkAssetHandler()
			h.checkAsset = func(context.Context, string) (components.Component, error) {
				return components.Component{ID: "hook-1", LibraryID: "react-component-library:useOverlaySurface", LatestVersion: "1.0.0", AssetKind: components.AssetKindHook}, nil
			}
			h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
				t.Fatal("hook checks must not request a browser report")
				return componenttests.Report{}, false, nil
			}
			var gateCalls []string
			h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
				gateCalls = append(gateCalls, gate)
				if gate == "dist-resolution" {
					if tc.packageErr != nil {
						return nil, tc.packageErr
					}
					return connect.NewResponse(&catalogv1.RunGateResponse{Findings: tc.packageFindings}), nil
				}
				return connect.NewResponse(&catalogv1.RunGateResponse{Findings: tc.allFindings}), nil
			}

			response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "useOverlaySurface"}))
			require.NoError(t, err)
			require.Equal(t, tc.wantVerdict, response.Msg.GetVerdict())
			require.Equal(t, "n/a", response.Msg.GetStages()[2].GetStatus())
			require.Equal(t, tc.wantPackage, response.Msg.GetStages()[3].GetStatus())
			require.Equal(t, []string{"all", "dist-resolution"}, gateCalls)
		})
	}
}

func TestCheckAssetTestServiceFailureReturnsFaultStage(t *testing.T) {
	h := checkAssetHandler()
	h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
		return componenttests.Report{}, false, errors.New("report store unavailable")
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "FAULT", response.Msg.GetVerdict())
	require.Equal(t, "fault", response.Msg.GetStages()[2].GetStatus())
	require.Contains(t, response.Msg.GetStages()[2].GetDetail(), "report store unavailable")
}

func TestCheckAssetGateRunnerFailureReturnsGateFaultStage(t *testing.T) {
	h := checkAssetHandler()
	h.checkGate = func(context.Context, string, string) (*connect.Response[catalogv1.RunGateResponse], error) {
		return nil, errors.New("catalog gate runner unavailable")
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "FAULT", response.Msg.GetVerdict())
	require.Equal(t, "fault", response.Msg.GetStages()[1].GetStatus())
	require.Contains(t, response.Msg.GetStages()[1].GetDetail(), "catalog gate runner unavailable")
}

func TestCheckAssetFaultsWhenCleanAssetHasNoComponentTestReader(t *testing.T) {
	h := checkAssetHandler()
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "FAULT", response.Msg.GetVerdict())
	require.Equal(t, "fault", response.Msg.GetStages()[2].GetStatus())
	require.Contains(t, response.Msg.GetStages()[2].GetDetail(), "component-test service is not configured")
}

func TestCheckAssetTreatsErrorSeverityAsBlockingRegardlessOfCase(t *testing.T) {
	h := checkAssetHandler()
	h.checkGate = func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
		if gate == "all" {
			return connect.NewResponse(&catalogv1.RunGateResponse{Findings: []*catalogv1.GateFinding{{Code: "catalog.policy", Severity: "ERROR"}, {Code: "catalog.unclassified"}}}), nil
		}
		return connect.NewResponse(&catalogv1.RunGateResponse{Findings: []*catalogv1.GateFinding{{Code: "catalog.package", Severity: "Error"}, {Code: "catalog.package_unclassified"}}}), nil
	}
	h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
		return componenttests.Report{ID: "passed", Verdict: componenttests.VerdictPassed}, true, nil
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "BLOCKED", response.Msg.GetVerdict())
	require.Equal(t, "blocked", response.Msg.GetStages()[1].GetStatus())
	require.Equal(t, "blocked", response.Msg.GetStages()[3].GetStatus())
}

func TestCheckAssetEditProfilePinsRequestedVersion(t *testing.T) {
	h := checkAssetHandler()
	h.checkAsset = func(ctx context.Context, _ string) (components.Component, error) {
		require.Equal(t, assetCheckProfileEdit, ctx.Value(assetCheckProfileContextKey{}))
		return components.Component{ID: "component-1", LibraryID: "react-component-library:Card", LatestVersion: "2.0.0"}, nil
	}
	h.checkTest = func(ctx context.Context, request componenttests.Request) (componenttests.Report, bool, error) {
		require.Equal(t, assetCheckProfileEdit, ctx.Value(assetCheckProfileContextKey{}))
		require.Equal(t, "1.3.0", ctx.Value(assetCheckVersionContextKey{}))
		require.Equal(t, "1.3.0", request.Version)
		return componenttests.Report{ID: "edit-report", SourceRevision: "rev-edit", Verdict: componenttests.VerdictPassed}, true, nil
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card", Profile: assetCheckProfileEdit, Version: "1.3.0"}))
	require.NoError(t, err)
	require.Equal(t, "PUBLISHABLE", response.Msg.GetVerdict())
	require.Equal(t, "edit-report", response.Msg.GetReusedReportId())
}

func TestCheckAssetBlockingGateFindingSurfacesPackageGateFaultWhenTestServiceIsAbsent(t *testing.T) {
	h := &handler{
		repoRoot:   t.TempDir(),
		checkBuild: func(string, catalogbuild.Options) (catalogbuild.Report, error) { return catalogbuild.Report{}, nil },
		checkGate: func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
			if gate == "all" {
				return connect.NewResponse(&catalogv1.RunGateResponse{Findings: []*catalogv1.GateFinding{{Code: "catalog.invalid", Severity: "error", Blocking: true}}}), nil
			}
			t.Fatalf("unexpected injected gate %q", gate)
			return nil, nil
		},
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "card"}))
	require.NoError(t, err)
	require.Equal(t, "FAULT", response.Msg.GetVerdict())
	require.Equal(t, "skipped", response.Msg.GetStages()[2].GetStatus())
	require.Equal(t, "fault", response.Msg.GetStages()[3].GetStatus())
	require.Contains(t, response.Msg.GetStages()[3].GetDetail(), "load catalog gate definitions")
}

func TestCheckAssetBlockingGateStillRunsPackageGateWithoutTestService(t *testing.T) {
	h := &handler{
		repoRoot:   catalogTestRepoRoot(t),
		checkBuild: func(string, catalogbuild.Options) (catalogbuild.Report, error) { return catalogbuild.Report{}, nil },
		checkGate: func(_ context.Context, gate, _ string) (*connect.Response[catalogv1.RunGateResponse], error) {
			require.Equal(t, "all", gate)
			return connect.NewResponse(&catalogv1.RunGateResponse{Findings: []*catalogv1.GateFinding{{Code: "catalog.invalid", Severity: "error", Blocking: true}}}), nil
		},
	}
	response, err := h.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "react-component-library:NotificationBadge"}))
	require.NoError(t, err)
	require.Equal(t, "BLOCKED", response.Msg.GetVerdict())
	require.Equal(t, "skipped", response.Msg.GetStages()[2].GetStatus())
	require.Equal(t, "passed", response.Msg.GetStages()[3].GetStatus())
	require.Contains(t, response.Msg.GetStages()[3].GetDetail(), "package finding(s)")
}

func TestCheckAssetReusesClonedCacheForCurrentPublishedRevision(t *testing.T) {
	assets := appearanceAssetsStub{component: components.Component{
		ID: "notification-badge", CatalogID: "components.notification-badge", LibraryID: "react-component-library:NotificationBadge", LatestVersion: "1.0.5",
	}}
	buildCalls, gateCalls, testCalls := 0, 0, 0
	h := &handler{
		repoRoot: catalogTestRepoRoot(t), assets: assets, checkCache: map[string]*catalogv1.CheckAssetResponse{},
		checkBuild: func(string, catalogbuild.Options) (catalogbuild.Report, error) {
			buildCalls++
			return catalogbuild.Report{}, nil
		},
		checkGate: func(context.Context, string, string) (*connect.Response[catalogv1.RunGateResponse], error) {
			gateCalls++
			return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
		},
		checkTest: func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
			testCalls++
			return componenttests.Report{ID: "report-1", SourceRevision: "revision-1", Verdict: componenttests.VerdictPassed}, true, nil
		},
	}
	revision, err := catalogcoverage.CurrentRevisionForVersion(h.repoRoot, assets.component.LibraryID, assets.component.LatestVersion)
	require.NoError(t, err)
	staleKey := assets.component.LibraryID + "@" + assets.component.LatestVersion + "#" + revision + "#profile=" + assetCheckProfilePublish
	h.checkCache[staleKey] = &catalogv1.CheckAssetResponse{AssetId: "components.notification-badge", Verdict: "STALE_TESTS"}
	request := connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "components.notification-badge"})
	first, err := h.CheckAsset(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHABLE", first.Msg.GetVerdict())
	first.Msg.Verdict = "mutated-by-caller"
	second, err := h.CheckAsset(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHABLE", second.Msg.GetVerdict(), "cache responses must not alias a caller-mutated response")
	require.Equal(t, 1, buildCalls)
	require.Equal(t, 2, gateCalls)
	require.Equal(t, 1, testCalls)
}
