import { useQuery } from "@tanstack/react-query";
import { HealthCard as LibraryHealthCard } from "@vrooli/react-component-library/HealthCard/0.1.6";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { fetchHealth } from "../../api/health";

export function HealthCard() {
  const { t } = useTranslation();
  const { data, error, isFetching, isLoading, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
  });

  const handleRefresh = () => {
    return refetch().then(() => undefined);
  };

  return (
    <LibraryHealthCard
      data={data ? { status: data.status, service: data.service, timestamp: data.timestamp } : undefined}
      loading={isLoading}
      error={error ? t(strings.health.error) : undefined}
      refreshing={isFetching && !isLoading}
      onRefresh={handleRefresh}
      title={t(strings.health.title)}
      statusLabel={t(strings.health.statusLabel)}
      serviceLabel={t(strings.health.serviceLabel)}
      timestampLabel={t(strings.health.timestampLabel)}
      loadingLabel={t(strings.health.loading)}
      errorLabel={t(strings.health.error)}
      refreshLabel={t(strings.health.refresh)}
      formatTimestamp={(timestamp) => formatDate(new Date(timestamp), { dateStyle: "medium", timeStyle: "short" })}
      testIds={selectors.health}
    />
  );
}
