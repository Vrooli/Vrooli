import { useResizeObserver } from "./useResizeObserver";
export function Default() {
  const measured = useResizeObserver();
  return (
    <div ref={measured.ref}>
      <div role="status">{measured.rect ? "measured" : "unmeasured"}</div>
    </div>
  );
}
