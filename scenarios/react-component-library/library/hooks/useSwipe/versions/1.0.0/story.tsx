import { useSwipe } from "./useSwipe";
export function Default({ log }: StoryHarnessProps) {
  return (
    <div data-testid="hooks.use-swipe" {...useSwipe((direction) => log("swipe", direction))}>
      Swipe surface
    </div>
  );
}
