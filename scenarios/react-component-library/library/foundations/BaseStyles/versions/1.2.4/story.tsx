import { BaseStyles } from "./BaseStyles";
import { useEffect, useRef, useState, type CSSProperties } from "react";

/** The foundation mounted with the library's own viewport defaults. */
export function Mounted() {
  return (
    <div data-testid="foundations.base-styles">
      <BaseStyles />
      <button data-rcl-control data-control-size="md" type="button">
        Control
      </button>
    </div>
  );
}

/**
 * A host that manages its own scrolling narrows the usable viewport and says
 * so through the contract. Every library surface then reads the host's numbers
 * rather than the raw environment.
 */
export function ViewportContractOverridden() {
  return (
    <div
      data-testid="foundations.base-styles.host-override"
      style={
        {
          "--rcl-viewport-height": "480px",
          "--rcl-safe-bottom": "34px",
          "--rcl-keyboard-inset": "280px",
        } as CSSProperties
      }
    >
      <BaseStyles />
      <div style={{ blockSize: "var(--rcl-viewport-height)" }}>Host-sized region</div>
    </div>
  );
}

/** Unlayered host declarations win over the library's layered defaults. */
export function UnlayeredOverride() {
  return (
    <div
      data-testid="foundations.base-styles.unlayered-override"
      style={{ "--space-md": "7px" } as CSSProperties}
    >
      <BaseStyles />
      <div
        data-testid="foundations.base-styles.measured"
        style={{ padding: "var(--space-md)", borderRadius: "var(--radius-panel)" }}
      >
        Scenario-owned token override
      </div>
    </div>
  );
}

/** Browser-measured proof that the foundation is sufficient without kit CSS. */
export function CanonicalTokenDefaults() {
  const measuredRef = useRef<HTMLDivElement>(null);
  const [measurement, setMeasurement] = useState({ padding: "pending", radius: "pending" });

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      if (!measuredRef.current) return;
      const style = getComputedStyle(measuredRef.current);
      setMeasurement({ padding: style.paddingTop, radius: style.borderTopLeftRadius });
    });
    return () => cancelAnimationFrame(frame);
  }, []);

  return (
    <div data-testid="foundations.base-styles.canonical-defaults">
      <BaseStyles />
      <div
        ref={measuredRef}
        style={{ padding: "var(--space-md)", borderRadius: "var(--radius-panel)" }}
      >
        Canonical defaults
      </div>
      <output
        data-testid="foundations.base-styles.computed"
        data-padding={measurement.padding}
        data-radius={measurement.radius}
      >
        padding={measurement.padding}; radius={measurement.radius}
      </output>
    </div>
  );
}
