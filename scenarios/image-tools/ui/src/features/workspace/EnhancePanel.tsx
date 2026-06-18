import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { SegmentedControl } from "../../components/ui/segmented-control";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { listAIOperations, type AIParamsInput } from "../../api/ai";
import { AI_FALLBACK_ICON, TIER_LABEL, UPSCALE_SCALES, aiPresentation } from "./aiCatalog";
import { MODE_LABEL } from "./modeLabels";
import { isEnhanceActive, type UseEnhance } from "./useEnhance";

const AI_OPS_QUERY_KEY = ["ai-operations"] as const;

export interface EnhancePanelProps {
  enhance: UseEnhance;
  /** The current canvas image (base or latest output), or null. */
  input: File | null;
  /** Current image dimensions, used for the upscale target-resolution preview. */
  inputWidth: number;
  inputHeight: number;
  /** Pre-selected enhancement op (from a Home tile handoff); optional. */
  initialAction?: string;
}

/** Largest pre-upscale megapixel count before we warn about memory/time. */
const LARGE_RESULT_MP = 24_000_000;

/**
 * The Enhance-mode inspector: a one-tap AI action list (the enhancement ops
 * discovered from `AIService.ListAIOperations`), the upscale factor, a
 * hardware-fit model badge, and the full durable-job lifecycle — install gate,
 * live progress with cancel, fallback-tier + warnings, success, and retry. The
 * heavy lifecycle lives in `useEnhance` (injected client); this panel is the
 * surface. The op's selected model is previewed on selection so the badge and
 * the install gate are honest before the user commits.
 */
