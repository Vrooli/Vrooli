import { AlertCircle, ArrowRight, BarChart3, CheckCircle2, Compass, Network, Shield, MousePointerClick, Map, LifeBuoy } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";

interface OrientationPageProps {
  onAnalyzeAll: () => void;
  onGoGraph: () => void;
  onGoDeployment: () => void;
  onGoCatalog: () => void;
  hasGraphData: boolean;
  hasScenarioSummaries: boolean;
  apiHealthy: boolean | null;
}

export function OrientationPage({
  onAnalyzeAll,
  onGoGraph,
  onGoDeployment,
  onGoCatalog,
  hasGraphData,
  hasScenarioSummaries,
  apiHealthy
}: OrientationPageProps) {
  const statusBadge = (() => {
    if (apiHealthy === null) return <Badge variant="secondary">Health: checking…</Badge>;
    if (apiHealthy) return <Badge className="bg-green-500/20 text-green-100 border-green-500/50">Health: OK</Badge>;
    return <Badge className="bg-red-500/20 text-red-100 border-red-500/50">Health: API unreachable</Badge>;
  })();

  return (
    <div className="space-y-6">
      <Card className="border border-primary/40 bg-primary/5">
        <CardHeader className="flex flex-col gap-2 pb-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="text-lg flex items-center gap-2">
              <Compass className="h-5 w-5 text-primary" />
              Orientation
            </CardTitle>
            <p className="text-sm text-muted-foreground">
              Follow these three steps to get real data, spot blockers, and fix gaps—without guesswork.
            </p>
          </div>
          {statusBadge}
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <StepCard
            title="1) Analyze everything"
            icon={<BarChart3 className="h-4 w-4 text-primary" />}
            body="Run a read-only scan to populate graphs, deployment readiness, and catalog detail."
            ctaLabel="Analyze all"
            onClick={onAnalyzeAll}
          />
          <StepCard
            title="2) Check deployment blockers"
            icon={<Shield className="h-4 w-4 text-primary" />}
            body="See tier fitness, blockers, and metadata gaps. Run Scan/Scan & Apply per scenario."
            ctaLabel="Open Deployment"
            onClick={onGoDeployment}
            disabled={!hasScenarioSummaries}
          />
          <StepCard
            title="3) Inspect a scenario"
            icon={<Network className="h-4 w-4 text-primary" />}
            body="Drill into one scenario’s drift, dependencies, and optimization recommendations."
            ctaLabel="Open Catalog"
            onClick={onGoCatalog}
            disabled={!hasScenarioSummaries}
          />
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="border border-border/40 bg-background/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-base flex items-center gap-2">
              <MousePointerClick className="h-4 w-4 text-primary" />
              Where should I click?
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <ClickRow
              icon={<Map className="h-4 w-4 text-primary" aria-hidden="true" />}
              title="Graph"
              body="Visual map of scenarios/resources. Best after Analyze All. Use search to spotlight a node."
              actionLabel="Open graph"
              onClick={onGoGraph}
              disabled={!hasGraphData}
            />
            <ClickRow
              icon={<Shield className="h-4 w-4 text-primary" aria-hidden="true" />}
              title="Deployment"
              body="Tier fitness, blockers, metadata gaps. Run Scan or Scan & Apply from the table."
              actionLabel="Open deployment"
              onClick={onGoDeployment}
              disabled={!hasScenarioSummaries}
            />
            <ClickRow
              icon={<Network className="h-4 w-4 text-primary" aria-hidden="true" />}
              title="Catalog"
              body="Pick a scenario to see drift, dependencies, and optimization hints."
              actionLabel="Open catalog"
              onClick={onGoCatalog}
              disabled={!hasScenarioSummaries}
            />
          </CardContent>
        </Card>

        <Card className="border border-border/40 bg-background/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Key terms</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-2 text-sm text-muted-foreground">
            <GlossaryItem term="Drift" description="Missing = detected in code but not declared. Declared-only = listed but not observed." />
            <GlossaryItem term="Fitness score" description="0–1 score per tier. Blockers make deployment impossible until fixed." />
            <GlossaryItem term="Scan vs Scan & Apply" description="Scan is read-only; Scan & Apply writes dependencies back to service.json." />
            <GlossaryItem term="Optimize" description="Advisory suggestions (e.g., swaps). Apply makes changes when supported." />
          </CardContent>
        </Card>
      </div>

      <Card className="border border-border/40 bg-background/50">
        <CardHeader className="pb-2">
          <CardTitle className="text-base flex items-center gap-2">
            <LifeBuoy className="h-4 w-4 text-primary" />
            Getting unstuck
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <TipCard
            icon={<AlertCircle className="h-4 w-4 text-primary" aria-hidden="true" />}
            title="API not responding"
            body="Ensure the scenario is running, then rerun Analyze All. Health shows above."
          />
          <TipCard
            icon={<Map className="h-4 w-4 text-primary" aria-hidden="true" />}
            title="Empty graph"
            body="Run Analyze All, then hit Refresh in Graph. Use search to highlight a node."
          />
          <TipCard
            icon={<Shield className="h-4 w-4 text-primary" aria-hidden="true" />}
            title="Don't know what to fix"
            body="Open Deployment, filter for issues/critical, then click a row to see details and Scan."
          />
        </CardContent>
      </Card>
    </div>
  );
}

function StepCard({
  title,
  body,
  icon,
  ctaLabel,
  onClick,
  disabled
}: {
  title: string;
  body: string;
  icon: React.ReactNode;
  ctaLabel: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <div className="rounded-xl border border-border/50 bg-background/60 p-4 shadow-sm">
      <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
        {icon}
        <span>{title}</span>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">{body}</p>
      <Button
        className="mt-3 w-full justify-center"
        disabled={disabled}
        onClick={onClick}
        variant={disabled ? "secondary" : "default"}
      >
        {ctaLabel}
      </Button>
    </div>
  );
}

function GlossaryItem({ term, description }: { term: string; description: string }) {
  return (
    <div className="rounded-lg border border-border/40 bg-background/40 p-3">
      <p className="text-xs font-semibold text-foreground">{term}</p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}

function ClickRow({
  icon,
  title,
  body,
  actionLabel,
  onClick,
  disabled
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  actionLabel: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-lg border border-border/50 bg-background/60 p-3">
      <div className="flex items-start gap-2">
        {icon}
        <div>
          <p className="text-sm font-semibold text-foreground">{title}</p>
          <p className="text-xs text-muted-foreground">{body}</p>
        </div>
      </div>
      <Button variant="ghost" size="sm" disabled={disabled} onClick={onClick} className="gap-2">
        {actionLabel}
        <ArrowRight className="h-3 w-3" aria-hidden="true" />
      </Button>
    </div>
  );
}

function TipCard({ icon, title, body }: { icon: React.ReactNode; title: string; body: string }) {
  return (
    <div className="rounded-lg border border-border/50 bg-background/60 p-3">
      <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
        {icon}
        <span>{title}</span>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{body}</p>
    </div>
  );
}
