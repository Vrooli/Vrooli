import { useState } from "react";
import { Check, Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Sheet } from "../../components/ui/sheet";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { CandidateModel } from "../../api/models";
import {
  PICKER_TONE_CLASS,
  alsoNeedsBackend,
  hostSummaryLine,
  present,
  type PickerStringKey,
} from "./modelPickerPresentation";
import type { UseModelPicker } from "./useModelPicker";

export interface ModelPickerProps {
  open: boolean;
  onClose: () => void;
  /** Operation the menu is for (used in the title). */
  operation: string;
  /** Human label for the operation (already localized). */
  operationLabel: string;
  picker: UseModelPicker;
  /** The currently-chosen model id (override) or "" for auto. */
  value: string;
  /** Choose a model and close. */
  onSelect: (id: string) => void;
}

function Chip({ keyText, values, className }: { keyText: PickerStringKey; values?: Record<string, string | number>; className: string }) {
  const { t } = useTranslation();
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${className}`}>
      {values ? t(keyText, values) : t(keyText)}
    </span>
  );
}

/**
 * ModelPicker is the host-aware model menu behind every AI action. It lists ALL
 * models that serve the operation — not just the one that would run — each with
 * an affirmative fit badge ("Runs on your GPU"), a ready-state status, and the
 * exact inline action it needs to become usable (download weights, install the
 * engine one-click, enable it, or manual setup steps). Models that can't run on
 * this host are shown de-emphasized for transparency, never hidden.
 */
export function ModelPicker({
  open,
  onClose,
  operation,
  operationLabel,
  picker,
  value,
  onSelect,
}: ModelPickerProps) {
  const { t } = useTranslation();
  const [manualOpen, setManualOpen] = useState<string>("");

  const activeId = value || picker.selectedId;
  const hostLine = hostSummaryLine(picker.host);
  const subtitle = (
    <span data-testid={selectors.models.picker.host}>
      {hostLine.values ? t(hostLine.key, hostLine.values) : t(hostLine.key)}
      {picker.host?.hasGpu
        ? ` · ${t(strings.models.picker.host.specs, {
            cores: picker.host.cpuCores,
            ram: picker.host.ramGb,
          })}`
        : ""}
    </span>
  );

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title={t(strings.models.picker.titleFor, { op: operationLabel })}
      subtitle={subtitle}
      closeLabel={t(strings.models.picker.close)}
      testId={selectors.models.picker.sheet}
    >
      <div className="flex flex-col gap-2 p-3">
        {picker.loading && picker.candidates.length === 0 ? (
          <p data-testid={selectors.models.picker.loading} className="p-4 text-sm text-app-muted-foreground">
            {t(strings.models.picker.loading)}
          </p>
        ) : picker.error ? (
          <p data-testid={selectors.models.picker.error} className="p-4 text-sm text-app-danger">
            {picker.error}
          </p>
        ) : (
          picker.candidates.map((candidate) => (
            <ModelRow
              key={candidate.model?.id ?? ""}
              candidate={candidate}
              picker={picker}
              active={candidate.model?.id === activeId}
              manualOpen={manualOpen === candidate.model?.id}
              onToggleManual={() =>
                setManualOpen((prev) => (prev === candidate.model?.id ? "" : candidate.model?.id ?? ""))
              }
              onSelect={() => {
                onSelect(candidate.model?.id ?? "");
                onClose();
              }}
            />
          ))
        )}

        <p
          data-testid={selectors.models.picker.footer}
          className="px-1 pt-1 text-[11px] text-app-muted-foreground"
        >
          {picker.candidates.length <= 1
            ? t(strings.models.picker.footer.none)
            : t(strings.models.picker.footer.all, { count: picker.candidates.length })}
        </p>
        <p className="sr-only" aria-hidden="true">
          {operation}
        </p>
      </div>
    </Sheet>
  );
}

function ModelRow({
  candidate,
  picker,
  active,
  manualOpen,
  onToggleManual,
  onSelect,
}: {
  candidate: CandidateModel;
  picker: UseModelPicker;
  active: boolean;
  manualOpen: boolean;
  onToggleManual: () => void;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  const model = candidate.model;
  const id = model?.id ?? "";
  const view = present(candidate, picker.host);
  const busy = picker.busyId === id;
  const rowError = picker.rowError[id];
  const backend = candidate.backend;

  return (
    <div
      data-testid={selectors.models.pickerRow({ id })}
      className={
        active
          ? "rounded-control border border-app-primary bg-app-primary/5 p-3"
          : view.dimmed
            ? "rounded-control border border-app-border bg-app-surface-muted/40 p-3 opacity-70"
            : "rounded-control border border-app-border bg-app-surface p-3"
      }
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            {active ? <Check aria-hidden="true" className="h-4 w-4 shrink-0 text-app-primary" /> : null}
            <span className="truncate text-sm font-medium text-app-foreground">{model?.name}</span>
            <span className="shrink-0 rounded bg-app-surface-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-muted-foreground">
              {model?.tier}
            </span>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <Chip
              keyText={view.fit.key}
              values={view.fit.values}
              className={PICKER_TONE_CLASS[view.fit.tone]}
            />
            <Chip
              keyText={view.status.key}
              values={view.status.values}
              className={PICKER_TONE_CLASS[view.status.tone]}
            />
            {model && model.sizeMbApprox > 0 ? (
              <span className="text-[11px] text-app-muted-foreground">
                {t(strings.models.picker.size, { size: model.sizeMbApprox })}
              </span>
            ) : null}
          </div>
          {candidate.fit?.vramShortfallGb && view.dimmed ? (
            <p className="mt-1 text-[11px] text-app-muted-foreground">
              {t(strings.models.picker.fit.insufficientVram, {
                gb: candidate.fit.vramShortfallGb,
              })}
            </p>
          ) : null}
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          <RowAction
            candidate={candidate}
            view={view}
            active={active}
            busy={busy}
            onSelect={onSelect}
            onToggleManual={onToggleManual}
            installModel={() => picker.installModel(id)}
            installBackend={() => picker.installBackend(backend?.hostTool ?? "", id)}
            enable={() => picker.enable(id)}
          />
          {alsoNeedsBackend(candidate) ? (
            <button
              type="button"
              data-testid={selectors.models.pickerInstallBackend({ id })}
              disabled={busy}
              onClick={() => picker.installBackend(backend?.hostTool ?? "", id)}
              className="text-[11px] text-app-info hover:underline disabled:opacity-60"
            >
              {t(strings.models.picker.action.installBackend)}
            </button>
          ) : null}
        </div>
      </div>

      {manualOpen && backend ? (
        <div
          data-testid={selectors.models.pickerManual({ id })}
          className="mt-2 rounded-control border border-app-border bg-app-surface-muted p-2"
        >
          <p className="text-[11px] font-medium text-app-foreground">
            {t(strings.models.picker.manualTitle)}
          </p>
          <pre className="mt-1 whitespace-pre-wrap break-words text-[11px] text-app-muted-foreground">
            {backend.manualHint || backend.detail}
          </pre>
        </div>
      ) : null}

      {rowError ? (
        <p
          data-testid={selectors.models.pickerRowError({ id })}
          className="mt-2 text-[11px] text-app-danger"
        >
          {rowError}
        </p>
      ) : null}
    </div>
  );
}

function RowAction({
  candidate,
  view,
  active,
  busy,
  onSelect,
  onToggleManual,
  installModel,
  installBackend,
  enable,
}: {
  candidate: CandidateModel;
  view: ReturnType<typeof present>;
  active: boolean;
  busy: boolean;
  onSelect: () => void;
  onToggleManual: () => void;
  installModel: () => void;
  installBackend: () => void;
  enable: () => void;
}) {
  const { t } = useTranslation();
  const id = candidate.model?.id ?? "";

  if (busy) {
    return (
      <span className="inline-flex items-center gap-1 text-[11px] text-app-muted-foreground">
        <Loader2 aria-hidden="true" className="h-3.5 w-3.5 animate-spin text-app-brand" />
        {t(strings.models.picker.action.installing)}
      </span>
    );
  }

  switch (view.action) {
    case "select":
      return active ? (
        <span data-testid={selectors.models.pickerInUse({ id })} className="text-[11px] font-medium text-app-primary">
          {t(strings.models.picker.action.selected)}
        </span>
      ) : (
        <Button
          type="button"
          data-testid={selectors.models.pickerSelect({ id })}
          onClick={onSelect}
          className="px-2 py-1 text-xs"
        >
          {t(strings.models.picker.action.select)}
        </Button>
      );
    case "install-model":
      return (
        <Button
          type="button"
          variant="outline"
          data-testid={selectors.models.pickerInstallModel({ id })}
          onClick={installModel}
          className="px-2 py-1 text-xs"
        >
          {t(strings.models.picker.action.installModel)}
        </Button>
      );
    case "install-backend":
      return (
        <Button
          type="button"
          variant="outline"
          data-testid={selectors.models.pickerInstallBackend({ id })}
          onClick={installBackend}
          className="px-2 py-1 text-xs"
        >
          {t(strings.models.picker.action.installBackend)}
        </Button>
      );
    case "enable":
      return (
        <Button
          type="button"
          variant="outline"
          data-testid={selectors.models.pickerEnable({ id })}
          onClick={enable}
          className="px-2 py-1 text-xs"
        >
          {t(strings.models.picker.action.enable)}
        </Button>
      );
    case "manual":
      return (
        <Button
          type="button"
          variant="outline"
          data-testid={selectors.models.pickerManualToggle({ id })}
          onClick={onToggleManual}
          className="px-2 py-1 text-xs"
        >
          {t(strings.models.picker.action.manualSteps)}
        </Button>
      );
    default:
      return null;
  }
}
