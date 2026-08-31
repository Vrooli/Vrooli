import { useState } from "react";
import { Tabs } from "./Tabs";

const SECTIONS = [
  "Sessions",
  "Workspace",
  "Input",
  "Output",
  "Shortcuts",
  "Defaults",
  "Links",
];

/** The default rendered anatomy: a short strip that fits its container. */
export function Default() {
  return (
    <Tabs ariaLabel="Sections" items={["Sessions", "Workspace", "Input"]} />
  );
}

/**
 * More tabs than fit: the strip scrolls horizontally in place instead of
 * wrapping into rows and growing the surface around it.
 */
export function Overflowing() {
  return (
    <div style={{ inlineSize: 320 }}>
      <Tabs ariaLabel="Sections" items={SECTIONS} defaultActive="Links" />
    </div>
  );
}

/** Icons and badges sit inside the trigger without displacing the label. */
export function WithIconsAndBadges() {
  return (
    <Tabs
      ariaLabel="Sections"
      items={[
        {
          id: "sessions",
          label: "Sessions",
          badge: 3,
          icon: (
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path d="M4 5h16v14H4z" />
            </svg>
          ),
        },
        { id: "workspace", label: "Workspace" },
      ]}
    />
  );
}

/** A controlled strip renders the panel its caller selects. */
export function ControlledWithPanels() {
  const [active, setActive] = useState("workspace");
  return (
    <Tabs
      ariaLabel="Sections"
      active={active}
      onChange={setActive}
      items={[
        { id: "sessions", label: "Sessions" },
        { id: "workspace", label: "Workspace" },
      ]}
      panels={{
        sessions: <p>Session settings</p>,
        workspace: <p>Workspace settings</p>,
      }}
    />
  );
}

/** A caller may route its own automation selectors onto individual tabs. */
export function CustomItemTestIds() {
  return (
    <Tabs
      ariaLabel="Sections"
      items={["Sessions", "Workspace"]}
      itemTestId={(item) => `settings-tab-${item.toLowerCase()}`}
    />
  );
}

/** Dense secondary navigation without the primary-navigation underline. */
export function Compact() {
  return (
    <Tabs
      ariaLabel="Change filters"
      density="compact"
      items={["All", "From agents", "Staged", "Conflicts"]}
      active="All"
    />
  );
}

/** Mutually-exclusive modes presented as one contained control. */
export function Segmented() {
  return (
    <Tabs
      ariaLabel="View modes"
      density="compact"
      variant="segmented"
      items={["Diff", "Full diff", "Source"]}
      active="Diff"
    />
  );
}
