/**
 * A small in-memory Hub for the settings tests: the owner session, devices,
 * channel addresses, quiet windows, the escalation chain, the notification
 * catalog and the browser's push state, behind the same client surfaces the UI
 * calls. It applies writes the way the real Hub does (one quiet window per
 * weekday, the whole escalation order replaced at once) so tests assert what
 * the screens do, not how they were mocked.
 *
 * Faults are queued per method. `reject` fails the call without applying it;
 * `lose` applies the call and then fails, the way a dropped response does.
 *
 * Wire it with `vi.mock` factories (see `installFakeHub`'s doc) and call
 * `hub.reset()` in `beforeEach`.
 */
import { Code, ConnectError } from "@connectrpc/connect";
import { vi } from "vitest";

export interface FakeDevice { id: string; name: string; machineId: string; channels: string[] }
export interface FakeAddress { id: string; deviceId: string; channel: string; address: string; approvedLabels: string[] }
export interface FakeWindow { id: string; weekday: number; start: string; end: string; timezone: string; criticalOverride: boolean }
export interface FakeSource {
  source_scenario: string;
  display_name: string;
  icon: string;
  deep_link_base: string;
  declared: boolean;
  categories: { category: string; display_name: string; description: string; declared: boolean; default_enabled: boolean; default_min_urgency: string }[];
}
export interface FakePreference { source_scenario: string; category: string; enabled: boolean; min_urgency: string; channel_override: string }
export interface FakeBrowser {
  supported: boolean;
  permission: "granted" | "denied" | "default" | "unsupported";
  controlled: boolean;
  runningVersion: string | null;
  availableVersion: string | null;
  stale: boolean;
  subscribed: boolean;
  endpointHost: string;
}
export interface FakeDeviceApps { group: string; label: string; lastUsed: string; apps: { subscriptionId: string; scenario: string; displayName: string; healthy: boolean; status: string; confirmedAt: string }[] }

type Fault = "reject" | "lose";

interface State {
  session: { signedIn: boolean; subject: string; email: string };
  devices: FakeDevice[];
  addresses: FakeAddress[];
  windows: FakeWindow[];
  chain: string[];
  vapidPublicKey: string;
  sources: FakeSource[];
  preferences: FakePreference[];
  browser: FakeBrowser;
  deviceApps: FakeDeviceApps[];
}

const OWNER = { signedIn: true, subject: "owner-1", email: "owner@example.test" };

function freshState(): State {
  return {
    session: { ...OWNER },
    devices: [],
    addresses: [],
    windows: [],
    chain: [],
    vapidPublicKey: "AQAB",
    sources: [],
    preferences: [],
    browser: { supported: true, permission: "default", controlled: true, runningVersion: "1", availableVersion: "1", stale: false, subscribed: false, endpointHost: "" },
    deviceApps: [],
  };
}

const unavailable = () => new ConnectError("unavailable", Code.Unavailable);

class FakeHub {
  state: State = freshState();
  private queued = new Map<string, (Fault | undefined)[]>();
  private persistent = new Map<string, Fault>();
  private nextId = 1;
  /** Who the next successful sign-in signs in as. */
  loginSubject = OWNER.subject;

  reset(patch: Partial<State> = {}): void {
    this.state = { ...freshState(), ...patch };
    this.queued.clear();
    this.persistent.clear();
    this.nextId = 1;
    this.loginSubject = OWNER.subject;
    for (const mock of this.mocks) mock.mockClear();
  }

  /** Make `times` calls to `method` fail, after letting `skip` calls through. */
  failNext(method: string, fault: Fault = "reject", times = 1, skip = 0): void {
    this.queued.set(method, [
      ...(this.queued.get(method) ?? []),
      ...Array.from({ length: skip }, () => undefined),
      ...Array.from({ length: times }, () => fault),
    ]);
  }

  /** Make every call to `method` fail until `reset`. */
  failAlways(method: string, fault: Fault = "reject"): void {
    this.persistent.set(method, fault);
  }

  signOut(): void {
    this.state.session = { signedIn: false, subject: "", email: "" };
  }

  private mocks: ReturnType<typeof vi.fn>[] = [];

  private method<F extends (...args: never[]) => unknown>(name: string, apply: F) {
    const mock = vi.fn(async (...args: Parameters<F>): Promise<Awaited<ReturnType<F>>> => {
      const fault = this.queued.get(name)?.shift() ?? this.persistent.get(name);
      if (fault === "reject") throw unavailable();
      const result = (await apply(...args)) as Awaited<ReturnType<F>>;
      if (fault === "lose") throw unavailable();
      return result;
    });
    this.mocks.push(mock);
    return mock;
  }

  readonly identityClient = {
    getSession: this.method("getSession", () => ({ ...this.state.session })),
    login: this.method("login", () => { this.state.session = { signedIn: true, subject: this.loginSubject, email: `${this.loginSubject}@example.test` }; return {}; }),
    logout: this.method("logout", () => { this.signOut(); return {}; }),
  };

