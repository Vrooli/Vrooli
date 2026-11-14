import { useQuery } from "@tanstack/react-query";
import { RefreshCw, Smartphone } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyNotice, QueryError } from "../components/QueryNotice";
import { ListGroup, ListItem, ListRow } from "../components/ListRow";
import { SkeletonRows } from "../components/Skeleton";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";
import { BrowserPushCard } from "../features/reach/BrowserPushCard";
import { DeviceAppsPanel } from "../features/reach/DeviceAppsPanel";
import { useRouteNotice } from "../features/reach/routeNotice";
import { isThisBrowser } from "../features/reach/webPush";
import { channelLabel } from "../features/settings/channelLabel";
import { Notice } from "../features/settings/Notice";
import { OwnerGate } from "../features/settings/OwnerGate";
import { devicesQuery } from "../features/settings/ownerQueries";
import { SettingsScreen } from "../features/settings/SettingsScreen";
import { useSettingsPath } from "../features/settings/testContext";

/** Devices and channels: where Chimesail can reach you, and what each may carry. */
export function ReachPage() {
  const { t } = useTranslation();
  return (
    <SettingsScreen title={t(strings.settings.section.reach)} description={t(strings.reach.description)} backToSettings>
      <OwnerGate><ReachContent /></OwnerGate>
    </SettingsScreen>
  );
}

function ReachContent() {
  const { t } = useTranslation();
  const path = useSettingsPath();
  const notice = useRouteNotice();
  // Devices and registrations change elsewhere; never show cached rows as current on arrival.
  const devices = useQuery({ ...devicesQuery, refetchOnWindowFocus: true, refetchOnMount: "always" });
  const rows = devices.data?.devices ?? [];

  return (
    <>
      {notice?.kind === "removed" && (
        <Notice tone="success" testId={selectors.reach.deviceRemoved}>
          {t(notice.uncertain ? strings.reach.devices.removedUncertain : strings.reach.devices.removed, { name: notice.name })}
        </Notice>
      )}
      <BrowserPushCard registeredHere={rows.some(isThisBrowser)} />
      <section aria-labelledby="reach-devices-heading" className="mt-2 flex flex-col gap-2" data-testid={selectors.reach.root}>
        <div className="flex items-center justify-between gap-3">
          <h2 id="reach-devices-heading" className="px-1 text-base font-semibold">{t(strings.reach.devices.title)}</h2>
          <Button className="min-h-11" type="button" variant="outline" size="sm" disabled={devices.isFetching} onClick={() => void devices.refetch()} aria-label={t(strings.reach.devices.refreshLabel)} data-testid={selectors.reach.refreshDevices}>
            <RefreshCw size={16} className={devices.isFetching ? "motion-safe:animate-spin" : undefined} aria-hidden="true" /> {t(strings.reach.devices.refresh)}
          </Button>
        </div>
        {devices.isPending ? <SkeletonRows rows={2} /> : devices.isError && !devices.data ? (
          <QueryError error={devices.error} title={t(strings.reach.devices.loadFailed)} detail={t(strings.reach.devices.loadFailedDetail)} onRetry={() => void devices.refetch()} retrying={devices.isFetching} />
        ) : rows.length === 0 ? (
          <EmptyNotice icon={<Smartphone size={28} />} title={t(strings.reach.devices.emptyTitle)} detail={t(strings.reach.devices.emptyDetail)} />
        ) : (
          <>
            {devices.isError && <Notice tone="warning">{t(strings.reach.devices.stale)}</Notice>}
            <ListGroup aria-label={t(strings.reach.devices.title)}>
              {rows.map((device) => (
                <ListItem key={device.id}>
                  <ListRow
                    to={path(`/settings/reach/devices/${encodeURIComponent(device.id)}`)}
                    data-testid={selectors.reach.device({ id: device.id })}
                    trailing={isThisBrowser(device) ? <span className="mt-0.5 shrink-0 rounded-full bg-app-surface-muted px-2 py-0.5 text-xs">{t(strings.reach.devices.thisBrowser)}</span> : undefined}
                  >
                    <span className="flex flex-col">
                      <span className="text-base font-semibold [overflow-wrap:anywhere]"><bdi dir="auto">{device.name}</bdi></span>
                      <span className="text-sm text-app-muted-foreground">
                        {device.channels.length ? device.channels.map((channel) => channelLabel(channel, t)).join(", ") : t(strings.reach.devices.noChannels)}
                      </span>
                    </span>
                  </ListRow>
                </ListItem>
              ))}
            </ListGroup>
          </>
        )}
      </section>
      <DeviceAppsPanel />
    </>
  );
}
