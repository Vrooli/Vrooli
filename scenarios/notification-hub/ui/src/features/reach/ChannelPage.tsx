import { useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Mail } from "lucide-react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { recipientsClient } from "../../api/notifications";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { dialogPrefixes, selectors } from "../../consts/selectors";
import { channelLabel } from "../settings/channelLabel";
import { formatList } from "../settings/format";
import { Notice } from "../settings/Notice";
import { OwnerGate } from "../settings/OwnerGate";
import { channelAddressesQuery, devicesQuery, readFresh } from "../settings/ownerQueries";
import { SaveBar } from "../settings/SaveBar";
import { SettingsScreen } from "../settings/SettingsScreen";
import { useSettingsPath } from "../settings/testContext";
import { UnsavedChangesGuard } from "../settings/UnsavedChangesGuard";
import { useSession } from "../session/useSession";
import { contentLabel } from "./contentLabel";
import { CONTENT_LABELS, isContentLabel, newStableAddressId, sameLabels, type ContentLabel } from "./webPush";

type Address = { id: string; deviceId: string; channel: string; address: string; approvedLabels: string[] };
type Problem = "conflict" | "uncertain" | "unconfirmed" | "unreadable";

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const PROBLEM = {
  conflict: { title: strings.reach.channel.problem.conflict.title, body: strings.reach.channel.problem.conflict.body },
  uncertain: { title: strings.reach.channel.problem.uncertain.title, body: strings.reach.channel.problem.uncertain.body },
  unconfirmed: { title: strings.reach.channel.problem.unconfirmed.title, body: strings.reach.channel.problem.unconfirmed.body },
  unreadable: { title: strings.reach.channel.problem.unreadable.title, body: strings.reach.channel.problem.unreadable.body },
} as const satisfies Record<Problem, { title: string; body: string }>;

const LABEL_HINT = {
  public: strings.reach.channel.hints.public,
  private: strings.reach.channel.hints.private,
  sensitive: strings.reach.channel.hints.sensitive,
  critical: strings.reach.channel.hints.critical,
} as const satisfies Record<ContentLabel, string>;

/**
 * Edit where a device is reached on one channel: the email address, and which
 * kinds of content may be sent there in full. Anything that widens what leaves
 * the Hub is confirmed in plain words before it is saved.
 */
export function ChannelPage() {
  const { t } = useTranslation();
  const { deviceId = "", channel = "" } = useParams();
  const devicePath = `/settings/reach/devices/${encodeURIComponent(deviceId)}`;
  return (
    <SettingsScreen title={channelLabel(channel, t)} description={t(strings.reach.channel.description)} back={{ to: devicePath, label: t(strings.reach.channel.back) }}>
      <OwnerGate><ChannelContent key={`${deviceId}/${channel}`} deviceId={deviceId} channel={channel} devicePath={devicePath} /></OwnerGate>
    </SettingsScreen>
  );
}

function ChannelContent({ deviceId, channel, devicePath }: { deviceId: string; channel: string; devicePath: string }) {
  const { t } = useTranslation();
  const path = useSettingsPath();
  const session = useSession();
  const owner = session.isSuccess && session.data.signedIn;
  const devices = useQuery({ ...devicesQuery, enabled: owner, refetchOnWindowFocus: true });
  const addresses = useQuery({ ...channelAddressesQuery, enabled: owner, refetchOnWindowFocus: true });
  // The form edits the address as it was when the form opened. A later read that
  // finds a different row (another tab, a save's own read-back) must not swap the
  // form underneath a draft; the save compares against fresh reads itself.
  const opened = useRef<{ existing: Address | undefined } | null>(null);
  const back = (
    <Link to={path(devicePath)} className="mt-2 inline-flex min-h-11 items-center font-medium text-app-primary underline underline-offset-4">{t(strings.reach.channel.back)}</Link>
  );

  if (devices.isPending || addresses.isPending) return <SkeletonRows rows={2} />;
  const failed = devices.isError && !devices.data ? devices : addresses.isError && !addresses.data ? addresses : undefined;
  if (failed) {
    return <QueryError error={failed.error} title={t(strings.reach.channel.loadFailed)} detail={t(strings.reach.channel.loadFailedDetail)} onRetry={() => { void devices.refetch(); void addresses.refetch(); }} retrying={failed.isFetching} />;
  }
  const device = devices.data?.devices.find((row) => row.id === deviceId);
  if (!device) {
    return <EmptyNotice icon={<Mail size={28} />} title={t(strings.reach.device.goneTitle)} detail={t(strings.reach.device.goneDetail)}>{back}</EmptyNotice>;
  }
  const existing = addresses.data?.addresses.find((row) => row.deviceId === deviceId && row.channel === channel);
  const isEmail = channel === "email";
  const isPush = channel === "web_push";
  // Browser push is created by turning notifications on, never here; and a
  // channel the device lists without a readable setting has nothing safe to edit.
  const editable = isEmail ? existing !== undefined || !device.channels.includes("email") : isPush && existing !== undefined;
  if (!editable) {
    return <EmptyNotice icon={<Mail size={28} />} title={t(strings.reach.channel.notEditableTitle)} detail={t(strings.reach.channel.notEditableDetail)}>{back}</EmptyNotice>;
  }
  opened.current ??= { existing };
  return <ChannelForm deviceId={deviceId} deviceName={device.name} channel={channel} existing={opened.current.existing} devicePath={devicePath} />;
}

