import { RefreshCw } from "lucide-react";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Suspense fallback shown while a code-split detail route's chunk loads. Kept in
 * its own module so `routeConfig` exports only route data (Fast Refresh only
 * hot-reloads files that export components exclusively, or data exclusively).
 */
export function RouteFallback() {
  const { t } = useTranslation();
  return (
    <div
      role="status"
      aria-live="polite"
      className="flex min-h-40 items-center justify-center text-app-muted-foreground"
    >
      <RefreshCw aria-hidden className="h-5 w-5 animate-spin" />
      <span className="sr-only">{t(strings.detail.loadingTitle)}</span>
    </div>
  );
}
