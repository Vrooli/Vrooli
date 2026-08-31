import { useRef } from "react";

import { useDirection } from "./useDirection";

function Readout({ testId }: { testId: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const direction = useDirection(ref);
  return (
    <div
      ref={ref}
      data-testid={testId}
      role="status"
      data-direction={direction}
    >
      {direction}
    </div>
  );
}

export function Default() {
  return <Readout testId="hooks.use-direction" />;
}

/**
 * The reason 2.x exists, and why the read is subtree-scoped: the region declares
 * `rtl` while the document does not. 1.x rendered `ltr` here and never
 * re-rendered; a document-scoped 2.0.0 would still render `ltr`.
 */
export function RightToLeft() {
  return (
    <div dir="rtl">
      <Readout testId="hooks.use-direction.rtl" />
    </div>
  );
}
