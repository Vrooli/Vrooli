import { useDelayedPending } from "./useDelayedPending";
export function Default() {
  const visible = useDelayedPending(false);
  return <div role="status">{visible ? "pending" : "idle"}</div>;
}
