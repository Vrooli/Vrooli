import { ScrollableTabs } from "./ScrollableTabs";

/** The delegate renders the tab strip it forwards to. */
export function Default() {
  return (
    <ScrollableTabs
      ariaLabel="Sections"
      items={["Sessions", "Workspace", "Input"]}
    />
  );
}

/** Overflow is carried by the strip itself, not by a wrapper that clips it. */
export function Overflowing() {
  return (
    <div style={{ inlineSize: 320 }}>
      <ScrollableTabs
        ariaLabel="Sections"
        items={[
          "Sessions",
          "Workspace",
          "Input",
          "Output",
          "Shortcuts",
          "Defaults",
          "Links",
        ]}
        defaultActive="Links"
      />
    </div>
  );
}
