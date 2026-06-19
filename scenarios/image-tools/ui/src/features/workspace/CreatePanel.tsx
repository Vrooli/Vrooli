import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { SegmentedControl } from "../../components/ui/segmented-control";
import { Textarea } from "../../components/ui/textarea";
import { Toggle } from "../../components/ui/toggle";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { listAIOperations, type AIOperationInfo, type AIParamsInput } from "../../api/ai";
import { needsConsent } from "../../api/safety";
import { useSafetyPolicy } from "../safety/useSafetyPolicy";
import { TIER_LABEL } from "./aiCatalog";
import {
  CREATE_FALLBACK_ICON,
  DEFAULT_SIZE_PRESET,
  SIZE_PRESETS,
  VARIATION_OPTIONS,
  createPresentation,
  type SizePreset,
} from "./createCatalog";
import { MaskBrush } from "./MaskBrush";
import { MODE_LABEL } from "./modeLabels";
import { isCreateActive, type CreateVariation, type UseCreate } from "./useCreate";
import { VariationGrid } from "./VariationGrid";

const AI_OPS_QUERY_KEY = ["ai-operations"] as const;

export interface CreatePanelProps {
  create: UseCreate;
  /** The current canvas image (img2img / inpaint / object-removal input), or null. */
  input: File | null;
  /** Object URL of the current image, used by the mask brush. */
  inputUrl: string | null;
  onSendToCanvas: (variation: CreateVariation) => void;
  onSendToEnhance: (variation: CreateVariation) => void;
  /** Pre-selected generation op (from a Home tile handoff); optional. */
  initialAction?: string;
}

const sizePresetByKey = (key: string): SizePreset =>
  SIZE_PRESETS.find((preset) => preset.key === key) ?? DEFAULT_SIZE_PRESET;

/**
 * The Create-mode inspector: the generation-op list (discovered from
 * `AIService.ListAIOperations`, category `generation`), a prompt, size / seed /
 * variations controls, an Advanced disclosure, the hardware-fit model badge,
 * and the full durable-job lifecycle (install gate → live progress → cancel /
 * retry) — then the N-variation result grid. Masked ops (inpaint / object
 * removal) surface the mask brush; img2img / inpaint compose on the current
 * canvas image. The lifecycle lives in `useCreate` (injected client).
 */
