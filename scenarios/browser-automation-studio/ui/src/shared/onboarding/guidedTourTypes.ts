import type { ReactNode } from "react";

export interface TourStep {
  id: string;
  title: string;
  description: string;
  icon: ReactNode;
  /** CSS selector for the element to anchor to */
  anchorSelector?: string;
  /** Which side of the anchor element to position on */
  anchorPosition?: "top" | "bottom" | "left" | "right";
  /** Action hint shown at the bottom of the step */
  actionHint?: string;
  /** If true, wait for user to interact with the target before auto-advancing */
  waitForInteraction?: boolean;
  /** Selector to watch for click to advance */
  advanceOnClick?: string;
  /** Called when this step becomes active */
  onEnter?: () => void;
  /** Called when leaving this step (for cleanup like closing modals) */
  onExit?: () => void;
  /** Action to auto-perform if user presses Next without completing the required interaction */
  autoAction?: () => Promise<void>;
  /** URL pattern this step requires (string for startsWith, RegExp for pattern match) */
  requiredUrl?: string | RegExp;
  /** Auto-navigate to this URL on step entry if not already there */
  navigateTo?: string;
}
