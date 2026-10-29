import { useState } from "react";
import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
} from "./IntegrationsOverview";

const icon = (path: string) => (
  <svg
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
  >
    <path d={path} />
  </svg>
);
const MIC = "M12 3a3 3 0 0 0-3 3v5a3 3 0 0 0 6 0V6a3 3 0 0 0-3-3zM5 11a7 7 0 0 0 14 0M12 18v3";
const SCREEN = "M3 4h18v12H3zM8 20h8M12 16v4";
const BELL = "M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a1.9 1.9 0 0 0 3.4 0";

const CHECKED_AT = "2026-10-28T23:41:00Z";
const noop = () => undefined;

function voice(overrides: Partial<IntegrationItem> = {}): IntegrationItem {
  return {
    id: "voice",
    name: "Voice",
    subtitle: "Dictation, read-aloud and wake word",
    provider: "Audio Tools",
    icon: icon(MIC),
    status: "ready",
    features: [
      { id: "dictation", label: "Dictate into the composer" },
      { id: "streaming", label: "Live transcription while you talk" },
      { id: "read-aloud", label: "Read replies aloud" },
      { id: "wake-word", label: "Wake word", available: false },
    ],
    facts: [
      { label: "ID", value: "audio-tools", monospace: true },
      { label: "Health", value: "scenario is healthy" },
    ],
    ...overrides,
  };
}

function groups({ bridgeOff = false, grokMissing = false } = {}): IntegrationGroup[] {
  return [
    {
      id: "accounts",
      label: "Accounts",
      width: "half",
      items: [
        {
          id: "openrouter-key",
          name: "OpenRouter",
          subtitle: "API key · verified today",
          initials: "OR",
          status: "ready",
          statusLabel: "Connected",
          selectable: true,
        },
      ],
    },
    {
      id: "agents",
      label: "Coding agents",
      width: "half",
      items: [
        {
          id: "claude",
          name: "Claude Code",
          subtitle: "Version 2.1.312",
          initials: "CC",
          status: "ready",
        },
        grokMissing
          ? {
              id: "grok",
              name: "Grok",
              subtitle: "Not installed",
              initials: "GK",
              status: "missing",
              inlineAction: true,
              action: { label: "Install", onRun: noop },
            }
          : {
              id: "grok",
              name: "Grok",
              subtitle: "Version 0.2.72",
              initials: "GK",
              status: "ready",
            },
      ],
    },
    {
      id: "apps",
      label: "Powered by other apps",
      items: [
        voice(),
        {
          id: "remote",
          name: "Remote terminals",
          subtitle: "Terminals on your other computers",
          provider: "Bridge",
          icon: icon(SCREEN),
          status: bridgeOff ? "off" : "ready",
          problem: {
            title: "Remote terminals are off",
            description: "Your other computers can't open terminals here until Bridge is running.",
          },
          action: { label: "Start Bridge", onRun: noop },
        },
        {
          id: "notifications",
          name: "Notifications",
          subtitle: "Alerts when an agent needs you",
          provider: "Notification Hub",
          icon: icon(BELL),
          status: "ready",
        },
      ],
    },
  ];
}

function Frame({ width, children }: { width: number; children: React.ReactNode }) {
  return <div style={{ width, padding: 16, boxSizing: "border-box" }}>{children}</div>;
}

export const AllReadyPhone = () => (
  <Frame width={390}>
    <IntegrationsOverview
      groups={groups()}
      checkedAt={CHECKED_AT}
      onRefresh={noop}
      locale="en-GB"
    />
  </Frame>
);

export const NeedsAttentionPhone = () => (
  <Frame width={390}>
    <IntegrationsOverview
      groups={groups({ bridgeOff: true, grokMissing: true })}
      checkedAt={CHECKED_AT}
      onRefresh={noop}
      locale="en-GB"
    />
  </Frame>
);

export const DetailPhone = () => {
  const [selected, setSelected] = useState<string | null>("voice");
  return (
    <Frame width={390}>
      <IntegrationsOverview
        groups={groups()}
        checkedAt={CHECKED_AT}
        selectedId={selected}
        onSelectedIdChange={setSelected}
        locale="en-GB"
      />
    </Frame>
  );
};

export const Desktop = () => (
  <Frame width={1000}>
    <IntegrationsOverview
      groups={groups({ bridgeOff: true, grokMissing: true })}
      checkedAt={CHECKED_AT}
      onRefresh={noop}
      locale="en-GB"
    />
  </Frame>
);

export const Loading = () => (
  <Frame width={390}>
    <IntegrationsOverview
      groups={[{ id: "agents", label: "Coding agents", items: [] }]}
      loading
      onRefresh={noop}
    />
  </Frame>
);

export const LoadFailed = () => (
  <Frame width={390}>
    <IntegrationsOverview
      groups={[
        { id: "agents", label: "Coding agents", items: [], emptyText: "Nothing to show yet." },
      ]}
      error="The service did not answer."
      onRefresh={noop}
    />
  </Frame>
);
