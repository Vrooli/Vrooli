import { useDrag } from "./useDrag";
export function Default({ log }: StoryHarnessProps) { return <div {...useDrag(() => log("moved"))}>Drag surface</div>; }
