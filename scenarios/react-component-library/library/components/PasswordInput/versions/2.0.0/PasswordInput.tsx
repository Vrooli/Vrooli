/**
 * @libraryId react-component-library:PasswordInput
 * @displayName Password Input
 * @description Secret input with reveal and paste-safe handling
 * @version 2.0.0
 * @tags ["forms","secret","setup"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PasswordInput */
import {
  forwardRef,
  useCallback,
  useId,
  useState,
  type FocusEvent,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup/1";

/**
 * 2.0.0 is a rebuild, not a revision.
 *
 * 1.x rendered a bare `<input>` beside a bare `<button>Show</button>` inside a
 * `<label>` carrying two inline style objects. It read no design token, drew no
 * border, accepted no `id`, and wired no `aria-describedby`, so in an adopting
 * app it rendered as an unstyled black box next to plain blue text while every
 * field around it had a border and a height. It also exported only a default,
 * which no other control in this library does.
 *
 * What changed, and why each part is a composition rather than new chrome:
 *
 *  - The field's border, background, radius and focus ring belong to
 *    `InputGroup`, which is the asset that exists to put chrome one level above
 *    the control. That is what lets the reveal button sit *inside* the outline
 *    instead of beside it — the single visual difference that made 1.x read as
 *    foreign.
 *  - The text box is `Input`, so a secret field inherits the same height,
 *    placeholder colour, and the 16px coarse-pointer floor that stops iOS
 *    zooming the viewport on focus.
 *  - The reveal control is `IconButton`, so it gets the tap target, the
 *    `aria-pressed` toggle semantics, and the icon morph the library already
 *    settled, rather than a text button that changes its own label.
 *  - Label, description, error and required marks belong to `FormField`, and
 *    are used only when a `label` is passed. Passing this component as
 *    `FormField`'s own `control` also works: the props `FormField` clones onto
 *    its control — `id`, `disabled`, `required`, `aria-invalid`,
 *    `aria-describedby` — all forward to the inner `<input>`.
 *
 * Two behaviours are the component's own, because nothing else owns them.
 *
 * `revealable={false}` is for a value the operator may replace but must never
 * read back: a credential being pushed to a node, a token held by a broker.
 * Rendering no reveal control at all is the honest expression of that, and it
 * is different from `disabled`, which would also block replacement.
 *
 * The caps-lock warning exists because a masked field is the one place where
 * the operating system's own indicator is not enough — the user cannot see the
 * value to notice it is wrong, and the error they get back is an
 * authentication failure that names nothing. It is announced politely rather
 * than as an alert: it is a hint about the keyboard, not a validation verdict.
 */

export const PASSWORD_INPUT_SIZES = ["sm", "md", "lg"] as const;
export const PASSWORD_INPUT_TONES = ["default", "invalid"] as const;

export type PasswordInputSize = (typeof PASSWORD_INPUT_SIZES)[number];
export type PasswordInputTone = (typeof PASSWORD_INPUT_TONES)[number];

type NativeInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "type" | "size" | "value" | "defaultValue" | "onChange"
>;

export interface PasswordInputProps extends NativeInputProps {
  value?: string;
  defaultValue?: string;
  /**
   * The DOM event handler, matching `Input`. Use `onValueChange` when only the
   * string is wanted; both fire, and a caller may use either or both.
   */
  onChange?: InputHTMLAttributes<HTMLInputElement>["onChange"];
  onValueChange?: (value: string) => void;
  /**
   * Renders the control inside a `FormField`. Omit it to get the bare control,
   * which is what to pass when the caller supplies its own `FormField`.
   */
  label?: ReactNode;
  description?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  optionalLabel?: ReactNode;
  size?: PasswordInputSize;
  tone?: PasswordInputTone;
  /** A write-only secret renders no reveal control at all. */
  revealable?: boolean;
  /** Controlled reveal state; omit for the component's own. */
  revealed?: boolean;
  defaultRevealed?: boolean;
  onRevealChange?: (revealed: boolean) => void;
  revealLabel?: string;
  concealLabel?: string;
  /** Set false where a physical keyboard is not in play and the notice is noise. */
  capsLockWarning?: boolean;
  capsLockLabel?: string;
  /**
   * Slot for a strength indicator, rendered under the field and wired into the
   * control's `aria-describedby`. A slot rather than a built-in meter because
   * password policy is the product's to state, never the library's to assume.
   */
  strength?: ReactNode;
  className?: string;
  testId?: string;
}

const styles = `
[data-rcl-password] { display: grid; gap: var(--space-2xs, 8px); min-inline-size: 0; }
[data-rcl-password-note] {
  display: flex; align-items: center; gap: var(--space-2xs, 8px);
  color: var(--color-warning, #b45309);
  font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans));
}
[data-rcl-password-note-mark] {
  display: inline-grid; place-items: center; flex: 0 0 auto;
  inline-size: 1.125rem; block-size: 1.125rem;
  border: var(--border-hairline, 1px) solid currentColor; border-radius: 50%;
  font: 700 .6875rem/1 system-ui, sans-serif;
}
[data-rcl-password-strength] { min-inline-size: 0; }
/* The masked dots are wider than the glyphs that replace them, so a field that
   does not letter-space its secret visibly reflows on every reveal. Holding the
   tracking constant across both states keeps the reveal a change of characters
   rather than a change of layout. */
[data-rcl-password-control] { letter-spacing: .04em; }
[data-rcl-password-control]:placeholder-shown { letter-spacing: normal; }
`;

function mergeIds(...ids: Array<string | undefined | false>): string | undefined {
  return ids.filter(Boolean).join(" ") || undefined;
}

const PasswordControl = forwardRef<HTMLInputElement, PasswordInputProps>(function PasswordControl(
  {
    value,
    defaultValue,
    onChange,
    onValueChange,
    label: _label,
    description: _description,
    error,
    required,
    optionalLabel: _optionalLabel,
    size = "md",
    tone,
    revealable = true,
    revealed,
    defaultRevealed = false,
    onRevealChange,
    revealLabel = "Show password",
    concealLabel = "Hide password",
    capsLockWarning = true,
    capsLockLabel = "Caps Lock is on",
    strength,
    className,
    testId,
    disabled,
    name = "password",
    autoComplete = "current-password",
    "aria-describedby": ariaDescribedBy,
    "aria-invalid": ariaInvalid,
    onKeyDown,
    onKeyUp,
    onBlur,
    ...rest
  },
  ref,
) {
  const generatedId = useId().replace(/:/g, "");
  const noteId = `rcl-password-${generatedId}-caps`;
  const strengthId = `rcl-password-${generatedId}-strength`;

  const [uncontrolledRevealed, setUncontrolledRevealed] = useState(defaultRevealed);
  const isRevealed = revealed ?? uncontrolledRevealed;
  const [capsLockOn, setCapsLockOn] = useState(false);

  const toggleReveal = useCallback(() => {
    const next = !isRevealed;
    if (revealed === undefined) setUncontrolledRevealed(next);
    onRevealChange?.(next);
  }, [isRevealed, onRevealChange, revealed]);

  /**
   * `getModifierState` is read on both keydown and keyup because the state
   * that matters is the one *after* the key that just fired: pressing Caps
   * Lock itself reports the outgoing state on keydown and the incoming state
   * on keyup, so reading only one of the two leaves the notice inverted for
   * exactly the keystroke that changes it.
   */
  const readCapsLock = useCallback(
    (event: KeyboardEvent<HTMLInputElement>) => {
      if (!capsLockWarning) return;
      if (typeof event.getModifierState !== "function") return;
      setCapsLockOn(event.getModifierState("CapsLock"));
    },
    [capsLockWarning],
  );

  const resolvedTone: PasswordInputTone = tone ?? (error ? "invalid" : "default");
  const showCapsLock = capsLockWarning && capsLockOn && !disabled;
  const describedBy = mergeIds(
    ariaDescribedBy,
    showCapsLock && noteId,
    strength ? strengthId : undefined,
  );

  return (
    <div data-rcl-password="true" className={className} data-testid={testId}>
      <StyleSheet name="password-input-2" css={styles} />
      <InputGroup
        size={size}
        tone={resolvedTone}
        disabled={disabled}
        testId={testId ? `${testId}-group` : undefined}
      >
        <Input
          {...rest}
          ref={ref}
          name={name}
          type={isRevealed ? "text" : "password"}
          autoComplete={autoComplete}
          value={value}
          defaultValue={defaultValue}
          disabled={disabled}
          required={required}
          aria-invalid={ariaInvalid ?? (error ? true : undefined)}
          aria-describedby={describedBy}
          data-rcl-password-control="true"
          data-testid={testId ? `${testId}-input` : "forms.password-input"}
          onChange={(event) => {
            onChange?.(event);
            onValueChange?.(event.target.value);
          }}
          onKeyDown={(event) => {
            readCapsLock(event);
            onKeyDown?.(event);
          }}
          onKeyUp={(event) => {
            readCapsLock(event);
            onKeyUp?.(event);
          }}
          onBlur={(event: FocusEvent<HTMLInputElement>) => {
            // The notice describes a keyboard the field no longer has.
            setCapsLockOn(false);
            onBlur?.(event);
          }}
        />
        {revealable && (
          <InputGroup.Action>
            <IconButton
              aria-label={isRevealed ? concealLabel : revealLabel}
              selected={isRevealed}
              disabled={disabled}
              size={size === "lg" ? "md" : "sm"}
              // Both states render the same component, so the morph has no
              // way to tell them apart from the child's identity alone and
              // would skip every swap. Naming the glyph is what makes the
              // lid animate instead of cutting.
              iconKey={isRevealed ? "eyeOff" : "eye"}
              tabIndex={-1}
              data-testid={testId ? `${testId}-reveal` : "forms.password-input-reveal"}
              onClick={toggleReveal}
            >
              <Icon name={isRevealed ? "eyeOff" : "eye"} size="sm" />
            </IconButton>
          </InputGroup.Action>
        )}
      </InputGroup>
      {strength && (
        <div id={strengthId} data-rcl-password-strength>
          {strength}
        </div>
      )}
      {showCapsLock && (
        <p id={noteId} data-rcl-password-note aria-live="polite">
          <span data-rcl-password-note-mark aria-hidden="true">
            !
          </span>
          {capsLockLabel}
        </p>
      )}
    </div>
  );
});

/**
 * The reveal button carries `tabIndex={-1}` deliberately. It sits between the
 * secret and whatever follows it, and a keyboard user tabbing out of a password
 * field is heading for the submit button, not for a control that changes how
 * the value looks. It stays reachable by pointer, by the screen-reader cursor,
 * and by the field's own label association — what it does not do is add a stop
 * to the middle of a login form.
 */
export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(
  function PasswordInput(props, ref) {
    const { label, description, error, required = false, optionalLabel, disabled, id } = props;

    if (label === undefined) {
      return <PasswordControl {...props} ref={ref} />;
    }

    return (
      <FormField
        id={id}
        label={label}
        description={description}
        error={error}
        required={required}
        disabled={disabled}
        {...(optionalLabel === undefined ? {} : { optionalLabel })}
        control={<PasswordControl {...props} ref={ref} label={undefined} />}
      />
    );
  },
);

export default PasswordInput;
