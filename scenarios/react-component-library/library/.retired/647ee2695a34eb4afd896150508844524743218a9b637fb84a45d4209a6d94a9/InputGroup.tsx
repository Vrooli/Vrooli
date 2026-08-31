/**
 * @libraryId react-component-library:InputGroup
 * @displayName Input Group
 * @description Field shell that owns border, background and focus ring so an input, its adornments and its buttons render as one control.
 * @version 1.2.0
 * @tags ["form","control","token-bound","compound"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:InputGroup */
import {
  createContext,
  forwardRef,
  useContext,
  useMemo,
  type ButtonHTMLAttributes,
  type ForwardRefExoticComponent,
  type HTMLAttributes,
  type ReactNode,
  type RefAttributes,
} from "react";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

import { inputGroupStyles } from "./styles";

export const INPUT_GROUP_SHAPES = ["rounded", "pill"] as const;
export const INPUT_GROUP_SIZES = ["sm", "md", "lg"] as const;
export const INPUT_GROUP_TONES = ["default", "invalid"] as const;
export const INPUT_GROUP_SIDES = ["leading", "trailing"] as const;
export const INPUT_GROUP_ALIGNMENTS = ["start", "center", "end"] as const;
export const INPUT_GROUP_EMPHASES = ["quiet", "solid"] as const;

export type InputGroupShape = (typeof INPUT_GROUP_SHAPES)[number];
export type InputGroupSize = (typeof INPUT_GROUP_SIZES)[number];
export type InputGroupTone = (typeof INPUT_GROUP_TONES)[number];
export type InputGroupSide = (typeof INPUT_GROUP_SIDES)[number];
export type InputGroupAlign = (typeof INPUT_GROUP_ALIGNMENTS)[number];
export type InputGroupEmphasis = (typeof INPUT_GROUP_EMPHASES)[number];

interface InputGroupContextValue {
  disabled: boolean;
  size: InputGroupSize;
  testId?: string;
}

const InputGroupContext = createContext<InputGroupContextValue>({
  disabled: false,
  size: "md",
});

/** Parts read the group's size and disabled state rather than repeating them. */
export function useInputGroup(): InputGroupContextValue {
  return useContext(InputGroupContext);
}

/**
 * The shell that owns the field's border, background, radius and focus ring,
 * so an input and the controls attached to it read and behave as one control.
 *
 * Geometry can be retuned per call site through two custom properties set in
 * `style`, which is the supported seam — overriding the component's own rules
 * is not:
 *
 * ```tsx
 * <InputGroup style={{ "--rcl-group-pad": "var(--space-2xs)" } as CSSProperties}>
 * ```
 *
 * - `--rcl-group-pad` — inline gutter shared by the control and its adornments
 * - `--rcl-group-control` — minimum block size of the control row
 */
export interface InputGroupProps
  extends Omit<HTMLAttributes<HTMLDivElement>, "children"> {
  children: ReactNode;
  /**
   * `rounded` matches `ControlBase`'s own default, so a field and a button
   * beside it agree without either overriding the other.
   *
   * `pill` means *as round as one row allows*, not a 9999px stadium: the
   * radius is half the resting row, so it is clamped to a true pill at rest
   * and stays a well-rounded rectangle once the field grows. A literal
   * stadium is only ever correct on a control that cannot grow, where the two
   * render identically anyway — so nothing is given up by never emitting one.
   */
  shape?: InputGroupShape;
  size?: InputGroupSize;
  tone?: InputGroupTone;
  /**
   * Presentation and pointer-blocking for the whole group. Controls inside
   * still carry their own `disabled` for semantics — this does not forge it,
   * because a group cannot know which of its parts should report as disabled
   * to assistive technology.
   */
  disabled?: boolean;
  /**
   * The field has grown past a single row, so the actions move to a row of
   * their own beneath it and the text takes the full width.
   *
   * The group cannot detect this for itself — only the host knows what a
   * "row" means for its control. Two ways to drive it, and callers should
   * pick one rather than both: pass this prop, or write `data-grown="true"`
   * straight to the group's node through a ref. The second exists because the
   * caller that needs this most is a composer already measuring its own height
   * on every keystroke, and routing that measurement through React state would
   * reintroduce a per-keystroke render. React leaves an attribute it never
   * rendered alone, so an imperatively-set one survives re-renders.
   */
  grown?: boolean;
  testId?: string;
}

/**
 * Written against the prop interfaces rather than `typeof` the parts, because
 * the parts are declared further down the file and a `typeof` forward
 * reference to a `const` is an error even in type position.
 */
type InputGroupComponent = ForwardRefExoticComponent<
  InputGroupProps & RefAttributes<HTMLDivElement>
> & {
  Action: ForwardRefExoticComponent<
    InputGroupActionProps & RefAttributes<HTMLDivElement>
  >;
  Adornment: ForwardRefExoticComponent<
    InputGroupAdornmentProps & RefAttributes<HTMLSpanElement>
  >;
  Field: ForwardRefExoticComponent<
    InputGroupFieldProps & RefAttributes<HTMLDivElement>
  >;
  Segment: ForwardRefExoticComponent<
    InputGroupSegmentProps & RefAttributes<HTMLButtonElement>
  >;
};

/**
 * The stamp plugin resolves an asset's owned root from the declaration whose
 * name matches the entry file, so this must be `InputGroup` and must be the
 * declaration that carries the JSX. The parts are attached to it after they
 * are defined, below.
 */
