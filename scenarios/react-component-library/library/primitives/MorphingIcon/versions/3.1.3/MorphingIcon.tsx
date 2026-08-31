/**
 * @libraryId react-component-library:MorphingIcon
 * @displayName MorphingIcon
 * @description Animates between any two icons, path-morphing when their geometry is compatible and crossfading when it is not.
 * @version 3.1.3
 * @tags ["primitive","motion","icon","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:MorphingIcon */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { isValidElement, useMemo, type CSSProperties, type ReactNode } from "react";
import { geometryPath, type IconGeometry } from "@vrooli/react-component-library/IconGeometry/1";
import { useIconMorph, type IconMorphMode } from "@vrooli/react-component-library/useIconMorph/1";
import { ICON_REGISTRY, type IconName } from "@vrooli/react-component-library/IconRegistry/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/**
 * Injected at runtime rather than imported as a side-effect stylesheet.
 *
 * 2.x shipped these rules in a sibling `MorphingIcon.css`. That works while the
 * component is the entry point a bundler is asked to build, and silently does
 * not when the component is reached transitively through a package subpath —
 * the CSS import is dropped and the rules never arrive. The failure is
 * invisible in this component's own stories (where it *is* the entry) and
 * appears only in consumers: `IconButton`'s bundle contained no `grid-area`
 * rule at all, so the morph frame stacked below the live icon instead of
 * overlaying it, and the live icon was never hidden.
 *
 * `useLibraryStyleSheet` is the library's own delivery mechanism and travels
 * with the component regardless of who imports it or how.
 */
const morphingIconStyles = `
[data-rcl-morphing-icon] { display: inline-grid; place-items: center; vertical-align: middle; position: relative; }
/* Every layer occupies the same cell so a swap overlays rather than stacks. */
[data-rcl-morphing-icon] > * { grid-area: 1 / 1; inline-size: 100%; block-size: 100%; display: inline-grid; place-items: center; }
[data-rcl-morphing-icon] > [data-rcl-hidden="true"] { visibility: hidden; }
[data-rcl-morphing-icon] svg { inline-size: 100%; block-size: 100%; display: block; }
`;

/**
 * 3.1.1 — the swap starts before paint, so the incoming icon never flashes.
 *
 * 3.1.0 — `swapIdentity`, so a control that moves between parents still animates.
 *
 * 3.0.1 — the stylesheet travels with the component. See `morphingIconStyles`.
 *
 * 3.0.0 — arbitrary icons, and geometry that is actually parsed.
 *
 * 2.x accepted only `icon={name}` against an eleven-glyph registry and rendered
 * from a private path parser that understood `M`, `L`, `T`, `H`, `V`, and `Z`.
 * Curve commands were dropped silently, which is why the registry contained
 * only straight-line glyphs — and why `icon="search"`, whose circle is two
 * arcs, shipped as a bare diagonal line.
 *
 * Geometry now lives in the `IconGeometry` foundation, which handles the whole
 * path grammar plus `rect`, `circle`, `ellipse`, `line`, `polyline`, and
 * `polygon`. That makes the interesting case possible: `children` may be any
 * icon component at all — a lucide glyph, an inline `<svg>` — and it is
 * measured off the DOM it renders rather than looked up in a table.
 *
 * The registry path still works unchanged. `icon="close"` renders the same
 * glyph it always did, only correctly.
 */

export type MorphingIconName = IconName | "copy";

/** Retained from 2.x for callers that pinned a strategy by name. */
export type MorphingIconStrategy = "morph" | "crossfade" | "transform";

export interface MorphingIconProps {
  /**
   * The icon to show. Any element renders; a change animates. Prefer this over
   * `icon` — it accepts the icon set the application already uses.
   */
  children?: ReactNode;
  /** Registry glyph. Ignored when `children` is provided. */
  icon?: MorphingIconName;
  /**
   * Identity of the current icon. Defaults to the child's component identity,
   * which distinguishes lucide glyphs correctly without any call-site work.
   * Supply it when one component renders different art from its props.
   */
  iconKey?: string;
  /**
   * A stable identity for the control, so an icon change still animates when
   * the control remounts — a responsive layout that renders it in a different
   * container per mode, a portal, a reordered list. Without it, a remount
   * silently skips the swap because the new instance has no memory of the old
   * icon. Must be unique among controls.
   */
  swapIdentity?: string;
  /**
   * `auto` animates every swap and upgrades to a path morph only when the two
   * shapes measure compatible. See `IconGeometry.morphCompatibility`.
   */
  morph?: IconMorphMode;
  /** @deprecated Use `morph`. Retained so 2.x call sites keep compiling. */
  strategy?: MorphingIconStrategy;
  /** Milliseconds. Ignored under `prefers-reduced-motion`. */
  duration?: number;
  size?: "sm" | "md" | "lg" | number;
  /** Names the icon for assistive technology. Omit inside a labelled control. */
  label?: string;
  className?: string;
  style?: CSSProperties;
}

const COPY_GLYPH = "M9 9h10v10H9zM5 15H4V4h11v1";

const sizeValue = (size: MorphingIconProps["size"]) =>
  typeof size === "number"
    ? `${Math.max(12, Math.min(size, 64))}px`
    : size === "sm"
      ? "var(--icon-size-sm, 1rem)"
      : size === "lg"
        ? "var(--icon-size-lg, 1.5rem)"
        : "var(--icon-size-md, 1.25rem)";

/**
 * Derive a stable identity for an icon element.
 *
 * Every icon in a set is a distinct component — lucide builds one per glyph —
 * so the element's `type` separates them without the caller naming anything.
 * Function identity is stable across renders for module-scope components, which
 * is what every icon library produces. A `displayName` is preferred when
 * present because it survives minification boundaries and reads better in the
 * geometry cache.
 */
