import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";
import { Search, X } from "lucide-react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { cn } from "../lib/classnames";

export interface SearchFieldProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "value" | "onChange" | "size"> {
  /** Stable test id for the input itself. */
  testId: string;
  /** Stable test id for the bordered group; defaults to `${testId}-group`. */
  groupTestId?: string;
  value: string;
  onValueChange: (value: string) => void;
  placeholder: string;
  /** Accessible name. Falls back to the placeholder. */
  label?: string;
  /** Parts rendered after the field and before the clear action. */
  children?: ReactNode;
  /** The clear affordance. `null` hides it. */
  onClear?: (() => void) | null;
  /** Accessible name for the clear affordance. */
  clearLabel?: string;
  /** A shortcut hint shown at the trailing edge. */
  shortcut?: ReactNode;
  shape?: "rounded" | "pill";
  size?: "sm" | "md" | "lg";
  className?: string;
  inputClassName?: string;
}

/**
 * The one search field in Aquila.
 *
 * The messages toolbar, the reader's find bar and settings all render this
 * control so a search box never looks like a different component depending on
 * where it lives. The border, pill geometry, focus ring and clear affordance
 * belong to the governed RCL `InputGroup`; callers only add the behaviour and
 * the context-specific actions (match stepping, filters, a shortcut hint).
 */
export const SearchField = forwardRef<HTMLInputElement, SearchFieldProps>(function SearchField({
  testId,
  groupTestId,
  value,
  onValueChange,
  placeholder,
  label,
  children,
  onClear,
  clearLabel = "Clear search",
  shortcut,
  shape = "pill",
  size = "md",
  className,
  inputClassName,
  ...inputProps
}, ref) {
  const clear = onClear === null
    ? null
    : (onClear ?? (() => { onValueChange(""); }));
  return (
    <InputGroup
      shape={shape}
      size={size}
      className={cn("min-w-0", className)}
      testId={groupTestId ?? `${testId}-group`}
      style={{
        background: "color-mix(in srgb, var(--color-surface-raised) 80%, transparent)",
        backdropFilter: "blur(4px)",
      }}
    >
      <InputGroup.Adornment side="leading">
        <Search aria-hidden />
      </InputGroup.Adornment>
      <InputGroup.Field>
        <Input
          {...inputProps}
          ref={ref}
          data-testid={testId}
          type="text"
          value={value}
          onChange={(changeEvent) => { onValueChange(changeEvent.target.value); }}
          placeholder={placeholder}
          aria-label={label ?? placeholder}
          className={cn("text-[13px]", inputClassName)}
        />
      </InputGroup.Field>
      {children}
      {clear && value && (
        <InputGroup.Action>
          <IconButton
            data-testid={`${testId}-clear`}
            aria-label={clearLabel}
            title={clearLabel}
            shape="circle"
            surface="ghost"
            size="xs"
            onClick={clear}
          >
            <X />
          </IconButton>
        </InputGroup.Action>
      )}
      {shortcut !== undefined && shortcut !== null && (
        <InputGroup.Action>
          {typeof shortcut === "string"
            ? <kbd className="hidden px-1 font-mono text-[10px] text-wc-text-faint sm:inline">{shortcut}</kbd>
            : shortcut}
        </InputGroup.Action>
      )}
    </InputGroup>
  );
});

export default SearchField;
