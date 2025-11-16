import { AlertTriangle, BadgeCheck, FoldVertical, Sparkles } from "lucide-react";
import type { OptimizationRecommendation } from "../types";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";

interface OptimizationPanelProps {
  recommendations: OptimizationRecommendation[];
}

const priorityVariant: Record<string, string> = {
  critical: "bg-red-500/20 text-red-200",
  high: "bg-orange-500/20 text-orange-200",
  medium: "bg-amber-500/20 text-amber-200",
  low: "bg-emerald-500/20 text-emerald-200"
};

const typeLabels: Record<string, string> = {
  dependency_reduction: "Dependency reduction",
  resource_swap: "Resource swap",
  shared_workflow_adoption: "Workflow",
  cost_optimization: "Cost",
  performance_improvement: "Performance"
};

function RecommendationCard({ recommendation }: { recommendation: OptimizationRecommendation }) {
  const priority = (recommendation.priority ?? "medium").toLowerCase();
  const priorityClass = priorityVariant[priority] ?? priorityVariant.medium;
  const typeLabel = typeLabels[recommendation.recommendation_type] ?? recommendation.recommendation_type;

  return (
    <div className="rounded-xl border border-border/30 bg-background/30 p-4">
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="text-xs uppercase text-muted-foreground">{typeLabel}</p>
          <h4 className="text-base font-medium text-foreground">{recommendation.title}</h4>
        </div>
        <Badge className={priorityClass} variant="outline">
          {priority}
        </Badge>
      </div>
      {recommendation.description ? (
        <p className="mt-2 text-sm text-muted-foreground">{recommendation.description}</p>
      ) : null}
      <div className="mt-3 grid gap-2 text-xs text-muted-foreground md:grid-cols-2">
        {typeof recommendation.confidence_score === "number" ? (
          <span>Confidence: {(recommendation.confidence_score * 100).toFixed(0)}%</span>
        ) : null}
        {recommendation.recommended_state?.action ? (
          <span>Action: {recommendation.recommended_state.action as string}</span>
        ) : null}
        {recommendation.recommended_state?.resource_name ? (
          <span>Resource: {recommendation.recommended_state.resource_name as string}</span>
        ) : null}
        {recommendation.recommended_state?.scenario_name ? (
          <span>Scenario: {recommendation.recommended_state.scenario_name as string}</span>
        ) : null}
      </div>
    </div>
  );
}

export function OptimizationPanel({ recommendations }: OptimizationPanelProps) {
  const hasRecommendations = recommendations && recommendations.length > 0;

  return (
    <Card className="border border-border/40 bg-background/40">
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm flex items-center gap-2">
          <Sparkles className="h-4 w-4" />
          Optimization intelligence
        </CardTitle>
        {hasRecommendations ? <BadgeCheck className="h-4 w-4 text-primary" /> : <FoldVertical className="h-4 w-4 text-muted-foreground" />}
      </CardHeader>
      <CardContent className="space-y-3">
        {!hasRecommendations ? (
          <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border/40 p-4 text-center text-sm text-muted-foreground">
            <AlertTriangle className="h-4 w-4" />
            <p>No optimization opportunities detected yet. Run an optimization sweep to refresh.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {recommendations.map((rec) => (
              <RecommendationCard key={rec.id} recommendation={rec} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
