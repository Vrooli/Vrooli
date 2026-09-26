import { useId, useRef, type ChangeEvent } from "react";

import { SegmentedControl } from "../../components/ui/segmented-control";
import { Slider } from "../../components/ui/slider";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DiffMode, type DiffResult } from "../../api/diff";
import { useCompare, type CompareClient, type CompareSlot } from "./useCompare";

export interface CompareViewProps {
  /** Injected seam; defaults to the live client. Injected in tests. */
  client?: CompareClient;
}

/** The two segmented-control mode keys (stable, locale-independent). */
type ModeKey = "pixel" | "perceptual";

const MODE_BY_KEY: Record<ModeKey, DiffMode> = {
  pixel: DiffMode.PIXEL,
  perceptual: DiffMode.PERCEPTUAL,
};

const KEY_BY_MODE = (mode: DiffMode): ModeKey => (mode === DiffMode.PERCEPTUAL ? "perceptual" : "pixel");

// Tolerance slider works in whole-percent steps (0..100) over the 0..1 param.
const TOLERANCE_MAX_PCT = 100;

/**
 * The visual-compare surface: two labelled dropzones (base + compare) each with
 * a file-picker fallback (no pointer-only — the button + hidden file input are
 * the keyboard/no-pointer path), a Pixel/Perceptual mode toggle + a tolerance
 * slider, a Compare action, then a results panel — the verdict chip, the
 * difference heat-map, a side-by-side of the two inputs, the metrics readout,
 * and any warnings.
 */
export function CompareView({ client }: CompareViewProps = {}) {
  const { t } = useTranslation();
  const c = useCompare(client);
  const headingId = useId();

  return (
    <section
      data-testid={selectors.compare.root}
      aria-labelledby={headingId}
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-2">
        <h2 id={headingId} className="text-2xl font-semibold">
          {t(strings.compare.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.compare.description)}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Dropzone
          slot="base"
          label={t(strings.compare.baseLabel)}
          chooseLabel={t(strings.compare.chooseBase)}
          dropHint={t(strings.compare.dropHint)}
          alt={t(strings.compare.baseAlt)}
          imageUrl={c.baseUrl}
          dropzoneTestId={selectors.compare.baseDropzone}
          fileInputTestId={selectors.compare.baseFileInput}
          imageTestId={selectors.compare.baseImage}
          onFile={c.setImage}
        />
        <Dropzone
          slot="compare"
          label={t(strings.compare.compareLabel)}
          chooseLabel={t(strings.compare.chooseCompare)}
          dropHint={t(strings.compare.dropHint)}
          alt={t(strings.compare.compareAlt)}
          imageUrl={c.compareUrl}
          dropzoneTestId={selectors.compare.compareDropzone}
          fileInputTestId={selectors.compare.compareFileInput}
          imageTestId={selectors.compare.compareImage}
          onFile={c.setImage}
        />
      </div>

      <div className="flex flex-wrap items-end gap-4">
        <div className="flex flex-col gap-1">
          <span className="text-xs text-app-muted-foreground">{t(strings.compare.mode)}</span>
          <SegmentedControl<ModeKey>
            label={t(strings.compare.mode)}
            value={KEY_BY_MODE(c.mode)}
            data-testid={selectors.compare.modeToggle}
            optionTestId={(mode) => selectors.compare.modeOption({ mode })}
            options={[
              { value: "pixel", label: t(strings.compare.modePixel) },
              { value: "perceptual", label: t(strings.compare.modePerceptual) },
            ]}
            onChange={(key) => c.setMode(MODE_BY_KEY[key])}
          />
        </div>
        <div className="min-w-[12rem] flex-1">
          <Slider
            label={t(strings.compare.tolerance)}
            value={Math.round(c.tolerance * TOLERANCE_MAX_PCT)}
            min={0}
            max={TOLERANCE_MAX_PCT}
            unit="%"
            defaultValue={0}
            resetLabel={t(strings.compare.toleranceReset)}
            data-testid={selectors.compare.toleranceInput}
            onChange={(pct) => c.setTolerance(pct / TOLERANCE_MAX_PCT)}
          />
        </div>
        <button
          type="button"
          data-testid={selectors.compare.runButton}
          disabled={!c.canCompare || c.phase === "comparing"}
          onClick={c.runCompare}
          className="rounded-control bg-app-primary px-4 py-2 text-sm font-medium text-app-primary-foreground disabled:opacity-50"
        >
          {t(strings.compare.run)}
        </button>
        <button
          type="button"
          data-testid={selectors.compare.reset}
          onClick={c.reset}
          className="rounded-control px-3 py-2 text-sm text-app-muted-foreground hover:text-app-foreground"
        >
          {t(strings.compare.reset)}
        </button>
      </div>

      {c.phase === "comparing" ? (
        <p data-testid={selectors.compare.status} className="text-sm text-app-muted-foreground">
          {t(strings.compare.comparing)}
        </p>
      ) : null}
      {c.error ? (
        <p data-testid={selectors.compare.error} role="alert" className="text-sm text-app-danger">
          {t(strings.compare.error)}
        </p>
      ) : null}

      {c.result ? <Results result={c.result} heatmapUrl={c.heatmapUrl} baseUrl={c.baseUrl} compareUrl={c.compareUrl} /> : null}
    </section>
  );
}

