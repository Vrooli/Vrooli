import { Fragment, useState } from "react";
import type { TFunction } from "i18next";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { StatusDot } from "../../../components/ui/status-dot";
import { Table, TBody, TD, TH, THead, TR } from "../../../components/ui/table";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import type { ModelHardware, ModelRow } from "../../../services/models";
import {
  capabilityLabel,
  commercialLabelKey,
  commercialTone,
  languageSummary,
  readinessLabelKey,
  readinessTone,
} from "../modelDisplay";
import { Detail } from "./Detail";

interface Props {
  models: ModelRow[];
}

/**
 * Registry table. The server guarantees at most one default per capability,
 * so `isDefault` renders as-is instead of being recomputed here.
 */
export function ModelsTable({ models }: Props) {
  const { t } = useTranslation();
  const dash = t(strings.common.dash);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  return (
    <Table>
        <THead>
          <TR>
            <TH className="w-[22%]">{t(strings.models.colModel)}</TH>
            <TH className="w-[10%]">{t(strings.models.colCapability)}</TH>
            <TH className="w-[11%]">{t(strings.models.colReadiness)}</TH>
            <TH className="w-[11%]">{t(strings.models.colCommercial)}</TH>
            <TH className="w-[10%]">{t(strings.models.colLicense)}</TH>
            <TH className="w-[11%]">{t(strings.models.colRuntime)}</TH>
            <TH className="w-[8%]">{t(strings.models.colSize)}</TH>
            <TH className="w-[12%]">{t(strings.models.colLanguages)}</TH>
          </TR>
        </THead>
        <TBody>
          {models.map((model) => {
            const expanded = expandedId === model.id;
            const detailsId = `model-details-${model.id}`;
            const summary = languageSummary(model.languages);
            return (
              <Fragment key={model.id}>
                <TR>
                  <TD>
                    <div className="flex flex-col gap-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{model.name || model.id}</span>
                        {model.isDefault ? (
                          <Badge variant="primary">
                            {t(strings.models.defaultBadge)}
                          </Badge>
                        ) : null}
                      </div>
                      <span className="font-mono text-xs text-app-muted-foreground">
                        {model.id}
                      </span>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto justify-start p-0 text-xs"
                        aria-expanded={expanded}
                        aria-controls={detailsId}
                        onClick={() => setExpandedId(expanded ? null : model.id)}
                      >
                        {expanded
                          ? t(strings.models.detailsHide)
                          : t(strings.models.detailsToggle)}
                      </Button>
                    </div>
                  </TD>
                  <TD className="text-xs">
                    {capabilityLabel(model.capability, t)}
                  </TD>
                  <TD>
                    <StatusDot
                      tone={readinessTone(model.readiness)}
                      label={t(readinessLabelKey(model.readiness))}
                    />
                  </TD>
                  <TD>
                    <StatusDot
                      tone={commercialTone(model.commercialUse)}
                      label={t(commercialLabelKey(model.commercialUse))}
                    />
                  </TD>
                  <TD className="text-xs">{model.license || dash}</TD>
                  <TD className="text-xs">{model.runtime || dash}</TD>
                  <TD className="text-xs">
                    {model.sizeMbApprox > 0
                      ? t(strings.models.sizeMb, { mb: model.sizeMbApprox })
                      : dash}
                  </TD>
                  <TD className="text-xs">
                    {summary.shown.length === 0 ? (
                      dash
                    ) : (
                      <span>
                        {summary.shown.join(", ")}
                        {summary.overflow > 0 ? (
                          <span className="ml-1 text-app-muted-foreground">
                            {t(strings.models.languageOverflow, {
                              count: summary.overflow,
                            })}
                          </span>
                        ) : null}
                      </span>
                    )}
                  </TD>
                </TR>
                {expanded ? (
                  <TR>
                    <TD colSpan={8} className="bg-app-surface-muted/40">
                      <div id={detailsId}>
                        <ModelDetails model={model} />
                      </div>
                    </TD>
                  </TR>
                ) : null}
              </Fragment>
            );
          })}
        </TBody>
    </Table>
  );
}

function ModelDetails({ model }: { model: ModelRow }) {
  const { t } = useTranslation();
  const dash = t(strings.common.dash);
  const notEstablished = t(strings.models.notEstablished);

  return (
    <dl className="grid gap-x-6 gap-y-3 px-1 py-2 text-xs sm:grid-cols-2">
      <Detail label={t(strings.models.detailsProviders)}>
        {joinOr(model.providerIds, dash)}
      </Detail>
      <Detail label={t(strings.models.detailsEngines)}>
        {joinOr(model.engineIds, dash)}
      </Detail>
      <Detail label={t(strings.models.detailsEngineDefaults)}>
        {joinOr(model.engineDefaults, dash)}
      </Detail>
      <Detail label={t(strings.models.detailsRoles)}>
        {joinOr(model.roles, dash)}
      </Detail>
      <Detail label={t(strings.models.detailsParams)}>
        {model.params || dash}
      </Detail>
      <Detail label={t(strings.models.detailsHardware)}>
        {hardwareText(model.hardware, t)}
      </Detail>
      <Detail label={t(strings.models.detailsBenchmarks)}>
        {benchmarksText(model, t)}
      </Detail>
      <Detail label={t(strings.models.detailsSource)}>
        {sourceText(model, dash)}
      </Detail>
      <Detail label={t(strings.models.detailsLicenseNotes)}>
        {model.licenseNotes || notEstablished}
      </Detail>
      <Detail label={t(strings.models.detailsNotes)} className="sm:col-span-2">
        {model.notes || notEstablished}
      </Detail>
    </dl>
  );
}

function joinOr(values: string[], fallback: string): string {
  return values.length > 0 ? values.join(", ") : fallback;
}

function hardwareText(hardware: ModelHardware, t: TFunction): string {
  const parts: string[] = [];
  if (hardware.cpuCapable) parts.push(t(strings.models.hardwareCpu));
  if (hardware.gpuRequired) parts.push(t(strings.models.hardwareGpu));
  if (hardware.minVramGb > 0) {
    parts.push(t(strings.models.hardwareVram, { gb: hardware.minVramGb }));
  }
  if (hardware.minRamGb > 0) {
    parts.push(t(strings.models.hardwareRam, { gb: hardware.minRamGb }));
  }
  if (hardware.osArch.length > 0) parts.push(hardware.osArch.join(", "));
  return parts.length > 0 ? parts.join(" · ") : t(strings.models.notEstablished);
}

function benchmarksText(model: ModelRow, t: TFunction): string {
  const parts: string[] = [];
  if (model.benchmarks.wer > 0) {
    parts.push(t(strings.models.benchmarkWer, { value: model.benchmarks.wer }));
  }
  if (model.benchmarks.sim > 0) {
    parts.push(t(strings.models.benchmarkSim, { value: model.benchmarks.sim }));
  }
  if (model.benchmarks.rtf > 0) {
    parts.push(t(strings.models.benchmarkRtf, { value: model.benchmarks.rtf }));
  }
  if (model.benchmarks.notes) parts.push(model.benchmarks.notes);
  return parts.length > 0 ? parts.join(" · ") : t(strings.models.notEstablished);
}

function sourceText(model: ModelRow, fallback: string) {
  if (model.sourceUrl) {
    return (
      <a
        href={model.sourceUrl}
        target="_blank"
        rel="noreferrer"
        className="text-app-primary underline-offset-4 hover:underline"
      >
        {model.sourceUrl}
      </a>
    );
  }
  return model.sourceRepo || fallback;
}