export const InputGroup = forwardRef<HTMLDivElement, InputGroupProps>(
  function InputGroup(
    {
      children,
      className,
      disabled = false,
      grown,
      shape = "rounded",
      size = "md",
      style,
      testId,
      tone = "default",
      ...rest
    },
    ref,
  ) {
    useLibraryStyleSheet("input-group-1-2-0", inputGroupStyles);
    const context = useMemo<InputGroupContextValue>(
      () => ({ disabled, size, testId }),
      [disabled, size, testId],
    );
    return (
      <InputGroupContext.Provider value={context}>
        <div
          {...rest}
          ref={ref}
          className={className}
          style={style}
          data-testid={testId ?? "forms.input-group"}
          data-rcl-input-group="true"
          data-shape={shape}
          data-size={size}
          data-tone={tone}
          data-disabled={disabled ? "true" : undefined}
          data-grown={grown ? "true" : undefined}
          aria-disabled={disabled ? true : undefined}
        >
          {children}
        </div>
      </InputGroupContext.Provider>
    );
  },
) as InputGroupComponent;

export interface InputGroupFieldProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  testId?: string;
}

/**
 * Optional wrapper around the control. A group works without it — the
 * neutralising rules are descendant selectors, not child selectors — but a
 * field gives a suffix somewhere to sit against the value and gives an
 * overlay a box to register against.
 */
export const InputGroupField = forwardRef<HTMLDivElement, InputGroupFieldProps>(
  function InputGroupField(
    { children, className, style, testId, ...rest },
    ref,
  ) {
    const { testId: groupTestId } = useInputGroup();
    return (
      <div
        {...rest}
        ref={ref}
        className={className}
        style={style}
        data-testid={
          testId ?? (groupTestId ? `${groupTestId}-field` : undefined)
        }
        data-rcl-input-group-field="true"
      >
        {children}
      </div>
    );
  },
);

export interface InputGroupAdornmentProps
  extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  side?: InputGroupSide;
  testId?: string;
}

/**
 * A non-interactive marking inside the border: a currency symbol, a unit, a
 * shell prompt, a search glyph. Placed as a sibling of the field it hugs the
 * border; placed inside the field it hugs the value.
 */
export const InputGroupAdornment = forwardRef<
  HTMLSpanElement,
  InputGroupAdornmentProps
>(function InputGroupAdornment(
  { children, className, side = "leading", style, testId, ...rest },
  ref,
) {
  const { testId: groupTestId } = useInputGroup();
  return (
    <span
      {...rest}
      ref={ref}
      className={className}
      style={style}
      data-testid={
        testId ?? (groupTestId ? `${groupTestId}-adornment-${side}` : undefined)
      }
      data-rcl-input-group-adornment="true"
      data-side={side}
      aria-hidden="true"
    >
      {children}
    </span>
  );
});

export interface InputGroupActionProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  /**
   * Where the action sits when the field is taller than one line. `center` is
   * correct for a fixed-height field and wrong for a composer that grows —
   * at four lines a centred button floats away from both the text and the
   * keyboard, which is why chat surfaces pin theirs to the bottom. On a
   * single line `center` and `end` render identically.
   */
  align?: InputGroupAlign;
  testId?: string;
}

/**
 * Wrapper for a control that keeps chrome of its own — typically an
 * `IconButton` — floating inside the field's gutter rather than beside it.
 */
export const InputGroupAction = forwardRef<
  HTMLDivElement,
  InputGroupActionProps
>(function InputGroupAction(
  { align = "center", children, className, style, testId, ...rest },
  ref,
) {
  const { testId: groupTestId } = useInputGroup();
  return (
    <div
      {...rest}
      ref={ref}
      className={className}
      style={style}
      data-testid={
        testId ?? (groupTestId ? `${groupTestId}-action` : undefined)
      }
      data-rcl-input-group-action="true"
      data-align={align}
    >
      {children}
    </div>
  );
});

export interface InputGroupSegmentProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  side?: InputGroupSide;
  emphasis?: InputGroupEmphasis;
  testId?: string;
}

/**
 * A full-height button that shares the group's outer border, divided from the
 * field by a hairline and squared on the inside corners. Use it when the
 * control is a peer of the field — a stepper, a unit picker, a submit — rather
 * than an ornament floating within it.
 */
export const InputGroupSegment = forwardRef<
  HTMLButtonElement,
  InputGroupSegmentProps
>(function InputGroupSegment(
  {
    children,
    className,
    disabled,
    emphasis = "quiet",
    side = "trailing",
    style,
    testId,
    type = "button",
    ...rest
  },
  ref,
) {
  const { disabled: groupDisabled, testId: groupTestId } = useInputGroup();
  const isDisabled = disabled ?? groupDisabled;
  return (
    <button
      {...rest}
      ref={ref}
      type={type}
      disabled={isDisabled}
      className={className}
      style={style}
      data-testid={
        testId ?? (groupTestId ? `${groupTestId}-segment-${side}` : undefined)
      }
      data-rcl-input-group-segment="true"
      data-side={side}
      data-emphasis={emphasis}
    >
      {children}
    </button>
  );
});

// Both spellings are supported deliberately: the dotted form reads well at a
// call site composing four parts, and the flat named exports stay
// tree-shakeable and survive the package's `export *` re-export barrels.
InputGroup.Action = InputGroupAction;
InputGroup.Adornment = InputGroupAdornment;
InputGroup.Field = InputGroupField;
InputGroup.Segment = InputGroupSegment;
