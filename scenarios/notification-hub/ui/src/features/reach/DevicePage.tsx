import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Mail, Smartphone } from "lucide-react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { recipientsClient } from "../../api/notifications";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { ListGroup, ListItem, ListRow } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { dialogPrefixes, selectors } from "../../consts/selectors";
import { channelLabel } from "../settings/channelLabel";
import { formatList } from "../settings/format";
import { Notice } from "../settings/Notice";
import { OwnerGate } from "../settings/OwnerGate";
import { channelAddressesQuery, devicesQuery, readFresh } from "../settings/ownerQueries";
import { SettingsScreen } from "../settings/SettingsScreen";
import { useSettingsPath } from "../settings/testContext";
import { useSession } from "../session/useSession";
import { useRouteNotice } from "./routeNotice";
import { contentLabel } from "./contentLabel";

/** The channels a device can be reached on are editable only where the Hub holds a setting to edit. */
export const EDITABLE_CHANNELS = ["email", "web_push"] as const;

export function DevicePage() {
  const { t } = useTranslation();
  const { deviceId = "" } = useParams();
  const session = useSession();
  const devices = useQuery({ ...devicesQuery, enabled: Boolean(session.data?.signedIn) });
  const device = devices.data?.devices.find((row) => row.id === deviceId);
  return (
    <SettingsScreen
      title={device?.name ?? t(strings.reach.device.title)}
      description={device?.machineId ? <><span>{t(strings.reach.device.machine)} </span><bdi dir="ltr">{device.machineId}</bdi></> : undefined}
      back={{ to: "/settings/reach", label: t(strings.reach.device.back) }}
    >
      <OwnerGate><DeviceContent deviceId={deviceId} /></OwnerGate>
    </SettingsScreen>
  );
}

