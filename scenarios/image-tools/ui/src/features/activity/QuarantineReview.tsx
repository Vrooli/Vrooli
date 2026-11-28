import { useEffect, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import type { QuarantinedOutput } from "../../api/quarantine";
import type { FeedbackChoice } from "../../api/safetyFeedback";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { cn } from "../../lib/utils";
import { liveQuarantineClient, type QuarantineClient } from "../safety/quarantineClient";

export interface QuarantineReviewProps {
  client?: QuarantineClient;
}

const CHOICES: FeedbackChoice[] = ["legitimate", "confirmed", "uncertain"];

const CHOICE_LABEL = {
  legitimate: strings.quarantine.legitimate,
  confirmed: strings.quarantine.confirmed,
  uncertain: strings.quarantine.uncertain,
} as const;

const CHOICE_SAVED = {
  legitimate: strings.quarantine.savedLegitimate,
  confirmed: strings.quarantine.savedConfirmed,
  uncertain: strings.quarantine.savedUncertain,
} as const;

function QuarantineCard({ item, client }: { item: QuarantinedOutput; client: QuarantineClient }) {
  const { t } = useTranslation();
  const [revealed, setRevealed] = useState(false);
  const [choice, setChoice] = useState<FeedbackChoice | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const id = item.id.toString();
  const reason = item.reasonCode || "—";
  const expires = item.expiresAt
    ? new Date(Number(item.expiresAt.seconds) * 1000).toLocaleString()
    : "";

  const save = async (next: FeedbackChoice) => {
    setPending(true);
    setError(null);
    try {
      await client.label(item.jobId, item.originalRef, next);
      setChoice(next);
    } catch (err) {
      setError(errorMessage(err, t));
    } finally {
      setPending(false);
    }
  };

  return (
    <li
      data-testid={selectors.quarantine.card({ id })}
      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-sm font-medium text-app-foreground">
          {t(strings.quarantine.cardLabel, { job: item.jobId, reason, score: item.score.toFixed(2) })}
        </span>
        {expires ? (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.quarantine.expires, { when: expires })}
          </span>
        ) : null}
      </div>

      {revealed ? (
        <img
          data-testid={selectors.quarantine.image({ id })}
          src={client.imageUrl(item.id)}
          alt={t(strings.quarantine.imageAlt, { job: item.jobId })}
          className="max-h-64 w-full rounded-control border border-app-border object-contain"
        />
      ) : (
        <div className="flex flex-col gap-2 rounded-control border border-dashed border-app-border bg-app-surface-muted p-2">
          <p className="text-xs text-app-muted-foreground">{t(strings.quarantine.warning)}</p>
          <Button
            type="button"
            size="sm"
            data-testid={selectors.quarantine.reveal({ id })}
            onClick={() => setRevealed(true)}
            shape="square"
          >
            {t(strings.quarantine.reveal)}
          </Button>
        </div>
      )}

      {revealed ? (
        <a
          data-testid={selectors.quarantine.download({ id })}
          href={client.imageUrl(item.id, true)}
          download
          className="w-fit text-xs font-medium text-app-primary underline underline-offset-2"
        >
          {t(strings.quarantine.download)}
        </a>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-app-muted-foreground">{t(strings.reviewerFeedback.prompt)}</span>
        {CHOICES.map((value) => (
          <Button
            key={value}
            type="button"
            size="sm"
            variant={value === "legitimate" ? "primary" : "ghost"}
            shape="square"
            data-testid={selectors.quarantine[value]({ id })}
            aria-pressed={choice === value}
            disabled={pending}
            onClick={() => void save(value)}
            className={cn(
              "border border-app-border",
              value === "legitimate"
                ? "bg-app-success/10 text-app-success hover:bg-app-success/20"
                : "text-app-foreground hover:bg-app-surface-muted",
              choice === value && "ring-2 ring-app-primary/40",
            )}
          >
            {t(CHOICE_LABEL[value])}
          </Button>
        ))}
      </div>

      {choice ? (
        <span data-testid={selectors.quarantine.saved({ id })} className="text-xs font-medium text-app-success">
          {t(CHOICE_SAVED[choice])}
        </span>
      ) : null}
      {error ? (
        <p data-testid={selectors.quarantine.labelError({ id })} role="alert" className="text-xs text-app-danger">
          {t(strings.quarantine.labelError)}: {error}
        </p>
      ) : null}
    </li>
  );
}

/**
 * The bounded admin review queue for refused outputs. Nothing is fetched until
 * the operator reveals an entry; the ordinary result/library surfaces never see
 * these bytes. Local deployments are ungated; a public deployment requires the
 * configured admin token (the CLI sends it; the local UI is the intended path).
 */
export function QuarantineReview({ client = liveQuarantineClient }: QuarantineReviewProps) {
  const { t } = useTranslation();
  const [items, setItems] = useState<QuarantinedOutput[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    setItems(null);
    setError(null);
    client
      .list()
      .then((outputs) => {
        if (live) setItems(outputs);
      })
      .catch((err: unknown) => {
        if (live) {
          setError(errorMessage(err, t));
          setItems([]);
        }
      });
    return () => {
      live = false;
    };
  }, [client, t]);

  if (error) {
    return (
      <section data-testid={selectors.activity.quarantine.root} className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.quarantine.heading)}
        </h3>
        <p data-testid={selectors.activity.quarantine.error} className="text-sm text-app-danger">
          {t(strings.quarantine.error)}
        </p>
      </section>
    );
  }

  if (items === null) {
    return (
      <section data-testid={selectors.activity.quarantine.root} className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.quarantine.heading)}
        </h3>
        <p data-testid={selectors.activity.quarantine.loading} className="text-sm text-app-muted-foreground">
          {t(strings.quarantine.loading)}
        </p>
      </section>
    );
  }

  if (items.length === 0) {
    return (
      <section data-testid={selectors.activity.quarantine.root} className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.quarantine.heading)}
        </h3>
        <p data-testid={selectors.activity.quarantine.empty} className="text-sm text-app-muted-foreground">
          {t(strings.quarantine.empty)}
        </p>
      </section>
    );
  }

  return (
    <section data-testid={selectors.activity.quarantine.root} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.quarantine.heading)}
        </h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.quarantine.description)}</p>
      </div>
      <ul data-testid={selectors.activity.quarantine.list} className="flex flex-col gap-2">
        {items.map((item) => (
          <QuarantineCard key={item.id.toString()} item={item} client={client} />
        ))}
      </ul>
      <p className="text-xs text-app-muted-foreground">{t(strings.quarantine.hint)}</p>
    </section>
  );
}
