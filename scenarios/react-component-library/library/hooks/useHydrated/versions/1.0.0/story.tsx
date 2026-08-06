import { useHydrated } from "./useHydrated";
export function Default() { return <div role="status">{useHydrated() ? "hydrated" : "hydrating"}</div>; }
