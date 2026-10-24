import { useRef } from "react";

import { HandednessProvider, useGestureDirection, useHandedness } from "./useHandedness";

function Readout({ testId }: { testId: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const handedness = useHandedness();
  const dismiss = useGestureDirection("dismiss", { elementRef: ref });
  const reveal = useGestureDirection("reveal", { elementRef: ref });
  return (
    <div
      ref={ref}
      data-testid={testId}
      role="status"
      data-handedness={handedness}
      data-anchor-edge={dismiss.anchorEdge}
      data-dismiss={dismiss.direction}
      data-reveal={reveal.direction}
    >
      {`${handedness} · dismiss ${dismiss.direction} · reveal ${reveal.direction}`}
    </div>
  );
}

export function Default() {
  return <Readout testId="hooks.use-handedness" />;
}

/** With no provider mounted, an app behaves exactly as it did before the hook existed. */
export function WithoutProvider() {
  return <Readout testId="hooks.use-handedness.default" />;
}

/**
 * Moving the reach side flips the gesture axis without touching the document
 * direction — the separation the whole module exists for.
 */
export function EndAnchored() {
  return (
    <HandednessProvider value="inline-end">
      <Readout testId="hooks.use-handedness.end" />
    </HandednessProvider>
  );
}

/**
 * The other input, moving on its own. The reach side is unchanged; only the
 * locale mirrors, and the resolved edge follows it.
 */
export function RightToLeft() {
  return (
    <div dir="rtl">
      <Readout testId="hooks.use-handedness.rtl" />
    </div>
  );
}
