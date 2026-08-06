import { useRef } from "react";
import { useFocusTrap } from "./useFocusTrap";

export function Default({ args }: StoryHarnessProps<{ active: boolean }>) {
  const ref = useRef<HTMLDivElement>(null);
  useFocusTrap(args.active, ref);
  return (
    <div ref={ref}>
      <button type="button">First</button>
      <div role="status">Focus contained</div>
      <button type="button">Last</button>
    </div>
  );
}
