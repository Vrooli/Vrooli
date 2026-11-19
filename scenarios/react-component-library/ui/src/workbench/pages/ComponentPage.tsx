import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowUpRight,
  CircleAlert,
  Layers,
  Play,
  RefreshCw,
  Share2,
  SlidersHorizontal,
  Sparkles,
} from "lucide-react";

import { runComponentTest } from "../../api/componentTests";
import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import {
  assetAdoptions,
  assetVersion,
  formatRelative,
  useComponentBundle,
  wbKeys,
  type WorkbenchStory,
} from "../data";
import { cx } from "../cx";
import { AssetDataDrawer, type DrawerTab } from "../component/AssetDataDrawer";
import { FixtureCanvas, type FixturePosition } from "../component/FixtureCanvas";
import { InContextChecksPanel } from "../component/InContextChecks";
import { VERDICT_ICON, VERDICT_KEY, asVerdict } from "../verdicts";
import { InContextStage, type Device, type InContextReport } from "../component/InContextStage";
import { readWorkbenchPreferences } from "../preferences";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { resolveTemplate, useLocalTemplates } from "../templates/local";
import {
  Banner,
  Chip,
  Dot,
  Empty,
  Ring,
  Segmented,
  Select,
  Skeleton,
  Tabs,
  TextArea,
  WbButton,
} from "../primitives";

type Mode = "fixtures" | "context";
type InspectorTab = "checks" | "props" | "usage";

function positionsKey(id: string): string {
  return `rcl.workbench.fixtures.${id}.v1`;
}

function readPositions(id: string): Record<string, FixturePosition> {
  try {
    const raw = window.localStorage.getItem(positionsKey(id));
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, FixturePosition>;
    return Array.isArray(parsed) ? {} : parsed;
  } catch {
    return {};
  }
}

function parseArgs(story: WorkbenchStory | undefined): Record<string, unknown> {
  return story?.args ?? {};
}

function displayPropValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (value === null || value === undefined) return "";
  return JSON.stringify(value);
}

function normalizeDrawerTab(value: string | null): DrawerTab {
  const tabs: DrawerTab[] = ["overview", "source", "versions", "experience", "tests", "relationships"];
  if (value && (tabs as string[]).includes(value)) return value as DrawerTab;
  return "overview";
}

