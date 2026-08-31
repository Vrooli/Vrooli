import { SplitPane } from "./SplitPane";

const pane = {
  blockSize: "100%",
  padding: "var(--space-sm, 16px)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
};

const Pane = ({ title, body }: { title: string; body: string }) => (
  <div style={pane}>
    <h2
      style={{
        margin: 0,
        font: "var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans))",
      }}
    >
      {title}
    </h2>
    <p
      style={{
        margin: "var(--space-2xs, 8px) 0 0",
        font: "var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans))",
      }}
    >
      {body}
    </p>
  </div>
);

export function Default() {
  return (
    <SplitPane
      min={160}
      max={420}
      defaultSize={280}
      secondaryMin={160}
      primaryName="Needs review"
      primary={<Pane title="Needs review" body="Three items are waiting for an owner." />}
      secondary={<Pane title="Ready to ship" body="The next handoff is prepared for release." />}
    />
  );
}

export function Vertical() {
  return (
    <SplitPane
      orientation="vertical"
      min={80}
      max={220}
      defaultSize={140}
      secondaryMin={60}
      primaryName="Editor"
      primary={<Pane title="Editor" body="Stacked panes share one splitter implementation." />}
      secondary={<Pane title="Console" body="The splitter is horizontal here." />}
    />
  );
}

export function Keyboard() {
  return Default();
}
