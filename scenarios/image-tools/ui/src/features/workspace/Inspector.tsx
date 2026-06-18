import { useState } from "react";

import { Button } from "../../components/ui/button";
import { ColorField } from "../../components/ui/color-field";
import { FilterThumbnailGrid } from "../../components/ui/filter-thumbnail-grid";
import { FormatPills } from "../../components/ui/format-pills";
import { Input } from "../../components/ui/input";
import { PositionPicker, type PositionToken } from "../../components/ui/position-picker";
import { SegmentedControl } from "../../components/ui/segmented-control";
import { Slider } from "../../components/ui/slider";
import { TargetSizeField } from "../../components/ui/target-size-field";
import { Toggle } from "../../components/ui/toggle";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import type { OperationInfo, OpParamValues } from "../../api/ops";
import {
  AXIS_OPTION_LABEL,
  FILTER_OPTION,
  FIT_OPTION_LABEL,
  POSITION_NAME_LABEL,
} from "./opCatalog";
import { OpPicker } from "./OpPicker";
import type { OpField, OpSpec } from "./opSpecs";
import { MODE_LABEL } from "./modeLabels";
import type { WorkspaceMode } from "./useWorkspace";

export interface InspectorProps {
  mode: WorkspaceMode;
  opsLoading: boolean;
  opsError: boolean;
  operations: OperationInfo[];
  operation: string;
  params: OpParamValues;
  spec: OpSpec | undefined;
  applying: boolean;
  runError: unknown;
  hasBase: boolean;
  hasSteps: boolean;
  /** Current canvas image URL, used to preview filter thumbnails. */
  previewUrl: string | null;
  onSelectOperation: (operation: string) => void;
  onParam: (name: string, value: string | number | boolean) => void;
  onApply: (overlay?: File) => void;
}

/** A `segmented`-control option-label map by op field name. */
const SEGMENTED_LABELS: Record<string, typeof FIT_OPTION_LABEL | typeof AXIS_OPTION_LABEL> = {
  fit: FIT_OPTION_LABEL,
  axis: AXIS_OPTION_LABEL,
};

/**
 * The mode-aware right panel. In Edit mode it renders the humanized
 * deterministic-op form: an icon+label op picker, the op description, and one
 * primitive per spec field (segmented / slider / position / color / format /
 * target-size / filter-grid / toggle), with crop's numeric box tucked under an
 * Advanced disclosure. Each control's primary interactive element keeps the
 * `fieldInput({ name })` test id so automation can drive the accessible path.
 * The non-edit modes keep the honest roadmap placeholder.
 */
