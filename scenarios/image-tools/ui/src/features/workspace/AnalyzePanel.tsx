import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Copy, Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  listAnalysisOperations,
  type AnalyzeNsfw,
  type AnalyzeOcr,
  type AnalyzeProbe,
} from "../../api/analysis";
import { ANALYZE_FALLBACK_ICON, analyzePresentation } from "./analyzeCatalog";
import { MODE_LABEL } from "./modeLabels";
import { isAnalyzeActive, type UseAnalyze } from "./useAnalyze";

const ANALYSIS_OPS_QUERY_KEY = ["analysis-operations"] as const;

/** Compact human byte size for the probe panel. */
const formatBytes = (n: number): string => {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)} MB`;
  if (n >= 1_000) return `${Math.round(n / 1_000)} KB`;
  return `${n} B`;
};

export interface AnalyzePanelProps {
  analyze: UseAnalyze;
  /** The current canvas image (base or latest output), or null. */
  input: File | null;
  /** Pre-selected analysis op (from a Home tile handoff); optional. */
  initialAction?: string;
}

/** A labeled metadata row in the probe panel. */
function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 py-0.5">
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="text-right font-medium text-app-foreground">{value}</dd>
    </div>
  );
}

/** Probe (pure-Go metadata) result: dimensions, codec facts, EXIF, palette. */
function ProbeView({ result }: { result: AnalyzeProbe }) {
  const { t } = useTranslation();
  const yes = t(strings.workspace.analyze.probe.yes);
  const no = t(strings.workspace.analyze.probe.no);
  return (
    <div data-testid={selectors.workspace.analyze.probe} className="flex flex-col gap-3">
      <dl className="rounded-control border border-app-border bg-app-surface-muted p-3 text-xs">
        <MetaRow
          label={t(strings.workspace.analyze.probe.dimensions)}
          value={`${result.width}×${result.height}`}
        />
        <MetaRow label={t(strings.workspace.analyze.probe.format)} value={result.format || "—"} />
        <MetaRow
          label={t(strings.workspace.analyze.probe.colorModel)}
          value={result.colorModel || "—"}
        />
        <MetaRow label={t(strings.workspace.analyze.probe.alpha)} value={result.hasAlpha ? yes : no} />
        <MetaRow
          label={t(strings.workspace.analyze.probe.megapixels)}
          value={`${result.megapixels}`}
        />
        <MetaRow
          label={t(strings.workspace.analyze.probe.size)}
          value={formatBytes(result.sizeBytes)}
        />
        {result.frameCount > 1 && (
          <MetaRow
            label={t(strings.workspace.analyze.probe.frames)}
            value={`${result.frameCount}`}
          />
        )}
        <MetaRow label={t(strings.workspace.analyze.probe.exif)} value={result.hasExif ? yes : no} />
        <MetaRow label={t(strings.workspace.analyze.probe.gps)} value={result.hasGps ? yes : no} />
        {result.orientation > 0 && (
          <MetaRow
            label={t(strings.workspace.analyze.probe.orientation)}
            value={`${result.orientation}`}
          />
        )}
      </dl>
      {result.dominantColors.length > 0 && (
        <div className="flex flex-col gap-1">
          <span className="text-xs text-app-muted-foreground">
            {t(strings.workspace.analyze.probe.palette)}
          </span>
          <div className="flex flex-wrap gap-1">
            {result.dominantColors.map((c, i) => (
              <span
                key={`${c.hex}-${i}`}
                title={`${c.hex} · ${Math.round(c.fraction * 100)}%`}
                className="h-6 w-6 rounded-sm border border-app-border"
                style={{ backgroundColor: c.hex }}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

/** OCR result: full text with one-tap copy, language, and block count. */
function OcrView({ result }: { result: AnalyzeOcr }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  // Reset the copied affordance whenever a fresh result arrives.
  useEffect(() => setCopied(false), [result]);

  const onCopy = () => {
    const text = result.fullText;
    if (!text) return;
    // `navigator.clipboard` is typed as always-present but is absent in some
    // contexts (insecure origin, older engines); the try/catch covers both the
    // sync access throw and the async rejection without an "unnecessary" guard.
    try {
      void navigator.clipboard.writeText(text).then(
        () => setCopied(true),
        () => setCopied(false),
      );
    } catch {
      setCopied(false);
    }
  };

  return (
    <div data-testid={selectors.workspace.analyze.ocr} className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-app-muted-foreground">
          {t(strings.workspace.analyze.ocr.language, { language: result.language || "—" })}
          {result.blocks.length > 0
            ? ` · ${t(strings.workspace.analyze.ocr.blocks, { count: result.blocks.length })}`
            : ""}
        </span>
        <Button
          variant="outline"
          size="sm"
          type="button"
          data-testid={selectors.workspace.analyze.copy}
          onClick={onCopy}
          disabled={!result.fullText}
        >
          {copied ? (
            <Check aria-hidden="true" className="mr-1 h-3.5 w-3.5 text-app-success" />
          ) : (
            <Copy aria-hidden="true" className="mr-1 h-3.5 w-3.5" />
          )}
          {copied ? t(strings.workspace.analyze.ocr.copied) : t(strings.workspace.analyze.ocr.copy)}
        </Button>
      </div>
      {result.fullText ? (
        <pre
          data-testid={selectors.workspace.analyze.ocrText}
          className="max-h-64 overflow-auto whitespace-pre-wrap rounded-control border border-app-border bg-app-surface-muted p-3 text-xs text-app-foreground"
        >
          {result.fullText}
        </pre>
      ) : (
        <p className="text-xs text-app-muted-foreground">
          {t(strings.workspace.analyze.ocr.empty)}
        </p>
      )}
    </div>
  );
}

/** NSFW safety classification result: verdict, score, per-label categories. */
function NsfwView({ result }: { result: AnalyzeNsfw }) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.workspace.analyze.nsfw} className="flex flex-col gap-2 text-xs">
      <span
        className={
          result.flagged
            ? "inline-flex w-fit items-center rounded-pill bg-app-danger/15 px-2 py-0.5 font-medium text-app-danger"
            : "inline-flex w-fit items-center rounded-pill bg-app-success/15 px-2 py-0.5 font-medium text-app-success"
        }
      >
        {result.flagged
          ? t(strings.workspace.analyze.nsfw.flagged)
          : t(strings.workspace.analyze.nsfw.safe)}
      </span>
      <p className="text-app-muted-foreground">
        {t(strings.workspace.analyze.nsfw.score, { score: Math.round(result.score * 100) })}
        {" · "}
        {t(strings.workspace.analyze.nsfw.threshold, {
          threshold: Math.round(result.threshold * 100),
        })}
      </p>
      {result.categories.length > 0 && (
        <dl className="flex flex-col gap-1">
          {result.categories.map((c) => (
            <div key={c.label} className="flex items-center gap-2">
              <dt className="w-20 shrink-0 truncate text-app-muted-foreground">{c.label}</dt>
              <dd className="flex-1">
                <span className="block h-1.5 w-full overflow-hidden rounded-pill bg-app-surface-muted">
                  <span
                    className="block h-full rounded-pill bg-app-accent"
                    style={{ width: `${Math.round(c.score * 100)}%` }}
                  />
                </span>
              </dd>
              <dd className="w-9 text-right tabular-nums text-app-muted-foreground">
                {Math.round(c.score * 100)}%
              </dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}

/**
 * The Analyze-mode inspector: a one-tap list of the analysis ops discovered
 * from `AnalysisService.ListAnalysisOperations` (Inspect/probe, Extract text,
 * Safety check), the run control, the install gate for the model-backed ops
 * (ocr / nsfw_classify; pure-Go probe needs no model), and the structured
 * result — a metadata panel (probe), copyable text + on-canvas boxes (ocr), or
 * a safety verdict (nsfw). Analysis is synchronous, so the lifecycle is simpler
 * than Enhance/Create; it lives in `useAnalyze` (injected client).
 */
export function AnalyzePanel({ analyze, input, initialAction }: AnalyzePanelProps) {
  const { t } = useTranslation();
  const opsQuery = useQuery({
    queryKey: ANALYSIS_OPS_QUERY_KEY,
    queryFn: listAnalysisOperations,
  });

  const [operation, setOperation] = useState(initialAction ?? "");

  const operations = useMemo(() => opsQuery.data?.operations ?? [], [opsQuery.data]);

  // Default to the first analysis op once discovery resolves.
  useEffect(() => {
    if (!operation && operations.length > 0) {
      setOperation(operations[0]?.name ?? "");
    }
  }, [operation, operations]);

  const selected = operations.find((op) => op.name === operation);
  const busy = isAnalyzeActive(analyze.phase);
  const { clear } = analyze;

  const onSelect = (name: string) => {
    if (name !== operation) {
      setOperation(name);
      clear();
    }
  };

  return (
    <section
      data-testid={selectors.workspace.analyze.panel}
      aria-label={t(MODE_LABEL.analyze)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">{t(MODE_LABEL.analyze)}</h3>
      <p
        data-testid={selectors.workspace.analyze.intro}
        className="mt-1 text-xs text-app-muted-foreground"
      >
        {t(strings.workspace.analyze.intro)}
      </p>

      {opsQuery.isLoading ? (
        <p
          data-testid={selectors.workspace.analyze.loading}
          className="mt-3 text-sm text-app-foreground"
        >
          {t(strings.workspace.analyze.loading)}
        </p>
      ) : opsQuery.error ? (
        <p
          data-testid={selectors.workspace.analyze.error}
          className="mt-3 text-sm text-app-danger"
        >
          {t(strings.workspace.analyze.error)}
        </p>
      ) : (
        <div className="mt-3 flex flex-col gap-4">
          <div
            data-testid={selectors.workspace.analyze.actions}
            role="radiogroup"
            aria-label={t(MODE_LABEL.analyze)}
            className="flex flex-col gap-2"
          >
            {operations.map((op) => {
              const meta = analyzePresentation(op.name);
              const Icon = meta?.Icon ?? ANALYZE_FALLBACK_ICON;
              const isSelected = op.name === operation;
              return (
                <button
                  key={op.name}
                  type="button"
                  role="radio"
                  aria-checked={isSelected}
                  data-testid={selectors.workspace.analyzeAction({ name: op.name })}
                  onClick={() => onSelect(op.name)}
                  disabled={busy}
                  className={
                    isSelected
                      ? "flex items-start gap-3 rounded-control border border-app-primary bg-app-primary/10 p-3 text-left disabled:opacity-60"
                      : "flex items-start gap-3 rounded-control border border-app-border bg-app-surface p-3 text-left hover:border-app-primary disabled:opacity-60"
                  }
                >
                  <Icon aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-brand" />
                  <span className="flex flex-col">
                    <span className="text-sm font-medium text-app-foreground">
                      {meta ? t(meta.labelKey) : op.summary || op.name}
                    </span>
                    <span className="text-xs text-app-muted-foreground">
                      {meta ? t(meta.descKey) : op.summary}
                    </span>
                  </span>
                </button>
              );
            })}
          </div>

          {selected?.modelBacked && (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.workspace.analyze.modelNote)}
            </p>
          )}

          {!input && (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.workspace.analyze.needsImage)}
            </p>
          )}

          {analyze.phase === "needs-install" && analyze.model ? (
            <div
              data-testid={selectors.workspace.analyze.installGate}
              className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
            >
              <p className="text-sm font-medium text-app-foreground">
                {t(strings.workspace.enhance.install.title, { model: analyze.model.name })}
              </p>
              <p className="text-xs text-app-muted-foreground">
                {analyze.model.cpuCapable
                  ? t(strings.workspace.enhance.install.cpu)
                  : t(strings.workspace.enhance.install.gpu, { vram: analyze.model.minVramGb })}
                {analyze.model.sizeMb > 0
                  ? ` · ${t(strings.workspace.enhance.install.size, { size: analyze.model.sizeMb })}`
                  : ""}
              </p>
              <Button
                type="button"
                data-testid={selectors.workspace.analyze.install}
                onClick={analyze.installAndRun}
              >
                {t(strings.workspace.enhance.install.run)}
              </Button>
            </div>
          ) : busy ? (
            <div
              data-testid={selectors.workspace.analyze.progress}
              className="flex items-center gap-2 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground"
            >
              <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-app-brand" />
              <span>
                {analyze.phase === "installing"
                  ? t(strings.workspace.enhance.install.installing)
                  : t(strings.workspace.analyze.running)}
              </span>
            </div>
          ) : (
            <Button
              type="button"
              data-testid={selectors.workspace.analyze.run}
              disabled={!input || !operation}
              onClick={() => {
                if (input && operation) {
                  analyze.run(operation, input, selected?.modelBacked ?? false);
                }
              }}
            >
              {t(strings.workspace.analyze.run)}
            </Button>
          )}

          {analyze.phase === "failed" && (
            <div className="flex flex-col gap-2">
              <p
                data-testid={selectors.workspace.analyze.failed}
                className="text-sm text-app-danger"
              >
                {analyze.error ?? t(strings.workspace.analyze.failed)}
              </p>
              <Button
                variant="outline"
                type="button"
                data-testid={selectors.workspace.analyze.retry}
                onClick={analyze.retry}
              >
                {t(strings.workspace.analyze.retry)}
              </Button>
            </div>
          )}

          {analyze.phase === "done" && analyze.result && (
            <div data-testid={selectors.workspace.analyze.result} className="flex flex-col gap-2">
              {analyze.result.kind === "probe" && <ProbeView result={analyze.result} />}
              {analyze.result.kind === "ocr" && <OcrView result={analyze.result} />}
              {analyze.result.kind === "nsfw" && <NsfwView result={analyze.result} />}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
