import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SessionGate } from "../session/SessionGate";
import { optionLabel } from "./askView";

/** Every open decision at once, newest first, each linking to its page. */
export function AsksPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.asks}
      aria-labelledby="asks-heading"
      className="mx-auto flex w-full max-w-xl flex-col gap-4"
    >
      <h2 id="asks-heading" className="text-2xl font-semibold">
        {t(strings.asks.list.title)}
      </h2>
      <SessionGate>
        <OpenAsks />
      </SessionGate>
    </section>
  );
}

function OpenAsks() {
  const { t } = useTranslation();
  const asks = useQuery({
    queryKey: ["asks", "open"],
    queryFn: async () =>
      (await conversationsClient.listAsks({ openOnly: true, limit: 50 })).asks,
    refetchInterval: 60_000,
  });
  if (asks.isPending) return null;
  if (asks.isError) return <p role="alert">{t(strings.asks.list.loadError)}</p>;
  if (asks.data.length === 0)
    return (
      <p
        data-testid={selectors.asks.empty}
        className="text-app-muted-foreground"
      >
        {t(strings.asks.list.empty)}
      </p>
    );
  return (
    <ul data-testid={selectors.asks.list} className="flex flex-col gap-2">
      {asks.data.map((ask) => (
        <li key={ask.id} data-testid={selectors.asks.item}>
          <Link
            to={`/asks/${ask.id}`}
            className="block rounded-panel border border-app-border p-3 hover:bg-app-surface-muted"
          >
            <span className="block font-medium">{ask.question}</span>
            {ask.recommended && (
              <span className="block text-sm text-app-muted-foreground">
                {t(strings.asks.detail.recommended)}:{" "}
                {optionLabel(ask, ask.recommended)}
              </span>
            )}
          </Link>
        </li>
      ))}
    </ul>
  );
}
