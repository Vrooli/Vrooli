import { useDirection } from "./useDirection";
export function Default() {
  return (
    <div data-testid="hooks.use-direction" role="status">
      {useDirection()}
    </div>
  );
}
