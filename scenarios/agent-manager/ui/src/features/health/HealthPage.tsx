import { useState } from "react";
import { Link } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import type { HealthAuditFilters } from "./api/types";
import { ModelHealthTable } from "./components/ModelHealthTable";
import { RunnerHealthTable } from "./components/RunnerHealthTable";
import { HealthAuditDrawer } from "./components/HealthAuditDrawer";
import { useModelHealth, useRunnerHealth } from "./hooks/useHealth";
import { useModelPolicyDrift } from "./hooks/useHealth";
import { ModelPolicyDriftStatus } from "./components/ModelPolicyDriftStatus";
import { RunnerProbePanel } from "./components/RunnerProbePanel";
import { FallbackInsights } from "./components/FallbackInsights";

export function HealthPage() {
  const models = useModelHealth();
  const runners = useRunnerHealth();
  const drift = useModelPolicyDrift();
  const [auditFilters, setAuditFilters] = useState<HealthAuditFilters | null>(null);

  const onModelAudit = (runner: string, model: string) => {
    setAuditFilters({ scope: "model", runner, model, limit: 100 });
  };
  const onRunnerAudit = (runner: string) => {
    setAuditFilters({ scope: "runner", runner, limit: 100 });
  };

  return (
    <div className="h-full space-y-4 overflow-auto px-4 py-6 sm:px-6 lg:px-10" data-testid="health-page">
      <PageHeader
        title="Health"
        description="Persisted snapshot of every runner and model the system has observed. Failed entries surface first."
        actions={<Link to="/investigations" className="inline-flex min-h-11 items-center text-sm font-medium text-primary hover:underline sm:min-h-8">Investigations →</Link>}
      />

      <RunnerProbePanel />

      <FallbackInsights />

      <ModelPolicyDriftStatus snapshot={drift.data} />

      <Tabs
        mode="uncontrolled"
        defaultActive="models"
        ariaLabel="Health sections"
        items={[{ id: "models", label: "Models" }, { id: "runners", label: "Runners" }]}
        itemTestId={(id) => `health-tab-${id}`}
        panels={{
          models: <div className="pt-4">{models.isLoading ? <p className="text-sm text-muted-foreground">Loading…</p> : models.error ? <p className="text-sm text-destructive">Failed to load model health: {models.error.message}</p> : <ModelHealthTable rows={models.data?.models ?? []} onShowAudit={onModelAudit} />}</div>,
          runners: <div className="pt-4">{runners.isLoading ? <p className="text-sm text-muted-foreground">Loading…</p> : runners.error ? <p className="text-sm text-destructive">Failed to load runner health: {runners.error.message}</p> : <RunnerHealthTable rows={runners.data?.runners ?? []} onShowAudit={onRunnerAudit} />}</div>,
        }}
      />

      <HealthAuditDrawer
        open={auditFilters !== null}
        filters={auditFilters}
        onOpenChange={() => setAuditFilters(null)}
      />
    </div>
  );
}
