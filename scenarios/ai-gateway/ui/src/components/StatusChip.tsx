interface StatusChipProps {
  children: string;
  tone?: "neutral" | "success" | "warning" | "danger" | "info";
}

const toneClass: Record<NonNullable<StatusChipProps["tone"]>, string> = {
  neutral: "border-app-border bg-app-surface-muted text-app-muted-foreground",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  danger: "border-red-200 bg-red-50 text-red-700",
  info: "border-sky-200 bg-sky-50 text-sky-700",
};

export function StatusChip({ children, tone = "neutral" }: StatusChipProps) {
  return (
    <span
      className={[
        "inline-flex min-h-6 items-center rounded-control border px-2 py-0.5 text-xs font-medium",
        toneClass[tone],
      ].join(" ")}
    >
      {children}
    </span>
  );
}
