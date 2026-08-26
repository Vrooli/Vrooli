/**
 * @libraryId react-component-library:Switch
 * @displayName Switch
 * @description A native checkbox-backed switch with a clear on/off state, supporting context, and reduced-motion-safe thumb transition.
 * @version 1.0.2
 * @tags ["controls","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.switch */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import {
  SelectionControl,
  type SelectionControlProps,
} from "../../../../primitives/SelectionControl/versions/1.0.0/SelectionControl";

export type SwitchProps = Omit<SelectionControlProps, "kind">;

export const Switch = withClassName(function Switch(props: SwitchProps) {
  return <SelectionControl {...props} kind="switch" />;
});