function DeviceContent({ deviceId }: { deviceId: string }) {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const path = useSettingsPath();
  const notice = useRouteNotice();
  const devices = useQuery({ ...devicesQuery, refetchOnWindowFocus: true });
  const addresses = useQuery({ ...channelAddressesQuery, refetchOnWindowFocus: true });
  const [confirmRemove, setConfirmRemove] = useState(false);
  const device = devices.data?.devices.find((row) => row.id === deviceId);

  // Removal is confirmed by the Hub's own list no longer holding the device,
  // whatever the removal call itself answered.
  const remove = useMutation({
    mutationFn: async () => {
      let failure: unknown;
      try {
        await recipientsClient.removeDevice({ id: deviceId });
      } catch (error) {
        failure = error;
      }
      const latest = await readFresh(queryClient, devicesQuery);
      if (latest.devices.some((row) => row.id === deviceId)) throw failure instanceof Error ? failure : new Error("Device remains in the latest read");
      return { responseUnknown: failure !== undefined };
    },
    onSuccess: ({ responseUnknown }) => {
      void queryClient.invalidateQueries({ queryKey: channelAddressesQuery.queryKey });
      navigate(path("/settings/reach"), { state: { notice: { kind: "removed", name: device?.name ?? "", uncertain: responseUnknown } } });
    },
  });

  if (devices.isPending) return <SkeletonRows rows={2} />;
  if (devices.isError && !devices.data) {
    return <QueryError error={devices.error} title={t(strings.reach.devices.loadFailed)} detail={t(strings.reach.devices.loadFailedDetail)} onRetry={() => void devices.refetch()} retrying={devices.isFetching} />;
  }
  if (!device) {
    return (
      <EmptyNotice icon={<Smartphone size={28} />} title={t(strings.reach.device.goneTitle)} detail={t(strings.reach.device.goneDetail)}>
        <Link to={path("/settings/reach")} className="mt-2 inline-flex min-h-11 items-center font-medium text-app-primary underline underline-offset-4">{t(strings.reach.device.back)}</Link>
      </EmptyNotice>
    );
  }

  const hasEmail = device.channels.includes("email");
  const channelsHeading = t(strings.reach.device.channels);
  return (
    <>
      {notice?.kind === "channelSaved" && (
        <Notice tone="success" testId={selectors.reach.channelSaved}>
          {t(notice.uncertain ? strings.reach.channel.savedUncertain : strings.reach.channel.saved, { channel: channelLabel(notice.channel, t) })}
        </Notice>
      )}
      <section aria-labelledby="device-channels-heading" className="flex flex-col gap-2">
        <h2 id="device-channels-heading" className="px-1 text-base font-semibold">{channelsHeading}</h2>
        {addresses.isError && (
          <Notice tone="warning" title={t(strings.reach.device.addressesFailed)}>
            <p>{addresses.data ? t(strings.reach.device.addressesStale) : t(strings.reach.device.addressesFailedDetail)}</p>
            <Button type="button" variant="outline" size="sm" className="mt-2 self-start" disabled={addresses.isFetching} onClick={() => void addresses.refetch()}>{t(strings.common.retry)}</Button>
          </Notice>
        )}
        {device.channels.length === 0 ? (
          <p className="px-1 text-sm text-app-muted-foreground">{t(strings.reach.device.noChannels)}</p>
        ) : (
          <ListGroup aria-label={channelsHeading}>
            {device.channels.map((channel) => {
              const address = addresses.data?.addresses.find((row) => row.deviceId === device.id && row.channel === channel);
              const editable = (EDITABLE_CHANNELS as readonly string[]).includes(channel) && address !== undefined && !addresses.isError;
              const body = (
                <span className="flex flex-col">
                  <span className="text-base font-semibold">{channelLabel(channel, t)}</span>
                  {address ? (
                    <>
                      <span className="text-sm text-app-muted-foreground [overflow-wrap:anywhere]">
                        {channel === "web_push" ? t(strings.reach.channel.thisBrowserPrivate) : <bdi dir="ltr">{address.address}</bdi>}
                      </span>
                      <span className="text-sm text-app-muted-foreground">
                        {t(strings.reach.channel.fullContent, { labels: address.approvedLabels.length ? formatList(address.approvedLabels.map((label) => contentLabel(label, t)), lang) : t(strings.reach.channel.none) })}
                      </span>
                    </>
                  ) : (
                    <span className="text-sm text-app-muted-foreground">{t(strings.reach.channel.unreadable)}</span>
                  )}
                </span>
              );
              return (
                <ListItem key={channel} data-testid={selectors.reach.channel({ channel })}>
                  {editable
                    ? <ListRow to={path(`/settings/reach/devices/${encodeURIComponent(device.id)}/channels/${encodeURIComponent(channel)}`)} data-testid={selectors.reach.channelEdit({ channel })}>{body}</ListRow>
                    : <div className="min-h-14 px-4 py-3">{body}</div>}
                </ListItem>
              );
            })}
          </ListGroup>
        )}
        {!hasEmail && (
          <div>
            <Link
              to={path(`/settings/reach/devices/${encodeURIComponent(device.id)}/channels/email`)}
              data-testid={selectors.reach.addEmail}
              className="inline-flex min-h-11 items-center gap-2 rounded-full border border-app-border bg-app-surface px-4 text-sm font-medium hover:bg-app-surface-muted focus-visible:outline focus-visible:outline-2"
            >
              <Mail size={16} aria-hidden="true" /> {t(strings.reach.device.addEmail)}
            </Link>
          </div>
        )}
      </section>

      <section aria-labelledby="device-remove-heading" className="flex flex-col gap-2 border-t border-app-border pt-5">
        <h2 id="device-remove-heading" className="px-1 text-base font-semibold">{t(strings.reach.device.removeHeading)}</h2>
        <p className="px-1 text-sm text-app-muted-foreground">{t(strings.reach.device.removeBody)}</p>
        {remove.isError && <Notice tone="danger" title={t(strings.reach.device.removeFailed)}>{t(strings.reach.device.removeFailedDetail)}</Notice>}
        <div>
          <Button className="min-h-11" type="button" variant="danger" disabled={remove.isPending || devices.isFetching} onClick={() => setConfirmRemove(true)} data-testid={selectors.reach.removeDevice}>
            {remove.isPending ? t(strings.reach.device.removing) : t(strings.reach.device.remove)}
          </Button>
        </div>
      </section>

      <AlertDialog
        testIdPrefix={dialogPrefixes.deviceRemoval}
        open={confirmRemove}
        title={t(strings.reach.device.confirmTitle, { name: device.name })}
        description={t(strings.reach.device.confirmBody)}
        destructive
        confirmLabel={t(strings.reach.device.remove)}
        cancelLabel={t(strings.settings.action.cancel)}
        closeLabel={t(strings.settings.action.cancel)}
        onConfirm={() => { setConfirmRemove(false); remove.mutate(); }}
        onCancel={() => setConfirmRemove(false)}
      />
    </>
  );
}
