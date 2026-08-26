import { useReducedMotion } from "./useReducedMotion";
export function Default() {
  return (
    <div data-testid="hooks.use-reduced-motion" role="status">{useReducedMotion() ? "reduced" : "full-motion"}</div>
  );
}
