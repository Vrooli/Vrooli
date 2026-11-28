import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';

const { gamepadInstance, GamepadInputManager, controller, initSpatialNav } = vi.hoisted(() => {
  const gamepadInstance = { start: vi.fn(), dispose: vi.fn() };
  const controller = { dispose: vi.fn(), registerGroup: vi.fn(), pushScope: vi.fn(), popScope: vi.fn() };
  return {
    gamepadInstance,
    controller,
    GamepadInputManager: vi.fn(() => gamepadInstance),
    initSpatialNav: vi.fn(() => controller),
  };
});

vi.mock('@vrooli/iframe-bridge/spatial', () => ({
  GamepadInputManager,
  initSpatialNav,
}));

import { useGamepad } from './useGamepad';
import { useSpatialNav } from './useSpatialNav';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useGamepad', () => {
  it('starts a gamepad manager on mount and disposes it on unmount', () => {
    const onAction = vi.fn();
    const { result, unmount } = renderHook(() => useGamepad(onAction, { pollIntervalMs: 50 } as never));

    expect(GamepadInputManager).toHaveBeenCalledTimes(1);
    expect(gamepadInstance.start).toHaveBeenCalledTimes(1);
    expect(result.current.current).toBe(gamepadInstance);

    // The manager forwards raw actions through the latest callback.
    const opts = (GamepadInputManager as unknown as ReturnType<typeof vi.fn>).mock.calls[0]![0] as {
      onAction: (a: unknown) => void;
    };
    opts.onAction({ type: 'button', name: 'A' });
    expect(onAction).toHaveBeenCalledWith({ type: 'button', name: 'A' });

    unmount();
    expect(gamepadInstance.dispose).toHaveBeenCalledTimes(1);
  });
});

describe('useSpatialNav', () => {
  it('initialises spatial navigation on mount and disposes on unmount', () => {
    const { result, unmount } = renderHook(() => useSpatialNav());
    expect(initSpatialNav).toHaveBeenCalledTimes(1);
    expect(result.current.current).toBe(controller);
    unmount();
    expect(controller.dispose).toHaveBeenCalledTimes(1);
  });
});
