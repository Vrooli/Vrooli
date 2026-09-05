import {
  HandednessProvider,
  useGestureDirection,
  useHandedness,
} from "./useHandedness";

function Readout({ testId }: { testId: string }) {
  const handedness = useHandedness();
  const dismiss = useGestureDirection("dismiss");
  const reveal = useGestureDirection("reveal");
  return (
    <div
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
