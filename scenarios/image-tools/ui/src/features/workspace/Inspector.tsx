import { useState } from "react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import type { OperationInfo, OpParamValues } from "../../api/ops";
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
  onSelectOperation: (operation: string) => void;
  onParam: (name: string, value: string | number | boolean) => void;
  onApply: (overlay?: File) => void;
}

/**
 * The mode-aware right panel. In Stage 0b only Edit is wired — it hosts the
 * deterministic-op parameter form (migrated verbatim from the retired editor
 * card; humanized controls land in Stage 1). The AI modes render an honest
 * roadmap placeholder until their stage ships.
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
  onSelectOperation,
  onParam,
  onApply,
}: InspectorProps) {
  const { t } = useTranslation();
  const [overlay, setOverlay] = useState<File | null>(null);

  const renderField = (field: OpField) => {
    const value = params[field.name];
    const id = `workspace-field-${field.name}`;
    const testId = selectors.workspace.fieldInput({ name: field.name });

    if (field.kind === "checkbox") {
      return (
        <label key={field.name} className="flex items-center gap-2 text-sm text-app-foreground">
          <input
            id={id}
            data-testid={testId}
            type="checkbox"
            checked={Boolean(value)}
            onChange={(e) => onParam(field.name, e.target.checked)}
            className="h-4 w-4"
          />
          {t(field.labelKey)}
        </label>
      );
    }

    return (
      <div key={field.name} className="flex flex-col gap-1">
        <label htmlFor={id} className="text-xs text-app-muted-foreground">
          {t(field.labelKey)}
        </label>
        {field.kind === "select" ? (
          <select
            id={id}
            data-testid={testId}
            value={String(value ?? "")}
            onChange={(e) => onParam(field.name, e.target.value)}
            className="h-10 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground focus:outline-none focus:ring-2 focus:ring-app-primary/50"
          >
            {field.options?.map((option) => (
              <option key={option} value={option} className="bg-app-surface text-app-foreground">
                {option}
              </option>
            ))}
          </select>
        ) : (
          <Input
            id={id}
            data-testid={testId}
            type={field.kind === "number" ? "number" : "text"}
            value={String(value ?? "")}
            onChange={(e) =>
              onParam(field.name, field.kind === "number" ? Number(e.target.value) : e.target.value)
            }
          />
        )}
      </div>
    );
  };

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
          <div className="flex flex-col gap-1">
            <label
              htmlFor={selectors.workspace.operationSelect}
              className="text-xs text-app-muted-foreground"
            >
              {t(strings.workspace.operationLabel)}
            </label>
            <select
              id={selectors.workspace.operationSelect}
              data-testid={selectors.workspace.operationSelect}
              value={operation}
              onChange={(e) => onSelectOperation(e.target.value)}
              className="h-10 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground focus:outline-none focus:ring-2 focus:ring-app-primary/50"
            >
              {operations.map((op) => (
                <option key={op.name} value={op.name} className="bg-app-surface text-app-foreground">
                  {op.name}
                </option>
              ))}
            </select>
          </div>

          {spec && spec.fields.length > 0 && (
            <div className="grid grid-cols-2 gap-3">{spec.fields.map(renderField)}</div>
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