export function ComponentPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const bundle = useComponentBundle(id);

  const [mode, setMode] = useState<Mode>(searchParams.get("mode") === "context" ? "context" : "fixtures");
  const [selectedStoryId, setSelectedStoryId] = useState<string | null>(searchParams.get("story"));
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>("checks");
  const [drawerOpen, setDrawerOpen] = useState(Boolean(searchParams.get("tab")));
  const [drawerTab, setDrawerTab] = useState<DrawerTab>(() => normalizeDrawerTab(searchParams.get("tab")));
  const [version, setVersion] = useState("");
  const [templateId, setTemplateId] = useState(searchParams.get("template") ?? DEFAULT_TEMPLATE.id);
  const [device, setDevice] = useState<Device>(() => {
    const requested = searchParams.get("device");
    if (requested === "desktop" || requested === "tablet" || requested === "phone") return requested;
    // A desktop frame cannot be inspected on a phone without panning a clipped
    // surface. Default to the frame that matches the viewport; the device
    // switcher still lets a person choose a wider frame deliberately.
    if (typeof window !== "undefined" && window.matchMedia("(max-width: 640px)").matches) {
      return "phone";
    }
    return readWorkbenchPreferences().defaultDevice;
  });
  const [positions, setPositions] = useState<Record<string, FixturePosition>>(() => readPositions(id));
  const [overrides, setOverrides] = useState<Record<string, Record<string, unknown> | null>>({});
  const [reloads, setReloads] = useState<Record<string, number>>({});
  const [statuses, setStatuses] = useState<Record<string, "loading" | "ready" | "error" | "applied">>({});
  const [contextReport, setContextReport] = useState<InContextReport | null>(null);
  const [contextSlot, setContextSlot] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const locals = useLocalTemplates();

  const asset = bundle.data?.asset;
  const stories = useMemo(() => bundle.data?.stories ?? [], [bundle.data?.stories]);
  const latestVersion = asset ? assetVersion(asset) : "";
  const activeVersion = version || latestVersion;
  const selectedStory = stories.find((story) => story.id === selectedStoryId) ?? stories[0];
  const template = resolveTemplate(templateId, locals) ?? DEFAULT_TEMPLATE;

  useEffect(() => {
    setPositions(readPositions(id));
  }, [id]);

  useEffect(() => {
    if (!selectedStoryId && stories[0]) setSelectedStoryId(stories[0].id);
  }, [selectedStoryId, stories]);

  useEffect(() => {
    if (!asset || version) return;
    setVersion(assetVersion(asset));
  }, [asset, version]);

  const saveTimer = useRef<number | undefined>(undefined);
  const persistPositions = (next: Record<string, FixturePosition>) => {
    setPositions(next);
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => {
      try {
        window.localStorage.setItem(positionsKey(id), JSON.stringify(next));
      } catch {
        /* local draft only */
      }
    }, 200);
  };

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
    updateURL({ tab });
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
    return (
      <div className="wb-page">
        <Skeleton height={34} width={240} />
        <div style={{ height: 12 }} />
        <Skeleton height={18} width={520} />
        <div style={{ height: 24 }} />
        <Skeleton height={420} />
      </div>
    );
  }

  if (bundle.isError || !asset) {
    return (
      <div className="wb-page">
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>{t("workbench.component.loadError", { defaultValue: "The component could not be opened." })}</b>{" "}
          {bundle.error instanceof Error ? bundle.error.message : ""}
        </Banner>
        <div style={{ marginTop: 14 }}>
          <WbButton variant="ghost" size="md" onClick={() => void bundle.refetch()}>
            <RefreshCw aria-hidden size={14} />
            {t("workbench.component.retry", { defaultValue: "Retry" })}
          </WbButton>
        </div>
      </div>
    );
  }

  const breadcrumb = (asset.catalogId || asset.libraryId).split(".").filter(Boolean);

  return (
    <div data-experience-surface="component-preview" data-testid="component-page">
      <header className="wb-comp-head">
        <div className="wb-row" style={{ flexWrap: "wrap", gap: 6 }}>
          <Link to={appRoutes.components} className="wb-muted wb-small">
            {t("workbench.component.breadcrumbComponents", { defaultValue: "Components" })}
          </Link>
          {breadcrumb.map((part) => (
            <span key={part} className="wb-row" style={{ gap: 6 }}>
              <span className="wb-muted-2">/</span>
              <span className="wb-muted wb-small">{part}</span>
            </span>
          ))}
        </div>

        <div className="wb-comp-title-row">
          <div style={{ minWidth: 0 }}>
            <h1 className="wb-comp-title">{asset.displayName}</h1>
            <p className="wb-comp-desc">{asset.description}</p>
          </div>
          <div className="wb-actions">
            <WbButton variant="ghost" size="md" onClick={() => void share()}>
              <Share2 aria-hidden size={14} />
              {copied
                ? t("workbench.component.copied", { defaultValue: "Copied" })
                : t("workbench.component.share", { defaultValue: "Share" })}
            </WbButton>
            <WbButton
              variant="ghost"
              size="md"
              onClick={() =>
                void navigate(
                  `${appRoutes.design}?template=${templateId}&component=${encodeURIComponent(id)}${
                    selectedStory ? `&story=${encodeURIComponent(selectedStory.id)}` : ""
                  }`,
                )
              }
            >
              <Sparkles aria-hidden size={14} />
              {t("workbench.component.useInDesign", { defaultValue: "Use in design" })}
            </WbButton>
            <WbButton variant="primary" size="md" onClick={() => runEvidence.mutate()} disabled={runEvidence.isPending}>
              <Play aria-hidden size={14} />
              {runEvidence.isPending
                ? t("workbench.component.running", { defaultValue: "Running…" })
                : t("workbench.component.runEvidence", { defaultValue: "Run evidence" })}
            </WbButton>
          </div>
        </div>

        <div className="wb-comp-meta">
          <Chip tone="ice">{`v${latestVersion}`}</Chip>
          <Chip>{asset.slot || t("workbench.component.noSlot", { defaultValue: "no slot" })}</Chip>
          <Chip>
            {t("workbench.component.direct", {
              defaultValue: "{{count}} direct",
              count: assetAdoptions(asset),
            })}
          </Chip>
          <Chip>
            {t("workbench.component.effective", {
              defaultValue: "{{count}} transitive",
              count: asset.metrics?.effectiveAdoptionCount ?? 0,
            })}
          </Chip>
          {bundle.data?.experience ? (
            <Chip tone={bundle.data.experience.evidenceStatus === "fresh" ? "green" : "amber"}>
              <Dot tone={bundle.data.experience.evidenceStatus === "fresh" ? "green" : "amber"} />
              {bundle.data.experience.evidenceStatus}
            </Chip>
          ) : (
            <Chip tone="amber">{t("workbench.component.unmeasured", { defaultValue: "unmeasured" })}</Chip>
          )}
          <span className="wb-muted-2">·</span>
          <button type="button" className="wb-link" onClick={() => openDrawer("source")}>
            {t("workbench.component.source", { defaultValue: "Source" })}
          </button>
          <button type="button" className="wb-link" onClick={() => openDrawer("versions")}>
            {t("workbench.component.versions", { defaultValue: "Versions" })}
          </button>
          <button type="button" className="wb-link" onClick={() => openDrawer("experience")}>
            {t("workbench.component.experience", { defaultValue: "Experience" })}
          </button>
          <button type="button" className="wb-link" onClick={() => openDrawer("tests")}>
            {t("workbench.component.tests", { defaultValue: "Tests" })}
          </button>
          <button type="button" className="wb-link" onClick={() => openDrawer("relationships")}>
            {t("workbench.component.relationships", { defaultValue: "Relationships" })}
          </button>
        </div>
      </header>

      <div className="wb-modebar">
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <Segmented
            label={t("workbench.component.mode", { defaultValue: "Component mode" })}
            value={mode}
            onChange={chooseMode}
            items={[
              { id: "fixtures", label: t("workbench.component.fixtures", { defaultValue: "Fixtures" }) },
              { id: "context", label: t("workbench.component.inContext", { defaultValue: "In context" }) },
            ]}
          />
          <Select
            aria-label={t("workbench.component.story", { defaultValue: "Story" })}
            value={selectedStory?.id ?? ""}
            onChange={(event) => {
              const story = stories.find((entry) => entry.id === event.target.value);
              if (story) chooseStory(story);
            }}
            style={{ width: "auto", minWidth: 170 }}
          >
            {stories.length === 0 ? <option value="">{t("workbench.component.noStory", { defaultValue: "No stories" })}</option> : null}
            {stories.map((story) => (
              <option key={story.id} value={story.id}>
                {story.name || story.id}
              </option>
            ))}
          </Select>
          <button
            type="button"
            className="wb-btn ghost sm"
            onClick={() => {
              if (!selectedStory) return;
              setReloads((current) => ({ ...current, [selectedStory.id]: (current[selectedStory.id] ?? 0) + 1 }));
            }}
          >
            <RefreshCw aria-hidden size={13} />
            {t("workbench.component.rerun", { defaultValue: "Re-run" })}
          </button>
        </div>
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <Chip tone={checkRatio === 1 && checks.length > 0 ? "green" : "amber"}>
            {checks.length === 0
              ? t("workbench.component.noChecks", { defaultValue: "No checks" })
              : t("workbench.component.checksSummary", {
                  defaultValue: "{{passed}} pass · {{total}} checks",
                  passed,
                  total: checks.length,
                })}
          </Chip>
          <button type="button" className="wb-btn ghost sm" onClick={() => setDrawerOpen((open) => !open)}>
            <Layers aria-hidden size={13} />
            {t("workbench.component.assetData", { defaultValue: "Asset data" })}
          </button>
        </div>
      </div>

      {runEvidence.isError ? (
        <div style={{ padding: "10px 26px" }}>
          <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
            {runEvidence.error instanceof Error ? runEvidence.error.message : String(runEvidence.error)}
          </Banner>
        </div>
      ) : null}

      <div className={cx("wb-comp-layout", mode === "context" && "is-context")}>
        {mode === "fixtures" ? (
          <FixtureCanvas
            asset={asset}
            stories={stories}
            version={activeVersion}
            selectedId={selectedStory?.id ?? null}
            onSelect={chooseStory}
            positions={positions}
            onMove={(storyId, position) =>
              persistPositions({ ...positions, [storyId]: position })
            }
            overrides={overrides}
            reloads={reloads}
            statuses={statuses}
            onStatus={(storyId, status) =>
              setStatuses((current) => ({ ...current, [storyId]: status }))
            }
          />
        ) : (
          <InContextStage
            asset={asset}
            story={selectedStory}
            version={activeVersion}
            templateId={templateId}
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
        )}

        <aside className="wb-inspector" aria-label={t("workbench.component.inspector", { defaultValue: "Inspector" })}>
          <div className="wb-inspector-head">
            <div style={{ minWidth: 0 }}>
              <span className="wb-caption">
                {mode === "fixtures"
                  ? t("workbench.component.selectedFixture", { defaultValue: "Selected fixture" })
                  : t("workbench.component.inContextSettings", { defaultValue: "In-context settings" })}
              </span>
              <h2>{mode === "fixtures" ? selectedStory?.name || selectedStory?.id || "—" : asset.displayName}</h2>
            </div>
            <Ring
              value={mode === "fixtures" ? checkRatio * 100 : contextRatio * 100}
              size={58}
              tone={
                (mode === "fixtures" ? checkRatio : contextRatio) >= 1 &&
                (mode === "fixtures" ? checks.length : contextScored) > 0
                  ? "green"
                  : "amber"
              }
            />
          </div>
          <div className="wb-inspector-tabs">
            <Tabs
              label={t("workbench.component.inspectorTabs", { defaultValue: "Inspector sections" })}
              value={inspectorTab}
              onChange={setInspectorTab}
              items={[
                { id: "checks", label: t("workbench.component.checks", { defaultValue: "Checks" }) },
                { id: "props", label: t("workbench.component.props", { defaultValue: "Props" }) },
                {
                  id: "usage",
                  label:
                    mode === "fixtures"
                      ? t("workbench.component.usageTab", { defaultValue: "Usage" })
                      : t("workbench.component.pageTab", { defaultValue: "Page" }),
                },
              ]}
            />
          </div>
          <div className="wb-inspector-body">
            {inspectorTab === "checks" ? (
              mode === "fixtures" ? (
                <ChecksList
                  rows={checks.map((check) => ({
                    id: check.claim.id,
                    label: check.claim.statement,
                    verdict: check.verdict,
                    detail: check.checkedAt ? formatRelative(new Date(check.checkedAt)) : undefined,
                  }))}
                  empty={t("workbench.component.noChecksBody", {
                    defaultValue:
                      "This asset has no experience claims. Nothing is shown as a passing check.",
                  })}
                />
              ) : (
                <InContextChecksPanel rows={contextRows} />
              )
            ) : null}

            {inspectorTab === "props" && selectedStory ? (
              <PropsEditor
                story={selectedStory}
                override={overrides[selectedStory.id] ?? null}
                onApply={(next) => setOverrides((current) => ({ ...current, [selectedStory.id]: next }))}
                onReset={() => setOverrides((current) => ({ ...current, [selectedStory.id]: null }))}
                status={statuses[selectedStory.id] ?? "loading"}
              />
            ) : null}

            {inspectorTab === "usage" ? (
              mode === "fixtures" ? (
                <div className="wb-stack tight">
                  <div className="wb-row-between">
                    <span className="wb-caption">{t("workbench.component.catalogId", { defaultValue: "Catalog id" })}</span>
                    <span className="wb-mono wb-small">{asset.catalogId || "—"}</span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-caption">{t("workbench.component.source", { defaultValue: "Source" })}</span>
                    <span className="wb-mono wb-small" style={{ textAlign: "right", overflowWrap: "anywhere" }}>
                      {asset.sourcePath}
                    </span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-caption">{t("workbench.component.adoptions", { defaultValue: "Adoptions" })}</span>
                    <span className="wb-mono wb-small">{assetAdoptions(asset)}</span>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-caption">{t("workbench.component.versionsCount", { defaultValue: "Versions" })}</span>
                    <span className="wb-mono wb-small">{bundle.data?.versions.length ?? 0}</span>
                  </div>
                  <div className="wb-divider" />
                  <p className="wb-small wb-muted">
                    {t("workbench.component.fixtureNote", {
                      defaultValue:
                        "Drag a fixture by its header to arrange the canvas. Select a fixture to inspect it. Arrangement is a local draft and never changes a release.",
                    })}
                  </p>
                </div>
              ) : (
                <div className="wb-stack tight">
                  <span className="wb-caption">{t("workbench.component.templateIdea", { defaultValue: "Design idea" })}</span>
                  <p className="wb-small wb-muted">{template.idea}</p>
                  <span className="wb-caption">{t("workbench.component.slotHint", { defaultValue: "Slot" })}</span>
                  <p className="wb-small wb-muted">{template.slotHint}</p>
                  <div className="wb-divider" />
                  <span className="wb-caption">{t("workbench.component.placement", { defaultValue: "Placement" })}</span>
                  {contextReport?.decision.chosen ? (
                    <>
                      <div className="wb-row-between">
                        <span className="wb-small wb-muted">{t("workbench.component.ctxChosenSlot", { defaultValue: "Chosen slot" })}</span>
                        <span className="wb-small">{contextReport.decision.chosen.label}</span>
                      </div>
                      <div className="wb-row-between">
                        <span className="wb-small wb-muted">{t("workbench.component.ctxSlotRole", { defaultValue: "Role" })}</span>
                        <span className="wb-small wb-mono">{contextReport.decision.chosen.role ?? "open"}</span>
                      </div>
                      <div className="wb-row-between">
                        <span className="wb-small wb-muted">{t("workbench.component.ctxAssetKind", { defaultValue: "Asset kind" })}</span>
                        <span className="wb-small wb-mono">{contextReport.decision.profile.kind}</span>
                      </div>
                      <div className="wb-row-between">
                        <span className="wb-small wb-muted">{t("workbench.component.ctxScore", { defaultValue: "Affinity" })}</span>
                        <span className="wb-small wb-mono">{contextReport.decision.chosen.score}%</span>
                      </div>
                      <div className="wb-row-between">
                        <span className="wb-small wb-muted">{t("workbench.component.ctxViewport", { defaultValue: "Viewport" })}</span>
                        <span className="wb-small wb-mono">
                          {contextReport.viewport} · {contextReport.device}
                        </span>
                      </div>
                    </>
                  ) : (
                    <p className="wb-small wb-muted">
                      {t("workbench.component.ctxPlacementPending", { defaultValue: "Placing the component…" })}
                    </p>
                  )}
                  <div className="wb-divider" />
                  <WbButton
                    variant="primary"
                    size="md"
                    onClick={() =>
                      void navigate(
                        `${appRoutes.design}?template=${templateId}&component=${encodeURIComponent(id)}${
                          selectedStory ? `&story=${encodeURIComponent(selectedStory.id)}` : ""
                        }`,
                      )
                    }
                  >
                    {t("workbench.component.takeIntoStudio", { defaultValue: "Take into studio" })}
                    <ArrowRight aria-hidden size={14} />
                  </WbButton>
                  <Link to={appRoutes.design} className="wb-btn ghost sm wide">
                    {t("workbench.component.openStudio", { defaultValue: "Open design studio" })}
                    <ArrowUpRight aria-hidden size={13} />
                  </Link>
                </div>
              )
            ) : null}
          </div>
        </aside>
      </div>

      {drawerOpen && bundle.data ? (
        <AssetDataDrawer
          bundle={bundle.data}
          tab={drawerTab}
          onTab={(next) => {
            setDrawerTab(next);
            updateURL({ tab: next });
          }}
          version={activeVersion}
          onVersion={setVersion}
        />
      ) : null}
    </div>
  );
}

