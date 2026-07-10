import { useState, useCallback, useMemo } from "react";
import { ClipboardCheck, Loader2, AlertTriangle, ChevronDown, X, Camera, Settings } from "lucide-react";
import { Button } from "./ui/button";
import { buildCaptureScreenshotUrl, presetSuffix, presetLabel, presetKey } from "../lib/api";
import type { CapturePreset, CaptureTheme, SnapshotSetMeta, SnapshotStalenessInfo, AgentContextItem } from "../lib/api";
import { AttachToAgentButton } from "./AgentTab";
import { screenshotContextItem } from "../lib/agentContext";
import { Popover } from "./ui/popover";
import { MediaLightbox, MutationErrorBanner, sanitizePagePath, type LightboxItem } from "./ScenarioReviewPanelShared";
import { PresetConfigPanel, ScreenshotImage } from "./ScenarioReviewPanelPresets";
import { useBaselines, useDefaultBaseline, type CompareOnDemand } from "../lib/hooks-baselines";
import { useVisualCaptureDetail } from "../lib/hooks";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { SurfaceBaselineBar } from "../features/baselines/SurfaceBaselineBar";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";

export function ScreenshotsTab({
  capture,
  captureStaleness,
  scenarioSlug,
  repoId,
  isMobile,
  basAvailable,
  isCapturing,
  onCapture,
  presetConfig,
  onPresetConfigChange,
  mutationError,
  onDismissError,
  agentManagerAvailable,
  onAttachToAgent,
  onOpenBaselines,
  initialPresetIndex,
  initialSelectedPage,
  onPresetIndexChange,
  onSelectedPageChange,
}: {
  capture?: SnapshotSetMeta;
  captureStaleness?: SnapshotStalenessInfo;
  scenarioSlug: string;
  repoId?: string | null;
  isMobile: boolean;
  basAvailable: boolean;
  isCapturing: boolean;
  onCapture: () => void;
  presetConfig: CapturePreset[];
  onPresetConfigChange: (presets: CapturePreset[]) => void;
  mutationError?: Error | null;
  onDismissError?: () => void;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  onOpenBaselines: () => void;
  initialPresetIndex?: number;
  initialSelectedPage?: number;
  onPresetIndexChange?: (index: number) => void;
  onSelectedPageChange?: (page: number) => void;
}) {
  const [selectedPage, setSelectedPageInternal] = useState(initialSelectedPage ?? 0);
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  const [mobileDropdownOpen, setMobileDropdownOpen] = useState(false);
  const [configOpen, setConfigOpen] = useState(false);
  const [comparing, setComparing] = useState(false);

  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, repoId);

  // Compare mode for visuals shows the selected baseline's PINNED screenshots
  // beside the current loose capture (Plan C Phase 4). It is instantaneous —
  // it resolves the "before" snapshot from the manifest's visuals pointer
  // rather than re-running a server diff (unlike tests/rules/workflows), so it
  // drives SurfaceBaselineBar via a local CompareOnDemand handle.
  const baselinesQuery = useBaselines(scenarioSlug, { repoId });
  const { defaultBaselineName } = useDefaultBaseline(scenarioSlug);
  const selectedBaseline = baselinesQuery.data?.find((b) => b.name === defaultBaselineName);
  const baselineSnapshotId = selectedBaseline?.run?.runId ?? "";
  const showCompare = comparing && Boolean(baselineSnapshotId);
  const beforeDetail = useVisualCaptureDetail(baselineSnapshotId, scenarioSlug, showCompare, repoId);
  const before: SnapshotSetMeta | undefined = showCompare ? beforeDetail.data : undefined;

  const compare: CompareOnDemand = {
    comparing,
    start: () => setComparing(true),
    exit: () => setComparing(false),
    baselineName: defaultBaselineName ?? "",
    diff: undefined,
    isRunning: false,
    error: null,
  };

  const setSelectedPage = useCallback((page: number) => {
    setSelectedPageInternal(page);
    onSelectedPageChange?.(page);
  }, [onSelectedPageChange]);

  const after = capture;
  // Presets/pages come from whichever snapshot is the anchor for the page grid.
  const primarySnapshot = after ?? before;
  const capturedPresets = useMemo<CapturePreset[]>(
    () => primarySnapshot?.presets ?? [],
    [primarySnapshot],
  );

  const defaultPreset = capturedPresets[0] ?? presetConfig[0] ?? { name: "Desktop Light", width: 1440, height: 900, theme: "light" as CaptureTheme };
  // Active preset for filtering — restore from per-scenario state or use first available
  const [activePreset, setActivePresetInternal] = useState<CapturePreset>(
    () => {
      if (initialPresetIndex !== undefined && initialPresetIndex >= 0) {
        return capturedPresets[initialPresetIndex] ?? presetConfig[initialPresetIndex] ?? defaultPreset;
      }
      return defaultPreset;
    }
  );
  const setActivePreset = useCallback((preset: CapturePreset) => {
    setActivePresetInternal(preset);
    // Find the index and report upward
    const allPresets = capturedPresets.length > 0 ? capturedPresets : presetConfig;
    const idx = allPresets.findIndex(p => presetKey(p) === presetKey(preset));
    if (idx >= 0) onPresetIndexChange?.(idx);
  }, [capturedPresets, presetConfig, onPresetIndexChange]);

  // No current capture yet → two-action empty state (Decision 2).
  if (!after) {
    return (
      <div className="space-y-4">
        <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
        {baselineModal}
        <SurfaceCaptureEmptyState
          surface="visuals"
          hasService={basAvailable}
          onCaptureLoose={onCapture}
          onCaptureBaseline={openCaptureBaseline}
          captureLabel="Capture screenshots"
          isCapturing={isCapturing}
          serviceMessage="Start browser-automation-studio to enable visual captures"
          icon={<ClipboardCheck className="h-8 w-8 mb-3 opacity-50" />}
        />
      </div>
    );
  }

  const primaryPages = primarySnapshot?.pages ?? [];
  const currentPage = primaryPages[selectedPage] ?? "/";

  // Build filename for current preset
  const screenshotFilename = (pagePath: string) =>
    sanitizePagePath(pagePath) + presetSuffix(activePreset) + ".png";

  // Build lightbox items for the active preset: baseline (before) first, then current (after).
  const lightboxItems: LightboxItem[] = [];
  const beforePages = before?.pages ?? [];
  if (before) {
    for (const page of beforePages) {
      lightboxItems.push({
        label: `Baseline: ${page === "/" ? "/ (Home)" : page}`,
        sublabel: `${new Date(before.createdAt).toLocaleString()} (${presetLabel(activePreset)})`,
        type: "image",
        url: buildCaptureScreenshotUrl(before.id, scenarioSlug, screenshotFilename(page)),
      });
    }
  }
  const afterPages = after.pages ?? [];
  for (const page of afterPages) {
    lightboxItems.push({
      label: before ? `Current: ${page === "/" ? "/ (Home)" : page}` : page === "/" ? "/ (Home)" : page,
      sublabel: `${new Date(after.createdAt).toLocaleString()} (${presetLabel(activePreset)})`,
      type: "image",
      url: buildCaptureScreenshotUrl(after.id, scenarioSlug, screenshotFilename(page)),
    });
  }

  const beforeIndex = (pageIdx: number) => pageIdx;
  const afterIndex = (pageIdx: number) => beforePages.length + pageIdx;

  // Capture time estimate
  const numPages = primaryPages.length || 1;
  const numPresets = presetConfig.length || 1;
  const estimatedSeconds = numPages * numPresets * 3;

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
      {baselineModal}

      <SurfaceBaselineBar
        scenario={scenarioSlug}
        repoId={repoId}
        compare={compare}
        onOpenBaselines={onOpenBaselines}
        onCaptureBaseline={openCaptureBaseline}
        viewingLabel={`captured ${new Date(after.createdAt).toLocaleString()}`}
      />

      {comparing && !baselineSnapshotId && (
        <p className="rounded-lg border border-dashed border-slate-800 px-3 py-2 text-xs text-slate-500">
          The selected baseline didn't capture screenshots. Pick another baseline or capture one.
        </p>
      )}
      {showCompare && beforeDetail.isLoading && (
        <div className="flex items-center gap-2 text-xs text-slate-500">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Loading baseline screenshots…
        </div>
      )}

      {/* Capture action */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1">
          {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
          Re-capture
        </Button>
      </div>

      {/* Capture time estimate */}
      {(numPresets > 1 || numPages > 1) && (
        <p className="text-[10px] text-slate-600">
          ~{estimatedSeconds}s for {numPresets} preset{numPresets > 1 ? "s" : ""} &times; {numPages} page{numPages > 1 ? "s" : ""}
        </p>
      )}

      {/* Preset filter — desktop: chips + gear, mobile: dropdown */}
      {capturedPresets.length > 0 && (
        isMobile ? (
          <div className="relative">
            <button
              type="button"
              onClick={() => setMobileDropdownOpen(v => !v)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-800 text-xs text-slate-300 hover:text-white transition-colors"
            >
              {presetLabel(activePreset)}
              <ChevronDown className="h-3 w-3" />
            </button>
            {mobileDropdownOpen && (
              <div className="absolute left-0 top-full mt-1 z-50 min-w-[180px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl py-1">
                {capturedPresets.map(p => (
                  <button
                    key={presetKey(p)}
                    type="button"
                    onClick={() => { setActivePreset(p); setMobileDropdownOpen(false); }}
                    className={`block w-full text-left px-3 py-1.5 text-xs transition-colors ${
                      presetKey(p) === presetKey(activePreset)
                        ? "bg-blue-600 text-white"
                        : "text-slate-400 hover:text-slate-200 hover:bg-slate-800"
                    }`}
                  >
                    {presetLabel(p)}
                  </button>
                ))}
                <div className="border-t border-slate-700 mt-1 pt-1">
                  <button
                    type="button"
                    onClick={() => { setMobileDropdownOpen(false); setConfigOpen(true); }}
                    className="block w-full text-left px-3 py-1.5 text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800"
                  >
                    <Settings className="h-3 w-3 inline mr-1.5" />
                    Configure presets...
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-1.5">
            {capturedPresets.map(p => (
              <button
                key={presetKey(p)}
                type="button"
                onClick={() => setActivePreset(p)}
                className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                  presetKey(p) === presetKey(activePreset)
                    ? "bg-blue-600 text-white"
                    : "bg-slate-800 text-slate-400 hover:text-slate-200"
                }`}
              >
                {presetLabel(p)}
              </button>
            ))}
            <Popover
              trigger={<Settings className="h-3.5 w-3.5 text-slate-500 hover:text-slate-300 transition-colors" />}
              align="end"
            >
              <PresetConfigPanel
                config={presetConfig}
                onChange={onPresetConfigChange}
              />
            </Popover>
          </div>
        )
      )}

      {/* Preset config modal for mobile (triggered from dropdown) */}
      {configOpen && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/60" onClick={() => setConfigOpen(false)}>
          <div className="bg-slate-900 border border-slate-700 rounded-t-xl sm:rounded-xl w-full sm:max-w-sm p-4" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-medium text-slate-200">Configure Presets</span>
              <button type="button" onClick={() => setConfigOpen(false)} className="text-slate-500 hover:text-slate-300"><X className="h-4 w-4" /></button>
            </div>
            <PresetConfigPanel config={presetConfig} onChange={onPresetConfigChange} />
          </div>
        </div>
      )}

      {/* Staleness warning */}
      {captureStaleness?.isStale && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Files have changed since this capture. Re-capture to see the latest state.
          </p>
        </div>
      )}

      {/* Fallback warning */}
      {primarySnapshot?.pageDiscoveryMethod === "fallback" && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
          </p>
        </div>
      )}

      {/* Page selector */}
      {primaryPages.length > 1 && (
        <div className="flex gap-1.5 overflow-x-auto pb-2">
          {primaryPages.map((page, i) => (
            <button
              key={page}
              type="button"
              onClick={() => setSelectedPage(i)}
              className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                i === selectedPage
                  ? "bg-blue-600 text-white"
                  : "bg-slate-800 text-slate-400 hover:text-slate-200"
              }`}
            >
              {page === "/" ? "/ (Home)" : page}
            </button>
          ))}
        </div>
      )}

      {/* Current page URL label */}
      <div className="text-xs text-slate-500">
        Page: <code className="bg-slate-800 px-1.5 py-0.5 rounded text-slate-300">{currentPage}</code>
        {currentPage === "/" && " (Home)"}
      </div>

      {/* Side-by-side (compare) or single (current) */}
      <div className={`gap-4 ${isMobile ? "space-y-4" : before ? "grid grid-cols-2" : ""}`}>
        {before && (
          <div>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs font-medium text-slate-400">Baseline</span>
              <span className="text-[10px] text-slate-600">
                {new Date(before.createdAt).toLocaleString()}
              </span>
              {agentManagerAvailable && onAttachToAgent && (
                <AttachToAgentButton onClick={() => {
                  const filename = screenshotFilename(currentPage);
                  onAttachToAgent(screenshotContextItem(before, {
                    filename,
                    pagePath: currentPage,
                    pageLabel: currentPage === "/" ? "/ (Home)" : currentPage,
                    viewportWidth: activePreset.width,
                    viewportHeight: activePreset.height,
                    theme: activePreset.theme,
                    sizeBytes: 0,
                  }));
                }} />
              )}
            </div>
            <ScreenshotImage
              captureId={before.id}
              scenarioSlug={scenarioSlug}
              pagePath={currentPage}
              preset={activePreset}
              onClick={() => setLightboxIndex(beforeIndex(selectedPage))}
            />
          </div>
        )}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-medium text-slate-400">
              {before ? "Current" : "Screenshot"}
            </span>
            {captureStaleness?.isStale && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                Stale
              </span>
            )}
            {before && (
              <span className="text-[10px] text-slate-600">
                {new Date(after.createdAt).toLocaleString()}
              </span>
            )}
            {agentManagerAvailable && onAttachToAgent && (
              <AttachToAgentButton onClick={() => {
                const filename = screenshotFilename(currentPage);
                onAttachToAgent(screenshotContextItem(after, {
                  filename,
                  pagePath: currentPage,
                  pageLabel: currentPage === "/" ? "/ (Home)" : currentPage,
                  viewportWidth: activePreset.width,
                  viewportHeight: activePreset.height,
                  theme: activePreset.theme,
                  sizeBytes: 0,
                }));
              }} />
            )}
          </div>
          <ScreenshotImage
            captureId={after.id}
            scenarioSlug={scenarioSlug}
            pagePath={currentPage}
            preset={activePreset}
            onClick={() => setLightboxIndex(before ? afterIndex(selectedPage) : beforeIndex(selectedPage))}
          />
        </div>
      </div>

      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </div>
  );
}
