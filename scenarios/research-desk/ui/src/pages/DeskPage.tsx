import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { useState } from "react";

import { LENS, useOverview } from "../api/activity";
import { useCapabilities, type Capability } from "../api/desk";
import { launchApi, useLaunchOptions, useLaunchTest } from "../api/launch";
import { operatorApi, useNeedsYou, type TestCourierResponse } from "../api/operator";
import { AccessWords } from "../components/AccessWords";
import { Confirm, Exact } from "../components/Confirm";
import { CourierHealth } from "../components/CourierHealth";
import { PathText } from "../components/PathText";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { When } from "../components/When";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { toDate } from "../lib/format";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_LABEL = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
} as const satisfies Record<ThemeChoice, string>;

/**
 * The desk itself (I5): which optional scenarios it can reach, what reads
 * `unknown` because one is down, where it reads from, how it starts sessions
 * and delivers decisions, in plain words (rulings V8), and the two self-tests,
 * launch and courier, each one confirmed action (F15). Appearance lives here
 * too.
 */
export function DeskPage() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  return (
    <div data-testid={selectors.pages.desk} className="rd-page">
      <PageHeader headingId="desk-heading" title={t(strings.desk.title)} description={t(strings.desk.description)} />
      <Capabilities />
      <Sources />
      <Tests />
      <Section id="appearance" title={t(strings.desk.appearance)}>
        <label className="flex max-w-xs flex-col gap-space-2xs text-body-sm">
          <span className="font-medium">{t(strings.theme.switcherLabel)}</span>
          <Select
            data-testid={selectors.desk.themeSelect}
            value={choice}
            onValueChange={(value) => setTheme(value as ThemeChoice)}
            options={(Object.keys(THEME_LABEL) as ThemeChoice[]).map((c) => ({ value: c, label: t(THEME_LABEL[c]) }))}
          />
        </label>
      </Section>
    </div>
  );
}

function tone(status: string): "success" | "warning" | "danger" | "neutral" {
  if (status === "available" || status === "ok" || status === "healthy") return "success";
  if (status === "degraded" || status === "unknown") return "warning";
  if (status === "unavailable" || status === "down" || status === "error") return "danger";
  return "neutral";
}

function Capabilities() {
  const { t } = useTranslation();
  const query = useCapabilities();
  return (
    <Section id="capabilities" title={t(strings.desk.optional)} note={t(strings.desk.optionalNote)}>
      <QueryView query={query}>
        {(list) =>
          list.length === 0 ? (
            <p className="text-body-sm">{t(strings.desk.noCapabilities)}</p>
          ) : (
            <ul className="rd-rows">{list.map((c) => <CapabilityRow key={c.id} capability={c} />)}</ul>
          )
        }
      </QueryView>
    </Section>
  );
}

function CapabilityRow({ capability: c }: { capability: Capability }) {
  const { t } = useTranslation();
  const ok = tone(c.status) === "success";
  return (
    <li className="rd-row">
      <span className="flex flex-wrap items-center justify-between gap-space-xs">
        <span className="font-semibold">{c.name || c.id}</span>
        <StatusBadge tone={tone(c.status)}>{c.status}</StatusBadge>
      </span>
      {c.message ? <span className="text-body-sm">{c.message}</span> : null}
      {c.uses && c.uses.length > 0 ? (
        <span className="text-body-sm text-app-muted-foreground">
          {t(ok ? strings.desk.usedFor : strings.desk.unknownWhileDown, { uses: c.uses.map((u) => u.name || u.id).join(", ") })}
        </span>
      ) : null}
    </li>
  );
}

/**
 * Where the desk reads from, then how it reaches the world outside the home:
 * starting sessions in Web Console and delivering decisions, each in plain
 * words rather than a raw lens row.
 */
function Sources() {
  const { t } = useTranslation();
  const overview = useOverview();
  return (
    <Section id="sources" title={t(strings.desk.sources)}>
      <QueryView query={overview}>
        {(o) => (
          <ul className="rd-rows">
            {o.lenses
              .filter((lens) => lens.name !== LENS.courier && lens.name !== LENS.webConsoleAccess)
              .map((lens) => (
                <li key={lens.name} className="rd-row">
                  <span className="flex flex-wrap items-center justify-between gap-space-xs">
                    <span className="font-semibold">{lens.name}</span>
                    <StatusBadge tone={lens.available ? "success" : "danger"}>{t(lens.available ? strings.desk.available : strings.desk.unavailable)}</StatusBadge>
                  </span>
                  {lens.reason ? (lens.reason.startsWith("/") ? <PathText path={lens.reason} /> : <span className="text-body-sm">{lens.reason}</span>) : null}
                </li>
              ))}
          </ul>
        )}
      </QueryView>
      <StartingSessions />
      <Delivery />
    </Section>
  );
}