function ChecksList({
  rows,
  empty,
}: {
  rows: Array<{ id: string; label: string; verdict: string; detail?: string }>;
  empty: string;
}) {
  const { t } = useTranslation();
  if (rows.length === 0) {
    return <Empty title={t("workbench.component.noChecksTitle", { defaultValue: "No checks declared" })}>{empty}</Empty>;
  }
  return (
    <div>
      {rows.map((row) => {
        const verdict = asVerdict(row.verdict);
        const Icon = VERDICT_ICON[verdict];
        return (
          <div key={row.id} className={`wb-checkrow is-${verdict}`}>
            <span aria-hidden className="wb-checkrow-icon">
              <Icon size={12} />
            </span>
            <span title={row.label}>{row.label}</span>
            <b className="wb-checkrow-verdict">
              {t(VERDICT_KEY[verdict], { defaultValue: verdict })}
            </b>
          </div>
        );
      })}
      <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
        {t("workbench.component.checksNote", {
          defaultValue:
            "Checks are captured evidence. An absent capture is unmeasured, not a pass.",
        })}
      </p>
    </div>
  );
}

function PropsEditor({
  story,
  override,
  onApply,
  onReset,
  status,
}: {
  story: WorkbenchStory;
  override: Record<string, unknown> | null;
  onApply: (next: Record<string, unknown>) => void;
  onReset: () => void;
  status: string;
}) {
  const { t } = useTranslation();
  const baseline = useMemo(() => parseArgs(story), [story]);
  const [draft, setDraft] = useState(() => JSON.stringify(override ?? baseline, null, 2));
  const [error, setError] = useState("");

  useEffect(() => {
    setDraft(JSON.stringify(override ?? baseline, null, 2));
    setError("");
  }, [baseline, override, story.id]);

  const entries = Object.entries(override ?? baseline);

  const setValue = (key: string, value: unknown) => {
    const next = { ...(override ?? baseline), [key]: value };
    setDraft(JSON.stringify(next, null, 2));
    setError("");
  };

  const apply = () => {
    try {
      const parsed = JSON.parse(draft) as unknown;
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        setError(t("workbench.component.propsObject", { defaultValue: "Props must be a JSON object." }));
        return;
      }
      onApply(parsed as Record<string, unknown>);
    } catch {
      setError(t("workbench.component.propsParse", { defaultValue: "That is not valid JSON." }));
    }
  };

  return (
    <div className="wb-stack tight">
      <div className="wb-row-between">
        <span className="wb-caption">{t("workbench.component.publicProps", { defaultValue: "Public props" })}</span>
        <Chip tone={status === "applied" ? "green" : status === "error" ? "red" : "ice"}>{status}</Chip>
      </div>
      {entries.length === 0 ? (
        <p className="wb-small wb-muted">
          {t("workbench.component.noProps", { defaultValue: "This story declares no public props." })}
        </p>
      ) : (
        entries.map(([key, value]) => (
          <div key={key} className="wb-field">
            <label htmlFor={`prop-${story.id}-${key}`}>{key}</label>
            {typeof value === "boolean" ? (
              <Select
                id={`prop-${story.id}-${key}`}
                value={String(value)}
                onChange={(event) => setValue(key, event.target.value === "true")}
              >
                <option value="true">{String(true)}</option>
                <option value="false">{String(false)}</option>
              </Select>
            ) : typeof value === "number" ? (
              <input
                id={`prop-${story.id}-${key}`}
                className="wb-input"
                type="number"
                value={value}
                onChange={(event) => setValue(key, Number(event.target.value))}
              />
            ) : (
              <input
                id={`prop-${story.id}-${key}`}
                className="wb-input"
                value={displayPropValue(value)}
                onChange={(event) => setValue(key, event.target.value)}
              />
            )}
          </div>
        ))
      )}
      <div className="wb-field">
        <label htmlFor={`props-json-${story.id}`}>{t("workbench.component.propsJson", { defaultValue: "Override JSON" })}</label>
        <TextArea
          id={`props-json-${story.id}`}
          value={draft}
          onChange={(event) => {
            setDraft(event.target.value);
            setError("");
          }}
          spellCheck={false}
        />
      </div>
      {error ? <span className="wb-small" style={{ color: "var(--wb-red)" }}>{error}</span> : null}
      <div className="wb-row">
        <WbButton variant="primary" size="sm" onClick={apply}>
          <SlidersHorizontal aria-hidden size={13} />
          {t("workbench.component.applyProps", { defaultValue: "Apply props" })}
        </WbButton>
        <WbButton variant="ghost" size="sm" onClick={onReset}>
          {t("workbench.component.resetProps", { defaultValue: "Reset" })}
        </WbButton>
      </div>
      <p className="wb-muted-2 wb-small">
        {t("workbench.component.propsNote", {
          defaultValue:
            "Overrides are applied to the real harness. They are a local experiment and do not publish a version.",
        })}
      </p>
    </div>
  );
}
