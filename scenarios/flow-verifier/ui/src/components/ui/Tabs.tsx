/**
 * Tabs — accessible, keyboard-navigable, horizontally-scrollable tablist.
 *
 * Consumers control the active value:
 *
 *   <TabList
 *     idPrefix="run-detail"
 *     value={tab}
 *     onChange={setTab}
 *     aria-label="Run detail sections"
 *     items={[
 *       { value: "result", label: t("...") },
 *       { value: "raw", label: t("...") },
 *     ]}
 *   />
 *   <TabPanel idPrefix="run-detail" value="result" active={tab}>
 *     ...
 *   </TabPanel>
 *
 * Keyboard model is *automatic activation*: ArrowLeft / ArrowRight cycle,
 * Home / End jump, focus follows selection. This matches WAI-ARIA APG's
 * "tabs with automatic activation" pattern (cheap to swap panels here).
 */
import { type KeyboardEvent, type ReactNode, useId, useRef } from "react";

export interface TabItem<T extends string> {
  value: T;
  label: ReactNode;
  testid?: string;
}

interface TabListProps<T extends string> {
  idPrefix: string;
  value: T;
  onChange: (v: T) => void;
  items: TabItem<T>[];
  "aria-label": string;
  className?: string;
  testid?: string;
}

export function TabList<T extends string>({
  idPrefix,
  value,
  onChange,
  items,
  "aria-label": ariaLabel,
  className,
  testid,
}: TabListProps<T>) {
  const refs = useRef<Record<string, HTMLButtonElement | null>>({});

  const onKeyDown = (e: KeyboardEvent<HTMLButtonElement>) => {
    const idx = items.findIndex((it) => it.value === value);
    if (idx === -1) return;
    let nextIdx: number | null = null;
    if (e.key === "ArrowRight") nextIdx = (idx + 1) % items.length;
    else if (e.key === "ArrowLeft") nextIdx = (idx - 1 + items.length) % items.length;
    else if (e.key === "Home") nextIdx = 0;
    else if (e.key === "End") nextIdx = items.length - 1;
    if (nextIdx === null) return;
    e.preventDefault();
    const nextItem = items[nextIdx];
    if (!nextItem) return;
    onChange(nextItem.value);
    refs.current[nextItem.value]?.focus();
  };

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      data-testid={testid}
      className={[
        "flex gap-1 overflow-x-auto border-b border-app-border",
        className ?? "",
      ].join(" ")}
    >
      {items.map((it) => {
        const active = it.value === value;
        return (
          <button
            key={it.value}
            type="button"
            role="tab"
            id={`${idPrefix}-tab-${it.value}`}
            aria-selected={active}
            aria-controls={`${idPrefix}-panel-${it.value}`}
            tabIndex={active ? 0 : -1}
            data-testid={it.testid ?? `${idPrefix}-tab-${it.value}`}
            onClick={() => onChange(it.value)}
            onKeyDown={onKeyDown}
            ref={(el) => {
              refs.current[it.value] = el;
            }}
            className={[
              "-mb-px shrink-0 border-b-2 px-3 py-2 text-sm transition-colors",
              active
                ? "border-app-primary text-app-foreground"
                : "border-transparent text-app-muted-foreground hover:text-app-foreground",
            ].join(" ")}
          >
            {it.label}
          </button>
        );
      })}
    </div>
  );
}

interface TabPanelProps<T extends string> {
  idPrefix: string;
  value: T;
  active: T;
  children: ReactNode;
  className?: string;
}

export function TabPanel<T extends string>({
  idPrefix,
  value,
  active,
  children,
  className,
}: TabPanelProps<T>) {
  const generatedId = useId();
  if (value !== active) return null;
  return (
    <div
      role="tabpanel"
      id={`${idPrefix}-panel-${value}`}
      aria-labelledby={`${idPrefix}-tab-${value}`}
      data-testid={`${idPrefix}-panel-${value}`}
      key={generatedId}
      className={className}
    >
      {children}
    </div>
  );
}
