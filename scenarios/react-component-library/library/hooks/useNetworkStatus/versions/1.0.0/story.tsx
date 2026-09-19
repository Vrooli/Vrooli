import { useNetworkStatus } from "./useNetworkStatus";
export function Default() {
  return <div data-testid="hooks.use-network-status" role="status">{useNetworkStatus() ? "online" : "offline"}</div>;
}
