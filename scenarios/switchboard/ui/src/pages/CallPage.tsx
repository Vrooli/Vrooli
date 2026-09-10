import { LogIn, PhoneOff } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { Page } from "../components/console/Page";
import { Panel } from "../components/console/Panel";
import { Quiet, Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useSession } from "../features/session/SessionProvider";
import { useTranslation } from "../i18n";

/**
 * Call mode is declared by the experience contract and deliberately deferred
 * by the plan (OT-P2). The route exists so a deep link lands somewhere honest:
 * the transcript region states that calls are not available in this release
 * and points back at the thread. A signed-out visitor also gets the sign-in
 * prompt here rather than discovering the requirement after calls ship.
 */
export function CallPage() {
  const { t } = useTranslation();
  const { threadId = "" } = useParams<{ threadId: string }>();
  const { session, requireSession } = useSession();
  return (
    <Page headingId="call-heading" testId="page-call" title={t(strings.console.call.title)} description={t(strings.console.call.description)}>
      {session ? null : (
        <Panel role="note" data-testid="call-signin" className="flex flex-col gap-2 border-app-primary/40 bg-app-primary/5 px-3 py-2.5 text-sm sm:flex-row sm:items-center">
          <LogIn aria-hidden="true" className="h-4 w-4 shrink-0 text-app-primary" />
          <p className="flex-1">
            <span className="font-medium text-app-foreground">{t(strings.console.call.signInTitle)}</span>{" "}
            <span className="text-app-muted-foreground">{t(strings.console.call.signInDetail)}</span>
          </p>
          <button
            type="button"
            data-testid="call-signin-cta"
            onClick={() => void requireSession()}
            className="inline-flex min-h-11 shrink-0 items-center justify-center rounded-control bg-app-primary px-3 text-sm font-medium text-app-primary-foreground hover:opacity-90 md:min-h-9"
          >
            {t(strings.console.session.signIn)}
          </button>
        </Panel>
      )}
      <Region surfaceId="transcript-region" testId="call-transcript-region" state="ready">
        <Quiet
          icon={<PhoneOff className="h-6 w-6" />}
          title={t(strings.console.call.unavailableTitle)}
          description={t(strings.console.call.unavailableDetail)}
          action={
            <Link to={`/conversations/${encodeURIComponent(threadId)}`} className="inline-flex min-h-11 items-center rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground hover:bg-app-surface-muted">
              {t(strings.console.call.backToThread)}
            </Link>
          }
        />
      </Region>
    </Page>
  );
}
