import { useScrollState } from "./useScrollState";
export function Default() {
  const state = useScrollState(null);
  return <div role="status">{state.atStart ? "at-start" : "scrolled"}</div>;
}
