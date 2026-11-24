import type { Reading } from "../lib/api";
import { displayFormat } from "../lib/format";
import "./SetpointTarget.css";

export function SetpointTarget({ reading }: { reading: Reading }) {
  const target = reading.target;
  if (!target || typeof target.bar !== "number" || !Number.isFinite(target.bar)) return null;

  const direction = target.direction === "up" ? "at least" : target.direction === "down" ? "at most" : "";
  const format = displayFormat(target.bar, reading.format);
  const notation = format?.includes("compact") ? "compact" : "standard";
  const number = reading.unit === "usd" || format?.startsWith("currency")
    ? new Intl.NumberFormat(undefined, { style: "currency", currency: "USD", notation, maximumFractionDigits: 1 }).format(target.bar)
    : new Intl.NumberFormat(undefined, { notation, maximumFractionDigits: 2 }).format(target.bar);
  const distance = target.distanceToTarget;
  const distanceLabel = typeof distance === "number" && Number.isFinite(distance)
    ? distance === 0 ? "target met" : `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(distance)} to go`
    : "";
  const targetLabel = ["Target", direction, number, reading.unit && reading.unit !== "usd" ? reading.unit : ""].filter(Boolean).join(" ");
  const label = distanceLabel ? `${targetLabel} · ${distanceLabel}` : targetLabel;

  return <span className="cc-setpoint-target" data-setpoint-target data-target-direction={target.direction} aria-label={label}>{label}</span>;
}
