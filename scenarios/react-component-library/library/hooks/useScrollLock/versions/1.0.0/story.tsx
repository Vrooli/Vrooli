import { useScrollLock } from "./useScrollLock";
export function Default() { useScrollLock(false); return <div role="status">unlocked</div>; }
