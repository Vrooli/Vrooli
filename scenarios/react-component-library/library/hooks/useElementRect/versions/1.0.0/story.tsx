import { useElementRect } from "./useElementRect";
export function Default() {
  const rect = useElementRect(null);
  return <div role="status">{rect ? "measured" : "unmeasured"}</div>;
}
