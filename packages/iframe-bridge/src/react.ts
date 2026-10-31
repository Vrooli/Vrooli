/** React adapters share the controller owned by the application entry point. */
import {
  createContext, createElement, useContext, useEffect, useLayoutEffect, useRef,
  type ReactNode, type ReactElement, type RefObject, type HTMLAttributes,
} from 'react';
import {
  getSpatialNav, type SpatialNavController, type GamepadActionHandler,
  type FocusGroupMode, type FocusGroupOptions,
} from './spatialNavBridge.js';

const SpatialNavContext = createContext<SpatialNavController | null>(null);
const useClientLayoutEffect = typeof window === 'undefined' ? useEffect : useLayoutEffect;

/** The entry point owns disposal; StrictMode and nested providers do not restart input. */
export function SpatialNavProvider({ controller, children }: {
  controller: SpatialNavController;
  children: ReactNode;
}) {
  return createElement(SpatialNavContext.Provider, { value: controller }, children);
}

export function useSpatialNav(): SpatialNavController {
  const controller = useContext(SpatialNavContext) ?? getSpatialNav();
  if (!controller) throw new Error('Initialize spatial navigation in the application entry point before rendering');
  return controller;
}

/** Focus-scoped custom input. Return true to prevent default navigation/host relay. */
export function useGamepad(
  elementRef: RefObject<HTMLElement | null>,
  onAction: GamepadActionHandler,
  enabled = true,
): void {
  const controller = useSpatialNav();
  const handler = useRef(onAction);
  // Update only after commit, so an abandoned render cannot replace live input.
  useClientLayoutEffect(() => { handler.current = onAction; });
  useEffect(() => {
    const element = elementRef.current;
    if (!enabled || !element) return;
    return controller.registerActionHandler(element, action => handler.current(action));
  }, [controller, elementRef, enabled]);
}

/** Modal cleanup removes its own registration, even when dialogs unmount out of order. */
export function useSpatialScope(elementRef: RefObject<HTMLElement | null>, enabled = true): void {
  const controller = useSpatialNav();
  useEffect(() => {
    const element = elementRef.current;
    if (!enabled || !element) return;
    return controller.registerScope(element);
  }, [controller, elementRef, enabled]);
}

export interface SpatialGroupProps extends HTMLAttributes<HTMLDivElement> {
  mode: FocusGroupMode;
  options?: FocusGroupOptions;
}

/** Register a focus group against the application controller. */
export function SpatialGroup({ mode, options, children, style, ...props }: SpatialGroupProps): ReactElement {
  const controller = useSpatialNav();
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    return mode === 'modal'
      ? controller.registerScope(element)
      : controller.registerGroup(element, mode, options);
  }, [controller, mode, options]);
  return createElement('div', { ...props, ref, style: { display: 'contents', ...style } }, children);
}
