/**
 * @libraryId react-component-library:Checkbox
 * @displayName Checkbox
 * @description A native checkbox with a composed label, description, mixed state, error semantics, and token-bound focus treatment.
 * @version 1.0.2
 * @tags ["controls","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.checkbox */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import {
  SelectionControl,
  type SelectionControlProps,
} from "../../../../primitives/SelectionControl/versions/1.0.0/SelectionControl";

export type CheckboxProps = Omit<SelectionControlProps, "kind">;

export const Checkbox = withClassName(function Checkbox(props: CheckboxProps) {
  return <SelectionControl {...props} kind="checkbox" />;
});