function ChannelForm({ deviceId, deviceName, channel, existing, devicePath }: {
  deviceId: string;
  deviceName: string;
  channel: string;
  existing: Address | undefined;
  devicePath: string;
}) {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const path = useSettingsPath();
  const isEmail = channel === "email";
  const savedLabels = existing?.approvedLabels ?? ["public"];
  const [destination, setDestination] = useState(existing?.address ?? "");
  const [labels, setLabels] = useState<string[]>(savedLabels);
  const [saving, setSaving] = useState(false);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [confirming, setConfirming] = useState(false);
  // A new address keeps one ID across every attempt, so a create whose answer
  // was lost is recognised by the Hub instead of repeated.
  const createId = useRef<string | null>(null);

  const trimmed = destination.trim();
  const emailValid = EMAIL.test(trimmed);
  const preserved = labels.filter((label) => !isContentLabel(label));
  const added = labels.filter((label) => !savedLabels.includes(label));
  const destinationChanged = isEmail && trimmed !== (existing?.address ?? "");
  const changed = !existing || destinationChanged || !sameLabels(labels, savedLabels);
  // A new form with nothing typed or changed has nothing to lose by leaving.
  const touched = existing ? changed : trimmed.length > 0 || !sameLabels(labels, savedLabels);
  const valid = (!isEmail || emailValid) && labels.length > 0;
  const widens = !existing || added.length > 0 || destinationChanged;
  const edit = (apply: () => void) => { setProblem(null); apply(); };
  const toggle = (label: ContentLabel, on: boolean) => edit(() => setLabels(on ? [...labels, label] : labels.filter((item) => item !== label)));
  const labelList = (values: readonly string[]) => formatList(values.map((value) => contentLabel(value, t)), lang);

  const save = async () => {
    setConfirming(false);
    setSaving(true);
    setProblem(null);
    const address = isEmail ? trimmed : existing?.address ?? "";
    const id = existing?.id ?? (createId.current ??= newStableAddressId());
    let responseUnknown = false;
    try {
      if (existing) await recipientsClient.updateChannelAddress({ id, address, approvedLabels: labels });
      else await recipientsClient.createChannelAddress({ id, deviceId, channel, address, approvedLabels: labels });
    } catch {
      responseUnknown = true;
    }
    try {
      const latest = await readFresh(queryClient, channelAddressesQuery);
      const row = latest.addresses.find((candidate) => candidate.id === id);
      if (row && row.deviceId === deviceId && row.channel === channel && row.address === address && sameLabels(row.approvedLabels, labels)) {
        void queryClient.invalidateQueries({ queryKey: devicesQuery.queryKey });
        navigate(path(devicePath), { state: { notice: { kind: "channelSaved", channel, uncertain: responseUnknown } } });
        return;
      }
      const taken = !existing && latest.addresses.some((candidate) => candidate.deviceId === deviceId && candidate.channel === channel);
      setProblem(taken ? "conflict" : responseUnknown ? "uncertain" : "unconfirmed");
    } catch {
      setProblem("unreadable");
    }
    setSaving(false);
  };

  return (
    <form
      className="flex flex-col gap-5"
      onSubmit={(event) => { event.preventDefault(); if (valid && changed && !saving) { if (widens) setConfirming(true); else void save(); } }}
    >
      <UnsavedChangesGuard when={touched && !saving} />
      {isEmail && (
        <label className="flex flex-col gap-1 text-sm font-semibold">
          {t(strings.reach.channel.emailAddress)}
          <input
            type="email"
            dir="ltr"
            autoComplete="email"
            inputMode="email"
            value={destination}
            onChange={(event) => edit(() => setDestination(event.target.value))}
            aria-invalid={destination.length > 0 && !emailValid}
            aria-describedby="reach-email-help"
            data-testid={selectors.reach.emailDestination}
            className="min-h-11 w-full rounded-control border border-app-border bg-app-background px-3 text-base font-normal focus-visible:outline focus-visible:outline-2"
          />
          <span id="reach-email-help" className={destination.length > 0 && !emailValid ? "text-sm font-normal text-app-danger" : "text-sm font-normal text-app-muted-foreground"}>
            {destination.length > 0 && !emailValid
              ? t(existing ? strings.reach.channel.invalidEmailKept : strings.reach.channel.invalidEmail)
              : t(strings.reach.channel.emailHelp, { device: deviceName })}
          </span>
        </label>
      )}

      <fieldset className="flex flex-col gap-1">
        <legend className="text-sm font-semibold">{t(strings.reach.channel.fullContentHeading)}</legend>
        <p className="mb-1 text-sm text-app-muted-foreground">{t(strings.reach.channel.fullContentHelp)}</p>
        <div className="divide-y divide-app-border overflow-hidden rounded-panel border border-app-border bg-app-surface">
          {CONTENT_LABELS.map((label) => (
            <div key={label} className="relative flex min-h-14 items-start gap-3 px-4 py-3 has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:-outline-offset-2">
              <input id={`reach-label-${label}`} type="checkbox" className="mt-0.5 size-5 shrink-0 accent-[var(--color-primary)]" checked={labels.includes(label)} onChange={(event) => toggle(label, event.target.checked)} />
              <label htmlFor={`reach-label-${label}`} data-testid={selectors.reach.label({ label })} className="flex min-w-0 flex-1 cursor-pointer flex-col font-medium after:absolute after:inset-0 after:content-['']">
                {contentLabel(label, t)}
                <span className="text-sm font-normal text-app-muted-foreground">{t(LABEL_HINT[label])}</span>
              </label>
            </div>
          ))}
        </div>
        {preserved.length > 0 && <p className="text-sm text-app-muted-foreground" data-testid={selectors.reach.unknownLabelPreserved}>{t(strings.reach.channel.preserved, { labels: preserved.join(", ") })}</p>}
        {labels.length === 0 && <p role="status" className="text-sm text-app-danger">{t(strings.reach.channel.pickOne)}</p>}
        {added.some((label) => label === "sensitive" || label === "critical") && <Notice tone="warning">{t(isEmail ? strings.reach.channel.widenEmail : strings.reach.channel.widenPush)}</Notice>}
      </fieldset>

      {problem && <Notice tone="danger" title={t(PROBLEM[problem].title)} testId={selectors.reach.channelProblem}>{t(PROBLEM[problem].body)}</Notice>}

      <SaveBar>
        <Button className="min-h-11" type="submit" variant="primary" disabled={!valid || !changed || saving} data-testid={selectors.reach.channelSave}>
          {saving ? t(strings.settings.action.saving) : t(strings.settings.action.save)}
        </Button>
        <Button type="button" variant="outline" disabled={saving} onClick={() => navigate(path(devicePath))}>{t(strings.settings.action.cancel)}</Button>
      </SaveBar>

      <AlertDialog
        testIdPrefix={dialogPrefixes.channelWiden}
        open={confirming}
        title={t(strings.reach.channel.confirmTitle)}
        description={isEmail
          ? t(strings.reach.channel.confirmEmail, { address: trimmed, labels: labelList(labels.filter(isContentLabel)) })
          : t(strings.reach.channel.confirmPush, { labels: labelList(labels.filter(isContentLabel)) })}
        confirmLabel={t(strings.settings.action.save)}
        cancelLabel={t(strings.reach.channel.keepEditing)}
        closeLabel={t(strings.reach.channel.keepEditing)}
        onConfirm={() => void save()}
        onCancel={() => setConfirming(false)}
      />
    </form>
  );
}
