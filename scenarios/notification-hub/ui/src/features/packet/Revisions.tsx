import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { Time } from "../../components/Time";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DecisionText } from "./DecisionText";
import type { TextLinks } from "./decisionLinks";
import { FIELD_LABEL, revisionChanges } from "./revisions";

/**
 * "Updated N times": for each update, what the decision said before and what
 * replaced it, so the operator can tell a refreshed decision from a new one and
 * see exactly what the agent changed after a question.
 */
export function Revisions({ ask, links, open, onToggle }: { ask: Ask; links: TextLinks; open: boolean; onToggle: (open: boolean) => void }) {
  const { t } = useTranslation();
  if (!ask.revisionCount) return null;
  return (
    <details id="packet-revisions" className="rounded-panel border border-app-border bg-app-surface p-3" open={open} onToggle={(event) => onToggle(event.currentTarget.open)}>
      <summary className="min-h-11 cursor-pointer py-2 font-medium">{t(strings.packet.revisions.title, { count: ask.revisionCount })}</summary>
      <ol className="mt-2 flex flex-col gap-4">
        {ask.revisions.map((revision) => {
          const changes = revisionChanges(ask, revision);
          return (
            <li key={revision.revision} data-revision={revision.revision} className="flex flex-col gap-2 border-t border-app-border pt-3 first:border-t-0 first:pt-0">
              <p className="text-xs text-app-muted-foreground">
                {t(strings.packet.revisions.replaced)} <Time value={revision.revisedAt} />
              </p>
              {changes.length === 0 ? (
                <DecisionText value={revision.question} links={links} className="text-app-muted-foreground" />
              ) : (
                changes.map((change) => (
                  <div key={change.field} className="flex flex-col gap-1 text-sm">
                    <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(FIELD_LABEL[change.field])}</p>
                    <div className="rounded-control border-s-4 border-app-border bg-app-surface-muted px-3 py-2">
                      <span className="sr-only">{t(strings.packet.revisions.before)}: </span>
                      <DecisionText value={change.before || t(strings.packet.revisions.empty)} links={links} className="text-app-muted-foreground line-through decoration-app-muted-foreground/50" />
                    </div>
                    <div className="rounded-control border-s-4 border-app-primary bg-app-primary/10 px-3 py-2">
                      <span className="sr-only">{t(strings.packet.revisions.now)}: </span>
                      <DecisionText value={change.after || t(strings.packet.revisions.empty)} links={links} />
                    </div>
                  </div>
                ))
              )}
            </li>
          );
        })}
      </ol>
    </details>
  );
}
