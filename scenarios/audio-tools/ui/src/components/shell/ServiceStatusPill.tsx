import { useQuery } from "@tanstack/react-query";
import { fetchHealth } from "../../api/health";
import { StatusDot } from "../ui/status-dot";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export function ServiceStatusPill() {
  const { t } = useTranslation();
  const { data, error, isLoading } = useQuery({
    queryKey: ["health", "pill"],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
    retry: false,
  });

  if (isLoading) return <StatusDot tone="neutral" pulse label={t(strings.status.checking)} />;
  if (error || !data) return <StatusDot tone="danger" label={t(strings.status.offline)} />;
  const tone = data.status.toLowerCase() === "ok" ? "success" : "warning";
  return <StatusDot tone={tone} label={data.service || t(strings.overview.summaryApi)} />;
}
