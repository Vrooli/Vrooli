import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { optionLabel } from "../asks/askView";
import { DecisionText } from "./DecisionText";
import type { TextLinks } from "./decisionLinks";

/** What the requester recommends and why, so a cold reader gets the answer in one glance. */
export function Recommendation({ ask, links }: { ask: Ask; links: TextLinks }) {
  const { t } = useTranslation();
  if (!ask.recommended || !ask.options.some((option) => option.key === ask.recommended)) return null;
  return (
    <section aria-labelledby="recommendation-heading" className="rounded-panel border border-app-border border-s-4 border-s-app-warning bg-app-surface p-3">
      <h2 id="recommendation-heading" className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.packet.cover.recommended)}</h2>
      <p dir="auto" className="mt-1 break-words font-semibold leading-snug">{optionLabel(ask, ask.recommended)}</p>
      {ask.recommendationReason && <DecisionText value={ask.recommendationReason} links={links} className="mt-1 block text-app-muted-foreground" />}
    </section>
  );
}
