import { BaseStyles } from "./BaseStyles";

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
        } as React.CSSProperties
      }
    >
      <BaseStyles />
      <div style={{ blockSize: "var(--rcl-viewport-height)" }}>Host-sized region</div>
    </div>
  );
}
