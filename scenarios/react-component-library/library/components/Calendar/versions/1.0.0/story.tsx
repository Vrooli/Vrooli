import { useState, type ReactNode } from "react";
import { Calendar, type CalendarValue } from "./Calendar";

const shell = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 430px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: ReactNode;
}) {
  return (
    <section style={shell}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Date selection
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

const month = new Date(2026, 7, 1);

export function Default() {
  return (
    <Showcase
      title="A calm month at a glance"
      detail="Single-date selection stays keyboard-operable, localized, and legible when the calendar is embedded in a form."
    >
      <Calendar
        month={month}
        defaultValue={new Date(2026, 7, 12)}
        label="Release date"
      />
    </Showcase>
  );
}

export function Range() {
  return (
    <Showcase
      title="Range selection shows the whole interval"
      detail="Start, in-range, and end dates have distinct visual and semantic states rather than relying on color alone."
    >
      <Calendar
        month={month}
        mode="range"
        defaultValue={{
          start: new Date(2026, 7, 12),
          end: new Date(2026, 7, 18),
        }}
        label="Release window"
      />
    </Showcase>
  );
}

export function Multiple() {
  return (
    <Showcase
      title="Multiple dates remain easy to scan"
      detail="Selected dates can be added or removed without collapsing the surrounding form context."
    >
      <Calendar
        month={month}
        mode="multiple"
        defaultValue={[
          new Date(2026, 7, 5),
          new Date(2026, 7, 12),
          new Date(2026, 7, 22),
        ]}
        label="Review dates"
      />
    </Showcase>
  );
}

export function Constrained() {
  return (
    <Showcase
      title="Constraints are visible at the point of choice"
      detail="Unavailable dates stay in the grid for orientation while native disabled semantics prevent invalid selection."
    >
      <Calendar
        month={month}
        minDate={new Date(2026, 7, 8)}
        maxDate={new Date(2026, 7, 24)}
        label="Booking date"
      />
    </Showcase>
  );
}

export function Interactive() {
  const [value, setValue] = useState<CalendarValue>(null);
  return (
    <Showcase
      title="Selection is a real form interaction"
      detail="Choose a date, then use arrow keys, Home, End, and month navigation to continue without leaving the control."
    >
      <Calendar
        month={month}
        value={value}
        onChange={setValue}
        label="Interactive date"
      />
    </Showcase>
  );
}
