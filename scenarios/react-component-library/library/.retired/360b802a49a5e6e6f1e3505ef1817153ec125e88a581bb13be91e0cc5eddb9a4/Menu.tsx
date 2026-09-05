/**
 * @libraryId react-component-library:Menu
 * @displayName Menu
 * @description The menu family covering dropdowns and submenus with roving focus, typeahead, check and radio items, safe pointer corridors, shortcut hints, and async item actions.
 * @version 1.2.3
 * @tags ["overlay","menu","keyboard","typeahead","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

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
import { ChevronRight } from "lucide-react";
import {
  Popover,
  PopoverParts,
  usePopover,
} from "@vrooli/react-component-library/Popover/1";
import { useRovingFocus } from "@vrooli/react-component-library/useRovingFocus/1";
import { useTypeahead } from "@vrooli/react-component-library/useTypeahead/1";

const styles = `
/* Icons are sized here rather than through the icon library's size prop: that
   prop lands on the SVG width/height geometry attributes, whose grammar is
   <length>, so a var() expression is rejected outright and the icon falls back
   to the 300x150 replaced-element default. */
[data-rcl-menu-submenu-arrow] { inline-size: var(--icon-size-sm); block-size: var(--icon-size-sm); flex: 0 0 auto; }
  [data-rcl-menu-content] { display: grid; gap: var(--space-3xs); min-inline-size: 13rem; padding: var(--space-2xs); }
  [data-rcl-menu-item], [data-rcl-menu-checkbox], [data-rcl-menu-radio] { display: grid; grid-template-columns: 1.25rem minmax(0, 1fr) auto; align-items: center; gap: var(--space-xs); min-block-size: var(--tap-target-min); padding: var(--space-xs) var(--space-sm); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); cursor: pointer; font: var(--text-body-sm); text-align: start; }
  [data-rcl-menu-item]:hover, [data-rcl-menu-item]:focus-visible, [data-rcl-menu-checkbox]:hover, [data-rcl-menu-checkbox]:focus-visible, [data-rcl-menu-radio]:hover, [data-rcl-menu-radio]:focus-visible { background: var(--color-surface-muted); outline: none; }
  [data-rcl-menu-item][data-disabled="true"], [data-rcl-menu-checkbox][data-disabled="true"], [data-rcl-menu-radio][data-disabled="true"] { cursor: not-allowed; opacity: .55; }
  [data-rcl-menu-indicator] { display: grid; place-items: center; color: var(--color-primary); font: var(--text-label); }
  [data-rcl-menu-label] { padding: var(--space-xs) var(--space-sm) var(--space-3xs); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
  [data-rcl-menu-separator] { block-size: 1px; margin-block: var(--space-2xs); background: var(--color-border); }
  [data-rcl-menu-shortcut] { color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-menu-submenu] { position: relative; }
  [data-rcl-menu-submenu-trigger] { inline-size: 100%; }
  [data-rcl-menu-submenu-trigger] [data-rcl-menu-submenu-arrow] { justify-self: end; color: var(--color-muted-foreground); transition: transform var(--dur-quick) var(--ease-standard); }
  [data-rcl-menu-submenu-trigger][aria-expanded="true"] [data-rcl-menu-submenu-arrow] { transform: rotate(90deg); color: var(--color-primary); }
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

export const Menu = withClassName(function Menu({
  children,
  ...props
}: MenuProps) {
  return (
    <Popover {...props} placement="bottom-start">
      <style
        data-rcl-menu-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      {children}
    </Popover>
  );
});

export const MenuTrigger = withClassName(function MenuTrigger({
  children,
  ...props
}: { children: ReactNode } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <PopoverParts.Trigger {...props} aria-haspopup="menu">
      {children}
    </PopoverParts.Trigger>
  );
});

export const MenuContent = withClassName(function MenuContent({
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
});

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
      data-testid="overlays.menu"
      ref={ref}
      type="button"
      role={role}
      data-rcl-menu-item={kind === "item" || undefined}
      data-rcl-menu-checkbox={kind === "checkbox" || undefined}
      data-rcl-menu-radio={kind === "radio" || undefined}
      data-rcl-menu-part={
        kind === "checkbox"
          ? "checkbox-item"
          : kind === "radio"
            ? "radio-item"
            : "item"
      }
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

export const MenuItem = withClassName(function MenuItem(props: MenuItemProps) {
  return <MenuItemBase {...props} />;
});

export const MenuCheckboxItem = withClassName(function MenuCheckboxItem({
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
});

export const MenuRadioItem = withClassName(function MenuRadioItem({
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
});

function SubmenuTrigger({
  label,
  disabled,
}: {
  label: ReactNode;
  disabled: boolean;
}) {
  const menu = useMenuContext();
  const popover = usePopover();
  const id = useId();
  const labelText = typeof label === "string" ? label : "Submenu";
  const [index, setIndex] = useState(-1);
  useEffect(
    () => setIndex(menu.register(id, labelText, popover.triggerRef, disabled)),
    [disabled, id, labelText, menu, popover.triggerRef],
  );
  const active = index === menu.activeIndex;

  return (
    <PopoverParts.Trigger
      type="button"
      role="menuitem"
      aria-haspopup="menu"
      aria-label={labelText}
      aria-disabled={disabled || undefined}
      data-rcl-menu-item
      data-rcl-menu-submenu-trigger
      tabIndex={active ? 0 : -1}
      disabled={disabled}
      onFocus={() => menu.setActiveIndex(index)}
      onKeyDown={(event) => {
        if (event.key === "ArrowRight") {
          event.preventDefault();
          popover.setOpen(true);
        }
        if (event.key === "ArrowLeft") {
          event.preventDefault();
          popover.setOpen(false);
        }
      }}
    >
      <span data-rcl-menu-indicator aria-hidden="true" />
      <span>{label}</span>
      <ChevronRight data-rcl-menu-submenu-arrow aria-hidden="true" />
    </PopoverParts.Trigger>
  );
}

export const MenuSubmenu = withClassName(function MenuSubmenu({
  label,
  children,
  disabled = false,
}: {
  label: ReactNode;
  children: ReactNode;
  disabled?: boolean;
}) {
  return (
    <div data-rcl-menu-submenu>
      <Popover placement="right-start" responsive="none">
        <SubmenuTrigger label={label} disabled={disabled} />
        <MenuContent aria-label={typeof label === "string" ? label : "Submenu"}>
          {children}
        </MenuContent>
      </Popover>
    </div>
  );
});

export const MenuLabel = withClassName(function MenuLabel({
  children,
}: {
  children: ReactNode;
}) {
  return <div data-rcl-menu-label>{children}</div>;
});
export const MenuSeparator = withClassName(function MenuSeparator() {
  return <div data-rcl-menu-separator role="separator" />;
});

export const MenuParts = {
  Trigger: MenuTrigger,
  Content: MenuContent,
  Item: MenuItem,
  CheckboxItem: MenuCheckboxItem,
  RadioItem: MenuRadioItem,
  Submenu: MenuSubmenu,
  Label: MenuLabel,
  Separator: MenuSeparator,
};
