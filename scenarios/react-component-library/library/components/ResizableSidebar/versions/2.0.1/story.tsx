import { ResizableSidebar } from "./ResizableSidebar";

const surface = {
  blockSize: "100%",
  padding: "var(--space-sm, 16px)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
};

export function Default() {
  return (
    <ResizableSidebar
      label="Filters"
      min={200}
      max={380}
      defaultSize={260}
      contentMin={140}
      content={
        <div style={surface}>
          <h2
            style={{
              margin: 0,
              font: "var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans))",
            }}
          >
            Results
          </h2>
          <p style={{ margin: "var(--space-2xs, 8px) 0 0" }}>
            Stays usable at every sidebar width.
          </p>
        </div>
      }
    >
      <div style={surface}>
        <p style={{ margin: 0 }}>Filters</p>
      </div>
    </ResizableSidebar>
  );
}

export function LongContent() {
  return (
    <ResizableSidebar
      label="Filters"
      min={200}
      max={380}
      defaultSize={260}
      contentMin={140}
      content={<div style={surface}>Results</div>}
    >
      <div style={surface}>
        <p style={{ margin: 0 }}>
          A filter label long enough to need wrapping rather than horizontal overflow.
        </p>
      </div>
    </ResizableSidebar>
  );
}
