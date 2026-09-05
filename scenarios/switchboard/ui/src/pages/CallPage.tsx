import { PhoneOff } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Call mode is declared by the experience contract and deliberately deferred
 * by the plan (OT-P2). The route exists so a deep link lands somewhere honest:
 * the transcript region states that calls are not available in this release
 * and points back at the thread.
 */
export function CallPage() {
  const { t } = useTranslation();
  const { threadId = "" } = useParams<{ threadId: string }>();
  return (
    <Page headingId="call-heading" testId="page-call" title={t(strings.console.call.title)} description={t(strings.console.call.description)}>
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