export function CreatePanel({
  create,
  input,
  inputUrl,
  onSendToCanvas,
  onSendToEnhance,
  initialAction,
}: CreatePanelProps) {
  const { t } = useTranslation();
  const aiOpsQuery = useQuery({ queryKey: AI_OPS_QUERY_KEY, queryFn: listAIOperations });
  const policyQuery = useSafetyPolicy();

  const [operation, setOperation] = useState(initialAction ?? "");
  const [consentAffirmed, setConsentAffirmed] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [negative, setNegative] = useState("");
  const [sizeKey, setSizeKey] = useState<string>(DEFAULT_SIZE_PRESET.key);
  const [seed, setSeed] = useState("");
  const [seedLocked, setSeedLocked] = useState(false);
  const [variations, setVariations] = useState<string>(VARIATION_OPTIONS[0]);
  const [modelOverride, setModelOverride] = useState("");
  const [allowByok, setAllowByok] = useState(false);
  const [mask, setMask] = useState<File | null>(null);

  const generationOps = useMemo(
    () => (aiOpsQuery.data?.operations ?? []).filter((op) => op.category === "generation"),
    [aiOpsQuery.data],
  );

  const opInfo: AIOperationInfo | undefined = useMemo(
    () => generationOps.find((op) => op.name === operation),
    [generationOps, operation],
  );

  // Default to the first generation op once discovery resolves.
  useEffect(() => {
    if (!operation && generationOps.length > 0) {
      setOperation(generationOps[0]?.name ?? "");
    }
  }, [operation, generationOps]);

  // Preview the selected op's model so the badge + install gate are honest
  // before the user commits.
  const { preview } = create;
  useEffect(() => {
    if (operation) {
      preview(operation);
    }
  }, [operation, preview]);

  // A new op means a new input contract; drop a stale mask and re-affirm consent.
  useEffect(() => {
    setMask(null);
    setConsentAffirmed(false);
  }, [operation]);

  const requiresImage = opInfo?.requiresImage ?? false;
  const requiresMask = opInfo?.requiresMask ?? false;
  const promptDriven = opInfo?.promptDriven ?? false;
  // Public-tier identity-altering ops need an affirmed-consent checkbox; on the
  // local tier `requireConsent` is false so this is always false (no checkbox).
  const requiresConsent = needsConsent(policyQuery.data, operation);
  const showSize = operation === "text_to_image";
  const showVariations = !requiresMask;

  const busy = isCreateActive(create.phase);
  const { model } = create;
  const tierKey = TIER_LABEL[create.tier];

  // The backend reports per-variation progress as "produced i/N"; localize it
  // when present, otherwise fall through to the raw status message.
  const producedMatch = /produced\s+(\d+)\/(\d+)/i.exec(create.progress.message);
  const produced = producedMatch
    ? { done: producedMatch[1] ?? "", total: producedMatch[2] ?? "" }
    : null;

  const params: AIParamsInput = useMemo(() => {
    const preset = sizePresetByKey(sizeKey);
    const out: AIParamsInput = { variations: showVariations ? Number(variations) : 1 };
    if (promptDriven && prompt.trim()) {
      out.prompt = prompt.trim();
    }
    if (negative.trim()) {
      out.negativePrompt = negative.trim();
    }
    if (showSize) {
      out.width = preset.width;
      out.height = preset.height;
    }
    if (seedLocked && seed.trim()) {
      out.seed = BigInt(seed.trim());
    }
    if (modelOverride.trim()) {
      out.modelOverride = modelOverride.trim();
    }
    if (allowByok) {
      out.allowByok = true;
    }
    if (consentAffirmed) {
      out.consentAffirmed = true;
    }
    return out;
  }, [
    sizeKey,
    showSize,
    showVariations,
    variations,
    promptDriven,
    prompt,
    negative,
    seedLocked,
    seed,
    modelOverride,
    allowByok,
    consentAffirmed,
  ]);

  const missingImage = requiresImage && !input;
  const missingMask = requiresMask && !mask;
  const missingPrompt = promptDriven && !prompt.trim();
  const missingConsent = requiresConsent && !consentAffirmed;
  const canRun =
    !busy && !!operation && !missingImage && !missingMask && !missingPrompt && !missingConsent;

  const randomizeSeed = () => {
    setSeed(String(Math.floor(Math.random() * 1_000_000_000)));
    setSeedLocked(true);
  };

  const run = () => {
    if (!canRun) {
      return;
    }
    create.start(
      operation,
      params,
      requiresImage ? input ?? undefined : undefined,
      requiresMask ? mask ?? undefined : undefined,
    );
  };

  return (
    <section
      data-testid={selectors.workspace.create.panel}
      aria-label={t(MODE_LABEL.create)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">{t(MODE_LABEL.create)}</h3>
      <p
        data-testid={selectors.workspace.create.intro}
        className="mt-1 text-xs text-app-muted-foreground"
      >
        {t(strings.workspace.create.intro)}
      </p>

      {aiOpsQuery.isLoading ? (
        <p
          data-testid={selectors.workspace.create.loading}
          className="mt-3 text-sm text-app-foreground"
        >
          {t(strings.workspace.create.loading)}
        </p>
      ) : aiOpsQuery.error ? (
        <p data-testid={selectors.workspace.create.error} className="mt-3 text-sm text-app-danger">
          {t(strings.workspace.create.error)}
        </p>
      ) : (
        <div className="mt-3 flex flex-col gap-4">
          <div
            data-testid={selectors.workspace.create.actions}
            role="radiogroup"
            aria-label={t(MODE_LABEL.create)}
            className="grid grid-cols-2 gap-2"
          >
            {generationOps.map((op) => {
              const meta = createPresentation(op.name);
              const Icon = meta?.Icon ?? CREATE_FALLBACK_ICON;
              const selected = op.name === operation;
              return (
                <button
                  key={op.name}
                  type="button"
                  role="radio"
                  aria-checked={selected}
                  data-testid={selectors.workspace.createAction({ name: op.name })}
                  onClick={() => setOperation(op.name)}
                  disabled={busy}
                  className={
                    selected
                      ? "flex items-start gap-2 rounded-control border border-app-primary bg-app-primary/10 p-2 text-left disabled:opacity-60"
                      : "flex items-start gap-2 rounded-control border border-app-border bg-app-surface p-2 text-left hover:border-app-primary disabled:opacity-60"
                  }
                >
                  <Icon aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-brand" />
                  <span className="text-xs font-medium text-app-foreground">
                    {meta ? t(meta.labelKey) : op.name}
                  </span>
                </button>
              );
            })}
          </div>

          {promptDriven && (
            <label className="flex flex-col gap-1">
              <span className="text-xs text-app-muted-foreground">
                {t(strings.workspace.create.promptLabel)}
              </span>
              <Textarea
                data-testid={selectors.workspace.create.prompt}
                value={prompt}
                placeholder={t(strings.workspace.create.promptPlaceholder)}
                onChange={(e) => setPrompt(e.target.value)}
              />
            </label>
          )}

          {requiresImage &&
            (missingImage ? (
              <p
                data-testid={selectors.workspace.create.needsImage}
                className="text-sm text-app-muted-foreground"
              >
                {t(strings.workspace.create.needsImage)}
              </p>
            ) : (
              <p
                data-testid={selectors.workspace.create.usesCurrent}
                className="text-xs text-app-muted-foreground"
              >
                {t(strings.workspace.create.usesCurrent)}
              </p>
            ))}

          {requiresMask && !missingImage && (
            <>
              <MaskBrush imageUrl={inputUrl} onMask={setMask} />
              {missingMask && (
                <p
                  data-testid={selectors.workspace.create.needsMask}
                  className="text-xs text-app-warning"
                >
                  {t(strings.workspace.create.needsMask)}
                </p>
              )}
            </>
          )}

          {showSize && (
            <div className="flex flex-col gap-1">
              <span className="text-xs text-app-muted-foreground">
                {t(strings.workspace.create.sizeLabel)}
              </span>
              <SegmentedControl<string>
                label={t(strings.workspace.create.sizeLabel)}
                value={sizeKey}
                options={SIZE_PRESETS.map((preset) => ({
                  value: preset.key,
                  label: t(preset.labelKey),
                }))}
                onChange={setSizeKey}
                data-testid={selectors.workspace.create.size}
              />
            </div>
          )}

          {showVariations && (
            <div className="flex flex-col gap-1">
              <span className="text-xs text-app-muted-foreground">
                {t(strings.workspace.create.variationsLabel)}
              </span>
              <SegmentedControl<string>
                label={t(strings.workspace.create.variationsLabel)}
                value={variations}
                options={VARIATION_OPTIONS.map((n) => ({ value: n, label: n }))}
                onChange={setVariations}
                data-testid={selectors.workspace.create.variations}
              />
            </div>
          )}

          <div className="flex flex-col gap-1">
            <span className="text-xs text-app-muted-foreground">
              {t(strings.workspace.create.seedLabel)}
            </span>
            <div className="flex items-center gap-2">
              <Input
                data-testid={selectors.workspace.create.seed}
                type="number"
                value={seed}
                aria-label={t(strings.workspace.create.seedLabel)}
                placeholder={t(strings.workspace.create.seedRandom)}
                onChange={(e) => setSeed(e.target.value)}
                className="flex-1"
              />
              <button
                type="button"
                data-testid={selectors.workspace.create.seedRandomize}
                onClick={randomizeSeed}
                className="rounded-control border border-app-border px-2 py-2 text-xs text-app-foreground hover:border-app-primary"
              >
                {t(strings.workspace.create.seedRandomize)}
              </button>
            </div>
            <Toggle
              label={t(strings.workspace.create.seedLock)}
              checked={seedLocked}
              onChange={setSeedLocked}
              data-testid={selectors.workspace.create.seedLock}
            />
          </div>

          <details className="rounded-control border border-app-border bg-app-surface-muted p-2">
            <summary
              data-testid={selectors.workspace.create.advanced}
              className="cursor-pointer text-xs font-medium text-app-muted-foreground"
            >
              {t(strings.workspace.create.advanced)}
            </summary>
            <div className="mt-2 flex flex-col gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-xs text-app-muted-foreground">
                  {t(strings.workspace.create.negativeLabel)}
                </span>
                <Textarea
                  data-testid={selectors.workspace.create.negative}
                  value={negative}
                  placeholder={t(strings.workspace.create.negativePlaceholder)}
                  onChange={(e) => setNegative(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-app-muted-foreground">
                  {t(strings.workspace.create.modelLabel)}
                </span>
                <Input
                  data-testid={selectors.workspace.create.model}
                  type="text"
                  value={modelOverride}
                  placeholder={t(strings.workspace.create.modelPlaceholder)}
                  onChange={(e) => setModelOverride(e.target.value)}
                />
              </label>
              <Toggle
                label={t(strings.workspace.create.byokLabel)}
                checked={allowByok}
                onChange={setAllowByok}
                data-testid={selectors.workspace.create.byok}
              />
            </div>
          </details>

          {requiresConsent && (
            <div className="flex flex-col gap-1 rounded-control border border-app-warning/50 bg-app-surface-muted p-3">
              <Toggle
                label={t(strings.workspace.create.consent.label)}
                checked={consentAffirmed}
                onChange={setConsentAffirmed}
                data-testid={selectors.workspace.create.consent}
              />
              {missingConsent && (
                <p
                  data-testid={selectors.workspace.create.consentRequired}
                  className="text-xs text-app-warning"
                >
                  {t(strings.workspace.create.consent.required)}
                </p>
              )}
            </div>
          )}

          {model && model.id !== "" && (
            <p
              data-testid={selectors.workspace.create.modelBadge}
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

          {create.phase === "needs-install" && model ? (
            <div
              data-testid={selectors.workspace.create.installGate}
              className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
            >
              <p className="text-sm font-medium text-app-foreground">
                {t(strings.workspace.enhance.install.title, { model: model.name })}
              </p>
              <p className="text-xs text-app-muted-foreground">
                {model.cpuCapable
                  ? t(strings.workspace.enhance.install.cpu)
                  : t(strings.workspace.enhance.install.gpu, { vram: model.minVramGb })}
                {model.sizeMb > 0
                  ? ` · ${t(strings.workspace.enhance.install.size, { size: model.sizeMb })}`
                  : ""}
              </p>
              <Button
                type="button"
                data-testid={selectors.workspace.create.install}
                onClick={create.installAndRun}
              >
                {t(strings.workspace.enhance.install.run)}
              </Button>
            </div>
          ) : busy ? (
            <div
              data-testid={selectors.workspace.create.progress}
              className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
            >
              <div className="flex items-center gap-2 text-sm text-app-foreground">
                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-app-brand" />
                <span>
                  {create.phase === "installing"
                    ? t(strings.workspace.enhance.install.installing)
                    : t(strings.workspace.create.running)}
                </span>
              </div>
              {create.phase === "running" && (
                <>
                  <p className="text-xs text-app-muted-foreground">
                    {t(strings.workspace.create.progress, { percent: create.progress.percent })}
                    {tierKey ? ` · ${t(tierKey)}` : ""}
                  </p>
                  {produced ? (
                    <p className="text-xs text-app-muted-foreground">
                      {t(strings.workspace.create.produced, produced)}
                    </p>
                  ) : create.progress.message ? (
                    <p className="text-xs text-app-muted-foreground">{create.progress.message}</p>
                  ) : null}
                </>
              )}
              <Button
                variant="outline"
                type="button"
                data-testid={selectors.workspace.create.cancel}
                onClick={create.cancel}
              >
                {t(strings.workspace.create.cancel)}
              </Button>
            </div>
          ) : (
            <Button
              type="button"
              data-testid={selectors.workspace.create.run}
              disabled={!canRun}
              onClick={run}
            >
              {t(strings.workspace.create.run)}
            </Button>
          )}

          {create.warnings.length > 0 && (
            <div
              data-testid={selectors.workspace.create.warnings}
              className="rounded-control border border-app-warning/50 bg-app-surface-muted p-2 text-xs text-app-foreground"
            >
              <p className="font-medium">{t(strings.workspace.create.warningsLabel)}</p>
              <ul className="mt-1 list-disc pl-4 text-app-muted-foreground">
                {create.warnings.map((warning, index) => (
                  <li key={`${index}-${warning}`}>{warning}</li>
                ))}
              </ul>
            </div>
          )}

          {create.phase === "succeeded" && (
            <p
              data-testid={selectors.workspace.create.succeeded}
              className="text-sm text-app-success"
            >
              {t(strings.workspace.create.succeeded, { count: create.results.length })}
            </p>
          )}

          {create.phase === "failed" && (
            <div className="flex flex-col gap-2">
              <p data-testid={selectors.workspace.create.failed} className="text-sm text-app-danger">
                {create.error ?? t(strings.workspace.create.failed)}
              </p>
              {create.consentBlocked && (
                <p
                  data-testid={selectors.workspace.create.consentBlocked}
                  className="text-xs text-app-warning"
                >
                  {t(strings.workspace.create.consent.required)}
                </p>
              )}
              <Button
                variant="outline"
                type="button"
                data-testid={selectors.workspace.create.retry}
                onClick={create.retry}
              >
                {t(strings.workspace.create.retry)}
              </Button>
            </div>
          )}

          <VariationGrid
            results={create.results}
            requestedCount={create.requestedCount}
            busy={busy}
            onSendToCanvas={onSendToCanvas}
            onSendToEnhance={onSendToEnhance}
          />
        </div>
      )}
    </section>
  );
}
