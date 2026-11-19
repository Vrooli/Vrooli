import { Button } from "@vrooli/react-component-library/Button/2";
import type { ButtonProps } from "@vrooli/react-component-library/Button/2";
import {
  type CSSProperties,
  type ReactElement,
  type ReactNode,
  cloneElement,
  isValidElement,
  useId,
  useRef,
  type SelectHTMLAttributes,
  type InputHTMLAttributes,
  type TextareaHTMLAttributes,
  type HTMLAttributes,
} from "react";
import { ChevronDown, Search } from "lucide-react";

import { cx } from "./cx";

/**
 * The workbench action control. RCL's Button owns the control contract; this
 * wrapper only supplies Nocturne defaults so a page does not repeat
 * `shape="square" variant="ghost"` at every call site.
 */
export function WbButton({
  variant = "ghost",
  size = "md",
  shape = "square",
  className,
  ...props
}: ButtonProps) {
  return (
    <Button
      variant={variant}
      size={size}
      shape={shape}
      density="compact"
      className={cx("wb-btn", variant, className)}
      {...props}
    />
  );
}

export function Panel({
  children,
  className,
  ...props
}: HTMLAttributes<HTMLDivElement> & { children: ReactNode }) {
  return (
    <section className={cx("wb-panel", className)} {...props}>
      {children}
    </section>
  );
}

export function PanelHead({
  title,
  meta,
  actions,
}: {
  title: ReactNode;
  meta?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <header className="wb-panel-head">
      <div className="wb-panel-title">{title}</div>
      <div className="wb-row">
        {meta ? <span className="wb-panel-meta">{meta}</span> : null}
        {actions}
      </div>
    </header>
  );
}

export function PanelBody({
  children,
  flush,
  className,
}: {
  children: ReactNode;
  flush?: boolean;
  className?: string;
}) {
  return <div className={cx("wb-panel-body", flush && "flush", className)}>{children}</div>;
}

export function Kpi({
  label,
  value,
  detail,
  tone,
}: {
  label: ReactNode;
  value: ReactNode;
  detail?: ReactNode;
  tone?: "green" | "amber" | "red" | "signal";
}) {
  return (
    <div className="wb-kpi">
      <div className="wb-kpi-k">{label}</div>
      <div className={cx("wb-kpi-v", tone && `is-${tone}`)}>{value}</div>
      {detail ? <div className="wb-kpi-d">{detail}</div> : null}
    </div>
  );
}

export function Chip({
  children,
  tone,
  active,
  title,
}: {
  children: ReactNode;
  tone?: "green" | "amber" | "red" | "ice" | "violet";
  active?: boolean;
  title?: string;
}) {
  return (
    <span className={cx("wb-chip", active && "is-active", tone && `is-${tone}`)} title={title}>
      {children}
    </span>
  );
}

export function Dot({ tone }: { tone?: "green" | "amber" | "red" | "ice" | "signal" }) {
  const color =
    tone === "green"
      ? "var(--wb-green)"
      : tone === "amber"
        ? "var(--wb-amber)"
        : tone === "red"
          ? "var(--wb-red)"
          : tone === "ice"
            ? "var(--wb-ice)"
            : "var(--wb-signal)";
  return <span className="wb-dot" style={{ background: color }} aria-hidden />;
}

export function Ring({
  value,
  size = 62,
  tone = "green",
  label,
}: {
  value: number;
  size?: number;
  tone?: "green" | "amber" | "red" | "signal" | "ice";
  label?: string;
}) {
  const color =
    tone === "green"
      ? "var(--wb-green)"
      : tone === "amber"
        ? "var(--wb-amber)"
        : tone === "red"
          ? "var(--wb-red)"
          : tone === "ice"
            ? "var(--wb-ice)"
            : "var(--wb-signal)";
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <span
      className="wb-ring"
      style={
        {
          "--wb-ring-size": `${size}px`,
          "--wb-ring-color": color,
          "--wb-ring-value": clamped,
        } as CSSProperties
      }
      role="img"
      aria-label={label ?? `${Math.round(clamped)}%`}
    >
      <span>{Math.round(clamped)}%</span>
    </span>
  );
}

export function Empty({
  title,
  children,
  action,
}: {
  title: ReactNode;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="wb-empty">
      <div className="wb-empty-title">{title}</div>
      {children ? <p>{children}</p> : null}
      {action}
    </div>
  );
}

export function Skeleton({
  height = 14,
  width = "100%",
}: {
  height?: number;
  width?: number | string;
}) {
  return <span className="wb-skeleton" style={{ display: "block", height, width }} aria-hidden />;
}

export function Banner({
  tone,
  icon,
  children,
}: {
  tone?: "red" | "amber" | "ice";
  icon?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div
      className={cx("wb-banner", tone && `is-${tone}`)}
      role={tone === "red" ? "alert" : undefined}
    >
      {icon}
      <div>{children}</div>
    </div>
  );
}

