/** @vrooliComponentSource foundations.viewport-axis */
import { selectors } from "../../consts/selectors";

/**
 * DeviceVisionFilterDefs renders a hidden SVG carrying the `<filter>`
 * definitions consumed by the emulator viewport's `filter: url(#...)`
 * binding (req DV-002). Matrices match the values app-monitor's
 * DeviceVisionFilterDefs uses — the same widely-cited approximations
 * used by Chrome DevTools' rendering pane.
 *
 * Rendered once at the EmulatorChrome root so every viewport that
 * references `#rcl-vision-<name>` resolves locally without bleed.
 */
export function DeviceVisionFilterDefs() {
  return (
    <svg
      data-testid={selectors.components.emulator.filterDefs}
      width={0}
      height={0}
      aria-hidden="true"
      style={{ position: "absolute", width: 0, height: 0 }}
    >
      <defs>
        <filter id="rcl-vision-grayscale">
          <feColorMatrix
            type="matrix"
            values="0.2126 0.7152 0.0722 0 0
                    0.2126 0.7152 0.0722 0 0
                    0.2126 0.7152 0.0722 0 0
                    0 0 0 1 0"
          />
        </filter>
        <filter id="rcl-vision-protanopia">
          <feColorMatrix
            type="matrix"
            values="0.567 0.433 0     0 0
                    0.558 0.442 0     0 0
                    0     0.242 0.758 0 0
                    0     0     0     1 0"
          />
        </filter>
        <filter id="rcl-vision-deuteranopia">
          <feColorMatrix
            type="matrix"
            values="0.625 0.375 0   0 0
                    0.7   0.3   0   0 0
                    0     0.3   0.7 0 0
                    0     0     0   1 0"
          />
        </filter>
        <filter id="rcl-vision-tritanopia">
          <feColorMatrix
            type="matrix"
            values="0.95 0.05  0     0 0
                    0    0.433 0.567 0 0
                    0    0.475 0.525 0 0
                    0    0     0     1 0"
          />
        </filter>
      </defs>
    </svg>
  );
}
