/**
 * @libraryId react-component-library:Switch
 * @displayName Switch
 * @description The binary settings control with spring-driven thumb motion, label association, loading state for remotely persisted settings, and read-only and disabled treatments.
 * @version 1.0.12
 * @tags ["controls","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.switch */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  SelectionControl,
  type SelectionControlProps,
  type SelectionLabelPlacement,
} from "@vrooli/react-component-library/SelectionControl/1";

export type { SelectionLabelPlacement };

/**
 * Omit `label` for a bare switch — a settings row, table cell, or toolbar that
 * already owns its labelling should pass `aria-label` instead of rendering a
 * second copy of the text. `labelPlacement="end"` puts the switch on the
 * trailing edge, which is the usual settings-row arrangement.
 */
export type SwitchProps = Omit<SelectionControlProps, "kind">;

export const Switch = withClassName(function Switch(props: SwitchProps) {
  return <SelectionControl data-testid="controls.switch" {...props} kind="switch" />;
});
