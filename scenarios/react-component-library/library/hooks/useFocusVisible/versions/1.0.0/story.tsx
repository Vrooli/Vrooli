import { useFocusVisible } from "./useFocusVisible";
export function Default() {
  return <div data-testid="hooks.use-focus-visible" role="status">{useFocusVisible() ? "keyboard" : "pointer"}</div>;
}
