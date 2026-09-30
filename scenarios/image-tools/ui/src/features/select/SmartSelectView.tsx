import { useId, useRef, useState, type ChangeEvent, type MouseEvent } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type SuggestedEdit } from "../../api/selection";
import { useSmartSelect, type SmartSelectClient } from "./useSmartSelect";

export interface SmartSelectViewProps {
  /** Injected seam; defaults to the live client. Injected in tests. */
  client?: SmartSelectClient;
}

const REGION_LABEL: Record<string, (typeof strings.select.class)[keyof typeof strings.select.class]> = {
  person: strings.select.class.person,
  sky: strings.select.class.sky,
  foliage: strings.select.class.foliage,
  background: strings.select.class.background,
  object: strings.select.class.object,
};

/**
 * The smart-select surface: load an image, click (or auto/coordinate-select) a
 * region — the edge snaps to the silhouette and classifies it — then pick a
 * context-aware edit that runs as a masked AI op. The click is the pointer
 * path; the "Auto-select subject" button and the X/Y coordinate inputs are the
 * keyboard / no-pointer fallbacks (the no-pointer-only tenet).
 */
export function SmartSelectView({ client }: SmartSelectViewProps = {}) {
  const { t } = useTranslation();
  const s = useSmartSelect(client);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const [px, setPx] = useState("50");
  const [py, setPy] = useState("50");
  const headingId = useId();

  const onFile = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      s.setImage(file);
    }
  };

  const onImageClick = (e: MouseEvent<HTMLImageElement>) => {
    const el = imageRef.current;
    if (!el) {
      return;
    }
    const rect = el.getBoundingClientRect();
    if (rect.width === 0 || rect.height === 0) {
      return;
    }
    const nx = (e.clientX - rect.left) / rect.width;
    const ny = (e.clientY - rect.top) / rect.height;
    s.selectPoint(nx, ny);
  };

  const onSelectPointButton = () => {
    const nx = clampPct(px) / 100;
    const ny = clampPct(py) / 100;
    s.selectPoint(nx, ny);
  };

  return (
    <section
      data-testid={selectors.select.root}
      aria-labelledby={headingId}
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-2">
        <h2 id={headingId} className="text-2xl font-semibold">
          {t(strings.select.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.select.description)}</p>
      </div>

      {!s.image ? (
        <div
          data-testid={selectors.select.dropzone}
          className="flex flex-col items-center gap-3 rounded-card border border-dashed border-app-border bg-app-surface-muted p-10"
        >
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className="rounded-control bg-app-primary px-4 py-2 text-sm font-medium text-app-primary-foreground"
          >
            {t(strings.select.chooseImage)}
          </button>
          <input
            ref={fileInputRef}
            data-testid={selectors.select.fileInput}
            type="file"
            accept="image/*"
            className="sr-only"
            aria-label={t(strings.select.chooseImage)}
            onChange={onFile}
          />
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-[2fr_1fr]">
          {/* Work surface: image + mask overlay + the pointer + fallback controls. */}
          <div className="flex flex-col gap-3">
            <p className="text-sm text-app-muted-foreground">{t(strings.select.pointerHint)}</p>
            <div className="relative inline-block max-w-full overflow-hidden rounded-card border border-app-border">
              {s.imageUrl ? (
                // The image click is a pointer ENHANCEMENT for placing a seed
                // point; the keyboard / no-pointer path is the X/Y coordinate
                // inputs + the auto-select button below (the no-pointer-only
                // tenet), so the click handler needs no key listener of its own.
                // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions, jsx-a11y/click-events-have-key-events
                <img
                  ref={imageRef}
                  data-testid={selectors.select.image}
                  src={s.imageUrl}
                  alt={t(strings.select.imageAlt)}
                  onClick={onImageClick}
                  className="block max-h-[60vh] w-auto cursor-crosshair select-none"
                />
              ) : null}
              {s.maskUrl ? (
                <img
                  data-testid={selectors.select.maskOverlay}
                  src={s.maskUrl}
                  alt={t(strings.select.maskAlt)}
                  className="pointer-events-none absolute inset-0 h-full w-full object-contain opacity-50 mix-blend-screen"
                />
              ) : null}
            </div>

            <div className="flex flex-wrap items-end gap-3">
              <button
                type="button"
                data-testid={selectors.select.autoButton}
                onClick={s.selectAuto}
                className="rounded-control border border-app-border px-3 py-1.5 text-sm font-medium hover:bg-app-surface-muted"
              >
                {t(strings.select.autoSelect)}
              </button>
              <label className="flex flex-col text-xs text-app-muted-foreground">
                {t(strings.select.pointX)}
                <input
                  data-testid={selectors.select.pointXInput}
                  type="number"
                  min={0}
                  max={100}
                  value={px}
                  onChange={(e) => setPx(e.target.value)}
                  className="mt-1 w-20 rounded-control border border-app-border bg-app-surface px-2 py-1 text-sm text-app-foreground"
                />
              </label>
              <label className="flex flex-col text-xs text-app-muted-foreground">
                {t(strings.select.pointY)}
                <input
                  data-testid={selectors.select.pointYInput}
                  type="number"
                  min={0}
                  max={100}
                  value={py}
                  onChange={(e) => setPy(e.target.value)}
                  className="mt-1 w-20 rounded-control border border-app-border bg-app-surface px-2 py-1 text-sm text-app-foreground"
                />
              </label>
              <button
                type="button"
                data-testid={selectors.select.selectPointButton}
                onClick={onSelectPointButton}
                className="rounded-control border border-app-border px-3 py-1.5 text-sm font-medium hover:bg-app-surface-muted"
              >
                {t(strings.select.selectPoint)}
              </button>
              <label className="flex flex-col text-xs text-app-muted-foreground">
                {t(strings.select.tolerance)}
                <input
                  data-testid={selectors.select.toleranceInput}
                  type="range"
                  min={2}
                  max={60}
                  value={Math.round((s.tolerance || 0.14) * 100)}
                  onChange={(e) => s.setTolerance(Number(e.target.value) / 100)}
                  className="mt-1 w-28"
                />
              </label>
              <button
                type="button"
                data-testid={selectors.select.reset}
                onClick={s.reset}
                className="rounded-control px-3 py-1.5 text-sm text-app-muted-foreground hover:text-app-foreground"
              >
                {t(strings.select.reset)}
              </button>
            </div>
          </div>

          {/* Inspector: status, classification, and the contextual edit menu. */}
          <div className="flex flex-col gap-4">
            {s.phase === "segmenting" ? (
              <p data-testid={selectors.select.status} className="text-sm text-app-muted-foreground">
                {t(strings.select.segmenting)}
              </p>
            ) : null}
            {s.error ? (
              <p data-testid={selectors.select.error} role="alert" className="text-sm text-app-danger">
                {t(strings.select.error)}
              </p>
            ) : null}
            {s.result ? (
              <SelectionMenu
                regionClass={s.result.regionClass}
                regionLabel={t(REGION_LABEL[s.result.regionClass] ?? strings.select.class.object)}
                confidence={s.result.confidence}
                edits={s.result.suggestedEdits}
                applying={s.phase === "applying"}
                onApply={s.applyEdit}
              />
            ) : null}
            {s.outcome ? <Outcome outcome={s.outcome} /> : null}
          </div>
        </div>
      )}
    </section>
  );
}

