import { useHydrated } from "./useHydrated";
export function Default() {
  return <div data-testid="hooks.use-hydrated" role="status">{useHydrated() ? "hydrated" : "hydrating"}</div>;
}
