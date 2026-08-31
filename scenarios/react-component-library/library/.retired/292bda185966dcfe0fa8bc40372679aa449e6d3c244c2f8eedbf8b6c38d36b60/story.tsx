import { useDrag } from "./useDrag";
export function Default({ log }: StoryHarnessProps) {
  return (
    <div data-testid="hooks.use-drag" {...useDrag(() => log("moved"))}>
      Drag surface
    </div>
  );
}
