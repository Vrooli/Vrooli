/**
 * @libraryId react-component-library:Checkbox
 * @displayName Checkbox
 * @description
 * @version 1.0.5
 * @tags ["controls","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.checkbox */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  SelectionControl,
  type SelectionControlProps,
} from "@vrooli/react-component-library/SelectionControl/1.0.0";

export type CheckboxProps = Omit<SelectionControlProps, "kind">;

export const Checkbox = withClassName(function Checkbox(props: CheckboxProps) {
  return <SelectionControl data-testid="controls.checkbox" {...props} kind="checkbox" />;
});
