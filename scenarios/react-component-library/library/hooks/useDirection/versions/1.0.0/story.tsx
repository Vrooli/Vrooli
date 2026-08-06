import { useDirection } from "./useDirection";
export function Default() { return <div role="status">{useDirection()}</div>; }
