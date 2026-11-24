import "../component.css";
import { useEffect, useMemo, useState } from "react";
import { ComponentPageHeader } from "../component/ComponentPageHeader";
import { ComponentPageInspector } from "../component/ComponentPageInspector";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";

import { runComponentTest } from "../../api/componentTests";
import { getWorkspace, WorkspaceProvenance } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { appRoutes, workspacePath } from "../../routes";
import { assetVersion, useComponentBundle, wbKeys, type WorkbenchStory } from "../data";
import { cx } from "../cx";
import { AssetDataDrawer, type DrawerTab } from "../component/AssetDataDrawer";
import { ComponentPageLoadState } from "../component/ComponentPageLoadState";
import { FixtureCanvas } from "../component/FixtureCanvas";
import { useFixturePositions } from "../component/useFixturePositions";
import { InContextStage, type Device, type InContextReport } from "../component/InContextStage";
import { readWorkbenchPreferences } from "../preferences";
import {
  createComponentWorkspace,
  curatedWorkspaceIdForTemplate,
  DEFAULT_COMPONENT_WORKSPACE_ID,
  workspaceTemplate,
} from "../workspaces/componentContext";
import { useWorkspaces } from "../workspaces/useWorkspaces";
import { Banner } from "../primitives";

type Mode = "fixtures" | "context";
type InspectorTab = "checks" | "props" | "usage";

function normalizeDrawerTab(value: string | null): DrawerTab {
  const tabs: DrawerTab[] = [
    "overview",
    "source",
    "versions",
    "experience",
    "tests",
    "relationships",
  ];
  if (value && (tabs as string[]).includes(value)) return value as DrawerTab;
  return "overview";
}

