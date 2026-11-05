import { useEffect, useRef, useState } from "react";
import { Button } from "../../../components/ui/button";
import { streamProviderLogs } from "../../../api/providerLifecycle";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

// Cap the visible buffer so a long-running tail doesn't eat the page.
// TODO(virtualization): swap the <pre> for a virtualized list when the
// component graduates from operator-only usage to general dashboards.
const MAX_LINES = 1000;

interface LogsDrawerProps {
  providerId: string;
  open: boolean;
  onClose: () => void;
}

export function LogsDrawer({ providerId, open, onClose }: LogsDrawerProps) {
  const { t } = useTranslation();
  const [lines, setLines] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!open) return;
    setLines([]);
    setError(null);
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    void (async () => {
      try {
        const stream = streamProviderLogs({ providerId, follow: true, tailLines: 200 }, ctrl.signal);
        for await (const evt of stream) {
          setLines((prev) => {
            const next = prev.length >= MAX_LINES ? prev.slice(-MAX_LINES + 1) : prev;
            return [...next, evt.line];
          });
        }
      } catch (e) {
        if (ctrl.signal.aborted) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      ctrl.abort();
      abortRef.current = null;
    };
  }, [open, providerId]);

  if (!open) return null;

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
      <pre
        data-testid="logs-stream"
        className="flex-1 overflow-auto whitespace-pre-wrap break-words bg-app-background p-3 font-mono text-xs text-app-foreground"
      >
        {lines.length === 0 ? t(strings.status.logsWaiting) : lines.join("\n")}
      </pre>
    </div>
  );
}

export default LogsDrawer;
