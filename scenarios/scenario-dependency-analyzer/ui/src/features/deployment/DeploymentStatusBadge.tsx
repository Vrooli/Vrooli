import { AlertTriangle, CheckCircle2, Clock, TrendingDown, TrendingUp } from "lucide-react";

import { Badge } from "../../components/ui/badge";
import { statusTone } from "../../theme/status";
import type { DeploymentStatusKind } from "./deploymentStatus";

export function DeploymentStatusBadge({ status }: { status: DeploymentStatusKind }) {
  switch (status) {
    case "ready":
      return (
        <Badge className={statusTone("success").badge}>
          <CheckCircle2 className="mr-1 h-3 w-3" />
          Ready
        </Badge>
      );
    case "critical":
      return (
        <Badge variant="outline" className={statusTone("danger").badge}>
          <AlertTriangle className="mr-1 h-3 w-3" />
          Critical
        </Badge>
      );
    case "issues":
      return (
        <Badge variant="outline" className={statusTone("warning").badge}>
          <AlertTriangle className="mr-1 h-3 w-3" />
          Issues
        </Badge>
      );
    default:
      return (
        <Badge variant="secondary">
          <Clock className="mr-1 h-3 w-3" />
          Not Scanned
        </Badge>
      );
  }
}

export function TierFitnessBadge({ tierFitness }: { tierFitness: { best: number; worst: number } | null }) {
  if (!tierFitness) {
    return <span className="text-xs text-muted-foreground">N/A</span>;
  }

  const avg = (tierFitness.best + tierFitness.worst) / 2;
  const Icon = avg >= 0.7 ? TrendingUp : TrendingDown;
  const color = avg >= 0.7 ? statusTone("success").text : avg >= 0.5 ? statusTone("warning").text : statusTone("danger").text;

  return (
    <span className="inline-flex items-center gap-2 text-xs">
      <Icon className={`h-3 w-3 ${color}`} />
      <span>
        {Math.round(tierFitness.best * 100)}% / {Math.round(tierFitness.worst * 100)}%
      </span>
    </span>
  );
}