export function ComponentPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const [version, setVersion] = useState("");
  const bundle = useComponentBundle(id, version || undefined);

  const [mode, setMode] = useState<Mode>(
    searchParams.get("mode") === "context" ? "context" : "fixtures",
  );
  const [selectedStoryId, setSelectedStoryId] = useState<string | null>(searchParams.get("story"));
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>("checks");
  const initialTestReportID = searchParams.get("testReport");
  const [drawerOpen, setDrawerOpen] = useState(
    Boolean(searchParams.get("tab") || initialTestReportID),
  );
  const [drawerTab, setDrawerTab] = useState<DrawerTab>(() =>
    initialTestReportID ? "tests" : normalizeDrawerTab(searchParams.get("tab")),
  );
  const [templateId, setTemplateId] = useState(() =>
    curatedWorkspaceIdForTemplate(searchParams.get("template") ?? DEFAULT_COMPONENT_WORKSPACE_ID),
  );
  const [device, setDevice] = useState<Device>(() => {
    const requested = searchParams.get("device");
    if (requested === "desktop" || requested === "tablet" || requested === "phone")
      return requested;
    // A desktop frame cannot be inspected on a phone without panning a clipped
    // surface. Default to the frame that matches the viewport; the device
    // switcher still lets a person choose a wider frame deliberately.
    if (typeof window !== "undefined" && window.matchMedia("(max-width: 640px)").matches) {
      return "phone";
    }
    return readWorkbenchPreferences().defaultDevice;
  });
  const { positions, persistPositions } = useFixturePositions(id);
  const [overrides, setOverrides] = useState<Record<string, Record<string, unknown> | null>>({});
  const [reloads, setReloads] = useState<Record<string, number>>({});
  const [savedSourceRevision, setSavedSourceRevision] = useState<{
    version: string;
    filePath: string;
    sha256: string;
  } | null>(null);
  const [statuses, setStatuses] = useState<
    Record<string, "loading" | "ready" | "error" | "applied">
  >({});
  const [contextReport, setContextReport] = useState<InContextReport | null>(null);
  const [contextSlot, setContextSlot] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [openingWorkspace, setOpeningWorkspace] = useState(false);
  const [workspaceError, setWorkspaceError] = useState<string | null>(null);
  const templateWorkspaces = useWorkspaces();
  const templateWorkspaceQuery = useQuery({
    queryKey: ["wb", "workspace", templateId],
    queryFn: async () => {
      const workspace = await getWorkspace(templateId);
      return workspace?.provenance === WorkspaceProvenance.CURATED ? workspace : null;
    },
    enabled: Boolean(templateId),
    staleTime: 30_000,
    retry: 0,
  });

  const asset = bundle.data?.asset;
  const stories = useMemo(() => bundle.data?.stories ?? [], [bundle.data?.stories]);
  const latestVersion = asset ? assetVersion(asset) : "";
  const activeVersion = version || latestVersion;
  const sourceRevision =
    savedSourceRevision?.version === activeVersion
      ? `${savedSourceRevision.filePath}:${savedSourceRevision.sha256}`
      : "";
  const selectedStory = stories.find((story) => story.id === selectedStoryId) ?? stories[0];
  const template = templateWorkspaceQuery.data
    ? workspaceTemplate(templateWorkspaceQuery.data, device)
    : null;
  const curatedTemplates = (templateWorkspaces.data ?? [])
    .filter((entry) => entry.provenance === WorkspaceProvenance.CURATED)
    .map((entry) => ({ id: entry.id, name: entry.name, summary: entry.summary }));

  useEffect(() => {
    if (!selectedStoryId && stories[0]) setSelectedStoryId(stories[0].id);
  }, [selectedStoryId, stories]);

  useEffect(() => {
    if (!asset || version) return;
    setVersion(assetVersion(asset));
  }, [asset, version]);

  const runEvidence = useMutation({
    mutationFn: () =>
      runComponentTest({ componentId: id, version: activeVersion, includeClosure: false }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(id) });
      setDrawerTab("tests");
      setDrawerOpen(true);
    },
  });

  const checks = useMemo(() => {
    const claims = bundle.data?.experience?.claims ?? [];
    const evidence = bundle.data?.experience?.evidence ?? [];
    return claims.map((claim) => {
      const rows = evidence.filter((row) => row.claimId === claim.id);
      const verdict = rows[0]?.verdict ?? "unmeasured";
      return { claim, verdict, checkedAt: rows[0]?.checkedAt };
    });
  }, [bundle.data?.experience]);

  const passed = checks.filter((check) => check.verdict === "pass").length;
  const checkRatio = checks.length === 0 ? 0 : passed / checks.length;

  // State coverage is the one check that comes from the experience contract
  // rather than from a canvas instrument: a state counts as covered only when
  // a capture names it.
  const contextStates = useMemo(() => {
    const experience = bundle.data?.experience;
    if (!experience || experience.states.length === 0) return null;
    const covered = experience.states.filter((state) =>
      experience.evidence.some((row) => row.stateId === state.id),
    ).length;
    return { total: experience.states.length, covered, status: experience.evidenceStatus };
  }, [bundle.data?.experience]);

  const contextRows = contextReport?.rows ?? [];
  const contextPassed = contextReport?.passed ?? 0;
  const contextScored = contextReport?.scored ?? 0;
  const contextRatio = contextScored === 0 ? 0 : contextPassed / contextScored;

  const updateURL = (next: Record<string, string | undefined>) => {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(next)) {
      if (value) params.set(key, value);
      else params.delete(key);
    }
    setSearchParams(params, { replace: true });
  };

  const chooseMode = (next: Mode) => {
    setMode(next);
    updateURL({ mode: next === "context" ? "context" : undefined });
  };

  const chooseStory = (story: WorkbenchStory) => {
    setSelectedStoryId(story.id);
    updateURL({ story: story.id });
  };

  const openDrawer = (tab: DrawerTab) => {
    setDrawerTab(tab);
    setDrawerOpen(true);
    updateURL({
      tab,
      testReport: tab === "tests" ? (searchParams.get("testReport") ?? undefined) : undefined,
    });
  };

  const openComponentWorkspace = async () => {
    if (!selectedStory || !templateWorkspaceQuery.data || openingWorkspace) return;
    setOpeningWorkspace(true);
    setWorkspaceError(null);
    try {
      const workspace = await createComponentWorkspace({
        templateWorkspace: templateWorkspaceQuery.data,
        componentId: id,
        storyId: selectedStory.id,
        componentVersion: selectedStory.version,
      });
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setWorkspaceError(cause instanceof Error ? cause.message : "Could not open workspace.");
    } finally {
      setOpeningWorkspace(false);
    }
  };

  const share = async () => {
    const url = `${window.location.origin}${appRoutes.preview.replace(":id", encodeURIComponent(id))}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  if (bundle.isLoading) {
    return <ComponentPageLoadState loading onRetry={() => void bundle.refetch()} />;
  }

  if (bundle.isError || !asset) {
    return (
      <ComponentPageLoadState
        loading={false}
        error={bundle.error}
        onRetry={() => void bundle.refetch()}
      />
    );
  }

  return (
    <div data-experience-surface="component-preview" data-testid="component-page">
      {workspaceError ? <Banner tone="red">{workspaceError}</Banner> : null}
      <ComponentPageHeader
        asset={asset}
        latestVersion={latestVersion}
        mode={mode}
        stories={stories}
        selectedStory={selectedStory}
        copied={copied}
        openingWorkspace={openingWorkspace}
        canOpenWorkspace={Boolean(templateWorkspaceQuery.data)}
        runningEvidence={runEvidence.isPending}
        experience={bundle.data?.experience}
        checksCount={checks.length}
        passed={passed}
        checkRatio={checkRatio}
        onShare={() => void share()}
        onOpenWorkspace={() => void openComponentWorkspace()}
        onRunEvidence={() => runEvidence.mutate()}
        onOpenDrawer={openDrawer}
        onChooseMode={chooseMode}
        onChooseStory={chooseStory}
        onRerunStory={() => {
          if (selectedStory)
            setReloads((current) => ({
              ...current,
              [selectedStory.id]: (current[selectedStory.id] ?? 0) + 1,
            }));
        }}
        onToggleDrawer={() => setDrawerOpen((open) => !open)}
      />
      {runEvidence.isError ? (
        <div style={{ padding: "10px 26px" }}>
          <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
            {runEvidence.error instanceof Error
              ? runEvidence.error.message
              : String(runEvidence.error)}
          </Banner>
        </div>
      ) : null}

      <div className={cx("wb-comp-layout", mode === "context" && "is-context")}>
        {mode === "fixtures" ? (
          <FixtureCanvas
            asset={asset}
            stories={stories}
            version={activeVersion}
            sourceRevision={sourceRevision}
            selectedId={selectedStory?.id ?? null}
            onSelect={chooseStory}
            positions={positions}
            onMove={(storyId, position) => persistPositions({ ...positions, [storyId]: position })}
            overrides={overrides}
            reloads={reloads}
            statuses={statuses}
            onStatus={(storyId, status) =>
              setStatuses((current) => ({ ...current, [storyId]: status }))
            }
          />
        ) : template ? (
          <InContextStage
            asset={asset}
            story={selectedStory}
            version={activeVersion}
            template={template}
            templates={curatedTemplates}
            onTemplateChange={(next) => {
              setTemplateId(next);
              setContextReport(null);
              updateURL({ template: next });
            }}
            device={device}
            onDeviceChange={(next) => {
              setDevice(next);
              updateURL({ device: next });
            }}
            slotChoice={contextSlot}
            onSlotChoice={setContextSlot}
            onReport={setContextReport}
            states={contextStates}
          />
        ) : (
          <div className="wb-empty-inline" role="status">
            {templateWorkspaceQuery.isLoading
              ? t("workbench.component.loadingWorkspaceTemplate", {
                  defaultValue: "Loading curated workspace…",
                })
              : t("workbench.component.workspaceTemplateUnavailable", {
                  defaultValue:
                    "Curated workspace unavailable. The saved template reference is preserved in the URL.",
                })}
          </div>
        )}

        <ComponentPageInspector
          asset={asset}
          versionsCount={bundle.data?.versions.length ?? 0}
          mode={mode}
          selectedStory={selectedStory}
          checks={checks}
          checkRatio={checkRatio}
          contextRatio={contextRatio}
          contextScored={contextScored}
          contextRows={contextRows}
          contextReport={contextReport}
          inspectorTab={inspectorTab}
          setInspectorTab={setInspectorTab}
          overrides={overrides}
          setOverrides={setOverrides}
          statuses={statuses}
          template={template}
          openingWorkspace={openingWorkspace}
          canOpenWorkspace={Boolean(templateWorkspaceQuery.data)}
          onOpenWorkspace={() => void openComponentWorkspace()}
        />
      </div>

      {drawerOpen && bundle.data ? (
        <AssetDataDrawer
          bundle={bundle.data}
          tab={drawerTab}
          onTab={(next) => {
            setDrawerTab(next);
            updateURL({
              tab: next,
              testReport:
                next === "tests" ? (searchParams.get("testReport") ?? undefined) : undefined,
            });
          }}
          version={activeVersion}
          onVersion={setVersion}
          onSourceSaved={(savedVersion, filePath, sha256) =>
            setSavedSourceRevision((current) =>
              current?.version === savedVersion &&
              current.filePath === filePath &&
              current.sha256 === sha256
                ? current
                : { version: savedVersion, filePath, sha256 },
            )
          }
          selectedTestReportID={searchParams.get("testReport")}
          onSelectTestReport={(reportID) => {
            setDrawerOpen(true);
            setDrawerTab("tests");
            updateURL({ tab: "tests", testReport: reportID ?? undefined });
          }}
        />
      ) : null}
    </div>
  );
}
