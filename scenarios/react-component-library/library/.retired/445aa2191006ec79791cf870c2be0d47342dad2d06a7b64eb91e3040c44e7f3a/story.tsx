import { Resizable } from "./Resizable";

const frame = {
  inlineSize: "min(100%, 620px)",
  blockSize: 260,
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface-muted, #f1f5f9)",
  overflow: "hidden",
};

const pane = {
  blockSize: "100%",
  padding: "var(--space-sm, 16px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
};

const Panel = ({ title, body }: { title: string; body: string }) => (
  <div style={pane}>
    <h2 style={{ margin: 0, font: "var(--text-title, inherit)" }}>{title}</h2>
    <p
      style={{
        margin: "var(--space-2xs, 8px) 0 0",
        font: "var(--text-body, inherit)",
      }}
    >
      {body}
    </p>
  </div>
);

export function Default() {
  return (
    <Resizable
      min={160}
      max={380}
      defaultSize={240}
      adjacentMin={140}
      panelName="Inspector"
      className="rcl-story-resizable"
      adjacent={
        <Panel
          title="Workspace"
          body="Takes the remaining space at every size."
        />
      }
    >
      <Panel
        title="Inspector"
        body="Drag the seam, or focus it and use the arrow keys."
      />
    </Resizable>
  );
}

export function AxisBlock() {
  return (
    <div style={frame}>
      <Resizable
        axis="block"
        min={80}
        max={200}
        defaultSize={120}
        adjacentMin={60}
        panelName="Output"
        adjacent={
          <Panel
            title="Terminal"
            body="The block axis is the same implementation."
          />
        }
      >
        <Panel title="Output" body="Resized along the block axis." />
      </Resizable>
    </div>
  );
}

export function Keyboard() {
  return Default();
}

export function Collapsed() {
  return (
    <Resizable
      min={120}
      max={380}
      defaultSize={240}
      collapseBelow={180}
      panelName="Inspector"
      adjacent={
        <Panel title="Workspace" body="The inspector can latch shut." />
      }
    >
      <Panel
        title="Inspector"
        body="Below its threshold this region reports itself collapsed."
      />
    </Resizable>
  );
}
