import { AlertCircle, RefreshCw } from "lucide-react";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** Shared loading placeholder for every drill-down detail view. */
export function DetailLoading({ testId }: { testId?: string }) {
  const { t } = useTranslation();
  return (
    <div data-testid={testId}>
      <EmptyState
        title={t(strings.detail.loadingTitle)}
        description={t(strings.detail.loadingDescription)}
        icon={<RefreshCw aria-hidden className="h-5 w-5 animate-spin" />}
      />
    </div>
  );
}

/** Shared error placeholder for every drill-down detail view. */
export function DetailError({ testId }: { testId?: string }) {
  const { t } = useTranslation();
  return (
    <div data-testid={testId}>
      <EmptyState
        title={t(strings.detail.errorTitle)}
        description={t(strings.detail.errorDescription)}
        icon={<AlertCircle aria-hidden className="h-5 w-5" />}
      />
    </div>
  );
}
