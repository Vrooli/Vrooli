import { useState } from "react";
import { BellOff, BellRing, Smartphone } from "lucide-react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { ConnectionStatus } from "../../api/serviceWorker";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import { Notice, type NoticeTone } from "../settings/Notice";
import { useBrowserEnrollment, type EnrollResult } from "./useBrowserEnrollment";

const RESULT = {
  registered: { tone: "success", copy: strings.reach.browser.result.registered },
  registeredNoSubscription: { tone: "warning", copy: strings.reach.browser.result.registeredNoSubscription },
  unsupported: { tone: "warning", copy: strings.reach.browser.result.unsupported },
  denied: { tone: "warning", copy: strings.reach.browser.result.denied },
  dismissed: { tone: "info", copy: strings.reach.browser.result.dismissed },
  noKey: { tone: "danger", copy: strings.reach.browser.result.noKey },
  unconfirmed: { tone: "danger", copy: strings.reach.browser.result.unconfirmed },
  failed: { tone: "danger", copy: strings.reach.browser.result.failed },
} as const satisfies Record<EnrollResult, { tone: NoticeTone; copy: string }>;

const PERMISSION_STATE = {
  granted: strings.reach.browser.permissionState.granted,
  denied: strings.reach.browser.permissionState.denied,
  default: strings.reach.browser.permissionState.default,
  unsupported: strings.reach.browser.permissionState.unsupported,
  unknown: strings.reach.browser.permissionState.unknown,
} as const;

type CardState = "checking" | "unsupported" | "denied" | "active" | "inactive";

function cardState(status: ConnectionStatus | undefined, registeredHere: boolean): CardState {
  if (!status) return "checking";
  if (!status.supported) return "unsupported";
  if (status.permission === "denied") return "denied";
  return registeredHere && status.subscribed ? "active" : "inactive";
}

/**
 * This browser's place in the list of ways to reach you: whether it can receive
 * push, whether it is registered, and the one action that fixes the state it is
 * in. The technical readings sit behind "Details" for the rare debugging visit.
 */
export function BrowserPushCard({ registeredHere }: { registeredHere: boolean }) {
  const { t } = useTranslation();
  const { status, update, phase, result, enable } = useBrowserEnrollment();
  const [confirming, setConfirming] = useState(false);
  const state = cardState(status.data, registeredHere);
  const busy = phase !== "idle";
  const data = status.data;

  const headline: Record<CardState, string> = {
    checking: t(strings.reach.browser.checking),
    unsupported: t(strings.reach.browser.unsupportedTitle),
    denied: t(strings.reach.browser.deniedTitle),
    active: t(strings.reach.browser.onTitle),
    inactive: t(strings.reach.browser.offTitle),
  };
  const detail: Record<CardState, string> = {
    checking: "",
    unsupported: t(strings.reach.browser.unsupportedBody),
    denied: t(strings.reach.browser.deniedBody),
    active: t(strings.reach.browser.onBody),
    inactive: t(strings.reach.browser.offBody),
  };
  const Icon = state === "active" ? BellRing : state === "inactive" || state === "checking" ? Smartphone : BellOff;

  return (
    <section aria-labelledby="reach-browser-heading" data-testid={selectors.reach.connectionStatus} className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      <div className="flex items-start gap-3">
        <span className={`mt-0.5 grid size-9 shrink-0 place-items-center rounded-full ${state === "active" ? "bg-app-success/15 text-app-success" : "bg-app-surface-muted text-app-muted-foreground"}`} aria-hidden="true">
          <Icon size={20} />
        </span>
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 id="reach-browser-heading" className="text-base font-semibold">{t(strings.reach.browser.title)}</h2>
          <p className="text-sm font-medium" data-testid={selectors.reach.browserState}>{headline[state]}</p>
          {detail[state] && <p className="text-sm text-app-muted-foreground">{detail[state]}</p>}
        </div>
      </div>

      {status.isError && <Notice tone="warning">{t(strings.reach.browser.statusUnavailable)}</Notice>}
      {phase === "asking" && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.reach.browser.asking)}</p>}
      {phase === "registering" && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.reach.browser.registering)}</p>}
      {result && <Notice tone={RESULT[result].tone} testId={selectors.reach.enrollResult}>{t(RESULT[result].copy)}</Notice>}

      <div className="flex flex-wrap gap-2">
        {state === "inactive" && (
          <Button type="button" variant="primary" disabled={busy} onClick={() => setConfirming(true)} data-testid={selectors.reach.addBrowser}>
            {t(strings.reach.browser.turnOn)}
          </Button>
        )}
        {(state === "denied" || state === "unsupported") && (
          <Button type="button" variant="outline" disabled={status.isFetching} onClick={() => void status.refetch()}>{t(strings.reach.browser.checkAgain)}</Button>
        )}
        {data?.stale && (
          <Button type="button" variant="outline" disabled={update.isPending} onClick={() => update.mutate()}>
            {update.isPending ? t(strings.reach.browser.updating) : t(strings.reach.browser.update)}
          </Button>
        )}
      </div>
      {data?.stale && <p className="text-sm text-app-muted-foreground">{t(strings.reach.browser.updateHelp)}</p>}

      {data && (
        <details className="text-sm">
          <summary className="min-h-11 cursor-pointer py-2 font-medium text-app-primary">{t(strings.reach.browser.details)}</summary>
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5 pb-1">
            <dt className="text-app-muted-foreground">{t(strings.reach.browser.support)}</dt>
            <dd>{t(data.supported ? strings.reach.browser.available : strings.reach.browser.unavailable)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.browser.permission)}</dt>
            <dd>{t(PERMISSION_STATE[permissionKey(data.permission)])}</dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.browser.subscription)}</dt>
            <dd data-testid={selectors.reach.subscriptionValue} className="[overflow-wrap:anywhere]">{data.subscribed ? <>{t(strings.reach.browser.subscribed)} (<bdi dir="ltr">{data.endpointHost}</bdi>)</> : t(strings.reach.browser.notSubscribed)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.reach.browser.worker)}</dt>
            <dd>{!data.controlled
              ? t(strings.reach.browser.noWorker)
              : data.stale
                ? t(strings.reach.browser.staleWorker, { running: data.runningVersion ?? t(strings.reach.browser.unknown), available: data.availableVersion ?? t(strings.reach.browser.unknown) })
                : t(strings.reach.browser.currentWorker, { version: data.runningVersion ?? data.availableVersion ?? t(strings.reach.browser.unknown) })}</dd>
          </dl>
        </details>
      )}

      <AlertDialog
        open={confirming}
        title={t(strings.reach.browser.confirmTitle)}
        description={t(strings.reach.browser.confirmBody)}
        confirmLabel={t(strings.reach.browser.confirm)}
        cancelLabel={t(strings.settings.action.cancel)}
        closeLabel={t(strings.settings.action.cancel)}
        onConfirm={() => { setConfirming(false); void enable(); }}
        onCancel={() => setConfirming(false)}
      />
    </section>
  );
}

function permissionKey(permission: string): "granted" | "denied" | "default" | "unsupported" | "unknown" {
  return permission === "granted" || permission === "denied" || permission === "default" || permission === "unsupported" ? permission : "unknown";
}
