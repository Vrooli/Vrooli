/**
 * @libraryId react-component-library:Avatar
 * @displayName Avatar
 * @description
 * @version 1.3.4
 * @tags ["identity","media","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource primitives.avatar */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { Children, useMemo, type CSSProperties, type ReactNode } from "react";
import { ProgressiveImage } from "@vrooli/react-component-library/ProgressiveImage/1.1.0";
import { Presence } from "@vrooli/react-component-library/Presence/1.0.0";
import { Text } from "@vrooli/react-component-library/Text/1.0.0";

export type AvatarSize = "xs" | "sm" | "md" | "lg" | "xl";
export type AvatarShape = "circle" | "rounded" | "square";
export type AvatarPresence = "online" | "away" | "busy" | "offline";
export type AvatarImageState = "loading" | "loaded" | "error";

export interface AvatarProps {
  name: string;
  src?: string;
  alt?: string;
  size?: AvatarSize;
  shape?: AvatarShape;
  presence?: AvatarPresence;
  presenceLabel?: string;
  /** An externally resolved image state for deterministic integrations and previews. */
  imageState?: AvatarImageState;
  loading?: "eager" | "lazy";
  placeholder?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-avatar-shell] { position: relative; display: inline-grid; place-items: center; inline-size: max-content; }
