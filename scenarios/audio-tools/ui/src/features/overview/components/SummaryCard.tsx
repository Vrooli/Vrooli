import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";
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
    <Card>
      <CardContent>
        <CardHeader
          className="items-center justify-between gap-2"
          style={{ flexDirection: "row", borderBlockEnd: "none", padding: 0 }}
        >
          <CardTitle>{title}</CardTitle>
          <span className="text-app-muted-foreground">{icon}</span>
        </CardHeader>
        <p className="mt-2 text-2xl font-semibold text-app-foreground">{value}</p>
        {hint ? <CardDescription className="mt-1">{hint}</CardDescription> : null}
        <div className="mt-2">
          <StatusDot tone={tone} label={statusLabel} />
        </div>
      </CardContent>
    </Card>
  );
}
