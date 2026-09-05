/**
 * @libraryId react-component-library:Transition
 * @displayName Transition
 * @description
 * @version 1.0.6
 * @tags ["primitive","motion","reduced-motion","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource motion.transition */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import {
  Presence,
  type PresenceDuration,
} from "@vrooli/react-component-library/Presence/1";
import {
  MotionPrimitive,
  type MotionDuration,
  type MotionPrimitiveProps,
  type MotionValue,
} from "@vrooli/react-component-library/MotionPrimitive/1";

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

export const Transition = withClassName(function Transition({
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
      data-testid="motion.transition"
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
});