interface DropzoneProps {
  slot: CompareSlot;
  label: string;
  chooseLabel: string;
  dropHint: string;
  alt: string;
  imageUrl: string | null;
  dropzoneTestId: string;
  fileInputTestId: string;
  imageTestId: string;
  onFile: (slot: CompareSlot, file: File) => void;
}

/**
 * One labelled input slot. The visible button opens the native file picker (the
 * keyboard / no-pointer path); a hidden, labelled `<input type="file">` carries
 * the actual selection so there is never a pointer-only route to choosing an
 * image. Once a file is picked its preview replaces the prompt.
 */
function Dropzone({
  slot,
  label,
  chooseLabel,
  dropHint,
  alt,
  imageUrl,
  dropzoneTestId,
  fileInputTestId,
  imageTestId,
  onFile,
}: DropzoneProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const labelId = useId();

  const onChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      onFile(slot, file);
    }
  };

  return (
    <div className="flex flex-col gap-2" aria-labelledby={labelId}>
      <span id={labelId} className="text-sm font-medium">
        {label}
      </span>
      <div
        data-testid={dropzoneTestId}
        className="flex flex-col items-center gap-3 rounded-card border border-dashed border-app-border bg-app-surface-muted p-6"
      >
        {imageUrl ? (
          <img
            data-testid={imageTestId}
            src={imageUrl}
            alt={alt}
            className="block max-h-48 w-auto rounded-control object-contain"
          />
        ) : (
          <p className="text-sm text-app-muted-foreground">{dropHint}</p>
        )}
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          className="rounded-control border border-app-border px-3 py-1.5 text-sm font-medium hover:bg-app-surface"
        >
          {chooseLabel}
        </button>
        <input
          ref={fileInputRef}
          data-testid={fileInputTestId}
          type="file"
          accept="image/*"
          className="sr-only"
          aria-label={chooseLabel}
          onChange={onChange}
        />
      </div>
    </div>
  );
}

interface ResultsProps {
  result: DiffResult;
  heatmapUrl: string | null;
  baseUrl: string | null;
  compareUrl: string | null;
}