interface SelectionMenuProps {
  regionClass: string;
  regionLabel: string;
  confidence: number;
  edits: SuggestedEdit[];
  applying: boolean;
  onApply: (edit: SuggestedEdit, promptText: string) => void;
}

function SelectionMenu({ regionLabel, confidence, edits, applying, onApply }: SelectionMenuProps) {
  const { t } = useTranslation();
  const [prompts, setPrompts] = useState<Record<string, string>>({});

  return (
    <div className="flex flex-col gap-3">
      <p data-testid={selectors.select.regionClass} className="text-sm font-medium">
        {t(strings.select.region, { class: regionLabel })}{" "}
        <span className="text-app-muted-foreground">
          {t(strings.select.confidence, { percent: Math.round(confidence * 100) })}
        </span>
      </p>
      <h3 data-testid={selectors.select.editsHeading} className="text-sm font-semibold text-app-muted-foreground">
        {t(strings.select.editsHeading)}
      </h3>
      {edits.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.select.noEdits)}</p>
      ) : (
        <ul data-testid={selectors.select.editsList} className="flex flex-col gap-2">
          {edits.map((edit) => (
            <li key={edit.id} className="flex flex-col gap-1">
              {edit.requiresPrompt ? (
                <input
                  data-testid={selectors.select.promptInput}
                  type="text"
                  value={prompts[edit.id] ?? ""}
                  placeholder={t(strings.select.promptPlaceholder)}
                  onChange={(e) => setPrompts((p) => ({ ...p, [edit.id]: e.target.value }))}
                  className="rounded-control border border-app-border bg-app-surface px-2 py-1 text-sm text-app-foreground"
                />
              ) : null}
              <button
                type="button"
                data-testid={selectors.select.editButton({ id: edit.id })}
                disabled={applying}
                onClick={() => onApply(edit, prompts[edit.id] ?? "")}
                className="rounded-control border border-app-border px-3 py-2 text-left text-sm hover:bg-app-surface-muted disabled:opacity-50"
              >
                <span className="font-medium">{edit.label}</span>
                <span className="block text-xs text-app-muted-foreground">{edit.description}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {applying ? <p className="text-sm text-app-muted-foreground">{t(strings.select.applying)}</p> : null}
    </div>
  );
}

function Outcome({ outcome }: { outcome: NonNullable<ReturnType<typeof useSmartSelect>["outcome"]> }) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.select.applyResult} className="rounded-card border border-app-border bg-app-surface-muted p-3 text-sm">
      {outcome.kind === "submitted" ? (
        <p>{t(strings.select.submitted, { jobId: outcome.jobId ?? "" })}</p>
      ) : outcome.kind === "gated" ? (
        <p className="text-app-warning">{t(strings.select.gate)}</p>
      ) : (
        <p className="text-app-danger">{outcome.message}</p>
      )}
    </div>
  );
}

function clampPct(value: string): number {
  const n = Number(value);
  if (!Number.isFinite(n)) return 50;
  if (n < 0) return 0;
  if (n > 100) return 100;
  return n;
}
