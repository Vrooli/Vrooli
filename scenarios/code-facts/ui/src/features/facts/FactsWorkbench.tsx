import { useMutation } from "@tanstack/react-query";
import {
  AlertTriangle,
  Braces,
  Database,
  FileSearch,
  Layers3,
  Loader2,
  Play,
  RefreshCcw,
} from "lucide-react";
import { FormEvent, useMemo, useState } from "react";

import {
  FactFamily,
  TargetKind,
  describeCodeFacts,
  moduleTarget,
  pathTarget,
  projectTarget,
  scenarioTarget,
} from "../../api/facts";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import type {
  CodeFactsReport,
  CodeTarget,
  Evidence,
  GenericFact,
  ParseUnit,
  Warning,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";
import {
  EvidenceStatus,
  SurfaceKind,
  SurfaceStatus,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

type TargetMode = "scenario" | "path" | "module" | "project";
type FamilyLabelKey = (typeof strings.facts.family)[keyof typeof strings.facts.family];

interface FamilyOption {
  family: FactFamily;
  labelKey: FamilyLabelKey;
}

const FAMILY_OPTIONS: FamilyOption[] = [
  { family: FactFamily.SURFACES, labelKey: strings.facts.family.surfaces },
  { family: FactFamily.PARSE_UNITS, labelKey: strings.facts.family.parseUnits },
  { family: FactFamily.IMPORTS, labelKey: strings.facts.family.imports },
  { family: FactFamily.SYMBOLS, labelKey: strings.facts.family.symbols },
  { family: FactFamily.REFERENCES, labelKey: strings.facts.family.references },
  { family: FactFamily.CALLS, labelKey: strings.facts.family.calls },
  { family: FactFamily.PROTO_ADOPTION, labelKey: strings.facts.family.protoAdoption },
  { family: FactFamily.ENDPOINT_PROOFS, labelKey: strings.facts.family.endpointProofs },
];

const DEFAULT_FAMILIES = new Set<FactFamily>([
  FactFamily.SURFACES,
  FactFamily.PARSE_UNITS,
  FactFamily.IMPORTS,
  FactFamily.SYMBOLS,
  FactFamily.REFERENCES,
  FactFamily.CALLS,
  FactFamily.PROTO_ADOPTION,
  FactFamily.ENDPOINT_PROOFS,
]);

export function FactsWorkbench() {
  const { t } = useTranslation();
  const [targetMode, setTargetMode] = useState<TargetMode>("scenario");
  const [targetValue, setTargetValue] = useState("code-facts");
  const [useCache, setUseCache] = useState(true);
  const [selectedFamilies, setSelectedFamilies] = useState<Set<FactFamily>>(
    () => new Set(DEFAULT_FAMILIES),
  );

  const selectedFamilyList = useMemo(
    () => FAMILY_OPTIONS.map((option) => option.family).filter((family) => selectedFamilies.has(family)),
    [selectedFamilies],
  );

  const factsMutation = useMutation({
    mutationFn: () =>
      describeCodeFacts({
        target: buildTarget(targetMode, targetValue.trim()),
        include: selectedFamilyList.length > 0 ? selectedFamilyList : [FactFamily.SURFACES],
        useCache,
      }),
  });

  const report = factsMutation.data;
  const allEvidence = useMemo(() => collectEvidence(report), [report]);
  const degradedWarnings = report?.warnings.filter((warning) => warning.status !== EvidenceStatus.PROVEN) ?? [];

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!targetValue.trim()) return;
    factsMutation.mutate();
  };

  const toggleFamily = (family: FactFamily) => {
    setSelectedFamilies((current) => {
      const next = new Set(current);
      if (next.has(family)) {
        next.delete(family);
      } else {
        next.add(family);
      }
      return next;
    });
  };

  return (
    <section
      data-testid={selectors.facts.workbench}
      aria-label={t(strings.facts.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <form onSubmit={submit} className="grid gap-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-app-foreground">
              <FileSearch aria-hidden="true" className="h-4 w-4" />
              {t(strings.facts.title)}
            </h2>
            <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">{t(strings.facts.description)}</p>
          </div>
          <div
            data-testid={selectors.facts.cache}
            className="inline-flex w-fit items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-xs text-app-muted-foreground"
          >
            <Database aria-hidden="true" className="h-3.5 w-3.5" />
            {report?.cache?.hit ? t(strings.facts.cacheHit) : t(strings.facts.cacheMiss)}
          </div>
        </div>

        <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]">
          <div className="grid gap-3 md:grid-cols-[12rem_minmax(0,1fr)]">
            <label className="grid gap-1 text-xs font-medium text-app-muted-foreground">
              {t(strings.facts.targetKind)}
              <select
                data-testid={selectors.facts.targetKind}
                className="h-10 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground"
                value={targetMode}
                onChange={(event) => setTargetMode(event.currentTarget.value as TargetMode)}
              >
                <option value="scenario">{t(strings.facts.target.scenario)}</option>
                <option value="path">{t(strings.facts.target.path)}</option>
                <option value="module">{t(strings.facts.target.module)}</option>
                <option value="project">{t(strings.facts.target.project)}</option>
              </select>
            </label>
            <label className="grid gap-1 text-xs font-medium text-app-muted-foreground">
              {t(strings.facts.targetValue)}
              <Input
                data-testid={selectors.facts.targetInput}
                value={targetValue}
                onChange={(event) => setTargetValue(event.currentTarget.value)}
                placeholder={t(strings.facts.targetPlaceholder)}
                className="border-app-border bg-app-surface-muted text-app-foreground placeholder:text-app-muted-foreground"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-end gap-2">
            <label className="inline-flex h-10 items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground">
              <input
                data-testid={selectors.facts.cacheToggle}
                type="checkbox"
                checked={useCache}
                onChange={(event) => setUseCache(event.currentTarget.checked)}
              />
              {t(strings.facts.useCache)}
            </label>
            <Button
              data-testid={selectors.facts.analyzeButton}
              type="submit"
              disabled={factsMutation.isPending || !targetValue.trim()}
              className="gap-2"
            >
              {factsMutation.isPending ? (
                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
              ) : (
                <Play aria-hidden="true" className="h-4 w-4" />
              )}
              {factsMutation.isPending ? t(strings.facts.loading) : t(strings.facts.analyze)}
            </Button>
          </div>
        </div>

        <div
          data-testid={selectors.facts.familyControls}
          className="flex flex-wrap gap-2"
          aria-label={t(strings.facts.familyControls)}
        >
          {FAMILY_OPTIONS.map((option) => {
            const active = selectedFamilies.has(option.family);
            return (
              <button
                key={option.family}
                type="button"
                data-testid={selectors.facts.familyToggle({ family: String(option.family) })}
                className={`h-9 rounded-control border px-3 text-sm transition-colors ${
                  active
                    ? "border-app-primary bg-app-primary text-app-primary-foreground"
                    : "border-app-border bg-app-surface-muted text-app-muted-foreground hover:text-app-foreground"
                }`}
                aria-pressed={active}
                onClick={() => toggleFamily(option.family)}
              >
                {t(option.labelKey)}
              </button>
            );
          })}
        </div>
      </form>

      {factsMutation.isPending && (
        <p data-testid={selectors.facts.loading} className="mt-4 text-sm text-app-muted-foreground">
          {t(strings.facts.loadingDetail)}
        </p>
      )}
      {factsMutation.error && (
        <div
          data-testid={selectors.facts.error}
          className="mt-4 rounded-control border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {errorMessage(factsMutation.error, t)}
        </div>
      )}
      {!factsMutation.isPending && !factsMutation.error && !report && (
        <div
          data-testid={selectors.facts.empty}
          className="mt-4 rounded-control border border-dashed border-app-border bg-app-surface-muted p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.facts.empty)}
        </div>
      )}

      {report && (
        <div className="mt-4 grid gap-4">
          <SummaryGrid report={report} evidenceCount={allEvidence.length} warningCount={degradedWarnings.length} />
          <TargetContextPanel report={report} />
          <CachePanel report={report} />
          <InventoryPanel report={report} />
          <FactsTable facts={report.facts} />
          <EvidenceTable evidence={allEvidence} />
          <WarningsPanel warnings={report.warnings} />
          <RawJsonPanel report={report} />
        </div>
      )}
    </section>
  );
}

function buildTarget(mode: TargetMode, value: string): CodeTarget {
  if (mode === "path") return pathTarget(value);
  if (mode === "module") return moduleTarget(value);
  if (mode === "project") return projectTarget(value);
  return scenarioTarget(value);
}

function collectEvidence(report?: CodeFactsReport): Evidence[] {
  if (!report) return [];
  return [
    ...report.evidence,
    ...report.surfaces.flatMap((surface) => surface.evidence),
    ...report.parseUnits.flatMap((unit) => unit.evidence),
    ...report.facts.flatMap((fact) => fact.evidence),
  ];
}

function SummaryGrid({
  report,
  evidenceCount,
  warningCount,
}: {
  report: CodeFactsReport;
  evidenceCount: number;
  warningCount: number;
}) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.facts.summary} className="grid gap-3 md:grid-cols-5">
      <FactStat label={t(strings.facts.surfaces)} value={report.surfaces.length} />
      <FactStat label={t(strings.facts.parseUnits)} value={report.parseUnits.length} />
      <FactStat label={t(strings.facts.facts)} value={report.facts.length} />
      <FactStat label={t(strings.facts.evidence)} value={evidenceCount} />
      <FactStat label={t(strings.facts.warnings)} value={warningCount} />
    </div>
  );
}

function TargetContextPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.targetContext}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.targetContext)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <Layers3 aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.targetContext)}
      </div>
      <dl className="grid gap-2 text-xs md:grid-cols-4">
        <InfoField label={t(strings.facts.resolvedKind)} value={enumLabel(TargetKind, report.target?.resolvedKind)} />
        <InfoField label={t(strings.facts.scenario)} value={report.target?.scenario || "-"} />
        <InfoField label={t(strings.facts.scenarioAware)} value={report.target?.scenarioAware ? "true" : "false"} />
        <InfoField label={t(strings.facts.rootPath)} value={report.target?.rootPath || "-"} wide />
      </dl>
    </section>
  );
}

function CachePanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.cachePanel}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.cachePanel)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <RefreshCcw aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.cachePanel)}
      </div>
      <dl className="grid gap-2 text-xs md:grid-cols-4">
        <InfoField label={t(strings.facts.cacheState)} value={report.cache?.state || "miss"} />
        <InfoField label={t(strings.facts.cacheKey)} value={report.cache?.cacheKey || "-"} wide />
        <InfoField label={t(strings.facts.cacheReason)} value={report.cache?.reason || "-"} />
        <InfoField label={t(strings.facts.cacheScope)} value={report.cache?.scope || "-"} />
        <InfoField label={t(strings.facts.sourceHash)} value={report.cache?.sourceHash || "-"} />
        <InfoField label={t(strings.facts.configHash)} value={report.cache?.configHash || "-"} />
        <InfoField label={t(strings.facts.providerVersion)} value={report.cache?.providerVersion || "-"} />
        <InfoField label={t(strings.facts.hitCount)} value={String(report.cache?.hitCount ?? 0)} />
      </dl>
    </section>
  );
}

function InventoryPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section className="grid gap-4 lg:grid-cols-2">
      <DataTable
        testId={selectors.facts.surfacesTable}
        title={t(strings.facts.surfaceInventory)}
        empty={t(strings.facts.noSurfaces)}
        headers={[
          t(strings.facts.table.id),
          t(strings.facts.table.kind),
          t(strings.facts.table.status),
          t(strings.facts.table.path),
        ]}
        rows={report.surfaces.map((surface) => [
          surface.id,
          enumLabel(SurfaceKind, surface.kind),
          enumLabel(SurfaceStatus, surface.status),
          surface.path,
        ])}
      />
      <DataTable
        testId={selectors.facts.parseUnitsTable}
        title={t(strings.facts.parseUnitInventory)}
        empty={t(strings.facts.noParseUnits)}
        headers={[
          t(strings.facts.table.id),
          t(strings.facts.table.language),
          t(strings.facts.table.status),
          t(strings.facts.table.path),
        ]}
        rows={report.parseUnits.map((unit) => parseUnitRow(unit))}
      />
    </section>
  );
}

function FactsTable({ facts }: { facts: GenericFact[] }) {
  const { t } = useTranslation();
  return (
    <DataTable
      testId={selectors.facts.factsTable}
      title={t(strings.facts.factsTable)}
      empty={t(strings.facts.noFacts)}
      headers={[
        t(strings.facts.table.family),
        t(strings.facts.table.kind),
        t(strings.facts.table.subject),
        t(strings.facts.table.status),
        t(strings.facts.table.attributes),
      ]}
      rows={facts.map((fact) => [
        enumLabel(FactFamily, fact.family),
        fact.kind,
        fact.subject,
        summarizeEvidenceStatus(fact.evidence),
        attributesSummary(fact.attributes),
      ])}
    />
  );
}

function EvidenceTable({ evidence }: { evidence: Evidence[] }) {
  const { t } = useTranslation();
  return (
    <DataTable
      testId={selectors.facts.evidenceTable}
      title={t(strings.facts.evidenceTable)}
      empty={t(strings.facts.noEvidence)}
      headers={[
        t(strings.facts.table.status),
        t(strings.facts.table.file),
        t(strings.facts.table.range),
        t(strings.facts.table.analyzer),
        t(strings.facts.table.message),
      ]}
      rows={evidence.map((entry) => [
        enumLabel(EvidenceStatus, entry.status),
        entry.range?.file || "-",
        formatRange(entry),
        entry.analyzer || "-",
        entry.message || entry.symbol || "-",
      ])}
    />
  );
}

