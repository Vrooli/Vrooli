import { useSwipe } from "./useSwipe";
export function Default({ log }: StoryHarnessProps) {
  return (
    <div
      data-testid="hooks.use-swipe"
      {...useSwipe({ direction: "down", onCommit: () => log("swipe", "down") })}
    >
      Swipe surface
    </div>
  );
}