export function Inspector({
  mode,
  opsLoading,
  opsError,
  operations,
  operation,
  params,
  spec,
  applying,
  runError,
  hasBase,
  hasSteps,
  previewUrl,
  onSelectOperation,
  onParam,
  onApply,
}: InspectorProps) {
  const { t } = useTranslation();
  const [overlay, setOverlay] = useState<File | null>(null);

  const renderControl = (field: OpField) => {
    const value = params[field.name];
    const testId = selectors.workspace.fieldInput({ name: field.name });
    const label = t(field.labelKey);

    switch (field.control) {
      case "toggle":
        return (
          <Toggle
            label={label}
            checked={Boolean(value)}
            onChange={(checked) => onParam(field.name, checked)}
            data-testid={testId}
          />
        );

      case "slider":
        return (
          <Slider
            label={label}
            value={Number(value ?? field.default)}
            min={field.min ?? 0}
            max={field.max ?? 100}
            step={field.step}
            unit={field.unit}
            defaultValue={Number(field.default)}
            resetLabel={t(strings.workspace.control.reset)}
            onChange={(next) => onParam(field.name, next)}
            data-testid={testId}
          />
        );

      case "segmented": {
        const labelMap = SEGMENTED_LABELS[field.name] ?? {};
        return (
          <div className="flex flex-col gap-1">
            <span className="text-xs text-app-muted-foreground">{label}</span>
            <SegmentedControl<string>
              label={label}
              value={String(value ?? field.default)}
              options={(field.options ?? []).map((token) => ({
                value: token,
                label: labelMap[token] ? t(labelMap[token]) : token,
              }))}
              onChange={(next) => onParam(field.name, next)}
              data-testid={testId}
            />
          </div>
        );
      }

      case "position":
        return (
          <div className="flex flex-col gap-1">
            <span className="text-xs text-app-muted-foreground">{label}</span>
            <PositionPicker
              label={label}
              value={String(value ?? "")}
              onChange={(token) => onParam(field.name, token)}
              cellLabel={(token: PositionToken) => t(POSITION_NAME_LABEL[token])}
              data-testid={testId}
            />
          </div>
        );

      case "color":
        return (
          <ColorField
            label={label}
            value={String(value ?? "")}
            onChange={(next) => onParam(field.name, next)}
            clearLabel={t(strings.workspace.color.clear)}
            alphaLabel={t(strings.workspace.color.alpha)}
            data-testid={testId}
          />
        );

      case "format":
        return (
          <div className="flex flex-col gap-1">
            <span className="text-xs text-app-muted-foreground">{label}</span>
            <FormatPills
              label={label}
              value={String(value ?? field.default)}
              options={field.options ?? []}
              onChange={(next) => onParam(field.name, next)}
              data-testid={testId}
            />
          </div>
        );

      case "targetSize":
        return (
          <TargetSizeField
            label={label}
            valueBytes={Number(value ?? 0)}
            onChange={(bytes) => onParam(field.name, bytes)}
            kbLabel={t(strings.workspace.target.kb)}
            mbLabel={t(strings.workspace.target.mb)}
            noLimitLabel={t(strings.workspace.target.noLimit)}
            data-testid={testId}
          />
        );

      case "filterGrid":
        return (
          <div className="flex flex-col gap-1">
            <span className="text-xs text-app-muted-foreground">{label}</span>
            <FilterThumbnailGrid
              label={label}
              value={String(value ?? field.default)}
              options={(field.options ?? []).map((token) => ({
                value: token,
                label: FILTER_OPTION[token] ? t(FILTER_OPTION[token].labelKey) : token,
                css: FILTER_OPTION[token]?.css ?? "none",
              }))}
              previewUrl={previewUrl}
              onChange={(next) => onParam(field.name, next)}
              data-testid={testId}
            />
          </div>
        );

      case "number":
      case "text":
      default:
        return (
          <div className="flex flex-col gap-1">
            <label
              htmlFor={`workspace-field-${field.name}`}
              className="text-xs text-app-muted-foreground"
            >
              {label}
            </label>
            <Input
              id={`workspace-field-${field.name}`}
              data-testid={testId}
              type={field.control === "number" ? "number" : "text"}
              value={String(value ?? "")}
              onChange={(e) =>
                onParam(
                  field.name,
                  field.control === "number" ? Number(e.target.value) : e.target.value,
                )
              }
            />
          </div>
        );
    }
  };

  const primaryFields = (spec?.fields ?? []).filter((f) => !f.advanced);
  const advancedFields = (spec?.fields ?? []).filter((f) => f.advanced);

  return (
    <section
      data-testid={selectors.workspace.inspector}
      aria-label={t(MODE_LABEL[mode])}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-medium text-app-muted-foreground">{t(MODE_LABEL[mode])}</h3>

      {mode !== "edit" ? (
        <p
          data-testid={selectors.workspace.inspectorPlaceholder}
          className="mt-3 rounded-panel bg-app-surface-muted p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.workspace.mode.comingSoon, { mode: t(MODE_LABEL[mode]) })}
        </p>
      ) : opsLoading ? (
        <p data-testid={selectors.workspace.loading} className="mt-3 text-sm text-app-foreground">
          {t(strings.workspace.loading)}
        </p>
      ) : opsError ? (
        <p data-testid={selectors.workspace.error} className="mt-3 text-sm text-app-danger">
          {t(strings.workspace.error)}
        </p>
      ) : (
        <form
          data-testid={selectors.workspace.paramsForm}
          onSubmit={(e) => {
            e.preventDefault();
            onApply(spec?.acceptsOverlay ? overlay ?? undefined : undefined);
          }}
          className="mt-3 flex flex-col gap-4"
        >
          <OpPicker
            operations={operations}
            operation={operation}
            onSelect={onSelectOperation}
          />

          {operation === "crop" && (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.workspace.crop.hint)}
            </p>
          )}

          {primaryFields.length > 0 && (
            <div className="flex flex-col gap-3">
              {primaryFields.map((field) => (
                <div key={field.name}>{renderControl(field)}</div>
              ))}
            </div>
          )}

          {advancedFields.length > 0 && (
            <details className="rounded-control border border-app-border bg-app-surface-muted p-2">
              <summary
                data-testid={selectors.workspace.crop.advanced}
                className="cursor-pointer text-xs font-medium text-app-muted-foreground"
              >
                {t(strings.workspace.crop.advanced)}
              </summary>
              <div className="mt-2 grid grid-cols-2 gap-3">
                {advancedFields.map((field) => (
                  <div key={field.name}>{renderControl(field)}</div>
                ))}
              </div>
            </details>
          )}

          {spec?.acceptsOverlay && (
            <div className="flex flex-col gap-1">
              <label
                htmlFor={selectors.workspace.overlayInput}
                className="text-xs text-app-muted-foreground"
              >
                {t(strings.workspace.overlayLabel)}
              </label>
              <input
                id={selectors.workspace.overlayInput}
                data-testid={selectors.workspace.overlayInput}
                type="file"
                accept="image/*"
                onChange={(e) => setOverlay(e.target.files?.[0] ?? null)}
                className="block w-full text-xs text-app-muted-foreground file:mr-3 file:rounded-control file:border-0 file:bg-app-primary file:px-3 file:py-2 file:text-app-primary-foreground"
              />
            </div>
          )}

          <Button
            data-testid={selectors.workspace.applyButton}
            type="submit"
            disabled={!hasBase || !operation || applying}
          >
            {applying ? t(strings.workspace.running) : t(strings.workspace.run)}
          </Button>

          {runError != null && (
            <p data-testid={selectors.workspace.runError} className="text-sm text-app-danger">
              {errorMessage(runError, t)}
            </p>
          )}

          {hasBase && !hasSteps && !applying && (
            <p data-testid={selectors.workspace.empty} className="text-sm text-app-muted-foreground">
              {t(strings.workspace.empty)}
            </p>
          )}
        </form>
      )}
    </section>
  );
}
