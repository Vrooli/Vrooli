/** @vrooliComponentSource overlays.responsive-panel */
import { selectors } from "../../consts/selectors";

const VISION_MATRICES = {
  grayscale: "0.2126 0.7152 0.0722 0 0 0.2126 0.7152 0.0722 0 0 0.2126 0.7152 0.0722 0 0 0 0 0 1 0",
  protanopia: "0.567 0.433 0 0 0 0.558 0.442 0 0 0 0 0.242 0.758 0 0 0 0 0 1 0",
  deuteranopia: "0.625 0.375 0 0 0 0.7 0.3 0 0 0 0 0.3 0.7 0 0 0 0 0 1 0",
  tritanopia: "0.95 0.05 0 0 0 0 0.433 0.567 0 0 0 0.475 0.525 0 0 0 0 0 1 0",
} as const;

/** Supplies the SVG matrices consumed by the emulator's filter URL chain. */
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
        {Object.entries(VISION_MATRICES).map(([name, values]) => (
          <filter key={name} id={`rcl-vision-${name}`}>
            <feColorMatrix type="matrix" values={values} />
          </filter>
        ))}
      </defs>
    </svg>
  );
}
