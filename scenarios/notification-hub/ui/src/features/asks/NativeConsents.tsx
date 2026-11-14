import { useInfiniteQuery } from "@tanstack/react-query";
import { conversationsClient } from "../../api/notifications";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

/** Native consent is reviewed at its source. This component has no mutation client. */
export function NativeConsents({ query }: { query: string }) {
  const { t, i18n } = useTranslation();
  const list = useInfiniteQuery({
    queryKey: ["native-consents", query], initialPageParam: "",
    queryFn: ({ pageParam, signal }) => conversationsClient.listNativeConsents({ limit: 25, cursor: pageParam, query }, { signal }),
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false, staleTime: 0, refetchInterval: 60_000,
  });
  const consents = [...new Map(list.data?.pages.flatMap((page) => page.consents).map((c) => [c.authorizationId || c.notificationId, c]) ?? []).values()];
  const format = (value: string) => {
    const date = new Date(value);
    return Number.isNaN(date.valueOf()) ? "" : new Intl.DateTimeFormat(i18n.language, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit", timeZoneName: "short" }).format(date);
  };
  if (list.isSuccess && !consents.length && !list.hasNextPage) return null;
  return <section aria-labelledby="native-consent-heading" className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
    <h3 id="native-consent-heading" className="text-lg font-semibold">{t(strings.nativeConsent.title)}</h3>
    <p className="text-sm text-app-muted-foreground">{t(strings.nativeConsent.authority)}</p>
    {list.isPending && <p role="status">{t(strings.nativeConsent.loading)}</p>}
    {list.isError && <div role="alert"><p>{t(strings.nativeConsent.unavailable)}</p><button type="button" onClick={() => void list.refetch()} className="min-h-11 underline">{t(strings.asks.list.retry)}</button></div>}
    {consents.length > 0 && <ul className="divide-y divide-app-border">{consents.map((consent) => <li key={consent.authorizationId || consent.notificationId} className="py-3">
      <h4 className="font-medium">{consent.title}</h4>
      <p className="mt-1 whitespace-pre-line text-sm">{consent.body}</p>
      <p className="mt-2 text-sm text-app-muted-foreground">{t(consent.state === "pending" && !list.isError ? strings.nativeConsent.pending : strings.nativeConsent.unverified)}</p>
      {consent.state === "pending" && !list.isError && <p className="text-xs text-app-muted-foreground">{t(strings.nativeConsent.checked, { time: format(consent.observedAt), expiry: format(consent.reviewExpiresAt) })}</p>}
      {consent.reviewUrl && <a href={consent.reviewUrl} className="mt-2 inline-flex min-h-11 items-center font-medium underline underline-offset-4" rel="noreferrer">{t(strings.nativeConsent.open)}</a>}
    </li>)}</ul>}
    {list.hasNextPage && <button type="button" disabled={list.isFetching} onClick={() => void list.fetchNextPage()} className="min-h-11 rounded-control border border-app-border p-3">{t(strings.nativeConsent.more)}</button>}
  </section>;
}
