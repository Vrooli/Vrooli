import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs";
import type { HealthAuditFilters } from "./api/types";
import { ModelHealthTable } from "./components/ModelHealthTable";
import { RunnerHealthTable } from "./components/RunnerHealthTable";
import { HealthAuditDrawer } from "./components/HealthAuditDrawer";
import { useModelHealth, useRunnerHealth } from "./hooks/useHealth";

export function HealthPage() {
  const models = useModelHealth();
  const runners = useRunnerHealth();
  const [auditFilters, setAuditFilters] = useState<HealthAuditFilters | null>(null);

  const onModelAudit = (runner: string, model: string) => {
    setAuditFilters({ scope: "model", runner, model, limit: 100 });
  };
  const onRunnerAudit = (runner: string) => {
    setAuditFilters({ scope: "runner", runner, limit: 100 });
  };

  return (
    <div className="h-full overflow-auto px-4 py-6 sm:px-6 lg:px-10" data-testid="health-page">
      <header className="mb-4">
        <h1 className="text-2xl font-semibold">Health</h1>
        <p className="text-sm text-muted-foreground">
          Persisted snapshot of every runner and model the system has observed. Failed entries surface first.
        </p>
      </header>

      <Tabs defaultValue="models" className="w-full">
        <TabsList>
          <TabsTrigger value="models" data-testid="health-tab-models">Models</TabsTrigger>
          <TabsTrigger value="runners" data-testid="health-tab-runners">Runners</TabsTrigger>
        </TabsList>
        <TabsContent value="models" className="mt-4">
          {models.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : models.error ? (
            <p className="text-sm text-destructive">Failed to load model health: {models.error.message}</p>
          ) : (
            <ModelHealthTable rows={models.data?.models ?? []} onShowAudit={onModelAudit} />
          )}
        </TabsContent>
        <TabsContent value="runners" className="mt-4">
          {runners.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : runners.error ? (
            <p className="text-sm text-destructive">Failed to load runner health: {runners.error.message}</p>
          ) : (
            <RunnerHealthTable rows={runners.data?.runners ?? []} onShowAudit={onRunnerAudit} />
          )}
        </TabsContent>
      </Tabs>

      <HealthAuditDrawer
        open={auditFilters !== null}
        filters={auditFilters}
        onOpenChange={(open) => {
          if (!open) setAuditFilters(null);
        }}
      />
    </div>
  );
}