/** The verdict chip + heat-map + side-by-side inputs + metrics + warnings. */
function Results({ result, heatmapUrl, baseUrl, compareUrl }: ResultsProps) {
  const { t } = useTranslation();

  const changedPercent = round1(result.changedFraction * 100);

  return (
    <div data-testid={selectors.compare.result} className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-sm font-semibold text-app-muted-foreground">{t(strings.compare.verdictHeading)}</span>
        <VerdictChip verdict={result.verdict} />
        {!result.dimensionsMatch ? (
          <span className="text-xs text-app-warning">{t(strings.compare.dimensionsMismatch)}</span>
        ) : null}
      </div>

      {heatmapUrl ? (
        <figure className="flex flex-col gap-2">
          <figcaption className="text-sm font-semibold text-app-muted-foreground">
            {t(strings.compare.heatmapHeading)}
          </figcaption>
          <img
            data-testid={selectors.compare.heatmap}
            src={heatmapUrl}
            alt={t(strings.compare.heatmapAlt)}
            className="block max-h-[50vh] w-auto rounded-card border border-app-border object-contain"
          />
        </figure>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2">
        {baseUrl ? (
          <figure className="flex flex-col gap-1">
            <figcaption className="text-xs text-app-muted-foreground">{t(strings.compare.baseLabel)}</figcaption>
            <img
              src={baseUrl}
              alt={t(strings.compare.baseAlt)}
              className="block max-h-64 w-full rounded-control border border-app-border object-contain"
            />
          </figure>
        ) : null}
        {compareUrl ? (
          <figure className="flex flex-col gap-1">
            <figcaption className="text-xs text-app-muted-foreground">{t(strings.compare.compareLabel)}</figcaption>
            <img
              src={compareUrl}
              alt={t(strings.compare.compareAlt)}
              className="block max-h-64 w-full rounded-control border border-app-border object-contain"
            />
          </figure>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold text-app-muted-foreground">{t(strings.compare.metricsHeading)}</h3>
        <dl data-testid={selectors.compare.metrics} className="grid gap-x-6 gap-y-2 sm:grid-cols-2">
        <Metric
          label={t(strings.compare.metricChanged)}
          value={t(strings.compare.metricChangedValue, {
            percent: changedPercent,
            changed: result.changedPixels.toString(),
            total: result.totalPixels.toString(),
          })}
        />
        <Metric label={t(strings.compare.metricMae)} value={round2(result.mae).toString()} />
        <Metric label={t(strings.compare.metricRmse)} value={round2(result.rmse).toString()} />
        <Metric label={t(strings.compare.metricPsnr)} value={t(strings.compare.metricPsnrValue, { value: round1(result.psnr) })} />
        <Metric label={t(strings.compare.metricPhashDistance)} value={result.phashDistance.toString()} />
        <Metric label={t(strings.compare.metricPhashSimilarity)} value={round3(result.phashSimilarity).toString()} />
          <Metric label={t(strings.compare.metricSsim)} value={round3(result.ssim).toString()} />
        </dl>
      </div>

      {result.warnings.length > 0 ? (
        <div data-testid={selectors.compare.warnings} className="flex flex-col gap-1">
          <span className="text-sm font-semibold text-app-warning">{t(strings.compare.warningsHeading)}</span>
          <ul className="list-disc pl-5 text-sm text-app-muted-foreground">
            {result.warnings.map((w) => (
              <li key={w}>{w}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-2 border-b border-app-border py-1">
      <dt className="text-sm text-app-muted-foreground">{label}</dt>
      <dd className="text-sm font-medium tabular-nums text-app-foreground">{value}</dd>
    </div>
  );
}

/** The verdict styled as a tone-coded chip; falls back to the raw verdict. */
function VerdictChip({ verdict }: { verdict: string }) {
  const { t } = useTranslation();
  const tone =
    verdict === "identical"
      ? "bg-app-success/15 text-app-success"
      : verdict === "similar"
        ? "bg-app-warning/15 text-app-warning"
        : "bg-app-danger/15 text-app-danger";
  const label =
    verdict === "identical"
      ? t(strings.compare.verdictIdentical)
      : verdict === "similar"
        ? t(strings.compare.verdictSimilar)
        : verdict === "different"
          ? t(strings.compare.verdictDifferent)
          : verdict;
  return (
    <span
      data-testid={selectors.compare.verdict}
      className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-semibold ${tone}`}
    >
      {label}
    </span>
  );
}

const round = (value: number, digits: number): number => {
  const f = 10 ** digits;
  return Math.round(value * f) / f;
};
const round1 = (v: number) => round(v, 1);
const round2 = (v: number) => round(v, 2);
const round3 = (v: number) => round(v, 3);
