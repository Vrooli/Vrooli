import {
  AppNotificationSettings,
  type NotificationPreferenceControlGroup,
  NotificationPreferencesPanel,
  type AppNotificationClient,
  type AppNotificationState,
  type AppNotificationSupport,
  type NotificationCategoryView,
} from "./AppNotificationSettings";

const offState: AppNotificationState = {
  scenario: "web-console",
  displayName: "Web Console",
  vapidPublicKey: "preview-key",
  eligible: true,
  ineligibleReason: "",
  publicUrl: "https://web-console.example.test",
  subscription: null,
  devices: [
    {
      group: "iphone-group",
      label: "iPhone",
      lastUsed: "2026-10-28T10:00:00Z",
      bestMatch: true,
      apps: ["Agent Manager"],
    },
    {
      group: "mac-group",
      label: "Mac · Chrome",
      lastUsed: "2026-10-25T10:00:00Z",
      bestMatch: false,
      apps: [],
    },
  ],
};

const pairedState: AppNotificationState = {
  ...offState,
  subscription: {
    id: "s1",
    deviceGroup: "iphone-group",
    deviceLabel: "iPhone",
    permission: "granted",
    confirmedAt: "2026-10-28T10:00:00Z",
    healthy: true,
    unhealthyReason: "",
    standalone: false,
  },
};

const categories: NotificationCategoryView[] = [
  {
    category: "session.turn_completed",
    displayName: "Session finished its turn",
    description: "A session is idle and waiting for you.",
    enabled: true,
    minUrgency: "normal",
  },
  {
    category: "session.approval_needed",
    displayName: "Approval needed",
    description: "A session is blocked on your decision.",
    enabled: true,
    minUrgency: "high",
  },
  {
    category: "session.digest",
    displayName: "Daily digest",
    description: "A summary of yesterday's sessions.",
    enabled: false,
    minUrgency: "low",
  },
];

const previewSubscription = { endpoint: "preview://push" } as unknown as PushSubscription;

/** A deterministic client: no service worker, push or network in Preview. */
function previewClient(
  state: AppNotificationState,
  support: AppNotificationSupport = "supported",
): AppNotificationClient {
  let current = state;
  return {
    support: () => support,
    getState: () => Promise.resolve(current),
    prepareSubscription: () => Promise.resolve(previewSubscription),
    enable: (_subscription, pairing) => {
      current = {
        ...current,
        subscription: {
          ...(pairedState.subscription as NonNullable<AppNotificationState["subscription"]>),
          deviceGroup: "group" in pairing ? pairing.group : "",
          deviceLabel:
            "group" in pairing
              ? (current.devices.find((d) => d.group === pairing.group)?.label ?? "")
              : "",
          standalone: "standalone" in pairing,
        },
      };
      return Promise.resolve(current);
    },
    disable: () => {
      current = { ...current, subscription: null };
      return Promise.resolve();
    },
    confirm: () => Promise.resolve(true),
    fetchCategories: () => Promise.resolve(categories),
    saveCategory: () => Promise.resolve(),
  };
}

const offClient = previewClient(offState);
const pairedClient = previewClient(pairedState);
const unhealthyClient = previewClient({
  ...pairedState,
  subscription: {
    ...(pairedState.subscription as NonNullable<AppNotificationState["subscription"]>),
    healthy: false,
    unhealthyReason: "Not opened for 30 days; using Chimesail until it is opened again",
  },
});
const ineligibleClient = previewClient({
  ...offState,
  eligible: false,
  ineligibleReason: "web-console only has a temporary address.",
});
const installFirstClient = previewClient(offState, "install-first");
const mutedOnDeviceClient: AppNotificationClient = {
  ...previewClient(pairedState),
  permission: () => "default",
};
const choosingClient = previewClient(offState);

export function OffWithCategories() {
  return <AppNotificationSettings client={offClient} />;
}

export function PairedWithDevice() {
  return <AppNotificationSettings client={pairedClient} showCategories={false} />;
}

export function UsingChimesailInstead() {
  return <AppNotificationSettings client={unhealthyClient} showCategories={false} />;
}

export function Ineligible() {
  return <AppNotificationSettings client={ineligibleClient} showCategories={false} />;
}

export function InstallFirst() {
  return <AppNotificationSettings client={installFirstClient} showCategories={false} />;
}

export function ChoosingDevice() {
  return <AppNotificationSettings client={choosingClient} showCategories={false} />;
}

export function MutedOnDevice() {
  return <AppNotificationSettings client={mutedOnDeviceClient} showCategories={false} />;
}

const preferenceGroups: NotificationPreferenceControlGroup[] = [
  {
    id: "agent-manager",
    label: "Agent Manager",
    rows: [
      {
        id: "agent-manager-run-failed",
        sourceScenario: "agent-manager",
        category: "run-failed",
        label: "Run failed",
        description: "A coding-agent run ended in error.",
        enabled: true,
        minUrgency: "high",
        onEnabledChange: () => undefined,
        onMinUrgencyChange: () => undefined,
      },
      {
        id: "agent-manager-run-completed",
        sourceScenario: "agent-manager",
        category: "run-completed",
        label: "Run completed",
        description: "A coding-agent run finished successfully.",
        enabled: false,
        minUrgency: "low",
        onEnabledChange: () => undefined,
        onMinUrgencyChange: () => undefined,
      },
    ],
  },
  {
    id: "git-control-tower",
    label: "Git Control Tower",
    rows: [
      {
        id: "git-control-tower-all",
        sourceScenario: "git-control-tower",
        category: "",
        label: "All notifications",
        description: "",
        enabled: true,
        minUrgency: "normal",
        onEnabledChange: () => undefined,
        onMinUrgencyChange: () => undefined,
      },
    ],
  },
];

export function NotificationPreferences() {
  return <NotificationPreferencesPanel groups={preferenceGroups} />;
}
