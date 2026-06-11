import { useState } from "react";
import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, FileCode2, GitBranch, RefreshCcw } from "lucide-react";
import { Severity } from "@vrooli/proto-types/proto-health/v1/validation/validation_pb";
import { TransportWorld } from "@vrooli/proto-types/proto-health/v1/shared/surface_pb";

import { describeScenarioProtos, validateScenario } from "../../api/protoHealth";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const QUICK_TARGETS = ["proto-health", "cli-health", "measures-health", "agent-manager"] as const;

export function ProtoHealthPanel() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("proto-health");
  const [target, setTarget] = useState("proto-health");

  const validation = useQuery({
    queryKey: ["proto-health", "validate", target],
    queryFn: () => validateScenario(target),
  });
  const surface = useQuery({
    queryKey: ["proto-health", "surface", target],
    queryFn: () => describeScenarioProtos(target),
  });

  const run = () => {
    const next = scenario.trim();
    if (next) {
      setTarget(next);
    }
  };

  const loading = validation.isLoading || surface.isLoading;
  const error = validation.error ?? surface.error;
  const report = validation.data;
  const protoSurface = surface.data?.surface;

  return (
    <section
      data-testid={selectors.protoHealth.panel}
      aria-labelledby="proto-health-heading"
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex flex-col gap-1">
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            Contract validator
          </p>
          <h3 id="proto-health-heading" className="text-xl font-semibold text-app-foreground">
            Proto Health
          </h3>
          <p className="max-w-3xl text-sm text-app-muted-foreground">
            Validate one scenario's proto contract and inspect the surface facts consumed by
            downstream quality-loop tools.
          </p>
        </div>

        <form
          className="flex flex-col gap-2 sm:flex-row"
          onSubmit={(event) => {
            event.preventDefault();
            run();
          }}
        >
          <Input
            data-testid={selectors.protoHealth.scenarioInput}
            aria-label="Scenario"
            value={scenario}
            onChange={(event) => setScenario(event.currentTarget.value)}
            className="min-w-0 flex-1"
          />
          <Button data-testid={selectors.protoHealth.runButton} type="submit">
            <RefreshCcw aria-hidden="true" className="me-2 h-4 w-4" />
            Run
          </Button>
        </form>

        <div className="flex flex-wrap gap-2">
          {QUICK_TARGETS.map((name) => (
            <button
              key={name}
              type="button"
              data-testid={selectors.protoHealth.quickTarget({ scenario: name })}
              onClick={() => {
                setScenario(name);
                setTarget(name);
              }}
              className={
                target === name
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {name}
            </button>
          ))}
        </div>
      </div>

      {loading && (
        <p data-testid={selectors.protoHealth.loading} className="text-sm text-app-muted-foreground">
          Loading proto surface...
        </p>
      )}

      {error && (
        <p data-testid={selectors.protoHealth.error} className="text-sm text-red-500">
          {errorMessage(error, t)}
        </p>
      )}

      {!loading && !error && report && protoSurface && (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <div className="rounded-panel border border-app-border bg-app-surface p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs font-semibold uppercase text-app-muted-foreground">
                  Validation
                </p>
                <h4 className="text-lg font-semibold">{report.scenario}</h4>
              </div>
              <span
                data-testid={selectors.protoHealth.status}
                className={
                  report.passed
                    ? "rounded-control bg-emerald-100 px-3 py-1 text-sm font-semibold text-emerald-800"
                    : "rounded-control bg-red-100 px-3 py-1 text-sm font-semibold text-red-800"
                }
              >
                {report.passed ? "Passed" : "Blocked"}
              </span>
            </div>
            <dl className="mt-4 grid grid-cols-3 gap-2 text-sm">
              <Metric label="Errors" value={report.summary?.errors ?? 0} />
              <Metric label="Warnings" value={report.summary?.warnings ?? 0} />
              <Metric label="Info" value={report.summary?.infos ?? 0} />
            </dl>
            <div className="mt-4 flex flex-col gap-2">
              {report.findings.length === 0 ? (
                <p data-testid={selectors.protoHealth.empty} className="text-sm text-app-muted-foreground">
                  No findings.
                </p>
              ) : (
                report.findings.slice(0, 8).map((finding) => (
                  <article
                    key={`${finding.code}:${finding.location}:${finding.message}`}
                    data-testid={selectors.protoHealth.finding}
                    className="rounded-panel border border-app-border bg-app-background p-3"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-xs font-semibold uppercase text-app-muted-foreground">
                        {severityLabel(finding.severity)}
                      </span>
                      <span className="font-mono text-xs text-app-foreground">{finding.code}</span>
                    </div>
                    <p className="mt-1 text-sm">{finding.message}</p>
                    {finding.location && (
                      <p className="mt-1 break-all font-mono text-xs text-app-muted-foreground">
                        {finding.location}
                      </p>
                    )}
                  </article>
                ))
              )}
            </div>
          </div>

          <div className="rounded-panel border border-app-border bg-app-surface p-4">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              Surface Facts
            </p>
            <div className="mt-3 grid gap-3 sm:grid-cols-3">
              <Fact icon={<FileCode2 aria-hidden="true" />} label="Files" value={protoSurface.files.length} />
              <Fact icon={<Activity aria-hidden="true" />} label="Services" value={protoSurface.services.length} />
              <Fact
                icon={<GitBranch aria-hidden="true" />}
                label="Transport"
                value={transportWorldLabel(protoSurface.transportWorld)}
              />
            </div>
            <div className="mt-4 overflow-hidden rounded-panel border border-app-border">
              <table className="w-full table-fixed text-left text-sm">
                <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                  <tr>
                    <th className="px-3 py-2">Domain</th>
                    <th className="px-3 py-2">Service</th>
                    <th className="px-3 py-2">RPCs</th>
                  </tr>
                </thead>
                <tbody>
                  {protoSurface.services.length === 0 ? (
                    <tr>
                      <td className="px-3 py-3 text-app-muted-foreground" colSpan={3}>
                        No services found.
                      </td>
                    </tr>
                  ) : (
                    protoSurface.services.map((service) => (
                      <tr key={service.fullName} className="border-t border-app-border">
                        <td className="px-3 py-2">{service.domain || "shared"}</td>
                        <td className="px-3 py-2 font-mono text-xs">{service.name}</td>
                        <td className="px-3 py-2">{service.rpcs.length}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-background p-3">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="mt-1 text-2xl font-semibold">{value}</dd>
    </div>
  );
}

function Fact({ icon, label, value }: { icon: ReactNode; label: string; value: number | string }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-background p-3">
      <div className="flex h-5 w-5 text-app-muted-foreground">{icon}</div>
      <p className="mt-2 text-xs uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
    </div>
  );
}

function severityLabel(severity: Severity): string {
  switch (severity) {
    case Severity.ERROR:
      return "Error";
    case Severity.WARNING:
      return "Warning";
    case Severity.INFO:
      return "Info";
    default:
      return "Unspecified";
  }
}

function transportWorldLabel(world: TransportWorld): string {
  switch (world) {
    case TransportWorld.CONNECT:
      return "Connect";
    case TransportWorld.HAND_ROLLED:
      return "Hand-rolled";
    case TransportWorld.MIXED:
      return "Mixed";
    case TransportWorld.NONE:
      return "None";
    default:
      return "Unknown";
  }
}