[data-rcl-avatar] { --rcl-avatar-size: var(--space-xl); position: relative; display: inline-grid; place-items: center; box-sizing: border-box; inline-size: var(--rcl-avatar-size); block-size: var(--rcl-avatar-size); flex: 0 0 var(--rcl-avatar-size); overflow: visible; border: var(--border-hairline) solid var(--color-border); background: var(--color-surface-muted); color: var(--color-foreground); box-shadow: 0 0 0 var(--space-3xs) var(--color-surface); }
[data-rcl-avatar][data-size="xs"] { --rcl-avatar-size: var(--space-md); }
[data-rcl-avatar][data-size="sm"] { --rcl-avatar-size: var(--space-lg); }
[data-rcl-avatar][data-size="lg"] { --rcl-avatar-size: var(--space-2xl); }
[data-rcl-avatar][data-size="xl"] { --rcl-avatar-size: calc(var(--space-2xl) + var(--space-md)); }
[data-rcl-avatar][data-shape="circle"] { border-radius: var(--radius-pill); }
[data-rcl-avatar][data-shape="rounded"] { border-radius: var(--radius-panel); }
[data-rcl-avatar][data-shape="square"] { border-radius: var(--radius-control); }
[data-rcl-avatar-fallback] { display: grid; place-items: center; box-sizing: border-box; inline-size: 100%; block-size: 100%; overflow: hidden; border-radius: inherit; background: color-mix(in srgb, var(--color-primary) 14%, var(--color-surface-muted)); color: var(--color-primary); font: var(--text-label); font-weight: 750; letter-spacing: .04em; }
[data-rcl-avatar] [data-rcl-progressive-image] [data-rcl-progressive-image-placeholder], [data-rcl-avatar] [data-rcl-progressive-image] [data-rcl-progressive-image-error] { padding: 0; border-radius: inherit; }
[data-rcl-avatar] [data-rcl-progressive-image-error-content] { inline-size: 100%; block-size: 100%; max-inline-size: none; }
[data-rcl-avatar-shell] > [data-rcl-presence] { position: absolute; inset-block-end: calc(var(--space-3xs) * -1); inset-inline-end: calc(var(--space-3xs) * -1); display: grid; place-items: center; box-sizing: border-box; inline-size: var(--space-sm); block-size: var(--space-sm); border: var(--border-strong) solid var(--color-surface); border-radius: var(--radius-pill); color: var(--color-surface); font: var(--text-caption); }
[data-rcl-avatar-presence] { display: grid; place-items: center; inline-size: 100%; block-size: 100%; border-radius: inherit; background: var(--color-muted-foreground); transition: background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-avatar-presence] > [aria-hidden="true"] { inline-size: var(--space-2xs); block-size: var(--space-2xs); border-radius: var(--radius-pill); background: currentColor; }
[data-rcl-avatar-presence="online"] { background: var(--color-success); }
[data-rcl-avatar-presence="away"] { background: var(--color-warning); }
[data-rcl-avatar-presence="busy"] { background: var(--color-danger); }
[data-rcl-avatar-group] { display: inline-flex; align-items: center; padding-inline-start: var(--space-3xs); }
[data-rcl-avatar-group] > [data-rcl-avatar-shell], [data-rcl-avatar-group] > [data-rcl-avatar], [data-rcl-avatar-group] > [data-rcl-avatar-overflow] { margin-inline-start: calc(var(--space-xs) * -1); }
[data-rcl-avatar-overflow] { display: grid; place-items: center; box-sizing: border-box; inline-size: var(--space-xl); block-size: var(--space-xl); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-pill); background: var(--color-surface-muted); color: var(--color-muted-foreground); box-shadow: 0 0 0 var(--space-3xs) var(--color-surface); font: var(--text-caption); font-weight: 700; }
@media (prefers-reduced-motion: reduce) { [data-rcl-avatar-presence] { transition: none; } }
@media (forced-colors: active) { [data-rcl-avatar] { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: 0 0 0 var(--space-3xs) Canvas; } [data-rcl-avatar-fallback], [data-rcl-avatar-overflow] { border: var(--border-hairline) solid CanvasText; background: Canvas; color: CanvasText; } [data-rcl-avatar-presence] { background: Highlight; } [data-rcl-avatar-shell] > [data-rcl-presence] { border-color: Canvas; } }
`;

function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  return parts
    .slice(0, 2)
    .map((part) => part[0]?.toLocaleUpperCase() ?? "")
    .join("");
}

export const Avatar = withClassName(function Avatar({
  name,
  src,
  alt = `${name} avatar`,
  size = "md",
  shape = "circle",
  presence,
  presenceLabel,
  imageState,
  loading = "lazy",
  placeholder,
  className,
  style,
}: AvatarProps) {
  const fallback = useMemo(() => initials(name), [name]);
  const accessiblePresence = presenceLabel ?? `${name} is ${presence ?? "offline"}`;
  const resolvedPlaceholder = placeholder ?? (
    <span data-testid="primitives.avatar" aria-hidden="true" />
  );
  return (
    <>
      <style data-rcl-avatar-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <span data-rcl-avatar-shell>
        <span
          data-rcl-avatar
          data-size={size}
          data-shape={shape}
          data-name={name}
          role="img"
          aria-label={name}
          className={className}
          style={style}
        >
          {src ? (
            <ProgressiveImage
              src={src}
              alt={alt}
              loading={loading}
              displayState={imageState}
              placeholder={resolvedPlaceholder}
              ratio="1 / 1"
              frameStyle={{
                inlineSize: "100%",
                blockSize: "100%",
                boxSizing: "border-box",
                overflow: "hidden",
                border: 0,
                borderRadius: "inherit",
                background: "transparent",
              }}
              errorFallback={<span data-rcl-avatar-fallback>{fallback}</span>}
            />
          ) : (
            <span data-rcl-avatar-fallback aria-hidden="true">
              {fallback}
            </span>
          )}
        </span>
        <Presence present={Boolean(presence)} as="span" duration="quick">
          {presence ? (
            <span data-rcl-avatar-presence={presence} role="status" aria-label={accessiblePresence}>
              <span aria-hidden="true" />
            </span>
          ) : null}
        </Presence>
      </span>
    </>
  );
});

export interface AvatarGroupProps {
  children: ReactNode;
  maxVisible?: number;
  overflowLabel?: (count: number) => string;
  label?: string;
  className?: string;
}

export const AvatarGroup = withClassName(function AvatarGroup({
  children,
  maxVisible,
  overflowLabel = (count) => `+${count} more people`,
  label = useStrings("primitives.avatar.people", "People"),
  className,
}: AvatarGroupProps) {
  const items = Children.toArray(children);
  const visibleCount =
    maxVisible === undefined
      ? items.length
      : Number.isFinite(maxVisible)
        ? Math.max(0, Math.floor(maxVisible))
        : 0;
  const visible = items.slice(0, visibleCount);
  const overflow = Math.max(0, items.length - visible.length);
  return (
    <>
      <style data-rcl-avatar-group-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <div data-rcl-avatar-group role="group" aria-label={label} className={className}>
        {visible}
        {overflow > 0 ? (
          <span data-rcl-avatar-overflow role="img" aria-label={overflowLabel(overflow)}>
            <Text as="span">+{overflow}</Text>
          </span>
        ) : null}
      </div>
    </>
  );
});

export const AvatarParts = { Group: AvatarGroup };
