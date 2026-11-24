import { Alert } from "@vrooli/react-component-library/Alert/1";
import { RefreshCw } from "lucide-react";

import { useTranslation } from "../../i18n";
import { Skeleton, WbButton } from "../primitives";

type ComponentPageLoadStateProps = {
  loading: boolean;
  error?: unknown;
  onRetry: () => void;
};

export function ComponentPageLoadState({
  loading,
  error,
  onRetry,
}: ComponentPageLoadStateProps) {
  const { t } = useTranslation();

  if (loading) {
    return (
      <div className="wb-page">
        <Skeleton height={34} width={240} />
        <div style={{ height: 12 }} />
        <Skeleton height={18} width={520} />
        <div style={{ height: 24 }} />
        <Skeleton height={420} />
      </div>
    );
  }

  return (
    <div className="wb-page">
      <Alert
        tone="danger"
        title={t("workbench.component.loadError", {
          defaultValue: "The component could not be opened.",
        })}
        description={error instanceof Error ? error.message : undefined}
        actions={
          <WbButton variant="ghost" size="md" onClick={onRetry}>
            <RefreshCw aria-hidden size={14} />
            {t("workbench.component.retry", { defaultValue: "Retry" })}
          </WbButton>
        }
      />
    </div>
  );
}
