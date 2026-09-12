import { Bot } from "lucide-react";

import { PageHeader, type PageHeaderProps } from "./PageHeader";

type Args = {
  level?: 1 | 2;
  density?: "comfortable" | "compact";
  eyebrow?: boolean;
  leading?: boolean;
  actions?: boolean;
  longTitle?: boolean;
};

const rigStyle = {
  inlineSize: "min(100%, 720px)",
  padding: "var(--space-sm)",
  background: "var(--color-background)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
};

const buttonStyle = {
  minBlockSize: "var(--control-height-sm)",
  paddingInline: "var(--space-sm)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  font: "var(--text-label)",
};

function Rig({ args }: { args: Args }) {
  const props: PageHeaderProps = {
    title: args.longTitle
      ? "A deliberately long page title that has to wrap on a phone without pushing the actions off the screen"
      : "Agents",
    description: "Every agent this install operates: where each is reachable and what it may do.",
    level: args.level,
    density: args.density,
    eyebrow: args.eyebrow ? <a href="#agents">Agents</a> : undefined,
    leading: args.leading ? <Bot aria-hidden="true" /> : undefined,
    actions: args.actions ? (
      <>
        <button type="button" style={buttonStyle}>
          Refresh
        </button>
        <button
          type="button"
          style={{
            ...buttonStyle,
            background: "var(--color-primary)",
            color: "var(--color-primary-foreground)",
            borderColor: "var(--color-primary)",
          }}
        >
          New agent
        </button>
      </>
    ) : undefined,
    testId: "story-page-header",
    headingId: "story-page-heading",
  };
  return (
    <div style={rigStyle}>
      <PageHeader {...props} />
    </div>
  );
}

export function Default({ args }: { args: Args }) {
  return <Rig args={{ level: 1, density: "comfortable", actions: true, ...args }} />;
}

export function LevelAxis({ args }: { args: Args }) {
  return <Rig args={{ density: "comfortable", ...args }} />;
}

export function DensityAxis({ args }: { args: Args }) {
  return <Rig args={{ level: 1, ...args }} />;
}

export function FullAnatomy({ args }: { args: Args }) {
  return (
    <Rig
      args={{
        level: 1,
        density: "comfortable",
        eyebrow: true,
        leading: true,
        actions: true,
        ...args,
      }}
    />
  );
}

export function LongTitle({ args }: { args: Args }) {
  return (
    <Rig args={{ level: 1, density: "comfortable", actions: true, longTitle: true, ...args }} />
  );
}
