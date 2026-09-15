import { Bot, Home, MessagesSquare, Radio, Users } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";

import {
  AppShell,
  type AppShellDensity,
  type AppShellMainMode,
  type AppShellMobileNav,
  type AppShellNavItem,
} from "./AppShell";

/**
 * Specimens for AppShell 2.0.0. One rig, driven by the story's args, so every
 * frame is the same shell answering a different question. The rig is sized to
 * the harness rather than the viewport: `100dvh` in the component becomes the
 * rig's block size here, which is what lets a phone-width frame and a desktop
 * frame both fit a capture.
 */

type Args = {
  density?: AppShellDensity;
  mobileNav?: AppShellMobileNav;
  mainMode?: AppShellMainMode;
  header?: boolean;
  badges?: boolean;
  longLabels?: boolean;
};

const rigStyle = {
  inlineSize: "min(100%, 960px)",
  blockSize: 520,
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  overflow: "hidden",
  position: "relative" as const,
  "--rcl-app-shell-block-size": "100%",
} as const;

function items(args: Args): AppShellNavItem[] {
  const long = args.longLabels;
  return [
    {
      id: "home",
      label: long ? "Home and everything waiting on you" : "Home",
      shortLabel: "Home",
      href: "#home",
      icon: <Home aria-hidden="true" />,
      current: true,
      testId: "story-nav-home",
      badge: args.badges ? { value: 3, tone: "warning", label: "3 waiting" } : undefined,
    },
    {
      id: "threads",
      label: long ? "Conversations across every channel" : "Threads",
      shortLabel: "Threads",
      href: "#threads",
      icon: <MessagesSquare aria-hidden="true" />,
      testId: "story-nav-threads",
      badge: args.badges ? { value: 12, tone: "neutral", label: "12 unread" } : undefined,
    },
    {
      id: "agents",
      label: "Agents",
      href: "#agents",
      icon: <Bot aria-hidden="true" />,
      testId: "story-nav-agents",
    },
    {
      id: "people",
      label: "People",
      href: "#people",
      icon: <Users aria-hidden="true" />,
      testId: "story-nav-people",
    },
    {
      id: "channels",
      label: long ? "Channels and the setup each one takes" : "Channels",
      shortLabel: "Channels",
      href: "#channels",
      icon: <Radio aria-hidden="true" />,
      testId: "story-nav-channels",
      disabled: long,
    },
  ];
}

function Page({ fill }: { fill?: boolean }) {
  const body: ReactNode = (
    <>
      <h1 style={{ margin: 0, font: "var(--text-title)" }}>Home</h1>
      <p style={{ margin: 0, maxInlineSize: "60ch", color: "var(--color-muted-foreground)" }}>
        Routed content. The shell pins its own chrome and this pane scrolls; nothing here knows how
        wide the navigation is.
      </p>
    </>
  );
  if (fill) {
    return (
      <div style={{ display: "flex", flex: 1, flexDirection: "column", minBlockSize: 0 }}>
        <div
          style={{
            flex: "none",
            padding: "var(--space-sm)",
            borderBlockEnd: "var(--border-hairline) solid var(--color-border)",
          }}
        >
          {body}
        </div>
        <div style={{ flex: 1, overflow: "auto", padding: "var(--space-sm)" }}>
          {Array.from({ length: 30 }, (_, index) => (
            <p key={index} style={{ margin: "0 0 var(--space-2xs)" }}>
              Row {index + 1} of a list that scrolls inside the pane the shell handed it.
            </p>
          ))}
        </div>
        <div
          style={{
            flex: "none",
            padding: "var(--space-sm)",
            borderBlockStart: "var(--border-hairline) solid var(--color-border)",
          }}
        >
          A composer pinned by the page, not the shell.
        </div>
      </div>
    );
  }
  return <div style={{ display: "grid", gap: "var(--space-sm)" }}>{body}</div>;
}

function Rig({ args }: { args: Args; log?: (name: string, ...eventArgs: unknown[]) => void }) {
  return (
    <div style={rigStyle as CSSProperties} data-testid="story-rig">
      <AppShell
        brand="Switchboard"
        brandMark={<Radio aria-hidden="true" />}
        brandHref="#home"
        items={items(args)}
        density={args.density}
        mobileNav={args.mobileNav}
        mainMode={args.mainMode}
        header={
          args.header ? (
            <span style={{ font: "var(--text-label)" }}>Workspace: production</span>
          ) : undefined
        }
        utility={
          <span
            style={{
              display: "block",
              padding: "var(--space-2xs)",
              font: "var(--text-caption)",
              color: "var(--color-muted-foreground)",
            }}
          >
            Signed in as MH
          </span>
        }
        onNavigate={() => undefined}
        testId="story-shell"
      >
        <Page fill={args.mainMode === "fill"} />
      </AppShell>
    </div>
  );
}

export function Default({ args }: { args: Args }) {
  return <Rig args={{ density: "sidebar", mobileNav: "tabs", mainMode: "scroll", ...args }} />;
}

export function DensityAxis({ args }: { args: Args }) {
  return <Rig args={{ mobileNav: "tabs", mainMode: "scroll", ...args }} />;
}

export function MobileNavAxis({ args }: { args: Args }) {
  return <Rig args={{ density: "sidebar", mainMode: "scroll", ...args }} />;
}

export function MainModeAxis({ args }: { args: Args }) {
  return <Rig args={{ density: "sidebar", mobileNav: "tabs", ...args }} />;
}

export function WithHeader({ args }: { args: Args }) {
  return (
    <Rig
      args={{ density: "sidebar", mobileNav: "tabs", mainMode: "scroll", header: true, ...args }}
    />
  );
}

export function Badges({ args }: { args: Args }) {
  return (
    <Rig args={{ density: "rail", mobileNav: "tabs", mainMode: "scroll", badges: true, ...args }} />
  );
}

export function LongLabels({ args }: { args: Args }) {
  return (
    <Rig
      args={{
        density: "sidebar",
        mobileNav: "tabs",
        mainMode: "scroll",
        longLabels: true,
        ...args,
      }}
    />
  );
}

export function RightToLeft({ args }: { args: Args }) {
  return (
    <div dir="rtl">
      <Rig args={{ density: "sidebar", mobileNav: "tabs", mainMode: "scroll", ...args }} />
    </div>
  );
}
