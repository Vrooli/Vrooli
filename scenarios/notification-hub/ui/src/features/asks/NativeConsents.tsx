import { Button } from "@vrooli/react-component-library/Button/2";
import { ShieldQuestion } from "lucide-react";

import { ListGroup, ListItem } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useNow } from "../../hooks/useNow";
import { useTranslation } from "../../i18n";
import { isUnauthenticated } from "../../lib/apiError";
import { formatAgo, formatDueIn } from "../../lib/ago";
import type { NativeConsentsRead } from "./useNativeConsents";

/**
 * Permission requests owned by Agent Manager. They are reviewed at their source:
 * this component has no mutation client, and a link opens the source page only.
 * A failed read is the queue page's to report (one message for everything that
 * could not load); here it only means the requests shown are unverified.
 */
export function NativeConsents({ list }: { list: NativeConsentsRead }) {
  const { t } = useTranslation();
  const now = useNow();
  const consents = [...new Map(list.data?.pages.flatMap((page) => page.consents).map((c) => [c.authorizationId || c.notificationId, c]) ?? []).values()];
  // A missing session is the list's to explain once; this section has nothing of its own to say.
  if (isUnauthenticated(list.error)) return null;
  if (list.isError && !consents.length) return null;
  if (list.isSuccess && !consents.length && !list.hasNextPage) return null;
  return (
    <section aria-labelledby="native-consent-heading" className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h2 id="native-consent-heading" className="flex items-center gap-2 text-base font-semibold">
          <ShieldQuestion size={18} aria-hidden="true" className="text-app-warning" />
          {t(strings.nativeConsent.title)}
        </h2>
        <details className="text-sm text-app-muted-foreground">
          <summary className="min-h-11 cursor-pointer py-2.5">
            {t(strings.nativeConsent.authorityShort)} <span className="font-medium text-app-primary">{t(strings.nativeConsent.authorityWhy)}</span>
          </summary>
          <p>{t(strings.nativeConsent.authority)}</p>
        </details>
      </div>
      {list.isPending && <SkeletonRows rows={1} />}
      {consents.length > 0 && (
        <ListGroup>
          {consents.map((consent) => {
            const verified = consent.state === "pending" && !list.isError;
            return (
              <ListItem key={consent.authorizationId || consent.notificationId} tone="attention" className="flex flex-col gap-1 px-4 py-3">
                <h3 className="font-semibold">{consent.title}</h3>
                <p className="whitespace-pre-line text-sm">{consent.body}</p>
                <p className="text-sm text-app-muted-foreground">{t(verified ? strings.nativeConsent.pending : strings.nativeConsent.unverified)}</p>
                {verified && (
                  <p className="text-xs text-app-muted-foreground">
                    {t(strings.nativeConsent.checked, { time: formatAgo(consent.observedAt, now) ?? "", expiry: formatDueIn(consent.reviewExpiresAt, now) ?? "" })}
                  </p>
                )}
                {consent.reviewUrl && (
                  <a href={consent.reviewUrl} rel="noreferrer" className="inline-flex min-h-11 w-fit items-center text-sm font-medium text-app-primary hover:underline">
                    {t(strings.nativeConsent.open)}
                  </a>
                )}
              </ListItem>
            );
          })}
        </ListGroup>
      )}
      {list.hasNextPage && (
        <Button type="button" variant="outline" disabled={list.isFetching} onClick={() => void list.fetchNextPage()}>
          {t(strings.nativeConsent.more)}
        </Button>
      )}
    </section>
  );
}
