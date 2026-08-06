import { useReducedMotion } from "./useReducedMotion";
export function Default() {
  return (
    <div role="status">{useReducedMotion() ? "reduced" : "full-motion"}</div>
  );
}