function WarningsPanel({ warnings }: { warnings: Warning[] }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.warningsPanel}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.warningsPanel)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <AlertTriangle aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.warningsPanel)}
      </div>
      {warnings.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.facts.noWarnings)}</p>
      ) : (
        <ul className="grid gap-2 text-sm">
          {warnings.map((warning, index) => (
            <li key={`${warning.code}-${index}`} className="rounded-control border border-app-border bg-app-surface p-2">
              <span className="font-mono text-xs text-app-muted-foreground">{warning.code}</span>
              <span className="ml-2 font-medium text-app-foreground">{enumLabel(EvidenceStatus, warning.status)}</span>
              <p className="mt-1 text-app-muted-foreground">{warning.message}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function RawJsonPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  const json = useMemo(() => JSON.stringify(report, jsonReplacer, 2), [report]);
  return (
    <details
      data-testid={selectors.facts.rawJson}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <summary className="flex cursor-pointer items-center gap-2 text-sm font-semibold text-app-foreground">
        <Braces aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.rawJson)}
      </summary>
      <pre className="mt-3 max-h-96 overflow-auto rounded-control border border-app-border bg-app-background p-3 text-xs text-app-foreground">
        {json}
      </pre>
    </details>
  );
}

function DataTable({
  testId,
  title,
  empty,
  headers,
  rows,
}: {
  testId: string;
  title: string;
  empty: string;
  headers: string[];
  rows: string[][];
}) {
  return (
    <section data-testid={testId} className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <h3 className="text-sm font-semibold text-app-foreground">{title}</h3>
      {rows.length === 0 ? (
        <p className="mt-2 text-sm text-app-muted-foreground">{empty}</p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full table-fixed text-left text-xs">
            <thead className="text-app-muted-foreground">
              <tr>
                {headers.map((header) => (
                  <th key={header} scope="col" className="border-b border-app-border px-2 py-2 font-medium">
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rowIndex) => (
                <tr key={rowIndex} className="border-b border-app-border/60 last:border-0">
                  {row.map((cell, cellIndex) => (
                    <td key={`${rowIndex}-${cellIndex}`} className="max-w-64 truncate px-2 py-2" title={cell}>
                      {cell || "-"}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function InfoField({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <div className={`min-w-0 ${wide ? "md:col-span-2" : ""}`}>
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="truncate font-mono text-app-foreground" title={value}>
        {value || "-"}
      </dd>
    </div>
  );
}

function FactStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <p className="truncate text-xs uppercase text-app-muted-foreground" title={label}>
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function parseUnitRow(unit: ParseUnit): string[] {
  return [
    unit.id,
    unit.language,
    enumLabel(EvidenceStatus, unit.status),
    unit.configPath || unit.rootPath,
  ];
}

function enumLabel(enumObject: Record<string, string | number>, value: number | undefined): string {
  if (value === undefined) return "-";
  const raw = enumObject[value];
  if (typeof raw !== "string") return String(value);
  return raw.toLowerCase().replace(/_/g, " ");
}

function summarizeEvidenceStatus(evidence: Evidence[]): string {
  if (evidence.length === 0) return "-";
  const precedence = [
    EvidenceStatus.CONTRADICTED,
    EvidenceStatus.MISSING,
    EvidenceStatus.UNKNOWN,
    EvidenceStatus.UNSUPPORTED,
    EvidenceStatus.PROVEN,
  ];
  const status = precedence.find((candidate) => evidence.some((entry) => entry.status === candidate));
  return enumLabel(EvidenceStatus, status);
}

function formatRange(entry: Evidence): string {
  const range = entry.range;
  if (!range || range.startLine === 0) return "-";
  if (range.endLine > 0 && range.endLine !== range.startLine) {
    return `${range.startLine}:${range.startColumn}-${range.endLine}:${range.endColumn}`;
  }
  return `${range.startLine}:${range.startColumn}`;
}

function attributesSummary(attributes: Record<string, string>): string {
  const entries = Object.entries(attributes);
  if (entries.length === 0) return "-";
  return entries.map(([key, value]) => `${key}=${value}`).join(", ");
}

function jsonReplacer(_key: string, value: unknown) {
  return typeof value === "bigint" ? value.toString() : value;
}
