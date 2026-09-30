import { useQuery } from "@tanstack/react-query";
import { Activity } from "lucide-react";

import { fetchHealth } from "../api/health";
import { useTranslation } from "../i18n";

export function HealthPill() {
  const { t } = useTranslation();
  const { data, error, isLoading } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
  });

  const tone = error
    ? "bg-app-danger/15 text-app-danger"
    : isLoading
      ? "bg-app-surface-muted text-app-muted-foreground"
      : "bg-app-success/15 text-app-success";

  const label = error
    ? t("health.pill.error", { defaultValue: "API offline" })
    : isLoading
      ? t("health.pill.loading", { defaultValue: "Checking…" })
      : t("health.pill.ok", { defaultValue: data?.status ?? "ok" });

  return (
    <span
      data-testid="health-pill"
      className={`inline-flex items-center gap-1.5 rounded-pill px-2.5 py-1 text-xs font-medium ${tone}`}
    >
      <Activity aria-hidden className="h-3.5 w-3.5" />
      {label}
    </span>
  );
}
