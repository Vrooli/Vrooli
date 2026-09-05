/**
 * Mock builders for `@vrooli/iframe-bridge/spatial`.
 *
 * The spatial-nav module owns gamepad input + focus-group registration
 * for the entire UI shell. Hook tests that exercise `useGamepad`,
 * `useSpatialNav`, and `<SpatialGroup>` need a substitutable shape so
 * mount/unmount lifecycles, callback indirection, and scope push/pop
 * can be asserted without driving a real gamepad.
 *
 * # Why mock builders, not inline mocks
 *
 * Each hook test still calls `vi.mock("@vrooli/iframe-bridge/spatial", ...)`
 * inline at the top of the file (Vitest's hoisting is non-negotiable; a
 * helper-wrapped `vi.mock` would be in the temporal dead zone). What
 * lives here are the *builders* the inline factory closure invokes —
 * one source of truth for the mock shape, so a future spatial-API
 * addition (e.g. a new `pause()` method) is one edit, not N edits
 * across every hook test.
 *
 * Usage shape:
 *
 *   import { makeMockSpatialNavController } from "@/test-utils/mocks/spatial";
 *
 *   vi.mock("@vrooli/iframe-bridge/spatial", () => {
 *     const controller = makeMockSpatialNavController();
 *     return { initSpatialNav: vi.fn().mockReturnValue(controller) };
 *   });
 *
 * # Production must not import this file
 *
 * The same `no-restricted-imports` ESLint rule that quarantines
 * test-utils covers this directory. If you find yourself reaching for
 * a spatial mock from production code, move the helper out of
 * test-utils into a real src/ location.
 */
import { vi } from "vitest";

/** Shape of the GamepadInputManager double; mirrors the surface the hook calls. */
export interface MockGamepadInputManager {
  start: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  /** The latest onAction passed to the constructor (last writer wins). */
  onAction: ((a: unknown) => void) | undefined;
}

/**
 * Build a fresh GamepadInputManager mock. `start` and `dispose` are
 * `vi.fn()` slots so tests assert on call counts; `onAction` is set by
 * the constructor double (see `makeGamepadInputManagerCtor` below).
 */
export const makeMockGamepadInputManager = (): MockGamepadInputManager => ({
  start: vi.fn(),
  dispose: vi.fn(),
  onAction: undefined,
});

/**
 * Build a constructor double that returns `instance` and captures the
 * `onAction` it was passed onto `instance.onAction`. Use this to spy on
 * the callback indirection contract in `useGamepad`:
 *
 *   const instance = makeMockGamepadInputManager();
 *   const Ctor = makeGamepadInputManagerCtor(instance);
 *   vi.mock("@vrooli/iframe-bridge/spatial", () => ({
 *     GamepadInputManager: Ctor,
 *   }));
 */
export const makeGamepadInputManagerCtor = (instance: MockGamepadInputManager) =>
  vi.fn().mockImplementation((opts: { onAction?: (a: unknown) => void }) => {
    instance.onAction = opts.onAction;
    return instance;
  });

/** Shape of the SpatialNavController double; mirrors the surface hooks call. */
export interface MockSpatialNavController {
  registerGroup: ReturnType<typeof vi.fn>;
  pushScope: ReturnType<typeof vi.fn>;
  popScope: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  /** The cleanup `vi.fn()` that `registerGroup` returns — exposed so tests can assert it was invoked on unmount. */
  cleanup: ReturnType<typeof vi.fn>;
}

/**
 * Build a fresh SpatialNavController mock. `registerGroup` returns a
 * `vi.fn()` cleanup so tests can verify the SpatialGroup component
 * runs the returned disposer on unmount.
 */
export const makeMockSpatialNavController = (): MockSpatialNavController => {
  const cleanup = vi.fn();
  return {
    registerGroup: vi.fn().mockReturnValue(cleanup),
    pushScope: vi.fn(),
    popScope: vi.fn(),
    dispose: vi.fn(),
    cleanup,
  };
};