/** How the desk starts sessions in Web Console, and whether it can (rulings F14, V8). */
function StartingSessions() {
  const { t } = useTranslation();
  const options = useLaunchOptions();
  const o = options.data;
  if (!o) return null;
  return (
    <div className="rd-rows">
      <div className="rd-row">
        <span className="flex flex-wrap items-center justify-between gap-space-xs">
          <span className="font-semibold">{t(strings.desk.automatic)}</span>
          <StatusBadge tone={o.automaticAvailable ? "success" : "neutral"}>{t(o.automaticAvailable ? strings.desk.automaticOn : strings.desk.automaticOff)}</StatusBadge>
        </span>
        <AccessWords said={o.webConsoleAccess} />
        {!o.automaticAvailable && o.automaticReason ? <span className="text-body-sm">{o.automaticReason}</span> : null}
      </div>
    </div>
  );
}

/** Whether decisions reach the phone: the courier's state and which instance sends. */
function Delivery() {
  const { t } = useTranslation();
  const courier = useNeedsYou().data?.courier;
  return (
    <div className="rd-rows">
      <CourierHealth />
      {courier ? (
        <div className="rd-row">
          <span className="flex flex-wrap items-center justify-between gap-space-xs">
            <span className="font-semibold">{t(strings.desk.courier)}</span>
            <StatusBadge tone={courier.writer ? "success" : "warning"}>{t(courier.writer ? strings.desk.writer : strings.desk.notWriter)}</StatusBadge>
          </span>
          {courier.writerReason ? <span className="text-body-sm">{courier.writerReason}</span> : null}
        </div>
      ) : null}
    </div>
  );
}

/**
 * The two self-tests, beside each other: each touches real services, never
 * runs on its own, and starts only after its confirmation (rulings F15).
 */
function Tests() {
  const { t } = useTranslation();
  return (
    <Section id="tests" title={t(strings.desk.tests)} note={t(strings.desk.testsNote)}>
      <ul className="rd-rows">
        <LaunchTest />
        <CourierTest />
      </ul>
    </Section>
  );
}

function LaunchTest() {
  const { t } = useTranslation();
  const report = useLaunchTest();
  const [open, setOpen] = useState(false);
  const r = report.data;
  const at = toDate(r?.at);
  const outcome = !r || !at ? t(strings.desk.neverRun) : r.running ? t(strings.desk.testRunning) : r.failedStep ? t(strings.launch.failedAt, { step: r.failedStep, reason: r.failedReason }) : t(strings.desk.testPassed, { status: r.statusFinal });
  return (
    <li className="rd-row">
      <span className="font-semibold">{t(strings.desk.launchTest)}</span>
      <span className="text-body-sm">
        {outcome} {at ? <When date={at} /> : null}
      </span>
      {r?.reportPath ? <PathText path={r.reportPath} /> : null}
      <div>
        <Button variant="secondary" disabled={r?.running} onClick={() => setOpen(true)}>{t(strings.desk.runLaunchTest)}</Button>
      </div>
      <Confirm
        open={open}
        title={t(strings.desk.launchTestTitle)}
        description={t(strings.desk.launchTestDescription)}
        confirmLabel={t(strings.desk.runLaunchTest)}
        onClose={() => setOpen(false)}
        onConfirm={async () => {
          await launchApi.testLaunch();
          await report.refetch();
        }}
      />
    </li>
  );
}

/**
 * "Send me a test decision": one ask through Notification Hub, then a wait of
 * up to a minute for a device to receive it. The 2026-11-11 failure was a
 * decision silently not arriving; this is how the operator proves it arrives.
 */
function CourierTest() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState<TestCourierResponse | null>(null);
  const delivered = toDate(result?.firstDeliveredAt);
  const expires = toDate(result?.expiresAt);
  return (
    <li className="rd-row">
      <span className="font-semibold">{t(strings.desk.courierTest)}</span>
      <span className="text-body-sm">{t(strings.desk.courierTestHint)}</span>
      {result ? (
        delivered ? (
          <Alert tone="success" title={t(strings.desk.courierDelivered, { seconds: result.waitedSeconds })} />
        ) : (
          <Alert
            tone="warning"
            title={t(strings.desk.courierNotDelivered, { seconds: result.waitedSeconds })}
            description={expires ? <>{t(strings.desk.courierExpires)} <When date={expires} /></> : undefined}
          />
        )
      ) : null}
      <div>
        <Button variant="secondary" onClick={() => setOpen(true)}>{t(strings.desk.runCourierTest)}</Button>
      </div>
      <Confirm
        open={open}
        title={t(strings.desk.courierTestTitle)}
        description={t(strings.desk.courierTestDescription)}
        confirmLabel={t(strings.desk.runCourierTest)}
        onClose={() => setOpen(false)}
        onConfirm={async () => {
          setResult(await operatorApi.testCourier());
        }}
      >
        <Exact label={t(strings.desk.courierTestSends)} text={t(strings.desk.courierTestText)} />
      </Confirm>
    </li>
  );
}
