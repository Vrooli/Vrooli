import { Card, CardContent } from "../../components/ui/card";
import { statusTone } from "../../theme/status";

interface DeploymentSummaryCardsProps {
  criticalCount: number;
  issuesCount: number;
  notScannedCount: number;
  readyCount: number;
}

export function DeploymentSummaryCards({
  criticalCount,
  issuesCount,
  notScannedCount,
  readyCount
}: DeploymentSummaryCardsProps) {
  return (
    <div className="grid gap-4 md:grid-cols-4">
      <Card className={`border ${statusTone("success").metric}`}>
        <CardContent className="pt-6">
          <div className={`text-2xl font-bold ${statusTone("success").text}`}>{readyCount}</div>
          <p className="text-xs text-muted-foreground">Ready for deployment</p>
        </CardContent>
      </Card>
      <Card className={`border ${statusTone("warning").metric}`}>
        <CardContent className="pt-6">
          <div className={`text-2xl font-bold ${statusTone("warning").text}`}>{issuesCount}</div>
          <p className="text-xs text-muted-foreground">With issues</p>
        </CardContent>
      </Card>
      <Card className={`border ${statusTone("danger").metric}`}>
        <CardContent className="pt-6">
          <div className={`text-2xl font-bold ${statusTone("danger").text}`}>{criticalCount}</div>
          <p className="text-xs text-muted-foreground">Critical gaps</p>
        </CardContent>
      </Card>
      <Card className="border border-border/40 bg-background/40">
        <CardContent className="pt-6">
          <div className="text-2xl font-bold text-foreground">{notScannedCount}</div>
          <p className="text-xs text-muted-foreground">Not yet scanned</p>
        </CardContent>
      </Card>
    </div>
  );
}
