import { useEffect, useState } from "react";
import { RollingNumber } from "./RollingNumber";

const ground = {
  display: "grid",
  gap: "var(--space-md, 24px)",
  padding: "var(--space-lg, 32px)",
  background: "var(--color-background, #f8fafc)",
  color: "var(--color-foreground, #0f172a)",
};

/** The same figure in every ink: solid, dimmed, hollow, dotted. The glyph box never changes. */
export function FourInks() {
  return (
    <div style={ground}>
      <RollingNumber value={1284} format="integer" ink="solid" scale="display" />
      <RollingNumber value={1284} format="integer" ink="dimmed" scale="display" />
      <RollingNumber value={1284} format="integer" ink="hollow" scale="display" />
      <RollingNumber value={1284} format="integer" ink="dotted" scale="display" />
    </div>
  );
}

/** Currency, percent and duration formats carry their affixes at a smaller weight beside the digits. */
export function Formats() {
  return (
    <div style={ground}>
      <RollingNumber
        value={12400}
        format="currency.compact"
        unit="usd"
        ink="solid"
        scale="display"
      />
      <RollingNumber value={0.58} format="percent" ink="solid" scale="display" />
      <RollingNumber value={1200} format="minutes" ink="solid" scale="display" />
      <RollingNumber value={null} ink="unavailable" scale="display" placeholder="––" />
    </div>
  );
}

/** A wall-scale hero whose last digit ticks: only that digit rolls, the rest hold still. */
export function Rolling() {
  const [value, setValue] = useState(1284);
  useEffect(() => {
    const timer = window.setInterval(() => setValue((v) => v + 1), 1600);
    return () => window.clearInterval(timer);
  }, []);
  return (
    <div style={ground}>
      <RollingNumber value={value} format="integer" ink="solid" scale="wall" />
    </div>
  );
}
