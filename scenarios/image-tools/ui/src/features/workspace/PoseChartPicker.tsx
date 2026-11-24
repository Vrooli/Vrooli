import { useCallback, useEffect, useRef, useState } from "react";

import { submitAI } from "../../api/ai";
import { fetchBlob } from "../../api/client";
import { jobsClient, JobState } from "../../api/jobs";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** One pose the chart split found, with a small preview cropped from the chart. */
export interface PoseCellThumb {
  number: number;
  url: string;
}

interface PoseChartShape {
  cells?: Array<{ number?: number; box?: [number, number, number, number] }>;
}

/**
 * Read a pose chart and show each pose as a tappable thumbnail. The split runs
 * the same `pose_chart_split` job the CLI uses, so a 3x3 grid, a strip or a
 * scatter of panels all work; the cell number is what the generation job
 * receives. A one-pose picture is selected automatically.
 */
export function PoseChartPicker({
  chart,
  onSelect,
  disabled = false,
}: {
  chart: File | null;
  onSelect: (cell: number) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const [cells, setCells] = useState<PoseCellThumb[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const active = useRef<{ controller: AbortController; jobId: string } | null>(null);

  const reset = useCallback(() => {
    const work = active.current;
    active.current = null;
    if (work) {
      work.controller.abort();
      if (work.jobId) void jobsClient.cancelJob({ id: work.jobId }).catch(() => undefined);
    }
    setCells([]);
    setSelected(null);
    setBusy(false);
    setError("");
  }, []);

  useEffect(() => {
    reset();
    if (!chart) return;
    const work = { controller: new AbortController(), jobId: "" };
    active.current = work;
    setBusy(true);
    void (async () => {
      try {
        const accepted = await submitAI("pose_chart_split", {}, {
          image: chart,
          operationId: crypto.randomUUID(),
          signal: work.controller.signal,
        });
        work.jobId = accepted.jobId;
        const response = await jobsClient.waitJob({ id: accepted.jobId }, { signal: work.controller.signal });
        if (active.current !== work) return;
        if (response.job?.state !== JobState.SUCCEEDED || !response.job.resultRef) {
          throw new Error(response.job?.error || response.job?.message || "the chart split did not finish");
        }
        const blob = await fetchBlob(response.job.resultRef);
        const doc = (JSON.parse(await blob.text()) as PoseChartShape) ?? {};
        const found = (doc.cells ?? []).filter(
          (cell): cell is { number: number; box: [number, number, number, number] } =>
            typeof cell.number === "number" && Array.isArray(cell.box) && cell.box.length === 4,
        );
        if (found.length === 0) throw new Error("the chart has no pose to select");
        const thumbs = await cropCells(chart, found);
        if (active.current !== work) return;
        setCells(thumbs);
        const first = thumbs.length === 1 ? thumbs[0]?.number : undefined;
        if (first) {
          setSelected(first);
          onSelect(first);
        }
      } catch (err) {
        if (active.current === work && !work.controller.signal.aborted) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (active.current === work) {
          active.current = null;
          setBusy(false);
        }
      }
    })();
    return () => { if (active.current === work) reset(); };
  }, [chart, onSelect, reset]);

  const choose = (number: number) => {
    setSelected(number);
    onSelect(number);
  };

  if (busy) {
    return <p className="text-xs text-app-muted-foreground" data-testid="pose-chart-busy">{t(strings.workspace.create.poseChart.busy)}</p>;
  }
  if (error) {
    return <p className="text-xs text-app-danger" data-testid="pose-chart-error">{t(strings.workspace.create.poseChart.error, { message: error })}</p>;
  }
  if (cells.length === 0) {
    return <p className="text-xs text-app-muted-foreground" data-testid="pose-chart-needs-chart">{t(strings.workspace.create.poseChart.needsChart)}</p>;
  }
  return (
    <div className="flex flex-col gap-2" data-testid="pose-chart-cells">
      <div className="grid grid-cols-3 gap-2">
        {cells.map((cell) => {
          const activeCell = cell.number === selected;
          return (
            <button
              key={cell.number}
              type="button"
              disabled={disabled}
              aria-pressed={activeCell}
              aria-label={t(strings.workspace.create.poseChart.selected, { cell: cell.number })}
              data-testid={`pose-chart-cell-${cell.number}`}
              onClick={() => choose(cell.number)}
              className={
                activeCell
                  ? "overflow-hidden rounded-control border-2 border-app-primary bg-app-primary/10"
                  : "overflow-hidden rounded-control border border-app-border hover:border-app-primary"
              }
            >
              <img src={cell.url} alt="" className="block h-auto w-full" />
              <span className="block py-1 text-center text-[11px] text-app-muted-foreground">{cell.number}</span>
            </button>
          );
        })}
      </div>
      {selected ? (
        <p className="text-xs text-app-muted-foreground" data-testid="pose-chart-selected">
          {t(strings.workspace.create.poseChart.selected, { cell: selected })}
        </p>
      ) : (
        <p className="text-xs text-app-warning" data-testid="pose-chart-needs-cell">
          {t(strings.workspace.create.poseChart.needsCell)}
        </p>
      )}
    </div>
  );
}

/** Crop each chart cell to a small JPEG data URL for the picker. */
async function cropCells(
  chart: File,
  cells: Array<{ number: number; box: [number, number, number, number] }>,
): Promise<PoseCellThumb[]> {
  const url = URL.createObjectURL(chart);
  try {
    const image = new Image();
    image.src = url;
    await image.decode();
    return cells.map((cell) => {
      const [x0, y0, x1, y1] = cell.box;
      const w = Math.max(1, x1 - x0);
      const h = Math.max(1, y1 - y0);
      const scale = Math.min(1, 180 / Math.max(w, h));
      const canvas = document.createElement("canvas");
      canvas.width = Math.max(1, Math.round(w * scale));
      canvas.height = Math.max(1, Math.round(h * scale));
      const ctx = canvas.getContext("2d");
      if (ctx) {
        ctx.drawImage(image, x0, y0, w, h, 0, 0, canvas.width, canvas.height);
      }
      return { number: cell.number, url: canvas.toDataURL("image/jpeg", 0.82) };
    });
  } finally {
    URL.revokeObjectURL(url);
  }
}
