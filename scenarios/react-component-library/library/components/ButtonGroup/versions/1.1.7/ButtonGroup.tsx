/**
 * @libraryId react-component-library:ButtonGroup
 * @displayName ButtonGroup
 * @version 1.1.7
 * @tags ["control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:ButtonGroup */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import type { HTMLAttributes } from "react";

const styles = `
[data-rcl-button-group] { display: inline-flex; box-sizing: border-box; min-inline-size: 0; align-items: center; flex-wrap: wrap; gap: var(--space-xs); max-inline-size: 100%; }
[data-rcl-button-group] > * { min-inline-size: 0; }
[data-rcl-button-group][data-align="start"] { justify-content: flex-start; }
[data-rcl-button-group][data-align="center"] { justify-content: center; }
[data-rcl-button-group][data-align="end"] { justify-content: flex-end; }
@media (max-width: 36rem) { [data-rcl-button-group][data-collapse="sm"] { inline-size: 100%; } [data-rcl-button-group][data-collapse="sm"] > * { flex: 1 1 auto; } }
`;

export type ButtonGroupAlign = "start" | "center" | "end";
export type ButtonGroupCollapse = "never" | "sm";

export const ButtonGroup = withClassName(function ButtonGroup({
  children,
  label,
  align = "start",
  collapse = "sm",
  className,
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  label?: string;
  children?: ReactNode;
  align?: ButtonGroupAlign;
  collapse?: ButtonGroupCollapse;
}) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("controls.button-group.actions", "Actions");
  return (
    <>
      <StyleSheet name="buttongroup-1-1-5-1" css={styles} />
      <div
        data-testid="controls.button-group"
        role="group"
        aria-label={label}
        data-rcl-button-group
        data-align={align}
        data-collapse={collapse}
        className={className}
        style={style}
        {...props}
      >
        {children}
      </div>
    </>
  );
});
