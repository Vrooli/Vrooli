import { useScrollLock } from "./useScrollLock";
export function Default() {
  useScrollLock(false);
  return (
    <div data-testid="hooks.use-scroll-lock" role="status">
      unlocked
    </div>
  );
}
