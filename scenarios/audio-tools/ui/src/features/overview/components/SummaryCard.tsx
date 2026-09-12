import { Card, CardDescription, CardHeader, CardTitle } from "../../../components/ui/card";
import { StatusDot } from "../../../components/ui/status-dot";

export function SummaryCard({
  icon,
  title,
  value,
  hint,
  tone,
  statusLabel,
}: {
  icon: React.ReactNode;
  title: string;
  value: string;
  hint?: string;
  tone: "neutral" | "info" | "success" | "warning" | "danger";
  statusLabel: string;
}) {
  return (
    <Card padding="md">
      <CardHeader className="gap-2">
        <CardTitle>{title}</CardTitle>
        <span className="text-app-muted-foreground">{icon}</span>
      </CardHeader>
      <p className="mt-2 text-2xl font-semibold text-app-foreground">{value}</p>
      {hint ? <CardDescription className="mt-1">{hint}</CardDescription> : null}
      <div className="mt-2">
        <StatusDot tone={tone} label={statusLabel} />
      </div>
    </Card>
  );
}
