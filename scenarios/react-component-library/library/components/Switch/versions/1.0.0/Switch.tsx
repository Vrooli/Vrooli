/** @vrooliComponentSource controls.switch */
import {
  SelectionControl,
  type SelectionControlProps,
} from "../../../../primitives/SelectionControl/versions/1.0.0/SelectionControl";

export type SwitchProps = Omit<SelectionControlProps, "kind">;

export function Switch(props: SwitchProps) {
  return <SelectionControl {...props} kind="switch" />;
}
