import { useScrollState } from "./useScrollState";
export function Default() {
  const state = useScrollState(null);
  return <div data-testid="hooks.use-scroll-state" role="status">{state.atStart ? "at-start" : "scrolled"}</div>;
}
