import { OverlayCanvas, type OverlaySubject } from "./OverlayCanvas";
export function OverlayCanvasStory({ args }: StoryHarnessProps) {
  return (
    <OverlayCanvas subjects={args.subjects as OverlaySubject[] | undefined} />
  );
}
