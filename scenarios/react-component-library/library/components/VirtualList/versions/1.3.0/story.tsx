import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useRef, useState, type CSSProperties } from "react";
import { VirtualList, type VirtualListHandle, type VirtualListPosition } from "./VirtualList";

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
  const libraryStrings = useStrings();
  return (
    <VirtualList
      items={items}
      height={dense ? 280 : 360}
      estimateItemHeight={dense ? 52 : 72}
      stickyIndices={[0]}
      title={dense ? "Command history" : "Recent activity"}
      description={libraryStrings(
        "data-display.virtual-list.description",
        "120 records · only the visible window is mounted",
      )}
      label={libraryStrings("data-display.virtual-list.label", "Activity history")}
      getItemKey={(item) => item.id}
      renderItem={(item, index) => (
        <div style={{ ...rowStyle, gap: dense ? 2 : 4 }}>
          <strong style={{ fontSize: dense ? 12 : 13 }}>{item.title}</strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              fontSize: 12,
            }}
          >
            {item.description ?? `Workspace signal ${index + 1} is ready for review.`}
          </span>
          <span
            style={{
              color: "var(--color-primary)",
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
  const libraryStrings = useStrings();
  return (
    <div style={shell}>
      <VirtualList
        items={[]}
        renderItem={() => null}
        title={libraryStrings("data-display.virtual-list.title", "Recent activity")}
        empty="No activity has arrived for this workspace."
      />
    </div>
  );
}
export function RestoredScroll() {
  const libraryStrings = useStrings();
  const [top] = useState(420);
  return (
    <div style={shell}>
      <VirtualList
        items={items}
        initialScrollTop={top}
        stickyIndices={[0]}
        title={libraryStrings(
          "data-display.virtual-list.title.restored-timeline",
          "Restored timeline",
        )}
        description={libraryStrings(
          "data-display.virtual-list.description.returning-here-preserves-your-reading-position-l",
          "Returning here preserves your reading position.",
        )}
        label={libraryStrings(
          "data-display.virtual-list.label.restored-activity",
          "Restored activity",
        )}
        getItemKey={(item) => item.id}
        renderItem={(item, index) => (
          <div style={rowStyle}>
            <strong style={{ fontSize: 13 }}>{item.title}</strong>
            <span
              style={{
                color: "var(--color-muted-foreground)",
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

export function Growing() {
  const libraryStrings = useStrings();
  return (
    <div style={shell}>
      <VirtualList
        items={items.slice(0, 12)}
        title="Growing row"
        label={libraryStrings("data-display.virtual-list.label.growing", "Growing activity")}
        renderItem={(item, index) => (
          <div style={rowStyle}>
            <strong>{item.title}</strong>
            <span>{index === 5 ? "This row can grow after first paint." : item.meta}</span>
          </div>
        )}
      />
    </div>
  );
}

const historyButtonStyle: CSSProperties = { minHeight: 44, paddingInline: 12 };

type HistoryRow = { id: string; title: string; size: number };
const historyRows: HistoryRow[] = Array.from({ length: 40 }, (_, index) => ({
  id: `message-${index}`,
  title: `Message ${index + 1}`,
  size: 48 + (index % 3) * 16,
}));
export function ScrollAnchoring() {
  const [rows, setRows] = useState(historyRows);
  const [position, setPosition] = useState<VirtualListPosition | null>(null);
  const controller = useRef<VirtualListHandle>(null);
  const older = useRef(-1);
  const newer = useRef(40);
  return (
    <div style={shell}>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginBottom: 12 }}>
        <button
          style={historyButtonStyle}
          type="button"
          onClick={() =>
            setRows((current) => {
              const prefix = Array.from({ length: 5 }, () => ({
                id: `message-${older.current--}`,
                title: "Earlier message",
                size: 64,
              })).reverse();
              return [...prefix, ...current];
            })
          }
        >
          Load earlier messages
        </button>
        <button
          style={historyButtonStyle}
          type="button"
          onClick={() =>
            setRows((current) => [
              ...current,
              {
                id: `message-${newer.current++}`,
                title: "New message",
                size: 80,
              },
            ])
          }
        >
          Append message
        </button>
        <button
          style={historyButtonStyle}
          type="button"
          onClick={() =>
            setRows((current) =>
              current.map((row, index) => (index === 0 ? { ...row, size: row.size + 160 } : row)),
            )
          }
        >
          Grow earlier message
        </button>
        <button
          style={historyButtonStyle}
          type="button"
          onClick={() =>
            setRows((current) =>
              current.map((row, index) =>
                index === current.length - 1 ? { ...row, size: row.size + 160 } : row,
              ),
            )
          }
        >
          Grow latest message
        </button>
        <button type="button" onClick={() => controller.current?.scrollToEnd()}>
          Go to latest
        </button>
        <button
          style={historyButtonStyle}
          type="button"
          onClick={() => setRows((current) => (current.length ? [] : historyRows))}
        >
          Toggle empty
        </button>
      </div>
      <VirtualList
        items={rows}
        getItemKey={(row) => row.id}
        height={360}
        estimateItemHeight={112}
        label="Conversation history"
        followEnd
        stickyIndices={[0]}
        controllerRef={controller}
        onViewportChange={setPosition}
        renderItem={(row) => (
          <div style={{ minHeight: row.size }}>
            <strong>{row.title}</strong>
            <br />
            <button type="button" data-history-action={row.id}>
              Inspect {row.id}
            </button>
          </div>
        )}
      />
      <output data-scroll-report aria-label="Scroll position">
        {position?.atEnd ? "Following latest messages" : "Reading earlier messages"}
      </output>
    </div>
  );
}

export function OpaqueKeys() {
  const records = [
    { id: "__proto__", label: "First record" },
    { id: "constructor", label: "Second record" },
    { id: "toString", label: "Third record" },
  ];
  return (
    <div style={shell}>
      <VirtualList
        items={records}
        getItemKey={(row) => row.id}
        height={240}
        label="Opaque item identifiers"
        renderItem={(row) => <strong>{row.label}</strong>}
      />
    </div>
  );
}
