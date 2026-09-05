/** @vrooliComponentSource hooks.use-press */
import {
  type KeyboardEvent as ReactKeyboardEvent,
  type SyntheticEvent,
} from "react";

export function usePress(onPress?: (event: SyntheticEvent) => void) {
  return {
    onClick: onPress,
    onKeyDown: (event: ReactKeyboardEvent) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        onPress?.(event);
      }
    },
  };
}
