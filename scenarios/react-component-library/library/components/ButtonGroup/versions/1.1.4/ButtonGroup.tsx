/**
 * @libraryId react-component-library:ButtonGroup
 * @displayName ButtonGroup
 * @description
 * @version 1.1.4
 * @tags ["control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:ButtonGroup */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
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
  label = useStrings("controls.button-group.actions", "Actions"),
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
  return (
    <>
      <style data-rcl-button-group-styles dangerouslySetInnerHTML={{ __html: styles }} />
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
