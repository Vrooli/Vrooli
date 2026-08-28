import { useState } from "react";

import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/2.4.0";
import { HandednessProvider } from "@vrooli/react-component-library/useHandedness/1.1.2";
import { SwipeActions, type SwipeAction } from "./SwipeActions";

const ROWS = [
  { id: "credential-blast-radius", name: "credential blast radius", time: "14:22" },
  { id: "host-cleanup-sweep", name: "host cleanup sweep", time: "09:41" },
];

function actionsFor(id: string, mark: (label: string) => void): SwipeAction[] {
  return [
    {
      id: "unread",
      label: "Unread",
      onSelect: () => {
        mark(`unread:${id}`);
      },
    },
    {
      id: "close",
      label: "Close",
      tone: "destructive",
      onSelect: () => {
        mark(`close:${id}`);
      },
    },
  ];
}

function Row({
  id,
  name,
  time,
  releaseMode,
  onAction,
}: {
  id: string;
  name: string;
  time: string;
  releaseMode?: "rest" | "commit";
  onAction: (label: string) => void;
}) {
  return (
    <SwipeActions
      testId={`patterns.swipe-actions.${id}`}
      actions={actionsFor(id, onAction)}
      releaseMode={releaseMode}
      label={`Actions for ${name}`}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-xs)",
          padding: "var(--space-2xs) var(--space-xs)",
          minBlockSize: "var(--tap-target-min)",
        }}
      >
        <span>{name}</span>
        <span style={{ color: "var(--color-muted-foreground)" }}>{time}</span>
      </div>
    </SwipeActions>
  );
}

function List({ releaseMode }: { releaseMode?: "rest" | "commit" }) {
  const [last, setLast] = useState("");
  return (
    <div style={{ inlineSize: "20rem" }}>
      <div data-testid="patterns.swipe-actions.last-action" role="status">
        {last}
      </div>
      {ROWS.map((row) => (
        <Row key={row.id} {...row} releaseMode={releaseMode} onAction={setLast} />
      ))}
    </div>
  );
}

export function Rows() {
  return <List />;
}

export function AutoCommit() {
  return <List releaseMode="commit" />;
}

/** Reveal follows the reach side, so an end-anchored drawer flips the track. */
export function EndAnchored() {
  return (
    <HandednessProvider value="inline-end">
      <List />
    </HandednessProvider>
  );
}

/**
 * The composition that motivated the component. SidebarShell locks every
 * descendant to `pan-y` so a drag down its nav list still scrolls; the row
 * publishes `data-rcl-pan-x` to claim the inline axis back. Without that hatch
 * the browser cancels the row's drag as a scroll, which is the failure this
 * story exists to keep fixed.
 */
export function InsideDrawer() {
  return (
    <SidebarShell
      mode="overlay"
      mobileOpen
      swipeToClose
      mobileLabel="Sessions"
      closeLabel="Close sessions"
      onMobileClose={() => {}}
      testId="patterns.swipe-actions.drawer"
    >
      <List />
    </SidebarShell>
  );
}
