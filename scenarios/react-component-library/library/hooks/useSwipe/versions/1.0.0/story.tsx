import { useSwipe } from "./useSwipe";
export function Default({ log }: StoryHarnessProps) { return <div {...useSwipe((direction) => log("swipe", direction))}>Swipe surface</div>; }
