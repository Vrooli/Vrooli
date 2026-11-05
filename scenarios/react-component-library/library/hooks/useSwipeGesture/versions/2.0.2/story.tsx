import { useRef, useState } from "react";

import { useSwipeGesture, type SwipeOutcome } from "./useSwipeGesture";

const STAGES = [64, 152];

function Specimen({ testId, releaseMode }: { testId: string; releaseMode?: "commit" | "rest" }) {
  const surface = useRef<HTMLDivElement>(null);
  const [stage, setStage] = useState(0);
  const [outcome, setOutcome] = useState<SwipeOutcome | "">("");

  const { onPointerDown } = useSwipeGesture({
    direction: "right",
    stages: STAGES,
    releaseMode,
    // The surface is written directly rather than through state: a gesture that
    // re-renders on every frame is the performance bug this hook exists to avoid.
    onMove: ({ translate }) => {
      if (surface.current) surface.current.style.transform = `translateX(${String(translate)}px)`;
    },
    onStageChange: setStage,
    onRelease: (release) => {
      setOutcome(release.outcome);
      if (surface.current) {
        surface.current.style.transform =
          release.outcome === "rest" ? `translateX(${String(STAGES[STAGES.length - 1])}px)` : "";
      }
    },
  });

  return (
    <div
      data-testid={testId}
      data-stage={stage}
      data-outcome={outcome}
      style={{ overflow: "hidden", width: "20rem", background: "var(--color-surface-muted)" }}
    >
      <div
        ref={surface}
        data-rcl-pan-x=""
        onPointerDown={onPointerDown}
        style={{
          padding: "var(--space-sm)",
          background: "var(--color-surface)",
          touchAction: "pan-y",
        }}
      >
        Drag me
      </div>
    </div>
  );
}

export function Default() {
  return <Specimen testId="hooks.use-swipe-gesture" />;
}

export function AutoCommit() {
  return <Specimen testId="hooks.use-swipe-gesture.commit" releaseMode="commit" />;
}
