export type StatusTone = "success" | "warning" | "danger" | "neutral" | "info";

interface StatusToneClasses {
  badge: string;
  panel: string;
  metric: string;
  text: string;
}

const statusToneClasses: Record<StatusTone, StatusToneClasses> = {
  danger: {
    badge: "border-destructive/60 bg-destructive/20 text-destructive-foreground",
    panel: "border-destructive/50 bg-destructive/10 text-destructive-foreground",
    metric: "border-destructive/50 bg-destructive/5",
    text: "text-destructive-foreground"
  },
  info: {
    badge: "border-primary/50 bg-primary/15 text-primary",
    panel: "border-primary/40 bg-primary/5 text-foreground",
    metric: "border-primary/40 bg-primary/5",
    text: "text-primary"
  },
  neutral: {
    badge: "border-transparent bg-secondary/60 text-secondary-foreground",
    panel: "border-border/40 bg-background/40 text-foreground",
    metric: "border-border/40 bg-background/40",
    text: "text-muted-foreground"
  },
  success: {
    badge: "border-accent/50 bg-accent/15 text-accent-foreground",
    panel: "border-accent/40 bg-accent/5 text-foreground",
    metric: "border-accent/40 bg-accent/5",
    text: "text-accent-foreground"
  },
  warning: {
    badge: "border-warning/60 bg-warning/15 text-warning-foreground",
    panel: "border-warning/50 bg-warning/10 text-warning-foreground",
    metric: "border-warning/50 bg-warning/5",
    text: "text-warning-foreground"
  }
};

export function statusTone(tone: StatusTone): StatusToneClasses {
  return statusToneClasses[tone];
}
