import { useEffect, useRef } from 'react';
import {
  GamepadInputManager,
  type GamepadAction,
  type GamepadInputOptions,
} from '@vrooli/iframe-bridge/spatial';

/**
 * Thin hook for raw gamepad input — fires `onAction` for every detected
 * button press or stick direction.  Use this for scenarios that need custom
 * gamepad handling without spatial navigation (e.g., game-like controls).
 *
 * For standard UI navigation, prefer {@link useSpatialNav} instead.
 */
export function useGamepad(
  onAction: (action: GamepadAction) => void,
  options?: Omit<GamepadInputOptions, 'onAction'>,
): React.RefObject<GamepadInputManager | null> {
  const managerRef = useRef<GamepadInputManager | null>(null);
  const callbackRef = useRef(onAction);
  callbackRef.current = onAction;

  useEffect(() => {
    const mgr = new GamepadInputManager({
      ...options,
      onAction: (a: GamepadAction) => callbackRef.current(a),
    });
    mgr.start();
    managerRef.current = mgr;
    return () => {
      mgr.dispose();
      managerRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return managerRef;
}
