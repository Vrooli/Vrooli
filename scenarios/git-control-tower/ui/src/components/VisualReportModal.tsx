import { useState } from "react";
import { X, Camera, RefreshCw, Loader2, Construction } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import { useVisualCaptures, useTriggerVisualCapture } from "../lib/hooks";
import { buildCaptureScreenshotUrl, buildCaptureVideoUrl } from "../lib/api";
import type { SnapshotSetMeta } from "../lib/api";

type Tab = "screenshots" | "videos" | "tests";

interface VisualReportModalProps {
  isOpen: boolean;
  onClose: () => void;
  scenarioSlug: string;
  repoId?: string | null;
}

export function VisualReportModal({ isOpen, onClose, scenarioSlug, repoId }: VisualReportModalProps) {
  const isMobile = useIsMobile();
  const [activeTab, setActiveTab] = useState<Tab>("screenshots");
  const capturesQuery = useVisualCaptures(scenarioSlug, isOpen, repoId);
  const triggerCapture = useTriggerVisualCapture(repoId);

  if (!isOpen) return null;

  const snapshots = capturesQuery.data?.snapshots ?? [];
  const completeSnapshots = snapshots.filter(s => s.status === "complete");
  const after = completeSnapshots[0] as SnapshotSetMeta | undefined;
  const before = completeSnapshots[1] as SnapshotSetMeta | undefined;

  const isCapturing = triggerCapture.isPending;

  const tabLabels: Record<Tab, string> = {
    screenshots: "Screenshots",
    videos: "Videos",
    tests: "Tests",
  };

  const content = (
    <>
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
        <div className="flex items-center gap-2">
          <Camera className="h-4 w-4 text-slate-400" />
          <div>
            <h2 className={`font-semibold text-slate-100 ${isMobile ? "text-base" : "text-sm"}`}>
              {scenarioSlug}
            </h2>
            {after && (
              <p className="text-[11px] text-slate-500 mt-0.5">
                Last captured: {new Date(after.createdAt).toLocaleString()}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="h-8 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
            onClick={() => triggerCapture.mutate(scenarioSlug)}
            disabled={isCapturing}
            title="Re-capture"
          >
            {isCapturing ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
          </button>
          <button
            type="button"
            className={`inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 ${
              isMobile ? "h-11 w-11" : "h-8 w-8"
            }`}
            onClick={onClose}
            aria-label="Close"
          >
            <X className={isMobile ? "h-5 w-5" : "h-4 w-4"} />
          </button>
        </div>
      </div>

      {/* Capture in progress banner */}
      {isCapturing && (
        <div className="flex items-center gap-2 px-4 py-2 bg-blue-950/50 border-b border-blue-900/50 text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Capturing screenshots...
        </div>
      )}

      {/* Tab Navigation */}
      <div className="flex border-b border-slate-800 px-4">
        {(Object.keys(tabLabels) as Tab[]).map((tab) => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-xs font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? "text-blue-400 border-blue-400"
                : "text-slate-400 border-transparent hover:text-slate-200"
            }`}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-4 py-4">
        {capturesQuery.isLoading ? (
          <div className="space-y-4">
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
          </div>
        ) : capturesQuery.error ? (
          <p className="text-red-400 text-sm">{capturesQuery.error.message}</p>
        ) : activeTab === "screenshots" ? (
          <ScreenshotsTab
            before={before}
            after={after}
            scenarioSlug={scenarioSlug}
            isMobile={isMobile}
          />
        ) : activeTab === "videos" ? (
          <VideosTab before={before} after={after} scenarioSlug={scenarioSlug} />
        ) : (
          <TestsTab />
        )}
      </div>
    </>
  );

  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Visual Report"
      >
        {content}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button variant="default" size="sm" onClick={onClose} className="w-full h-12 text-sm">
            Done
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Visual Report"
    >
      <div className="w-full max-w-4xl max-h-[85vh] flex flex-col rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        {content}
      </div>
    </div>
  );
}

function ScreenshotsTab({
  before,
  after,
  scenarioSlug,
  isMobile,
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
  isMobile: boolean;
}) {
  const [selectedPage, setSelectedPage] = useState(0);

  if (!after) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Camera className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">No captures yet</p>
        <p className="text-xs mt-1">Click the capture button to take screenshots</p>
      </div>
    );
  }

  const afterPages = after.pages ?? [];
  const currentPage = afterPages[selectedPage] ?? "/";

  return (
    <div className="space-y-4">
      {/* Page selector */}
      {afterPages.length > 1 && (
        <div className="flex gap-1.5 overflow-x-auto pb-2">
          {afterPages.map((page, i) => (
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
              {page === "/" ? "Home" : page}
            </button>
          ))}
        </div>
      )}

      {/* Comparison status */}
      {!before && (
        <div className="text-xs text-slate-500 bg-slate-900/50 rounded px-3 py-2">
          Capture again to compare before/after
        </div>
      )}

      {/* Side-by-side or stacked screenshots */}
      <div className={`gap-4 ${isMobile ? "space-y-4" : "grid grid-cols-2"}`}>
        {before && (
          <div>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs font-medium text-slate-400">Before</span>
              <span className="text-[10px] text-slate-600">
                {new Date(before.createdAt).toLocaleString()}
              </span>
            </div>
            <ScreenshotImage captureId={before.id} scenarioSlug={scenarioSlug} pagePath={currentPage} />
          </div>
        )}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-medium text-slate-400">{before ? "After" : "Current"}</span>
            {before && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                Changed
              </span>
            )}
          </div>
          <ScreenshotImage captureId={after.id} scenarioSlug={scenarioSlug} pagePath={currentPage} />
        </div>
      </div>
    </div>
  );
}

function ScreenshotImage({
  captureId,
  scenarioSlug,
  pagePath,
}: {
  captureId: string;
  scenarioSlug: string;
  pagePath: string;
}) {
  const filename = sanitizePagePath(pagePath) + ".png";
  const url = buildCaptureScreenshotUrl(captureId, scenarioSlug, filename);

  return (
    <div className="rounded-lg border border-slate-800 overflow-hidden bg-slate-900">
      <img
        src={url}
        alt={`Screenshot of ${pagePath}`}
        className="max-w-full object-contain"
        loading="lazy"
      />
    </div>
  );
}

function VideosTab({
  before,
  after,
  scenarioSlug,
}: {
  before?: SnapshotSetMeta;
  after?: SnapshotSetMeta;
  scenarioSlug: string;
}) {
  if (!after || after.videoCount === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm">No video recordings available</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4">
      {before && before.videoCount > 0 && (
        <div>
          <p className="text-xs font-medium text-slate-400 mb-2">Before</p>
          <video
            controls
            src={buildCaptureVideoUrl(before.id, scenarioSlug, "recording.webm")}
            className="w-full rounded-lg border border-slate-800"
          />
        </div>
      )}
      <div>
        <p className="text-xs font-medium text-slate-400 mb-2">
          {before ? "After" : "Current"}
        </p>
        <video
          controls
          src={buildCaptureVideoUrl(after.id, scenarioSlug, "recording.webm")}
          className="w-full rounded-lg border border-slate-800"
        />
      </div>
    </div>
  );
}

function TestsTab() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-slate-500">
      <Construction className="h-8 w-8 mb-3 opacity-50" />
      <p className="text-sm font-medium">Test results integration coming soon</p>
      <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
        Lighthouse scores, test-genie reports, and automated regression testing
      </p>
    </div>
  );
}

function sanitizePagePath(pagePath: string): string {
  if (pagePath === "/" || pagePath === "") return "_root_";
  let s = pagePath.startsWith("/") ? pagePath.slice(1) : pagePath;
  s = s.endsWith("/") ? s.slice(0, -1) : s;
  s = s.replace(/\//g, "_");
  return "_" + s + "_";
}