  readonly recipientsClient = {
    listDevices: this.method("listDevices", () => ({ devices: this.state.devices.map((device) => ({ ...device, channels: [...device.channels] })), vapidPublicKey: this.state.vapidPublicKey })),
    removeDevice: this.method("removeDevice", ({ id }: { id: string }) => {
      this.state.devices = this.state.devices.filter((device) => device.id !== id);
      this.state.addresses = this.state.addresses.filter((address) => address.deviceId !== id);
      return {};
    }),
    listChannelAddresses: this.method("listChannelAddresses", () => ({ addresses: this.state.addresses.map((address) => ({ ...address, approvedLabels: [...address.approvedLabels] })) })),
    createChannelAddress: this.method("createChannelAddress", (request: FakeAddress) => {
      if (this.state.addresses.some((address) => address.id === request.id)) return request;
      if (this.state.addresses.some((address) => address.deviceId === request.deviceId && address.channel === request.channel)) throw new ConnectError("exists", Code.AlreadyExists);
      this.state.addresses.push({ ...request });
      const device = this.state.devices.find((row) => row.id === request.deviceId);
      if (device && !device.channels.includes(request.channel)) device.channels.push(request.channel);
      return request;
    }),
    updateChannelAddress: this.method("updateChannelAddress", ({ id, address, approvedLabels }: { id: string; address: string; approvedLabels: string[] }) => {
      const row = this.state.addresses.find((candidate) => candidate.id === id);
      if (!row) throw new ConnectError("missing", Code.NotFound);
      row.address = address;
      row.approvedLabels = [...approvedLabels];
      return row;
    }),
    listQuietWindows: this.method("listQuietWindows", () => ({ windows: this.state.windows.map((window) => ({ ...window })) })),
    setQuietWindow: this.method("setQuietWindow", (request: Omit<FakeWindow, "id">) => {
      if (request.end === request.start) throw new ConnectError("equal", Code.InvalidArgument);
      this.state.windows = this.state.windows.filter((window) => window.weekday !== request.weekday);
      const window = { id: `window-${this.nextId++}`, ...request };
      this.state.windows.push(window);
      return window;
    }),
    deleteQuietWindow: this.method("deleteQuietWindow", ({ id }: { id: string }) => {
      this.state.windows = this.state.windows.filter((window) => window.id !== id);
      return {};
    }),
    getEscalationChain: this.method("getEscalationChain", () => ({ steps: this.state.chain.map((channel, ordinal) => ({ ordinal, channel })) })),
    setEscalationChain: this.method("setEscalationChain", ({ channels }: { channels: string[] }) => {
      this.state.chain = [...new Set(channels)];
      return { steps: this.state.chain.map((channel, ordinal) => ({ ordinal, channel })) };
    }),
  };

  readonly appDeliveryClient = {
    listDeviceApps: this.method("listDeviceApps", () => ({ devices: this.state.deviceApps })),
    removeDeviceApp: this.method("removeDeviceApp", ({ subscriptionId }: { subscriptionId: string }) => {
      for (const device of this.state.deviceApps) device.apps = device.apps.filter((app) => app.subscriptionId !== subscriptionId);
      return {};
    }),
  };

  readonly registerBrowserPushSubscription = this.method("registerBrowserPushSubscription", () => {
    this.state.devices.push({ id: `device-${this.nextId++}`, name: window.location.origin, machineId: "", channels: ["web_push"] });
    this.state.browser.subscribed = true;
    return {};
  });

  readonly fetchNotificationCatalog = this.method("fetchNotificationCatalog", () => ({
    sources: this.state.sources,
    preferences: this.state.preferences.map((pref) => ({ ...pref })),
  }));

  readonly updateNotificationPreference = this.method("updateNotificationPreference", (preference: FakePreference) => {
    this.state.preferences = [
      ...this.state.preferences.filter((pref) => !(pref.source_scenario === preference.source_scenario && pref.category === preference.category)),
      { ...preference },
    ];
    return preference;
  });

  readonly readConnectionStatus = this.method("readConnectionStatus", () => ({ ...this.state.browser }));
  readonly forceServiceWorkerUpdate = this.method("forceServiceWorkerUpdate", () => { this.state.browser.stale = false; });
}

export const hub = new FakeHub();

/** A device with sensible defaults. */
export function device(over: Partial<FakeDevice> & { id: string }): FakeDevice {
  return { name: "Work laptop", machineId: "", channels: [], ...over };
}

/** A quiet window with sensible defaults. */
export function quietWindow(weekday: number, over: Partial<FakeWindow> = {}): FakeWindow {
  return { id: `stored-${weekday}`, weekday, start: "12:00", end: "13:00", timezone: "Europe/London", criticalOverride: false, ...over };
}

/**
 * `vi.mock` factories for the three client modules the settings screens use.
 * Each test file wires them the same way, with a dynamic import so the factory
 * runs after the hub exists:
 *
 *     vi.mock("../../api/notifications", async () => (await import("./mocks/fakeHub")).notificationsModule());
 */
export const notificationsModule = () => ({
  identityClient: hub.identityClient,
  recipientsClient: hub.recipientsClient,
  appDeliveryClient: hub.appDeliveryClient,
  registerBrowserPushSubscription: hub.registerBrowserPushSubscription,
});

export const preferencesModule = async () => ({
  ...(await vi.importActual<typeof import("../../../api/preferences")>("../../../api/preferences")),
  fetchNotificationCatalog: hub.fetchNotificationCatalog,
  updateNotificationPreference: hub.updateNotificationPreference,
});

export const serviceWorkerModule = () => ({
  readConnectionStatus: hub.readConnectionStatus,
  forceServiceWorkerUpdate: hub.forceServiceWorkerUpdate,
});
