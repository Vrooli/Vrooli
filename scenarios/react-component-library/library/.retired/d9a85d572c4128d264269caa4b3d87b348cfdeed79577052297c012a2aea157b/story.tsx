import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  type CardProps,
} from "./Card";

type CardStoryArgs = Pick<CardProps, "className">;

function cardProps(args: Record<string, unknown>): CardStoryArgs {
  return {
    className: typeof args.className === "string" ? args.className : undefined,
  };
}

export function MetricCard({ args }: StoryHarnessProps) {
  return (
    <Card
      {...cardProps(args)}
      role="region"
      aria-label="Adoption health"
      data-rcl-card-metric="true"
    >
      <CardHeader>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "var(--space-sm)",
          }}
        >
          <CardTitle>Adoption health</CardTitle>
          <span aria-hidden="true" style={{ color: "var(--color-primary)" }}>
            ✓
          </span>
        </div>
        <CardDescription>
          Current coverage across active scenarios
        </CardDescription>
      </CardHeader>
      <CardContent style={{ display: "grid", gap: "var(--space-xs)" }}>
        <strong
          style={{
            font: "var(--text-title)",
            fontVariantNumeric: "tabular-nums",
          }}
        >
          8 scenarios
        </strong>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "var(--space-xs)",
          }}
        >
          <span
            role="status"
            style={{
              color: "var(--color-success)",
              border: "var(--badge-border)",
              borderRadius: "var(--radius-pill)",
              paddingInline: "var(--space-sm)",
              paddingBlock: "var(--space-3xs)",
            }}
          >
            Healthy
          </span>
          <span style={{ color: "var(--color-muted-foreground)" }}>
            All required checks are current.
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

export function EmptyStateCard({ args }: StoryHarnessProps) {
  return (
    <Card {...cardProps(args)}>
      <CardHeader>
        <CardTitle>No drift detected</CardTitle>
        <CardDescription>The review queue is clear.</CardDescription>
      </CardHeader>
      <CardContent>
        <span style={{ color: "var(--color-muted-foreground)" }}>
          There are no pending recommendations for this workspace.
        </span>
      </CardContent>
    </Card>
  );
}

export function LongContentCard({ args }: StoryHarnessProps) {
  return (
    <Card {...cardProps(args)}>
      <CardHeader>
        <CardTitle>Workspace configuration review</CardTitle>
        <CardDescription>
          A deliberately long description demonstrates wrapping without breaking
          the card header or pushing the primary content beyond its readable
          measure.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <span>
          Review and apply all pending workspace configuration changes before
          publishing the next adoption report.
        </span>
      </CardContent>
    </Card>
  );
}

export function CardGrid({ args }: StoryHarnessProps) {
  return (
    <div
      style={{
        display: "grid",
        gap: "var(--space-md)",
        padding: "var(--space-md)",
        background: "var(--color-surface-sunken)",
      }}
    >
      <MetricCard args={args} log={() => undefined} />
      <MetricCard args={args} log={() => undefined} />
    </div>
  );
}
