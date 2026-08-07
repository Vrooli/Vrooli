/** @vrooliComponentSource controls.checkbox */
import {
  SelectionControl,
  type SelectionControlProps,
} from "../../../../primitives/SelectionControl/versions/1.0.0/SelectionControl";

export type CheckboxProps = Omit<SelectionControlProps, "kind">;

export function Checkbox(props: CheckboxProps) {
  return <SelectionControl {...props} kind="checkbox" />;
}
