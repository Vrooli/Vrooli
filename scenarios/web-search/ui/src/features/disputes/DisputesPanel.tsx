import { useQuery } from "@tanstack/react-query";

import { findingsClient } from "../../api/clients";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { DisputeCard } from "./DisputeCard";

/**
 * DisputesPanel is the dispute review queue: every DISPUTED finding rendered as
 * a conflict card with a resolve action. It is the read+resolve surface over
 * FindingsService.ListDisputes / ResolveDispute.
 */
export function DisputesPanel() {
  const { t } = useTranslation();
  const queueKey = ["disputes"] as const;

  const query = useQuery({
    queryKey: queueKey,
    queryFn: async () => findingsClient.listDisputes({ limit: 100 }),
  });

  const disputes = query.data?.findings ?? [];

  return (
    <div data-testid={selectors.disputes.panel} className="flex flex-col gap-4">
      {query.isPending && (
        <p data-testid={selectors.disputes.loading} className="text-app-muted-foreground">
          {t(strings.disputes.loading)}
        </p>
      )}

      {query.isError && (
        <p data-testid={selectors.disputes.error} className="text-app-danger">
          {t(strings.disputes.error, { message: errorMessage(query.error, t) })}
        </p>
      )}

      {!query.isPending && !query.isError && disputes.length === 0 && (
        <p data-testid={selectors.disputes.empty} className="text-app-muted-foreground">
          {t(strings.disputes.empty)}
        </p>
      )}

      {disputes.length > 0 && (
        <ul data-testid={selectors.disputes.list} className="flex flex-col gap-3">
          {disputes.map((f) => (
            <DisputeCard key={f.id} finding={f} queueKey={queueKey} />
          ))}
        </ul>
      )}
    </div>
  );
}