export function Field({
  label,
  hint,
  children,
  htmlFor,
}: {
  label: ReactNode;
  hint?: ReactNode;
  children: ReactNode;
  htmlFor?: string;
}) {
  // Associate the label with the control even when a callsite did not think
  // about ids. A generated id is stable across renders, and an id the child
  // already carries wins so the caller keeps control of it.
  const generated = useId();
  const existing = isValidElement<{ id?: string }>(children) ? children.props.id : undefined;
  const id = htmlFor ?? existing ?? generated;
  const control: ReactNode =
    isValidElement<{ id?: string }>(children) && !existing
      ? cloneElement(children as ReactElement<{ id?: string }>, { id })
      : children;
  return (
    <div className="wb-field">
      <label htmlFor={id}>{label}</label>
      {control}
      {hint ? <span className="wb-muted-2 wb-small">{hint}</span> : null}
    </div>
  );
}

export function TextInput({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cx("wb-input", className)} {...props} />;
}

export function TextArea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cx("wb-textarea", className)} {...props} />;
}

export function Select({
  className,
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement> & { children: ReactNode }) {
  return (
    <span className="wb-select-wrap" style={{ position: "relative", display: "block" }}>
      <select className={cx("wb-select", className)} {...props}>
        {children}
      </select>
      <ChevronDown
        aria-hidden
        size={14}
        style={{
          position: "absolute",
          right: 10,
          top: "50%",
          transform: "translateY(-50%)",
          pointerEvents: "none",
          color: "var(--wb-muted-2)",
        }}
      />
    </span>
  );
}

export function SearchBox({
  value,
  onChange,
  placeholder,
  label,
  shortcut,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  label: string;
  shortcut?: string;
}) {
  return (
    <div className="wb-search">
      <Search aria-hidden size={15} style={{ color: "var(--wb-muted-2)", flex: "none" }} />
      <input
        type="search"
        value={value}
        aria-label={label}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {shortcut ? <kbd className="wb-kbd">{shortcut}</kbd> : null}
    </div>
  );
}

export function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      className="wb-toggle"
      onClick={() => onChange(!checked)}
    />
  );
}

export function Tabs<T extends string>({
  items,
  value,
  onChange,
  label,
}: {
  items: Array<{ id: T; label: ReactNode; count?: number }>;
  value: T;
  onChange: (id: T) => void;
  label: string;
}) {
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const move = (index: number) => {
    if (items.length === 0) return;
    const next = (index + items.length) % items.length;
    const item = items[next];
    if (!item) return;
    onChange(item.id);
    refs.current[next]?.focus();
  };
  return (
    <div className="wb-tabs" role="tablist" aria-label={label}>
      {items.map((item, index) => (
        <button
          key={item.id}
          ref={(node) => {
            refs.current[index] = node;
          }}
          type="button"
          role="tab"
          aria-selected={value === item.id}
          tabIndex={value === item.id ? 0 : -1}
          className="wb-tab"
          onClick={() => onChange(item.id)}
          onKeyDown={(event) => {
            if (event.key === "ArrowRight") {
              event.preventDefault();
              move(index + 1);
            } else if (event.key === "ArrowLeft") {
              event.preventDefault();
              move(index - 1);
            } else if (event.key === "Home") {
              event.preventDefault();
              move(0);
            } else if (event.key === "End") {
              event.preventDefault();
              move(items.length - 1);
            }
          }}
        >
          {item.label}
          {typeof item.count === "number" ? (
            <span className="wb-tab-count">{item.count}</span>
          ) : null}
        </button>
      ))}
    </div>
  );
}

export function Segmented<T extends string>({
  items,
  value,
  onChange,
  label,
}: {
  items: Array<{ id: T; label: ReactNode; ariaLabel?: string }>;
  value: T;
  onChange: (id: T) => void;
  label: string;
}) {
  return (
    <div className="wb-seg" role="group" aria-label={label}>
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          aria-label={item.ariaLabel}
          aria-pressed={value === item.id}
          onClick={() => onChange(item.id)}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

export function Meter({
  value,
  tone = "signal",
}: {
  value: number;
  tone?: "signal" | "green" | "amber" | "red" | "ice";
}) {
  const color =
    tone === "green"
      ? "var(--wb-green)"
      : tone === "amber"
        ? "var(--wb-amber)"
        : tone === "red"
          ? "var(--wb-red)"
          : tone === "ice"
            ? "var(--wb-ice)"
            : "var(--wb-signal)";
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <span
      className="wb-meter"
      style={{
        display: "block",
        height: 4,
        borderRadius: 999,
        background: "#241f27",
        overflow: "hidden",
      }}
      aria-hidden
    >
      <span
        className="wb-meter-fill"
        style={{ display: "block", width: `${clamped}%`, height: "100%", background: color }}
      />
    </span>
  );
}
