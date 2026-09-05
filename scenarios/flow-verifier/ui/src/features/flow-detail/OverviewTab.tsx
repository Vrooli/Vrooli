// OverviewTab is the default tab on /flows/:id and the answer to
// "what is this flow for?". It renders the prose+structure that
// flow.json carries but earlier tabs never surfaced: description,
// domain, model (module/seed/maxSteps), invariants, traces, runtime.
import type { FlowDetail } from "../../api/inventory";
import { useTranslation } from "../../i18n";

interface Props {
  detail: FlowDetail;
}

export function OverviewTab({ detail }: Props) {
  const { t } = useTranslation();
  return (
    <section data-testid="flow-overview" className="flex flex-col gap-4 text-sm text-app-foreground">
      <div data-testid="flow-overview-description" className="leading-relaxed">
        {detail.description ? (
          <p>{detail.description}</p>
        ) : (
          <p className="italic text-app-muted-foreground">
            {t("flowOverview.noDescription", { defaultValue: "No description set in flow.json." })}
          </p>
        )}
      </div>

      <dl
        data-testid="flow-overview-metadata"
        className="grid gap-x-4 gap-y-2 sm:grid-cols-2"
      >
        <Cell label={t("flowOverview.domain", { defaultValue: "Domain" })}>
          {detail.domain || "—"}
        </Cell>
        <Cell label={t("flowOverview.schemaVersion", { defaultValue: "Schema version" })}>
          {detail.schemaVersion}
        </Cell>
        <Cell label={t("flowOverview.language", { defaultValue: "Language" })}>
          {detail.language}
        </Cell>
        <Cell label={t("flowOverview.flowId", { defaultValue: "Flow id" })}>
          <span className="font-mono">{detail.flowId}</span>
        </Cell>
        <Cell label={t("flowOverview.contractPath", { defaultValue: "Contract path" })}>
          <span className="font-mono text-xs">{detail.contractPath}</span>
        </Cell>
      </dl>

      <ModelCard detail={detail} />

      {detail.invariants.length > 0 && (
        <section data-testid="flow-overview-invariants" className="flex flex-col gap-2">
          <h3 className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t("flowOverview.invariants", { defaultValue: "Invariants" })}
          </h3>
          <ul className="flex flex-col gap-2">
            {detail.invariants.map((inv) => (
              <li
                key={inv.id}
                data-testid={`flow-overview-invariant-${inv.id}`}
                className="rounded-panel border border-app-border bg-app-surface-muted p-2"
              >
                <p className="font-mono text-xs text-app-muted-foreground">{inv.quint}</p>
                <p>{inv.description}</p>
              </li>
            ))}
          </ul>
        </section>
      )}

      {detail.traces.length > 0 && (
        <section data-testid="flow-overview-traces" className="flex flex-col gap-2">
          <h3 className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t("flowOverview.traces", { defaultValue: "Named traces" })}
          </h3>
          <ul className="flex flex-wrap gap-2">
            {detail.traces.map((trace) => (
              <li
                key={trace.name}
                data-testid={`flow-overview-trace-${trace.name}`}
                className="rounded-pill border border-app-border bg-app-surface-muted px-2 py-0.5 font-mono text-xs"
              >
                {trace.name}
              </li>
            ))}
          </ul>
        </section>
      )}

      <RuntimeCard detail={detail} />
    </section>
  );
}

function ModelCard({ detail }: { detail: FlowDetail }) {
  const { t } = useTranslation();
  const model = detail.model;
  if (!model.module) return null;
  return (
    <dl
      data-testid="flow-overview-model"
      className="grid gap-x-4 gap-y-2 rounded-panel border border-app-border bg-app-surface-muted p-3 sm:grid-cols-2"
    >
      <Cell label={t("flowOverview.modelModule", { defaultValue: "Quint module" })}>
        <span className="font-mono">{model.module}</span>
      </Cell>
      <Cell label={t("flowOverview.modelSeed", { defaultValue: "Seed" })}>
        <span className="font-mono">{model.seed || "—"}</span>
      </Cell>
      <Cell label={t("flowOverview.modelMaxSteps", { defaultValue: "Max steps" })}>
        {model.maxSteps}
      </Cell>
      <Cell label={t("flowOverview.modelTraceCount", { defaultValue: "Trace count" })}>
        {model.traceCount}
      </Cell>
    </dl>
  );
}

function RuntimeCard({ detail }: { detail: FlowDetail }) {
  const { t } = useTranslation();
  const r = detail.runtime;
  if (!r.go && !r.typescript) return null;
  return (
    <dl
      data-testid="flow-overview-runtime"
      className="grid gap-x-4 gap-y-2 rounded-panel border border-app-border bg-app-surface-muted p-3 sm:grid-cols-2"
    >
      {r.go && (
        <>
          <Cell label={t("flowOverview.runtimeGoPackage", { defaultValue: "Go package" })}>
            <span className="font-mono">{r.go.package}</span>
          </Cell>
          <Cell label={t("flowOverview.runtimeGoStatus", { defaultValue: "Go status type" })}>
            <span className="font-mono">{r.go.statusType}</span>
          </Cell>
        </>
      )}
      {r.typescript && (
        <>
          <Cell label={t("flowOverview.runtimeTsStatus", { defaultValue: "TS status type" })}>
            <span className="font-mono">{r.typescript.statusType}</span>
          </Cell>
          <Cell label={t("flowOverview.runtimeTsEvent", { defaultValue: "TS event type" })}>
            <span className="font-mono">{r.typescript.eventType}</span>
          </Cell>
        </>
      )}
    </dl>
  );
}

function Cell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="text-app-foreground">{children}</dd>
    </div>
  );
}
