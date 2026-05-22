import * as React from "react";
import { Link, useRouteError } from "react-router-dom";

import { Button } from "./ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * RouteErrorFallback — the standard router `errorElement`. Renders when a
 * loader/action throws or when a route's render boundary catches an error.
 *
 * Per `react-stability` §2, this is the route-level boundary; feature-level
 * boundaries wrap heavier components inside the route.
 */
export function RouteErrorFallback() {
  const { t } = useTranslation();
  const error = useRouteError();

  React.useEffect(() => {
    // Surface the error for ops; the visible alert keeps the user oriented.
    if (error) console.error("[route-error]", error);
  }, [error]);

  return (
    <div
      data-testid={selectors.shared.routeErrorFallback.root}
      role="alert"
      className="flex min-h-[50vh] flex-col items-center justify-center gap-4 p-6 text-center"
    >
      <h2 className="text-xl font-semibold text-app-danger">
        {t(strings.shared.routeError.title)}
      </h2>
      <p className="max-w-md text-sm text-app-muted-foreground">
        {t(strings.shared.routeError.message)}
      </p>
      <div className="flex gap-2">
        <Button
          data-testid={selectors.shared.routeErrorFallback.retryButton}
          onClick={() => window.location.reload()}
        >
          {t(strings.shared.routeError.retry)}
        </Button>
        <Button asChild variant="outline">
          <Link
            to="/"
            data-testid={selectors.shared.routeErrorFallback.homeButton}
          >
            {t(strings.shared.routeError.home)}
          </Link>
        </Button>
      </div>
    </div>
  );
}
