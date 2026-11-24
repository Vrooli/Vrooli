import { useEffect, useRef, useState } from "react";
import { Button } from "../../../components/ui/button";
import { streamProviderLogs } from "../../../api/providerLifecycle";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

// Keep the stream bounded and render only the rows visible in the viewport.
const MAX_LINES = 1000;
const ROW_HEIGHT = 18;
const OVERSCAN = 12;

interface LogsDrawerProps {
  providerId: string;
  open: boolean;
  onClose: () => void;
}

export function LogsDrawer({ providerId, open, onClose }: LogsDrawerProps) {
  const { t } = useTranslation();
  const [lines, setLines] = useState<string[]>([]);
  const [viewport, setViewport] = useState({ top: 0, height: 0 });
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const viewportRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return;
    setLines([]);
    setError(null);
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    let pending: string[] = [];
    let flushTimer: ReturnType<typeof setTimeout> | undefined;
    const flush = () => {
      if (flushTimer) clearTimeout(flushTimer);
      flushTimer = undefined;
      if (pending.length === 0) return;
      const batch = pending;
      pending = [];
      setLines((prev) => [...prev, ...batch].slice(-MAX_LINES));
    };
    void (async () => {
      try {
        const stream = streamProviderLogs({ providerId, follow: true, tailLines: 200 }, ctrl.signal);
        for await (const evt of stream) {
          pending.push(evt.line);
          if (pending.length >= 25) flush();
          else if (!flushTimer) flushTimer = setTimeout(flush, 50);
        }
      } catch (e) {
        if (ctrl.signal.aborted) return;
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        flush();
      }
    })();
    return () => {
      ctrl.abort();
      if (flushTimer) clearTimeout(flushTimer);
      flush();
      abortRef.current = null;
    };
  }, [open, providerId]);

  useEffect(() => {
    const element = viewportRef.current;
    if (!open || !element) return;
    const updateHeight = () => setViewport((current) => ({ ...current, height: element.clientHeight }));
    updateHeight();
    const observer = new ResizeObserver(updateHeight);
    observer.observe(element);
    return () => observer.disconnect();
  }, [open]);

  if (!open) return null;

  const firstRow = Math.max(0, Math.floor(viewport.top / ROW_HEIGHT) - OVERSCAN);
  const visibleCount = Math.ceil(viewport.height / ROW_HEIGHT) + 2 * OVERSCAN;
  const visibleLines = lines.slice(firstRow, firstRow + visibleCount);

  return (
    <div
      role="dialog"
      aria-label={t(strings.status.logsDialogLabel, { providerId })}
      className="fixed inset-y-0 right-0 z-50 flex w-full max-w-2xl flex-col border-l border-app-border bg-app-surface shadow-xl"
    >
      <header className="flex items-center justify-between gap-2 border-b border-app-border p-3">
        <h2 className="text-base font-semibold text-app-foreground">{t(strings.status.logsHeading, { providerId })}</h2>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            abortRef.current?.abort();
            onClose();
          }}
        >
          {t(strings.status.logsClose)}
        </Button>
      </header>
      {error && (
        <p className="border-b border-app-border p-3 text-sm text-app-danger">{error}</p>
      )}
      <div
        data-testid="logs-stream"
        ref={viewportRef}
        role="log"
        aria-live="off"
        onScroll={(event) => setViewport((current) => ({ ...current, top: event.currentTarget.scrollTop }))}
        className="flex-1 overflow-auto bg-app-background p-3 font-mono text-xs leading-[18px] text-app-foreground"
      >
        {lines.length === 0 ? (
          <p>{t(strings.status.logsWaiting)}</p>
        ) : (
          <div style={{ height: lines.length * ROW_HEIGHT, position: "relative" }}>
            <div style={{ position: "absolute", top: firstRow * ROW_HEIGHT, left: 0, right: 0 }}>
              {visibleLines.map((line, index) => (
                <div key={firstRow + index} className="h-[18px] w-max min-w-full whitespace-pre">
                  {line}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default LogsDrawer;
