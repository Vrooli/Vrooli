import {
  cloneElement,
  forwardRef,
  isValidElement,
  type CSSProperties,
  type ElementType,
  type HTMLAttributes,
  type ReactElement,
  type ReactNode,
  type SVGProps,
} from "react";
import {
  ICON_REGISTRY,
  type IconName,
} from "../../foundations/IconRegistry/versions/1.0.0/IconRegistry";
import { TEXT_STYLES } from "../../foundations/Tokens/versions/1.0.0/Tokens";

const layoutStyle = (
  gap: string | undefined,
  extra?: CSSProperties,
): CSSProperties => ({
  gap: gap ? `var(--space-${gap})` : undefined,
  ...extra,
});
export function Slot({ children, ...props }: HTMLAttributes<HTMLElement>) {
  if (!isValidElement(children)) return null;
  return cloneElement(children as ReactElement, props);
}
export const Stack = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "md", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={layoutStyle(gap, { display: "flex", flexDirection: "column" })}
    {...props}
  />
));
export const Inline = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "sm", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={layoutStyle(gap, {
      display: "flex",
      flexWrap: "wrap",
      alignItems: "center",
    })}
    {...props}
  />
));
export const Cluster = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "sm", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={layoutStyle(gap, {
      display: "flex",
      flexWrap: "wrap",
      alignItems: "baseline",
    })}
    {...props}
  />
));
export const Container = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { width?: "content" | "wide" | "full" }
>(({ width = "content", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    data-container-width={width}
    style={{
      width: "100%",
      maxWidth:
        width === "full"
          ? "none"
          : width === "wide"
            ? "var(--container-wide)"
            : "var(--container-content)",
      marginInline: "auto",
      ...props.style,
    }}
    {...props}
  />
));
export function Separator({
  orientation = "horizontal",
  ...props
}: HTMLAttributes<HTMLHRElement> & {
  orientation?: "horizontal" | "vertical";
}) {
  return (
    <hr
      aria-orientation={orientation}
      data-orientation={orientation}
      style={{
        border: 0,
        background: "var(--app-border)",
        ...(orientation === "vertical"
          ? { width: "var(--separator-thickness)", height: "100%" }
          : { height: "var(--separator-thickness)", width: "100%" }),
        ...props.style,
      }}
      {...props}
    />
  );
}
export const ScrollArea = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    tabIndex={0}
    style={{
      overflow: "auto",
      WebkitOverflowScrolling: "touch",
      ...props.style,
    }}
    {...props}
  />
));
export const Surface = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & {
    elevation?: "flat" | "raised" | "floating" | "overlay";
  }
>(({ elevation = "flat", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    data-elevation={elevation}
    style={{
      background: "var(--app-surface)",
      border: "var(--surface-border)",
      borderRadius: "var(--panel-radius)",
      boxShadow: `var(--elev-${elevation})`,
      ...props.style,
    }}
    {...props}
  />
));
export function EdgeFade({
  side = "inline-end",
  ...props
}: HTMLAttributes<HTMLDivElement> & { side?: "inline-start" | "inline-end" }) {
  return (
    <div
      aria-hidden
      style={{
        pointerEvents: "none",
        [side === "inline-start" ? "insetInlineStart" : "insetInlineEnd"]: 0,
        position: "absolute",
        insetBlock: 0,
        width: "var(--edge-fade-width)",
        background: "var(--edge-fade)",
        ...props.style,
      }}
      {...props}
    />
  );
}
export function Text({
  style = "body",
  as = "span",
  ...props
}: HTMLAttributes<HTMLElement> & {
  style?: keyof typeof TEXT_STYLES;
  as?: keyof HTMLElementTagNameMap;
}) {
  const Component: ElementType = as;
  return (
    <Component data-text-style={style} className={`text-${style}`} {...props} />
  );
}
export function Heading({
  level = 2,
  style = "heading",
  ...props
}: HTMLAttributes<HTMLHeadingElement> & {
  level?: 1 | 2 | 3 | 4 | 5 | 6;
  style?: keyof typeof TEXT_STYLES;
}) {
  const Component: ElementType = `h${level}`;
  return <Component data-text-style={style} {...props} />;
}
export const Link = forwardRef<
  HTMLAnchorElement,
  HTMLAttributes<HTMLAnchorElement> & { href?: string }
>(({ className, ...props }, ref) => (
  <a ref={ref} className={className} data-link="true" {...props} />
));
export function Code({
  inline = false,
  ...props
}: HTMLAttributes<HTMLElement> & { inline?: boolean }) {
  const Component = inline ? "code" : "pre";
  return <Component data-text-style="code" {...props} />;
}
type SVGElementProps = SVGProps<SVGSVGElement>;
export function Icon({
  name,
  label,
  ...props
}: { name: IconName; label?: string } & SVGElementProps) {
  const icon = ICON_REGISTRY[name];
  return (
    <svg
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      viewBox={icon.viewBox}
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      data-icon={name}
      {...props}
    >
      <path d={icon.path} />
    </svg>
  );
}
const badgeTones = {
  neutral: "var(--app-muted-foreground)",
  info: "var(--app-info)",
  success: "var(--app-success)",
  warning: "var(--app-warning)",
  danger: "var(--app-danger)",
} as const;
export function Badge({
  tone = "neutral",
  children,
  ...props
}: HTMLAttributes<HTMLSpanElement> & { tone?: keyof typeof badgeTones }) {
  return (
    <span
      role="status"
      data-tone={tone}
      style={{
        color: badgeTones[tone],
        border: "var(--badge-border)",
        borderRadius: "var(--radius-pill)",
        paddingInline: "var(--space-sm)",
        paddingBlock: "var(--space-3xs)",
        ...props.style,
      }}
      {...props}
    >
      {children}
    </span>
  );
}
export function Skeleton({
  label = "Loading",
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      data-skeleton="true"
      style={{
        background: "var(--app-surface-muted)",
        borderRadius: "var(--radius-sm)",
        minHeight: "var(--skeleton-line-height)",
        ...props.style,
      }}
      {...props}
    />
  );
}
export function Spinner({
  label = "Loading",
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      data-spinner="true"
      style={{
        width: "var(--icon-size-md)",
        height: "var(--icon-size-md)",
        border: "var(--spinner-border)",
        borderTopColor: "var(--app-primary)",
        borderRadius: "var(--radius-pill)",
        animation: "vrooli-spin var(--dur-slow) linear infinite",
        ...props.style,
      }}
      {...props}
    />
  );
}
export function Presence({
  present,
  children,
}: {
  present: boolean;
  children: ReactNode;
}) {
  return present ? <>{children}</> : null;
}
export function LayoutGroup({
  children,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div data-layout-group="true" {...props}>
      {children}
    </div>
  );
}
