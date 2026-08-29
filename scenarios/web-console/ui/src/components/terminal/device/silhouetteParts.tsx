import type { DeviceGeometry } from "../../../lib/deviceGeometry";
import { KEYBOARD_COLUMNS, KEYBOARD_ROWS, keyboardColumns, keyboardKeyHeight, screenBox } from "../../../lib/deviceGeometry";

/**
 * Shared drawing primitives for the device silhouettes.
 *
 * [REQ:P0-002e] These draw a recognisable enclosure, never a specific product.
 * The leader's device class is self-declared and operator-editable, so the
 * silhouette must not imply a hardware identity the console cannot verify.
 *
 * Every shape is in device units from `DEVICE_GEOMETRY` and rendered through a
 * viewBox whose aspect matches its element, so nothing is stretched.
 */

/** The enclosure body: filled panel, rim highlight, and the recessed screen. */
export function Enclosure({ geometry, screenLit = false }: { geometry: DeviceGeometry; screenLit?: boolean }) {
  const box = screenBox(geometry);
  return <>
    <rect x="0" y="0" width={geometry.width} height={geometry.height} rx={geometry.radius} fill="var(--wc-device-body)" />
    <rect
      x="0.75"
      y="0.75"
      width={geometry.width - 1.5}
      height={geometry.height - 1.5}
      rx={Math.max(0, geometry.radius - 0.75)}
      fill="none"
      stroke="var(--wc-device-rim)"
      strokeWidth="1.5"
    />
    <rect
      x={box.x}
      y={box.y}
      width={box.width}
      height={box.height}
      rx={geometry.screenRadius}
      fill={screenLit ? "var(--wc-device-screen-lit, rgb(var(--wc-accent) / 0.32))" : "var(--wc-device-screen)"}
    />
    <rect
      x={box.x}
      y={box.y}
      width={box.width}
      height={box.height}
      rx={geometry.screenRadius}
      fill="none"
      stroke="var(--wc-device-screen-rim)"
      strokeWidth="1"
    />
  </>;
}

/** A laptop's wedge base, wider than its lid, drawn below the panel. */
export function WedgeBase({ geometry }: { geometry: DeviceGeometry }) {
  const top = geometry.height + 2;
  const bottom = geometry.height + geometry.baseHeight;
  const lip = geometry.width * 0.035;
  const inset = 9;
  return <>
    <path
      d={`M${String(-lip)} ${String(top)} H${String(geometry.width + lip)} L${String(geometry.width + lip - inset)} ${String(bottom)} H${String(-lip + inset)} Z`}
      fill="var(--wc-device-body-shade)"
    />
    <path d={`M${String(-lip)} ${String(top)} H${String(geometry.width + lip)}`} stroke="var(--wc-device-rim)" strokeWidth="1.2" fill="none" />
    {/* Trackpad recess — the one detail that reads "laptop" rather than "slab". */}
    <rect
      x={geometry.width / 2 - geometry.width * 0.09}
      y={top + (bottom - top) * 0.28}
      width={geometry.width * 0.18}
      height={(bottom - top) * 0.34}
      rx="2"
      fill="var(--wc-device-screen)"
      opacity="0.55"
    />
  </>;
}

/** A monitor's neck and foot, drawn below the panel. */
export function MonitorStand({ geometry }: { geometry: DeviceGeometry }) {
  const centre = geometry.width / 2;
  const top = geometry.height;
  const neckWidth = geometry.width * 0.09;
  const neckHeight = geometry.baseHeight * 0.72;
  const footWidth = geometry.width * 0.34;
  const footHeight = geometry.baseHeight - neckHeight;
  return <>
    <path
      d={`M${String(centre - neckWidth / 2)} ${String(top)} H${String(centre + neckWidth / 2)} L${String(centre + neckWidth / 2 + 4)} ${String(top + neckHeight)} H${String(centre - neckWidth / 2 - 4)} Z`}
      fill="var(--wc-device-body-shade)"
    />
    <rect
      x={centre - footWidth / 2}
      y={top + neckHeight}
      width={footWidth}
      height={footHeight}
      rx={footHeight / 2}
      fill="var(--wc-device-body)"
    />
  </>;
}

/** Phone speaker island, inside the screen's top margin. */
export function Island({ geometry }: { geometry: DeviceGeometry }) {
  const width = geometry.width * 0.21;
  const height = geometry.bezel * 0.95;
  return <rect
    x={geometry.width / 2 - width / 2}
    y={geometry.bezel + height * 0.55}
    width={width}
    height={height}
    rx={height / 2}
    fill="var(--wc-device-body-shade)"
  />;
}