export function EnhancePanel({
  enhance,
  input,
  inputWidth,
  inputHeight,
  initialAction,
}: EnhancePanelProps) {
  const { t } = useTranslation();
  const aiOpsQuery = useQuery({ queryKey: AI_OPS_QUERY_KEY, queryFn: listAIOperations });

  const [operation, setOperation] = useState(initialAction ?? "");
  const [scale, setScale] = useState<string>(UPSCALE_SCALES[0]);

  const enhancementOps = useMemo(
    () => (aiOpsQuery.data?.operations ?? []).filter((op) => op.category === "enhancement"),
    [aiOpsQuery.data],
  );

  // Default to the first enhancement op once discovery resolves.
  useEffect(() => {
    if (!operation && enhancementOps.length > 0) {
      setOperation(enhancementOps[0]?.name ?? "");
    }
  }, [operation, enhancementOps]);

  // Preview the selected op's model so the badge + install gate are honest
  // before the user runs anything.
  const { preview } = enhance;
  useEffect(() => {
    if (operation) {
      preview(operation);
    }
  }, [operation, preview]);

  const params: AIParamsInput = useMemo(
    () => (operation === "upscale" ? { scale: Number(scale) } : {}),
    [operation, scale],
  );

  const busy = isEnhanceActive(enhance.phase);
  const factor = Number(scale);
  const targetWidth = inputWidth * factor;
  const targetHeight = inputHeight * factor;
  const showTarget = operation === "upscale" && inputWidth > 0 && inputHeight > 0;
  const largeResult = showTarget && targetWidth * targetHeight > LARGE_RESULT_MP;

  const tierKey = TIER_LABEL[enhance.tier];
  const { model } = enhance;

  return (
    <section
      data-testid={selectors.workspace.enhance.panel}
      aria-label={t(MODE_LABEL.enhance)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">{t(MODE_LABEL.enhance)}</h3>
      <p
        data-testid={selectors.workspace.enhance.intro}
        className="mt-1 text-xs text-app-muted-foreground"
      >
        {t(strings.workspace.enhance.intro)}
      </p>

      {aiOpsQuery.isLoading ? (
        <p
          data-testid={selectors.workspace.enhance.loading}
          className="mt-3 text-sm text-app-foreground"
        >
          {t(strings.workspace.enhance.loading)}
        </p>
      ) : aiOpsQuery.error ? (
        <p
          data-testid={selectors.workspace.enhance.error}
          className="mt-3 text-sm text-app-danger"
        >
          {t(strings.workspace.enhance.error)}
        </p>
      ) : (
        <div className="mt-3 flex flex-col gap-4">
          <div
            data-testid={selectors.workspace.enhance.actions}
            role="radiogroup"
            aria-label={t(MODE_LABEL.enhance)}
            className="flex flex-col gap-2"
          >
            {enhancementOps.map((op) => {
              const meta = aiPresentation(op.name);
              const Icon = meta?.Icon ?? AI_FALLBACK_ICON;
              const selected = op.name === operation;
              return (
                <button
                  key={op.name}
                  type="button"
                  role="radio"
                  aria-checked={selected}
                  data-testid={selectors.workspace.enhanceAction({ name: op.name })}
                  onClick={() => setOperation(op.name)}
                  disabled={busy}
                  className={
                    selected
                      ? "flex items-start gap-3 rounded-control border border-app-primary bg-app-primary/10 p-3 text-left disabled:opacity-60"
                      : "flex items-start gap-3 rounded-control border border-app-border bg-app-surface p-3 text-left hover:border-app-primary disabled:opacity-60"
                  }
                >
                  <Icon aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-brand" />
                  <span className="flex flex-col">
                    <span className="text-sm font-medium text-app-foreground">
                      {meta ? t(meta.labelKey) : op.name}
                    </span>
                    {meta && (
                      <span className="text-xs text-app-muted-foreground">{t(meta.descKey)}</span>
                    )}
                  </span>
                </button>
              );
            })}
          </div>

          {operation === "upscale" && (
            <div className="flex flex-col gap-1">
              <span className="text-xs text-app-muted-foreground">
                {t(strings.workspace.enhance.scaleLabel)}
              </span>
              <SegmentedControl<string>
                label={t(strings.workspace.enhance.scaleLabel)}
                value={scale}
                options={UPSCALE_SCALES.map((s) => ({ value: s, label: `${s}×` }))}
                onChange={setScale}
                data-testid={selectors.workspace.enhance.scale}
              />
              {showTarget && (
                <p className="text-xs text-app-muted-foreground">
                  {t(strings.workspace.enhance.targetResolution, {
                    from: `${inputWidth}×${inputHeight}`,
                    to: `${targetWidth}×${targetHeight}`,
                  })}
                </p>
              )}
              {largeResult && (
                <p className="text-xs text-app-warning">
                  {t(strings.workspace.enhance.memoryWarning)}
                </p>
              )}
            </div>
          )}

          {model && model.id !== "" && (
            <p
              data-testid={selectors.workspace.enhance.modelBadge}
              className="text-xs text-app-muted-foreground"
            >
              <span className="font-medium text-app-foreground">{model.name}</span>
              {" · "}
              {model.cpuCapable
                ? t(strings.workspace.enhance.install.cpu)
                : t(strings.workspace.enhance.install.gpu, { vram: model.minVramGb })}
              {model.speedNote ? ` · ${model.speedNote}` : ""}
            </p>
          )}

          {!input && (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.workspace.enhance.needsImage)}
            </p>
          )}

          {enhance.phase === "needs-install" && model ? (
            <div
              data-testid={selectors.workspace.enhance.installGate}
              className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
            >
              <p className="text-sm font-medium text-app-foreground">
                {t(strings.workspace.enhance.install.title, { model: model.name })}
              </p>
              <p className="text-xs text-app-muted-foreground">
                {model.cpuCapable
                  ? t(strings.workspace.enhance.install.cpu)
                  : t(strings.workspace.enhance.install.gpu, { vram: model.minVramGb })}
                {model.sizeMb > 0 ? ` · ${t(strings.workspace.enhance.install.size, { size: model.sizeMb })}` : ""}
              </p>
              <Button
                type="button"
                data-testid={selectors.workspace.enhance.install}
                onClick={enhance.installAndRun}
              >
                {t(strings.workspace.enhance.install.run)}
              </Button>
            </div>
          ) : busy ? (
            <div
              data-testid={selectors.workspace.enhance.progress}
              className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
            >
              <div className="flex items-center gap-2 text-sm text-app-foreground">
                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-app-brand" />
                <span>
                  {enhance.phase === "installing"
                    ? t(strings.workspace.enhance.install.installing)
                    : t(strings.workspace.enhance.running)}
                </span>
              </div>
              {enhance.phase === "running" && (
                <>
                  <p className="text-xs text-app-muted-foreground">
                    {t(strings.workspace.enhance.progress, { percent: enhance.progress.percent })}
                    {tierKey ? ` · ${t(tierKey)}` : ""}
                  </p>
                  {enhance.progress.message && (
                    <p className="text-xs text-app-muted-foreground">{enhance.progress.message}</p>
                  )}
                </>
              )}
              <Button
                variant="outline"
                type="button"
                data-testid={selectors.workspace.enhance.cancel}
                onClick={enhance.cancel}
              >
                {t(strings.workspace.enhance.cancel)}
              </Button>
            </div>
          ) : (
            <Button
              type="button"
              data-testid={selectors.workspace.enhance.run}
              disabled={!input || !operation}
              onClick={() => {
                if (input && operation) {
                  enhance.start(operation, params, input);
                }
              }}
            >
              {t(strings.workspace.enhance.run)}
            </Button>
          )}

          {enhance.warnings.length > 0 && (
            <div
              data-testid={selectors.workspace.enhance.warnings}
              className="rounded-control border border-app-warning/50 bg-app-surface-muted p-2 text-xs text-app-foreground"
            >
              <p className="font-medium">{t(strings.workspace.enhance.warningsLabel)}</p>
              <ul className="mt-1 list-disc pl-4 text-app-muted-foreground">
                {enhance.warnings.map((warning, index) => (
                  <li key={`${index}-${warning}`}>{warning}</li>
                ))}
              </ul>
            </div>
          )}

          {enhance.phase === "succeeded" && (
            <p
              data-testid={selectors.workspace.enhance.succeeded}
              className="text-sm text-app-success"
            >
              {t(strings.workspace.enhance.succeeded)}
            </p>
          )}

          {enhance.phase === "failed" && (
            <div className="flex flex-col gap-2">
              <p
                data-testid={selectors.workspace.enhance.failed}
                className="text-sm text-app-danger"
              >
                {enhance.error ?? t(strings.workspace.enhance.failed)}
              </p>
              <Button
                variant="outline"
                type="button"
                data-testid={selectors.workspace.enhance.retry}
                onClick={enhance.retry}
              >
                {t(strings.workspace.enhance.retry)}
              </Button>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
