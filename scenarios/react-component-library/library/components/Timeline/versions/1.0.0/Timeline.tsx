/** @vrooliComponentSource react-component-library:Timeline */
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function Timeline({
  events = [],
}: {
  events?: Array<{ label: string; detail?: string }>;
}) {
  return (
    <ol
      aria-label="Timeline"
      style={{
        display: "grid",
        gap: 16,
        listStyle: "none",
        margin: 0,
        padding: 0,
      }}
    >
      {events.map((event, index) => (
        <li
          key={event.label + String(index)}
          style={{ display: "grid", gridTemplateColumns: "16px 1fr", gap: 12 }}
        >
          <span
            aria-hidden
            style={{
              width: 10,
              height: 10,
              marginTop: 5,
              borderRadius: "50%",
              background: "var(--color-primary, #2563eb)",
            }}
          />
          <span>
            <strong>{event.label}</strong>
            {event.detail && (
              <small style={{ display: "block", marginTop: 4, ...muted }}>
                {event.detail}
              </small>
            )}
          </span>
        </li>
      ))}
    </ol>
  );
}
