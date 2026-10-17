import { useElementRect } from "./useElementRect";
export function Default() {
  const rect = useElementRect(null);
  return <div data-testid="hooks.use-element-rect" role="status">{rect ? "measured" : "unmeasured"}</div>;
}
