import {
  Camera,
  Download,
  Trash2,
  CheckSquare,
  Square,
  Loader2,
} from "lucide-react";
import { Drawer, DrawerBody, DrawerHeader } from "../ui/drawer";
import { Button } from "../ui/button";
import { formatBytes } from "../../domain/download";
import { buildCaptureFileUrl } from "../../lib/api/captures";
import { useCapturesStore } from "../../store/capturesStore";

function timeAgo(
  value: { seconds: bigint; nanos: number } | undefined,
): string {
  if (!value) return "unknown";
  const ms =
    Date.now() - (Number(value.seconds) * 1_000 + value.nanos / 1_000_000);
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${String(seconds)}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${String(minutes)}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${String(hours)}h ago`;
  const days = Math.floor(hours / 24);
  return `${String(days)}d ago`;
}

export function CapturesDrawer() {
  const isOpen = useCapturesStore((s) => s.isOpen);
  const close = useCapturesStore((s) => s.close);
  const scenarioName = useCapturesStore((s) => s.scenarioName);
  const captures = useCapturesStore((s) => s.captures);
  const summary = useCapturesStore((s) => s.summary);
  const selectedIds = useCapturesStore((s) => s.selectedIds);
  const loading = useCapturesStore((s) => s.loading);
  const error = useCapturesStore((s) => s.error);
  const toggleSelect = useCapturesStore((s) => s.toggleSelect);
  const selectAll = useCapturesStore((s) => s.selectAll);
  const deselectAll = useCapturesStore((s) => s.deselectAll);
  const deleteCapture = useCapturesStore((s) => s.deleteCapture);
  const deleteAll = useCapturesStore((s) => s.deleteAll);
  const downloadSelected = useCapturesStore((s) => s.downloadSelected);

  const allSelected =
    captures.length > 0 && selectedIds.size === captures.length;

  return (
    <Drawer
      open={isOpen}
      onClose={close}
      side="right"
      panelClassName="md:w-[600px] md:max-w-2xl"
    >
      <DrawerHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-2">
            <Camera className="h-5 w-5 text-slate-400" />
            <div>
              <h2 className="text-base font-semibold text-slate-100">
                Captures
              </h2>
              {scenarioName && (
                <p className="text-xs text-slate-500">{scenarioName}</p>
              )}
            </div>
          </div>
          {summary && summary.totalBytes > 0n && (
            <span className="text-xs text-slate-400">
              Total: {formatBytes(Number(summary.totalBytes))}
            </span>
          )}
        </div>
      </DrawerHeader>
      <DrawerBody>
        {/* Toolbar */}
        {captures.length > 0 && (
          <div className="flex items-center gap-2 mb-4 flex-wrap">
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={allSelected ? deselectAll : selectAll}
              className="text-xs"
            >
              {allSelected ? (
                <>
                  <CheckSquare className="h-3.5 w-3.5 mr-1" /> Deselect All
                </>
              ) : (
                <>
                  <Square className="h-3.5 w-3.5 mr-1" /> Select All
                </>
              )}
            </Button>
            {selectedIds.size > 0 && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={downloadSelected}
                className="text-xs"
              >
                <Download className="h-3.5 w-3.5 mr-1" />
                Download ({selectedIds.size})
              </Button>
            )}
            <Button
              type="button"
              size="sm"
              variant="destructive"
              onClick={() => {
                if (
                  window.confirm(
                    `Delete all ${String(captures.length)} captures for "${scenarioName ?? "this scenario"}"?`,
                  )
                ) {
                  void deleteAll();
                }
              }}
              className="text-xs ml-auto"
            >
              <Trash2 className="h-3.5 w-3.5 mr-1" />
              Clean Up All
            </Button>
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className="flex items-center justify-center py-12 text-slate-400">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="rounded-lg border border-red-900/50 bg-red-950/20 p-3 text-sm text-red-300 mb-4">
            {error}
          </div>
        )}

        {/* Empty state */}
        {!loading && !error && captures.length === 0 && (
          <div className="text-center py-12">
            <Camera className="h-10 w-10 text-slate-600 mx-auto mb-3" />
            <p className="text-sm text-slate-400">No captures yet</p>
            <p className="text-xs text-slate-600 mt-1">
              Take screenshots or recordings in a desktop session
            </p>
          </div>
        )}

        {/* Grid */}
        {!loading && captures.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {captures.map((cap) => {
              const isSelected = selectedIds.has(cap.captureId);
              const fileUrl = scenarioName
                ? buildCaptureFileUrl(scenarioName, cap.captureId)
                : "";
              return (
                <div
                  key={cap.captureId}
                  className={`relative rounded-lg border bg-slate-900/50 overflow-hidden transition ${
                    isSelected
                      ? "border-blue-500 ring-1 ring-blue-500/30"
                      : "border-slate-800 hover:border-slate-700"
                  }`}
                >
                  {/* Checkbox */}
                  <button
                    type="button"
                    className="absolute top-1.5 left-1.5 z-10 rounded bg-slate-900/80 p-0.5"
                    onClick={() => {
                      toggleSelect(cap.captureId);
                    }}
                  >
                    {isSelected ? (
                      <CheckSquare className="h-4 w-4 text-blue-400" />
                    ) : (
                      <Square className="h-4 w-4 text-slate-500" />
                    )}
                  </button>

                  {/* Media */}
                  <div className="aspect-video bg-slate-950">
                    {cap.kind !== "recording" ? (
                      <img
                        src={fileUrl}
                        alt={cap.filename}
                        className="w-full h-full object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <video
                        src={fileUrl}
                        className="w-full h-full object-cover"
                        controls
                        preload="metadata"
                      />
                    )}
                  </div>

                  {/* Info */}
                  <div className="p-2 space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-slate-400 truncate">
                        {formatBytes(Number(cap.fileSizeBytes))}
                      </span>
                      <span className="text-xs text-slate-600">
                        {timeAgo(cap.createdAt)}
                      </span>
                    </div>
                    <div className="flex justify-end">
                      <button
                        type="button"
                        className="p-1 rounded hover:bg-red-950/50 text-slate-500 hover:text-red-400 transition"
                        onClick={() => {
                          void deleteCapture(cap.captureId);
                        }}
                        title="Delete"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </DrawerBody>
    </Drawer>
  );
}
