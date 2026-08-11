/** @vrooliComponentSource motion.transition */
import type { ReactNode } from "react";
import {
  Presence,
  type PresenceDuration,
} from "../../../Presence/versions/1.0.0/Presence";
import {
  MotionPrimitive,
  type MotionDuration,
  type MotionPrimitiveProps,
  type MotionValue,
} from "../../../MotionPrimitive/versions/1.0.0/MotionPrimitive";

export type TransitionKind = "fade" | "scale" | "slide" | "blur" | "crossfade";

export interface TransitionProps
  extends Omit<
    MotionPrimitiveProps,
    "active" | "children" | "variant" | "duration"
  > {
  children: ReactNode;
  present?: boolean;
  initial?: boolean;
  keepMounted?: boolean;
  kind?: TransitionKind;
  duration?: MotionDuration | number;
  exitDuration?: PresenceDuration | number;
  onExitComplete?: () => void;
  motionValues?: Record<string, MotionValue>;
}

function variantFor(kind: TransitionKind) {
  if (kind === "scale") return "scale" as const;
  if (kind === "slide") return "slide-up" as const;
  if (kind === "blur") return "blur" as const;
  return "fade" as const;
}

export function Transition({
  children,
  present = true,
  initial = false,
  keepMounted = false,
  kind = "fade",
  duration = "quick",
  exitDuration,
  onExitComplete,
  motionValues,
  ...props
}: TransitionProps) {
  const presenceDuration: PresenceDuration | number = duration;
  return (
    <Presence
      present={present}
      initial={initial}
      keepMounted={keepMounted}
      duration={presenceDuration}
      exitDuration={exitDuration}
      onExitComplete={onExitComplete}
    >
      <MotionPrimitive
        {...props}
        active={present}
        variant={variantFor(kind)}
        duration={duration}
        motionValues={motionValues}
        data-rcl-transition
        data-transition-kind={kind}
      >
        {children}
      </MotionPrimitive>
    </Presence>
  );
}
