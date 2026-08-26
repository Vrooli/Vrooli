import { useResizeObserver } from "./useResizeObserver";
export function Default() {
  const measured = useResizeObserver();
  return (
    <div data-testid="hooks.use-resize-observer" ref={measured.ref}>
      <div role="status">{measured.rect ? "measured" : "unmeasured"}</div>
    </div>
  );
}
