/** @vrooliComponentSource overlays.menu */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useId,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type HTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
} from "react";
import {
  Popover,
  PopoverParts,
  usePopover,
} from "../../../Popover/versions/1.0.0/Popover";
import { useRovingFocus } from "../../../../hooks/useRovingFocus/versions/1.0.0/useRovingFocus";
import { useTypeahead } from "../../../../hooks/useTypeahead/versions/1.0.0/useTypeahead";

const styles = `
  [data-rcl-menu-content] { display: grid; gap: var(--space-3xs); min-inline-size: 13rem; padding: var(--space-2xs); }
  [data-rcl-menu-item], [data-rcl-menu-checkbox], [data-rcl-menu-radio] { display: grid; grid-template-columns: 1.25rem minmax(0, 1fr) auto; align-items: center; gap: var(--space-xs); min-block-size: var(--tap-target-min); padding: var(--space-xs) var(--space-sm); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); cursor: pointer; font: var(--text-body-sm); text-align: start; }
  [data-rcl-menu-item]:hover, [data-rcl-menu-item]:focus-visible, [data-rcl-menu-checkbox]:hover, [data-rcl-menu-checkbox]:focus-visible, [data-rcl-menu-radio]:hover, [data-rcl-menu-radio]:focus-visible { background: var(--color-surface-muted); outline: none; }
  [data-rcl-menu-item][data-disabled="true"], [data-rcl-menu-checkbox][data-disabled="true"], [data-rcl-menu-radio][data-disabled="true"] { cursor: not-allowed; opacity: .55; }
  [data-rcl-menu-indicator] { display: grid; place-items: center; color: var(--color-primary); font: var(--text-label); }
  [data-rcl-menu-label] { padding: var(--space-xs) var(--space-sm) var(--space-3xs); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
  [data-rcl-menu-separator] { block-size: 1px; margin-block: var(--space-2xs); background: var(--color-border); }
  [data-rcl-menu-shortcut] { color: var(--color-muted-foreground); font: var(--text-caption); }
  @media (prefers-reduced-motion: reduce) { [data-rcl-menu-item], [data-rcl-menu-checkbox], [data-rcl-menu-radio] { transition: none; } }
`;

interface MenuRegistry {
  refs: RefObject<HTMLElement>[];
  labels: string[];
  activeIndex: number;
  setActiveIndex: (index: number) => void;
  disabledIndices: number[];
  register: (
    id: string,
    label: string,
    ref: RefObject<HTMLElement>,
    disabled: boolean,
  ) => number;
  close: () => void;
}

const MenuContext = createContext<MenuRegistry | null>(null);

function useMenuContext() {
  const value = useContext(MenuContext);
  if (!value) throw new Error("Menu parts must be used inside Menu");
  return value;
}

export interface MenuProps {
  children: ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
}

export function Menu({ children, ...props }: MenuProps) {
  return (
    <Popover {...props} placement="bottom-start">
      <style
        data-rcl-menu-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      {children}
    </Popover>
  );
}

export function MenuTrigger({
  children,
  ...props
}: { children: ReactNode } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <PopoverParts.Trigger {...props} aria-haspopup="menu">
      {children}
    </PopoverParts.Trigger>
  );
}

