import { useCallback, useRef } from 'react';
import { describe, expect, it, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import {
  computeAnchoredPopoverStyle,
  useAnchoredPopover,
} from './AnchoredPopover';

type RectParams = {
  x: number;
  y: number;
  width: number;
  height: number;
};

const createRect = ({ x, y, width, height }: RectParams): DOMRect => ({
  x,
  y,
  width,
  height,
  top: y,
  left: x,
  right: x + width,
  bottom: y + height,
  toJSON: () => ({}),
}) as DOMRect;

const setViewport = (width: number, height: number) => {
  Object.defineProperty(window, 'innerWidth', { value: width, configurable: true });
  Object.defineProperty(window, 'innerHeight', { value: height, configurable: true });
  if (!window.requestAnimationFrame) {
    window.requestAnimationFrame = (cb: FrameRequestCallback) => window.setTimeout(cb, 0);
  }
  if (!window.cancelAnimationFrame) {
    window.cancelAnimationFrame = (id: number) => window.clearTimeout(id);
  }
};

type HarnessProps = {
  anchorRect: DOMRect;
  popoverRect: DOMRect;
  delayFrames?: number;
};

const PopoverHarness = ({ anchorRect, popoverRect, delayFrames }: HarnessProps) => {
  const anchorRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  const { style, placement } = useAnchoredPopover({
    isOpen: true,
    anchorRef,
    popoverRef,
  });

  const setAnchorNode = useCallback((node: HTMLButtonElement | null) => {
    anchorRef.current = node;
    if (!node) {
      return;
    }
    Object.defineProperty(node, 'getBoundingClientRect', {
      value: () => anchorRect,
    });
  }, [anchorRect]);

  const setPopoverNode = useCallback((node: HTMLDivElement | null) => {
    popoverRef.current = node;
    if (!node) {
      return;
    }
    const scheduleMeasure = (framesLeft: number) => {
      if (framesLeft <= 0) {
        Object.defineProperty(node, 'getBoundingClientRect', {
          value: () => popoverRect,
        });
        return;
      }
      window.requestAnimationFrame(() => {
        scheduleMeasure(framesLeft - 1);
      });
    };

    if (typeof delayFrames === 'number' && delayFrames > 0) {
      scheduleMeasure(delayFrames);
      return;
    }

    Object.defineProperty(node, 'getBoundingClientRect', {
      value: () => popoverRect,
    });
  }, [delayFrames, popoverRect]);

  return (
    <div>
      <button ref={setAnchorNode} data-testid="anchor" />
      {style && <div ref={setPopoverNode} data-testid="popover" style={style} />}
      <div data-testid="placement">{placement}</div>
    </div>
  );
};

describe('computeAnchoredPopoverStyle', () => {
  beforeEach(() => {
    setViewport(1000, 800);
  });

  it('keeps the preferred placement when there is space below', () => {
    const anchor = createRect({ x: 700, y: 120, width: 40, height: 32 });
    const popoverSize = { width: 240, height: 120 };
    const viewport = { width: 1000, height: 800 };

    const result = computeAnchoredPopoverStyle(anchor, popoverSize, viewport);

    expect(result.placement).toBe('bottom-end');
    expect(result.style.transformOrigin).toBe('top right');
  });

  it('flips above when there is no space below', () => {
    const anchor = createRect({ x: 700, y: 700, width: 40, height: 32 });
    const popoverSize = { width: 240, height: 120 };
    const viewport = { width: 1000, height: 800 };

    const result = computeAnchoredPopoverStyle(anchor, popoverSize, viewport);

    expect(result.placement).toBe('top-end');
    expect(result.style.transformOrigin).toBe('bottom right');
  });

  it('clamps within viewport margins when the right edge is tight', () => {
    const anchor = createRect({ x: 260, y: 200, width: 40, height: 32 });
    const popoverSize = { width: 200, height: 120 };
    const viewport = { width: 300, height: 400 };

    const result = computeAnchoredPopoverStyle(anchor, popoverSize, viewport, {
      margin: 12,
      offset: 8,
    });

    expect(parseInt(result.style.left as string, 10)).toBe(288);
  });

  it('selects a placement based on anchor position when neither side fits', () => {
    const anchor = createRect({ x: 100, y: 40, width: 40, height: 32 });
    const popoverSize = { width: 240, height: 380 };
    const viewport = { width: 300, height: 300 };

    const result = computeAnchoredPopoverStyle(anchor, popoverSize, viewport);

    expect(result.placement).toBe('bottom-end');
  });
});

describe('useAnchoredPopover', () => {
  beforeEach(() => {
    setViewport(1000, 800);
  });

  it('updates placement after measuring the popover', async () => {
    const anchor = createRect({ x: 640, y: 700, width: 40, height: 32 });
    const popover = createRect({ x: 0, y: 0, width: 240, height: 120 });

    render(<PopoverHarness anchorRect={anchor} popoverRect={popover} delayFrames={3} />);

    await waitFor(() => {
      expect(screen.getByTestId('placement').textContent).toBe('top-end');
    });
  });
});
