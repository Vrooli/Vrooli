/**
 * @libraryId react-component-library:Checkbox
 * @displayName Checkbox
 * @description The binary or indeterminate selection control with animated state marks, validation treatment, group membership, and robust keyboard and screen-reader behavior.
 * @version 1.0.6
 * @tags ["controls","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.checkbox */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  SelectionControl,
  type SelectionControlProps,
} from "@vrooli/react-component-library/SelectionControl/1";

export type CheckboxProps = Omit<SelectionControlProps, "kind">;

export const Checkbox = withClassName(function Checkbox(props: CheckboxProps) {
  return <SelectionControl data-testid="controls.checkbox" {...props} kind="checkbox" />;
});