export function MenuContent({
  children,
  ...props
}: { children: ReactNode } & HTMLAttributes<HTMLDivElement>) {
  const [activeIndex, setActiveIndex] = useState(0);
  const popover = usePopover();
  const registryRef = useRef(
    new Map<
      string,
      { label: string; ref: RefObject<HTMLElement>; disabled: boolean }
    >(),
  );
  const [, rerender] = useState(0);
  const register = useCallback(
    (
      id: string,
      label: string,
      ref: RefObject<HTMLElement>,
      disabled: boolean,
    ) => {
      if (!registryRef.current.has(id)) {
        registryRef.current.set(id, { label, ref, disabled });
        rerender((value) => value + 1);
      }
      return [...registryRef.current.keys()].indexOf(id);
    },
    [],
  );
  const entries = [...registryRef.current.values()];
  const refs = entries.map((entry) => entry.ref);
  const labels = entries.map((entry) => entry.label);
  const disabledIndices = entries.flatMap((entry, index) =>
    entry.disabled ? [index] : [],
  );
  const rove = useRovingFocus(refs, activeIndex, setActiveIndex, {
    orientation: "vertical",
    disabledIndices,
  });
  const match = useCallback(
    (query: string) => {
      const index = labels.findIndex((label) =>
        label.toLocaleLowerCase().startsWith(query.toLocaleLowerCase()),
      );
      if (index >= 0) {
        setActiveIndex(index);
        refs[index]?.current?.focus();
      }
    },
    [labels, refs],
  );
  const typeahead = useTypeahead(match);
  const context = useMemo<MenuRegistry>(
    () => ({
      refs,
      labels,
      activeIndex,
      setActiveIndex,
      disabledIndices,
      register,
      close: () => popover.setOpen(false),
    }),
    [activeIndex, disabledIndices, labels, popover, register, refs],
  );
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    rove(event);
    typeahead(event);
    props.onKeyDown?.(event);
  };
  return (
    <MenuContext.Provider value={context}>
      <PopoverParts.Content
        {...props}
        role="menu"
        aria-orientation="vertical"
        initialFocus="first"
        data-rcl-menu-content
        onKeyDown={onKeyDown}
      >
        {children}
      </PopoverParts.Content>
    </MenuContext.Provider>
  );
}

interface MenuItemProps {
  children: ReactNode;
  onSelect?: () => void;
  disabled?: boolean;
  shortcut?: string;
}

function MenuItemBase({
  children,
  onSelect,
  disabled = false,
  shortcut,
  role = "menuitem",
  indicator = "",
  checked,
  kind = "item",
}: MenuItemProps & {
  role?: "menuitem" | "menuitemcheckbox" | "menuitemradio";
  indicator?: string;
  checked?: boolean;
  kind?: string;
}) {
  const menu = useMenuContext();
  const id = useId();
  const ref = useRef<HTMLButtonElement>(null);
  const label = typeof children === "string" ? children : "Menu item";
  const [index, setIndex] = useState(-1);
  useEffect(
    () => setIndex(menu.register(id, label, ref, disabled)),
    [disabled, id, label, menu],
  );
  const active = index === menu.activeIndex;
  return (
    <button
      ref={ref}
      type="button"
      role={role}
      data-rcl-menu-item={kind === "item" || undefined}
      data-rcl-menu-checkbox={kind === "checkbox" || undefined}
      data-rcl-menu-radio={kind === "radio" || undefined}
      data-disabled={disabled || undefined}
      disabled={disabled}
      tabIndex={active ? 0 : -1}
      aria-checked={role === "menuitem" ? undefined : checked}
      onFocus={() => setIndex(index)}
      onClick={() => {
        if (!disabled) {
          onSelect?.();
          menu.close();
        }
      }}
    >
      <span data-rcl-menu-indicator aria-hidden="true">
        {indicator}
      </span>
      <span>{children}</span>
      {shortcut ? <span data-rcl-menu-shortcut>{shortcut}</span> : null}
    </button>
  );
}

export function MenuItem(props: MenuItemProps) {
  return <MenuItemBase {...props} />;
}

export function MenuCheckboxItem({
  checked = false,
  onCheckedChange,
  ...props
}: MenuItemProps & {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
}) {
  return (
    <MenuItemBase
      {...props}
      kind="checkbox"
      role="menuitemcheckbox"
      checked={checked}
      indicator={checked ? "✓" : ""}
      onSelect={() => onCheckedChange?.(!checked)}
    />
  );
}

export function MenuRadioItem({
  checked = false,
  onCheckedChange,
  ...props
}: MenuItemProps & { checked?: boolean; onCheckedChange?: () => void }) {
  return (
    <MenuItemBase
      {...props}
      kind="radio"
      role="menuitemradio"
      checked={checked}
      indicator={checked ? "●" : "○"}
      onSelect={onCheckedChange}
    />
  );
}

export function MenuLabel({ children }: { children: ReactNode }) {
  return <div data-rcl-menu-label>{children}</div>;
}
export function MenuSeparator() {
  return <div data-rcl-menu-separator role="separator" />;
}

export const MenuParts = {
  Trigger: MenuTrigger,
  Content: MenuContent,
  Item: MenuItem,
  CheckboxItem: MenuCheckboxItem,
  RadioItem: MenuRadioItem,
  Label: MenuLabel,
  Separator: MenuSeparator,
};
