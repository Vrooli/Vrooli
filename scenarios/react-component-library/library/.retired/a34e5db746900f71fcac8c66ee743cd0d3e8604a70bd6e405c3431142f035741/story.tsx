import { useDirection } from "./useDirection";

export function Default() {
  return (
    <div
      data-testid="hooks.use-direction"
      role="status"
      data-direction={useDirection()}
    >
      {useDirection()}
    </div>
  );
}

/**
 * The reason 2.0.0 exists: the document declares `rtl`, and the hook reports it
 * without anything else forcing a render. Under 1.x this rendered `ltr`.
 */
export function RightToLeft() {
  const direction = useDirection();
  return (
    <div dir="rtl">
      <div
        data-testid="hooks.use-direction.rtl"
        role="status"
        data-direction={direction}
      >
        {direction}
      </div>
    </div>
  );
}