function deriveIconKey(node: ReactNode): string {
  if (!isValidElement(node)) return typeof node === "string" ? `text:${node}` : "unknown";
  const type = node.type as { displayName?: string; name?: string } | string;
  if (typeof type === "string") return `intrinsic:${type}`;
  const named = type.displayName ?? type.name;
  if (named) return `component:${named}`;
  // Anonymous components are rare; fall back to reference identity so distinct
  // components still compare unequal.
  return `anonymous:${anonymousKey(type)}`;
}

const anonymousKeys = new WeakMap<object, string>();
let anonymousCounter = 0;
function anonymousKey(type: object): string {
  const existing = anonymousKeys.get(type);
  if (existing) return existing;
  anonymousCounter += 1;
  const key = String(anonymousCounter);
  anonymousKeys.set(type, key);
  return key;
}

function registryGlyph(name: MorphingIconName): { path: string; viewBox: string } {
  if (name === "copy") return { path: COPY_GLYPH, viewBox: "0 0 24 24" };
  const definition = ICON_REGISTRY[name];
  return { path: definition.path, viewBox: definition.viewBox };
}

/** Render the registry glyph as an ordinary SVG so it measures like any other icon. */
function RegistryIcon({ name }: { name: MorphingIconName }) {
  const { path, viewBox } = registryGlyph(name);
  return (
    <svg
      viewBox={viewBox}
      width="100%"
      height="100%"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      focusable="false"
      aria-hidden="true"
    >
      <path d={path} />
    </svg>
  );
}

function MorphedSvg({ geometry }: { geometry: IconGeometry }) {
  return (
    <svg
      data-rcl-morphing-icon-frame=""
      viewBox={geometry.viewBox}
      width="100%"
      height="100%"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      focusable="false"
      aria-hidden="true"
    >
      {geometry.subpaths.map((subpath, index) => (
        <path key={index} d={geometryPath(subpath)} opacity={subpath.opacity} />
      ))}
    </svg>
  );
}

export const MorphingIcon = withClassName(function MorphingIcon({
  children,
  icon,
  iconKey,
  swapIdentity,
  morph,
  // Read here so 2.x call sites keep compiling; the mapping onto `morph`
  // below is the whole reason the deprecated prop is still destructured.
  // eslint-disable-next-line @typescript-eslint/no-deprecated
  strategy,
  duration = 320,
  size = "md",
  label,
  className,
  style,
}: MorphingIconProps) {
  useLibraryStyleSheet("morphing-icon", morphingIconStyles);
  // 2.x callers passed `strategy`; `transform` was never a distinct rendering
  // path, only a data attribute, so it maps onto the crossfade it always was.
  const mode: IconMorphMode =
    morph ?? (strategy === "crossfade" || strategy === "transform" ? "crossfade" : "auto");

  const content = children ?? (icon ? <RegistryIcon name={icon} /> : null);
  const key = iconKey ?? (children ? deriveIconKey(children) : `registry:${icon ?? "none"}`);

  const { technique, progress, active, geometry, previousKey, currentRef, previousRef } =
    useIconMorph({ iconKey: key, swapIdentity, mode, duration });

  // Held across the transition so the outgoing art can still be painted after
  // React has committed the incoming element.
  const outgoing = useOutgoing(content, key, active);

  const resolved = sizeValue(size);
  const computedStyle: CSSProperties = { inlineSize: resolved, blockSize: resolved, ...style };

  return (
    <span
      data-testid="motion.morphing-icon"
      className={className}
      data-rcl-morphing-icon=""
      data-size={size}
      data-rcl-technique={active ? technique : "idle"}
      data-rcl-transition-mode={mode}
      style={computedStyle}
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
    >
      {/*
        The live icon is always mounted, even while a morph paints over it: it
        is what `useIconMorph` measures, and unmounting it would destroy the
        geometry needed for the *next* swap. During a morph it is hidden rather
        than removed.
      */}
      <span
        ref={currentRef}
        data-rcl-morphing-icon-current=""
        data-rcl-hidden={active && technique === "morph" ? "true" : undefined}
        style={
          active && technique === "crossfade"
            ? { opacity: progress, transform: `scale(${0.7 + 0.3 * progress})` }
            : undefined
        }
      >
        {content}
      </span>

      {active && technique === "crossfade" && outgoing && previousKey ? (
        <span
          ref={previousRef}
          data-rcl-morphing-icon-previous=""
          style={{ opacity: 1 - progress, transform: `scale(${1 - 0.3 * progress})` }}
        >
          {outgoing}
        </span>
      ) : null}

      {active && technique === "morph" && geometry ? <MorphedSvg geometry={geometry} /> : null}
    </span>
  );
});

/**
 * Remember the previously rendered icon element for the duration of a swap.
 *
 * A crossfade needs both frames on screen at once, but React has already
 * replaced `children` by the time the transition starts. Holding the last
 * element in a ref — rather than in state — keeps this from causing a second
 * render pass on every icon change.
 */
function useOutgoing(content: ReactNode, key: string, active: boolean): ReactNode {
  const held = useMemo(
    () => ({ key, content, previous: null as ReactNode, previousKey: "" }),
    // A single mutable cell for the lifetime of the component.
    [],
  );
  if (held.key !== key) {
    held.previous = held.content;
    held.previousKey = held.key;
    held.key = key;
    held.content = content;
  } else if (!active) {
    held.content = content;
  }
  return held.previous;
}
