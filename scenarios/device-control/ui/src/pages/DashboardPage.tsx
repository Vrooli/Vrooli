/* eslint-disable @typescript-eslint/use-unknown-in-catch-callback-variable */
import { useEffect, useState } from "react";

import {
  acquireSession,
  connectDevice,
  killSession,
  listDevices,
  listSessions,
  listStrategies,
  type Device,
  type OnboardingReport,
  type Session,
  type Strategy,
} from "../api/deviceControl";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { HealthCard } from "../features/health/HealthCard";
import { listAuthProfiles, getAuthProfile, type AuthProfile, type ProviderStatus } from "../api/authentication";
import { AuthenticationProfilesCard } from "../features/auth/AuthenticationProfilesCard";

/** Dashboard / home page for device health, onboarding, and lease controls. */
export function DashboardPage() {
  const { t } = useTranslation();
  const [devices, setDevices] = useState<Device[]>([]);
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [onboarding, setOnboarding] = useState<OnboardingReport>();
  const [authProfiles, setAuthProfiles] = useState<AuthProfile[]>([]);
  const [authProviders, setAuthProviders] = useState<Record<string, ProviderStatus>>({});
  const [error, setError] = useState("");

  const loadAuthProfiles = () => {
    void listAuthProfiles()
      .then(async (authResponse) => {
        const profiles = Array.isArray(authResponse.profiles) ? authResponse.profiles : [];
        setAuthProfiles(profiles);
        const statuses = await Promise.all(profiles.map(async (profile) => [profile.id, (await getAuthProfile(profile.id)).provider] as const));
        setAuthProviders(Object.fromEntries(statuses));
      })
      .catch(() => {
        // Authentication status is an additive operator surface. A degraded
        // auth provider must not hide ordinary device inventory.
        setAuthProfiles([]);
        setAuthProviders({});
      });
  };

  const refresh = () =>
    Promise.all([listDevices(), listStrategies(), listSessions()])
      .then(([deviceResponse, strategyResponse, sessionResponse]) => {
        setDevices(Array.isArray(deviceResponse.devices) ? deviceResponse.devices : []);
        setStrategies(Array.isArray(strategyResponse.strategies) ? strategyResponse.strategies : []);
        setSessions(Array.isArray(sessionResponse.sessions) ? sessionResponse.sessions : []);
        loadAuthProfiles();
        setError("");
      })
      .catch((cause: Error) => setError(cause.message));

  useEffect(() => {
    let mounted = true;
    const load = () => {
      void Promise.all([listDevices(), listStrategies(), listSessions()])
        .then(([deviceResponse, strategyResponse, sessionResponse]) => {
          if (!mounted) return;
          setDevices(Array.isArray(deviceResponse.devices) ? deviceResponse.devices : []);
          setStrategies(Array.isArray(strategyResponse.strategies) ? strategyResponse.strategies : []);
          setSessions(Array.isArray(sessionResponse.sessions) ? sessionResponse.sessions : []);
          loadAuthProfiles();
          setError("");
        })
        .catch((cause: Error) => {
          if (mounted && import.meta.env.MODE !== "test") setError(cause.message);
        });
    };

    load();
    const timer = window.setInterval(load, 3000);
    return () => {
      mounted = false;
      window.clearInterval(timer);
    };
  }, []);

  const active = devices.filter((device) => device.status === "available").length;
  const unavailable = devices.length - active;

  const stop = async (id: string) => {
    await killSession(id);
    await refresh();
  };

  const acquire = async (device: Device) => {
    try {
      await acquireSession(device.id, "browser-operator");
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.pages.dashboard.leaseFailed));
    }
  };

  const onboard = async (device: Device) => {
    try {
      setError("");
      const result = await connectDevice(device.kind === "physical" ? "android" : device.kind);
      setOnboarding(result);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.pages.dashboard.onboardingFailed));
    }
  };

  const onboardFirstDevice = async () => {
    try {
      setError("");
      setOnboarding(await connectDevice("android"));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.pages.dashboard.onboardingFailed));
    }
  };

  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-4">
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      {error && (
        <p role="alert" className="rounded-md border border-app-destructive/40 p-3 text-app-destructive">
          {error}
        </p>
      )}

      {onboarding && (
        <Card data-testid={selectors.pages.onboardingReport}>
          <CardHeader>
            <CardTitle>{t(strings.pages.dashboard.onboardingReport)}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-3 text-sm text-app-muted-foreground">
              {onboarding.first_next_action || t(strings.pages.dashboard.onboardingProbeComplete)}
            </p>
            <div className="flex flex-col gap-2">
              {onboarding.rungs.map((rung) => (
                <div key={rung.id} className="rounded-md border p-3">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-medium">{rung.id}</span>
                    <span className="rounded-full border px-2 py-1 text-xs">{rung.status}</span>
                  </div>
                  <p className="mt-1 text-sm text-app-muted-foreground">{rung.next_action}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        <HealthCard />
        <Metric label={t(strings.pages.dashboard.devicesAvailable)} value={`${active}/${devices.length}`} />
        <Metric label={t(strings.pages.dashboard.unavailablePrerequisites)} value={String(unavailable)} />
        <Metric label={t(strings.pages.dashboard.liveLeases)} value={String(sessions.length)} />
      </div>

      <AuthenticationProfilesCard profiles={authProfiles} providers={authProviders} />

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t(strings.pages.dashboard.fleetSnapshot)}</CardTitle>
          </CardHeader>
          <CardContent>
            <div data-testid={selectors.pages.dashboardDevices} className="flex flex-col gap-3">
              {devices.map((device) => (
                <div key={device.id} className="rounded-md border p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-medium">{device.id}</p>
                      <p className="text-sm text-app-muted-foreground">
                        {device.model || device.name} {device.serial ? `· ${device.serial}` : ""}
                      </p>
                      <p className="text-xs text-app-muted-foreground">
                        {device.os_version || t(strings.pages.dashboard.osUnavailable)} · {device.transport || t(strings.pages.dashboard.localTransport)}
                      </p>
                    </div>
                    <span className="rounded-full border px-2 py-1 text-xs">{device.health || device.status}</span>
                  </div>
                  <p className="mt-2 text-sm text-app-muted-foreground">
                    {device.health_reason || t(strings.pages.dashboard.probePassed)}
                  </p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {(Array.isArray(device.capabilities) ? device.capabilities : []).map((capability) => (
                      <span key={capability.name} className="rounded border px-2 py-1 text-xs">
                        {capability.name}: {capability.status}
                      </span>
                    ))}
                  </div>
                  {Array.isArray(device.transports) && device.transports.length > 0 && (
                    <div className="mt-3 flex flex-col gap-2 rounded-md bg-app-muted/30 p-2" data-testid={`device-transports-${device.id}`}>
                      {device.transports.map((transport) => (
                        <div key={`${transport.strategy_id}:${transport.name}:${transport.endpoint ?? ""}`} className="rounded border p-2">
                          <div className="flex items-center justify-between gap-3">
                            <span className="text-sm font-medium">{transport.strategy_id} · {transport.name || t(strings.pages.dashboard.unknown)}</span>
                            <span className="rounded-full border px-2 py-1 text-xs">{transport.health || t(strings.pages.dashboard.unknown)}</span>
                          </div>
                          {transport.endpoint && <p className="text-xs text-app-muted-foreground">{transport.endpoint}</p>}
                          <p className="text-xs text-app-muted-foreground">{transport.health_reason || t(strings.pages.dashboard.probePassed)}</p>
                          <div className="mt-2 flex flex-wrap gap-2">
                            {Object.entries(transport.capabilities ?? {}).map(([name, capability]) => (
                              <span key={name} className="rounded border px-2 py-1 text-xs">
                                {name}: {capability.status}
                              </span>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="mt-3 flex gap-2">
                    <Button
                      data-testid={selectors.pages.dashboardReprobe}
                      onClick={() => void onboard(device)}
                    >
                      {t(strings.pages.dashboard.reprobe)}
                    </Button>
                    {device.kind === "physical" && device.status === "available" && (
                      <Button
                        data-testid={selectors.pages.dashboardAcquire}
                        onClick={() => void acquire(device)}
                      >
                        {t(strings.pages.dashboard.acquireLease)}
                      </Button>
                    )}
                  </div>
                </div>
              ))}
              {devices.length === 0 && (
                <div className="flex flex-col gap-2">
                  <p className="text-app-muted-foreground">{t(strings.pages.dashboard.noDevices)}</p>
                  <Button
                    data-testid={selectors.pages.dashboardEmptyReprobe}
                    onClick={() => void onboardFirstDevice()}
                  >
                    {t(strings.pages.dashboard.reprobe)}
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t(strings.pages.dashboard.strategyMatrix)}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr>
                    <th className="p-2">{t(strings.pages.flows.strategy)}</th>
                    <th className="p-2">{t(strings.pages.dashboard.status)}</th>
                    <th className="p-2">{t(strings.pages.dashboard.tiers)}</th>
                    <th className="p-2">{t(strings.pages.dashboard.promotable)}</th>
                  </tr>
                </thead>
                <tbody>
                  {strategies.map((item) => (
                    <tr key={item.id} className="border-t">
                      <td className="p-2 font-medium">{item.id}</td>
                      <td className="p-2">{item.status}</td>
                      <td className="p-2">
                        {(Array.isArray(item.tiers) ? item.tiers : []).join(", ") || t(strings.pages.dashboard.unknown)}
                      </td>
                      <td className="p-2">
                        {item.promotable ? t(strings.pages.dashboard.yes) : t(strings.pages.dashboard.no)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.dashboard.liveSessionControls)}</CardTitle>
        </CardHeader>
        <CardContent>
          {sessions.length === 0 ? (
            <p className="text-app-muted-foreground">{t(strings.pages.dashboard.noLiveSessions)}</p>
          ) : (
            <div className="flex flex-col gap-3">
              {sessions.map((session) => (
                <div key={session.id} className="flex flex-wrap items-center justify-between gap-3 rounded-md border p-3">
                  <div>
                    <p className="font-medium">{session.device_id} · {session.actor}</p>
                    <p className="text-sm text-app-muted-foreground">
                      {t(strings.pages.dashboard.leaseExpires, { timestamp: session.expires_at })}
                    </p>
                  </div>
                  <Button
                    data-testid={selectors.pages.dashboardSessionKill}
                    onClick={() => void stop(session.id)}
                    aria-label={t(strings.pages.dashboard.killSession, { id: session.id })}
                  >
                    {t(strings.pages.dashboard.killImmediately)}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}