/** Phone home indicator, in the chin. */
export function HomeBar({ geometry }: { geometry: DeviceGeometry }) {
  const width = geometry.width * 0.35;
  return <rect
    x={geometry.width / 2 - width / 2}
    y={geometry.height - geometry.chin / 2 - 1.4}
    width={width}
    height="2.8"
    rx="1.4"
    fill="var(--wc-device-detail)"
  />;
}

/** A small lens dot centred in the top bezel. */
export function Camera({ geometry }: { geometry: DeviceGeometry }) {
  return <circle cx={geometry.width / 2} cy={geometry.bezel / 2} r={Math.min(2.2, geometry.bezel / 3)} fill="var(--wc-device-detail)" />;
}

/** Phone side buttons, drawn just outside the enclosure edges. */
export function SideButtons({ geometry }: { geometry: DeviceGeometry }) {
  const unit = geometry.height / 400;
  return <>
    <rect x={geometry.width - 1} y={86 * unit} width="3" height={46 * unit} rx="1.5" fill="var(--wc-device-body-shade)" />
    <rect x="-2" y={74 * unit} width="3" height={24 * unit} rx="1.5" fill="var(--wc-device-body-shade)" />
    <rect x="-2" y={106 * unit} width="3" height={24 * unit} rx="1.5" fill="var(--wc-device-body-shade)" />
  </>;
}

/** A brand-neutral relief mark in a monitor's chin. */
export function ChinMark({ geometry }: { geometry: DeviceGeometry }) {
  const width = geometry.width * 0.07;
  return <rect
    x={geometry.width / 2 - width / 2}
    y={geometry.height - geometry.chin / 2 - 1.2}
    width={width}
    height="2.4"
    rx="1.2"
    fill="var(--wc-device-detail)"
    opacity="0.55"
  />;
}

/**
 * A stylised key plate filling screen space the leader's keyboard covers.
 *
 * It is deliberately glyphless and non-interactive: it explains why the
 * follower is suddenly seeing fewer rows, and must never read as something a
 * viewer could type on.
 */
export function KeyPlate({ x, y, width, height }: { x: number; y: number; width: number; height: number }) {
  const { gap, padX, keyWidth } = keyboardColumns(width);
  const keyHeight = keyboardKeyHeight(width, height);
  if (keyHeight <= 0.5 || keyWidth <= 0.5) return null;
  const radius = Math.min(keyWidth, keyHeight) / 4;
  // Padding is whatever the key rows leave over, split top and bottom, which
  // stands in for a real keyboard's suggestion strip.
  const rowsHeight = KEYBOARD_ROWS * keyHeight + (KEYBOARD_ROWS - 1) * gap;
  const padY = Math.max(0, (height - rowsHeight) / 2);
  const inner = width - padX * 2;

  // Row key counts and their left/right inset, which is what makes a keyboard
  // read as staggered rather than as a grid.
  const layout = [
    { count: KEYBOARD_COLUMNS, inset: 0 },
    { count: KEYBOARD_COLUMNS - 1, inset: 0.05 },
    { count: KEYBOARD_COLUMNS - 2, inset: 0.11 },
  ];

  const keys: { x: number; y: number; width: number; height: number }[] = [];
  layout.forEach(({ count, inset: insetRatio }, row) => {
    const inset = inner * insetRatio;
    const available = inner - inset * 2;
    const rowKeyWidth = (available - gap * (count - 1)) / count;
    for (let index = 0; index < count; index++) {
      keys.push({
        x: x + padX + inset + index * (rowKeyWidth + gap),
        y: y + padY + row * (keyHeight + gap),
        width: Math.max(0.5, rowKeyWidth),
        height: keyHeight,
      });
    }
  });
  // Bottom row: modifier, space, modifier.
  const bottomY = y + padY + 3 * (keyHeight + gap);
  const modifier = inner * 0.16;
  const space = inner - modifier * 2 - gap * 2;
  keys.push({ x: x + padX, y: bottomY, width: modifier, height: keyHeight });
  keys.push({ x: x + padX + modifier + gap, y: bottomY, width: Math.max(0.5, space), height: keyHeight });
  keys.push({ x: x + padX + modifier + space + gap * 2, y: bottomY, width: modifier, height: keyHeight });

  return <g aria-hidden="true">
    <rect x={x} y={y} width={width} height={height} fill="var(--wc-device-keyplate)" />
    {keys.map((key, index) => (
      <rect key={index} x={key.x} y={key.y} width={key.width} height={key.height} rx={radius} fill="var(--wc-device-keycap)" />
    ))}
  </g>;
}
