import { useDelayedPending } from "./useDelayedPending";
export function Default() {
  const visible = useDelayedPending(false);
  return <div data-testid="hooks.use-delayed-pending" role="status">{visible ? "pending" : "idle"}</div>;
}
