import { useFocusVisible } from "./useFocusVisible";
export function Default() {
  return <div role="status">{useFocusVisible() ? "keyboard" : "pointer"}</div>;
}
