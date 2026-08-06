import { useNetworkStatus } from "./useNetworkStatus";
export function Default() { return <div role="status">{useNetworkStatus() ? "online" : "offline"}</div>; }
