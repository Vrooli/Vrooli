import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState, type CSSProperties } from "react";
import { VirtualList } from "./VirtualList";

const shell: CSSProperties = { width: "min(100%, 38rem)", minWidth: 0 };
const items = Array.from({ length: 120 }, (_, index) => ({
  id: `event-${index}`,
  title:
    ["Deploy completed", "Review requested", "Backup verified", "Agent checkpoint"][index % 4] ??
    "Activity",
  meta: `${index + 1} min ago`,
  description:
    index % 5 === 0
      ? "A longer row demonstrates measured content without changing reading order."
      : undefined,
}));
const rowStyle: CSSProperties = { display: "grid", gap: 4, minWidth: 0 };

function Rows({ dense = false }: { dense?: boolean }) {
  return (
    <VirtualList
      items={items}
      height={dense ? 280 : 360}
      estimateItemHeight={dense ? 52 : 72}
      stickyIndices={[0]}
      title={dense ? "Command history" : "Recent activity"}
      description={resolveStrings(
        "data-display.virtual-list.description",
        "120 records · only the visible window is mounted",
      )}
      label={resolveStrings("data-display.virtual-list.label", "Activity history")}
      getItemKey={(item) => item.id}
      renderItem={(item, index) => (
        <div style={{ ...rowStyle, gap: dense ? 2 : 4 }}>
          <strong style={{ fontSize: dense ? 12 : 13 }}>{item.title}</strong>
          <span
            style={{
              color: "var(--color-muted-foreground, #64748b)",
              fontSize: 12,
            }}
          >
            {item.description ?? `Workspace signal ${index + 1} is ready for review.`}
          </span>
          <span
            style={{
              color: "var(--color-primary, #2563eb)",
              fontSize: 11,
              fontWeight: 750,
            }}
          >
            {item.meta}
          </span>
        </div>
      )}
    />
  );
}

export function Default() {
  return (
    <div style={shell}>
      <Rows />
    </div>
  );
}
export function Dense() {
  return (
    <div style={shell}>
      <Rows dense />
    </div>
  );
}
export function Empty() {
  return (
    <div style={shell}>
      <VirtualList
        items={[]}
        renderItem={() => null}
        title={resolveStrings("data-display.virtual-list.title", "Recent activity")}
        empty="No activity has arrived for this workspace."
      />
    </div>
  );
}
export function RestoredScroll() {
  const [top] = useState(420);
  return (
    <div style={shell}>
      <VirtualList
        items={items}
        initialScrollTop={top}
        stickyIndices={[0]}
        title={resolveStrings("data-display.virtual-list.title.restored-timeline", "Restored timeline")}
        description={resolveStrings(
          "data-display.virtual-list.description.returning-here-preserves-your-reading-position-l",
          "Returning here preserves your reading position.",
        )}
        label={resolveStrings("data-display.virtual-list.label.restored-activity", "Restored activity")}
        getItemKey={(item) => item.id}
        renderItem={(item, index) => (
          <div style={rowStyle}>
            <strong style={{ fontSize: 13 }}>{item.title}</strong>
            <span
              style={{
                color: "var(--color-muted-foreground, #64748b)",
                fontSize: 12,
              }}
            >
              Restored at activity {index + 1} · {item.meta}
            </span>
          </div>
        )}
      />
    </div>
  );
}
