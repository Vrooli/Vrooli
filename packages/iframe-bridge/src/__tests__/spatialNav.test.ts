import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SpatialNavManager, FOCUSABLE_SELECTOR, type Direction } from '../spatialNav.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a positioned button element with a known bounding rect. */
function makeButton(
  id: string,
  rect: { top: number; left: number; width: number; height: number },
): HTMLButtonElement {
  const btn = document.createElement('button');
  btn.id = id;
  btn.textContent = id;
  // Store the rect as a data attribute so our mock can read it.
  btn.dataset.mockTop = String(rect.top);
  btn.dataset.mockLeft = String(rect.left);
  btn.dataset.mockWidth = String(rect.width);
  btn.dataset.mockHeight = String(rect.height);
  return btn;
}

/** Mock getBoundingClientRect that reads from data attributes. */
function mockGetRect(el: HTMLElement): DOMRect {
  const top = Number(el.dataset.mockTop ?? 0);
  const left = Number(el.dataset.mockLeft ?? 0);
  const width = Number(el.dataset.mockWidth ?? 0);
  const height = Number(el.dataset.mockHeight ?? 0);
  return {
    top,
    left,
    width,
    height,
    right: left + width,
    bottom: top + height,
    x: left,
    y: top,
    toJSON: () => ({}),
  };
}

function setupDOM(...buttons: HTMLButtonElement[]): HTMLDivElement {
  const container = document.createElement('div');
  container.dataset.mockTop = '0';
  container.dataset.mockLeft = '0';
  container.dataset.mockWidth = '1000';
  container.dataset.mockHeight = '1000';
  for (const btn of buttons) container.appendChild(btn);
  document.body.appendChild(container);
  return container;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('SpatialNavManager', () => {
  let root: HTMLDivElement;

  afterEach(() => {
    document.body.innerHTML = '';
    document.documentElement.removeAttribute('data-spatial-active');
  });

  describe('focus movement', () => {
    it('moves focus right to the nearest element', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      const c = makeButton('c', { top: 100, left: 600, width: 80, height: 40 });
      root = setupDOM(a, b, c);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      // Should focus first element
      expect(document.activeElement).toBe(a);

      const moved = mgr.moveFocus('right');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(b);
      expect(b.getAttribute('data-spatial-focus')).toBe('true');
      expect(a.hasAttribute('data-spatial-focus')).toBe(false);

      mgr.dispose();
    });

    it('moves focus left', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      mgr.moveFocus('right'); // go to b
      expect(document.activeElement).toBe(b);

      mgr.moveFocus('left');
      expect(document.activeElement).toBe(a);

      mgr.dispose();
    });

    it('moves focus up and down', () => {
      const a = makeButton('a', { top: 50, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 150, left: 100, width: 80, height: 40 });
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(document.activeElement).toBe(a);

      mgr.moveFocus('down');
      expect(document.activeElement).toBe(b);

      mgr.moveFocus('up');
      expect(document.activeElement).toBe(a);

      mgr.dispose();
    });

    it('returns false when no candidate in direction', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      root = setupDOM(a);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(mgr.moveFocus('right')).toBe(false);
      expect(document.activeElement).toBe(a); // unchanged

      mgr.dispose();
    });

    it('prefers aligned elements over closer unaligned ones', () => {
      // A is at (100, 100). B is close but far off vertically.
      // C is farther along the axis but perfectly aligned.
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 300, left: 200, width: 80, height: 40 }); // close horizontally but far vertically
      const c = makeButton('c', { top: 100, left: 400, width: 80, height: 40 }); // aligned vertically
      root = setupDOM(a, b, c);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      const moved = mgr.moveFocus('right');
      expect(moved).toBe(true);
      // C should win because cross-axis distance is weighted 2x
      expect(document.activeElement).toBe(c);

      mgr.dispose();
    });

    it('handles partially overlapping elements (overlap tolerance)', () => {
      // B overlaps A slightly on the horizontal axis but is mostly to the right.
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 178, width: 80, height: 40 }); // left edge at 178, A's right at 180
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      const moved = mgr.moveFocus('right');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(b);

      mgr.dispose();
    });

    it('returns false when not in spatial mode', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      // Not active yet
      expect(mgr.moveFocus('right')).toBe(false);

      mgr.dispose();
    });

    it('focuses first element when nothing is focused', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(document.activeElement).toBe(a);

      mgr.dispose();
    });

    it('graceful no-op when there are no focusable elements', () => {
      root = document.createElement('div');
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(mgr.moveFocus('right')).toBe(false);

      mgr.dispose();
    });
  });

  describe('focus ring management', () => {
    it('sets and clears data-spatial-focus attribute', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(a.getAttribute('data-spatial-focus')).toBe('true');

      mgr.moveFocus('right');
      expect(a.hasAttribute('data-spatial-focus')).toBe(false);
      expect(b.getAttribute('data-spatial-focus')).toBe('true');

      mgr.exitSpatialMode();
      expect(b.hasAttribute('data-spatial-focus')).toBe(false);

      mgr.dispose();
    });
  });

  describe('mode management', () => {
    it('sets data-spatial-active on html when entering spatial mode', () => {
      root = setupDOM(makeButton('a', { top: 0, left: 0, width: 80, height: 40 }));
      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);

      mgr.enterSpatialMode();
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(true);

      mgr.exitSpatialMode();
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);

      mgr.dispose();
    });

    it('auto-exits on mousemove', () => {
      root = setupDOM(makeButton('a', { top: 0, left: 0, width: 80, height: 40 }));
      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);

      window.dispatchEvent(new MouseEvent('mousemove'));
      expect(mgr.isActive()).toBe(false);

      mgr.dispose();
    });

    it('auto-exits on touchstart', () => {
      root = setupDOM(makeButton('a', { top: 0, left: 0, width: 80, height: 40 }));
      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);

      window.dispatchEvent(new Event('touchstart'));
      expect(mgr.isActive()).toBe(false);

      mgr.dispose();
    });
  });

  describe('focus groups', () => {
    it('constrains navigation to the registered group', () => {
      // Two groups, each with a button.
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });

      const group1 = document.createElement('div');
      group1.dataset.mockTop = '0';
      group1.dataset.mockLeft = '0';
      group1.dataset.mockWidth = '200';
      group1.dataset.mockHeight = '200';
      group1.appendChild(a);

      const group2 = document.createElement('div');
      group2.dataset.mockTop = '0';
      group2.dataset.mockLeft = '250';
      group2.dataset.mockWidth = '200';
      group2.dataset.mockHeight = '200';
      group2.appendChild(b);

      root = document.createElement('div');
      root.appendChild(group1);
      root.appendChild(group2);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.registerGroup(group1, 'spatial');
      mgr.registerGroup(group2, 'spatial');

      mgr.enterSpatialMode();
      // Manually focus a
      a.focus();
      a.setAttribute('data-spatial-focus', 'true');

      // Moving right from a should NOT reach b (different group)
      const moved = mgr.moveFocus('right');
      expect(moved).toBe(false);

      mgr.dispose();
    });

    it('passthrough group blocks spatial nav moveFocus', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });

      const group = document.createElement('div');
      group.appendChild(a);
      group.appendChild(b);
      root = document.createElement('div');
      root.appendChild(group);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.registerGroup(group, 'passthrough');
      mgr.enterSpatialMode();

      // Focus a (which is inside passthrough)
      a.focus();
      a.setAttribute('data-spatial-focus', 'true');

      // moveFocus should return false because we're in passthrough
      expect(mgr.moveFocus('right')).toBe(false);

      mgr.dispose();
    });

    it('sets data-spatial-group attribute and removes on dispose', () => {
      const group = document.createElement('div');
      root = document.createElement('div');
      root.appendChild(group);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      const dispose = mgr.registerGroup(group, 'spatial');
      expect(group.getAttribute('data-spatial-group')).toBe('spatial');

      dispose();
      expect(group.hasAttribute('data-spatial-group')).toBe(false);

      mgr.dispose();
    });

    it('cycleFocusGroup moves between groups', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 400, width: 80, height: 40 });

      const group1 = document.createElement('div');
      group1.appendChild(a);
      const group2 = document.createElement('div');
      group2.appendChild(b);

      root = document.createElement('div');
      root.appendChild(group1);
      root.appendChild(group2);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.registerGroup(group1, 'spatial');
      mgr.registerGroup(group2, 'spatial');

      mgr.enterSpatialMode();
      expect(document.activeElement).toBe(a);

      mgr.cycleFocusGroup('next');
      expect(document.activeElement).toBe(b);

      mgr.cycleFocusGroup('prev');
      expect(document.activeElement).toBe(a);

      mgr.dispose();
    });
  });

  describe('grid mode', () => {
    it('navigates in grid pattern (right, down, left, up)', () => {
      // 2x2 grid
      const tl = makeButton('tl', { top: 0, left: 0, width: 80, height: 40 });
      const tr = makeButton('tr', { top: 0, left: 100, width: 80, height: 40 });
      const bl = makeButton('bl', { top: 60, left: 0, width: 80, height: 40 });
      const br = makeButton('br', { top: 60, left: 100, width: 80, height: 40 });

      const grid = document.createElement('div');
      grid.dataset.mockTop = '0';
      grid.dataset.mockLeft = '0';
      grid.dataset.mockWidth = '200';
      grid.dataset.mockHeight = '120';
      grid.append(tl, tr, bl, br);

      root = document.createElement('div');
      root.appendChild(grid);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.registerGroup(grid, 'grid');
      mgr.enterSpatialMode();

      // Focus top-left
      tl.focus();
      tl.setAttribute('data-spatial-focus', 'true');

      mgr.moveFocus('right');
      expect(document.activeElement).toBe(tr);

      mgr.moveFocus('down');
      expect(document.activeElement).toBe(br);

      mgr.moveFocus('left');
      expect(document.activeElement).toBe(bl);

      mgr.moveFocus('up');
      expect(document.activeElement).toBe(tl);

      mgr.dispose();
    });
  });

  describe('grid wrap behavior', () => {
    function makeGrid(wrap: boolean) {
      // 2x3 grid
      const buttons = [
        makeButton('r0c0', { top: 0, left: 0, width: 80, height: 40 }),
        makeButton('r0c1', { top: 0, left: 100, width: 80, height: 40 }),
        makeButton('r0c2', { top: 0, left: 200, width: 80, height: 40 }),
        makeButton('r1c0', { top: 60, left: 0, width: 80, height: 40 }),
        makeButton('r1c1', { top: 60, left: 100, width: 80, height: 40 }),
        makeButton('r1c2', { top: 60, left: 200, width: 80, height: 40 }),
      ];

      const grid = document.createElement('div');
      grid.dataset.mockTop = '0';
      grid.dataset.mockLeft = '0';
      grid.dataset.mockWidth = '300';
      grid.dataset.mockHeight = '120';
      for (const btn of buttons) grid.appendChild(btn);

      const root = document.createElement('div');
      root.appendChild(grid);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.registerGroup(grid, 'grid', { wrap });
      mgr.enterSpatialMode();

      return { mgr, buttons, root };
    }

    it('right from last column wraps to first column', () => {
      const { mgr, buttons } = makeGrid(true);

      // Focus last column of first row
      buttons[2].focus();
      buttons[2].setAttribute('data-spatial-focus', 'true');

      const moved = mgr.moveFocus('right');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(buttons[0]); // wraps to first column

      mgr.dispose();
    });

    it('down from last row wraps to first row', () => {
      const { mgr, buttons } = makeGrid(true);

      // Focus bottom-left
      buttons[3].focus();
      buttons[3].setAttribute('data-spatial-focus', 'true');

      const moved = mgr.moveFocus('down');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(buttons[0]); // wraps to first row, same col

      mgr.dispose();
    });

    it('left from first column wraps to last column', () => {
      const { mgr, buttons } = makeGrid(true);

      // Focus first column of first row
      buttons[0].focus();
      buttons[0].setAttribute('data-spatial-focus', 'true');

      const moved = mgr.moveFocus('left');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(buttons[2]); // wraps to last column

      mgr.dispose();
    });

    it('up from first row wraps to last row', () => {
      const { mgr, buttons } = makeGrid(true);

      // Focus top-left
      buttons[0].focus();
      buttons[0].setAttribute('data-spatial-focus', 'true');

      const moved = mgr.moveFocus('up');
      expect(moved).toBe(true);
      expect(document.activeElement).toBe(buttons[3]); // wraps to last row, same col

      mgr.dispose();
    });

    it('does NOT wrap when wrap is false', () => {
      const { mgr, buttons } = makeGrid(false);

      // Focus last column
      buttons[2].focus();
      buttons[2].setAttribute('data-spatial-focus', 'true');

      const moved = mgr.moveFocus('right');
      expect(moved).toBe(false); // no wrap
      expect(document.activeElement).toBe(buttons[2]); // unchanged

      mgr.dispose();
    });
  });

  describe('scroll-into-view edge cases', () => {
    it('element below viewport triggers container.scrollTo', () => {
      // Create a scrollable container
      const container = document.createElement('div');
      container.style.overflowY = 'auto';
      container.style.overflowX = 'auto';
      container.dataset.mockTop = '0';
      container.dataset.mockLeft = '0';
      container.dataset.mockWidth = '400';
      container.dataset.mockHeight = '200'; // viewport is 200px tall

      const a = makeButton('a', { top: 50, left: 50, width: 80, height: 40 });
      // b is below the container's visible area
      const b = makeButton('b', { top: 300, left: 50, width: 80, height: 40 });
      container.append(a, b);

      const root = document.createElement('div');
      root.appendChild(container);
      document.body.appendChild(root);

      const scrollToSpy = vi.fn();
      container.scrollTo = scrollToSpy;
      container.scrollTop = 0;
      container.scrollLeft = 0;

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      // Focus a first
      a.focus();
      a.setAttribute('data-spatial-focus', 'true');

      // Move down to b (which is below viewport)
      mgr.moveFocus('down');
      expect(document.activeElement).toBe(b);

      // scrollTo should have been called because b is below the container
      expect(scrollToSpy).toHaveBeenCalled();
      const callArgs = scrollToSpy.mock.calls[0][0];
      expect(callArgs.behavior).toBe('smooth');
      // scrollTop should increase to bring b into view
      expect(callArgs.top).toBeGreaterThan(0);

      mgr.dispose();
    });

    it('element already visible does NOT trigger scroll', () => {
      const container = document.createElement('div');
      container.style.overflow = 'auto';
      container.dataset.mockTop = '0';
      container.dataset.mockLeft = '0';
      container.dataset.mockWidth = '400';
      container.dataset.mockHeight = '400'; // large enough viewport

      // Both buttons well within the visible area (accounting for 16px scroll padding)
      const a = makeButton('a', { top: 50, left: 50, width: 80, height: 40 });
      const b = makeButton('b', { top: 50, left: 200, width: 80, height: 40 });
      container.append(a, b);

      const root = document.createElement('div');
      root.appendChild(container);
      document.body.appendChild(root);

      const scrollToSpy = vi.fn();
      container.scrollTo = scrollToSpy;
      container.scrollTop = 0;
      container.scrollLeft = 0;

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      a.focus();
      a.setAttribute('data-spatial-focus', 'true');

      mgr.moveFocus('right');
      expect(document.activeElement).toBe(b);

      // scrollTo should NOT have been called — b is already visible
      expect(scrollToSpy).not.toHaveBeenCalled();

      mgr.dispose();
    });
  });

  describe('stale focus ring cleanup', () => {
    it('clears orphaned data-spatial-focus attributes on move', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
      const c = makeButton('c', { top: 100, left: 500, width: 80, height: 40 });
      root = setupDOM(a, b, c);

      // Simulate a stale attribute (e.g., from an out-of-sync native focus event)
      a.setAttribute('data-spatial-focus', 'true');
      b.setAttribute('data-spatial-focus', 'true');

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      // enterSpatialMode focuses first element (a), clearing stale from b
      expect(a.getAttribute('data-spatial-focus')).toBe('true');
      expect(b.hasAttribute('data-spatial-focus')).toBe(false);

      mgr.moveFocus('right');
      expect(b.getAttribute('data-spatial-focus')).toBe('true');
      expect(a.hasAttribute('data-spatial-focus')).toBe(false);

      mgr.dispose();
    });

    it('only one element has data-spatial-focus after rapid navigation', () => {
      const buttons = [
        makeButton('a', { top: 100, left: 100, width: 80, height: 40 }),
        makeButton('b', { top: 100, left: 300, width: 80, height: 40 }),
        makeButton('c', { top: 100, left: 500, width: 80, height: 40 }),
      ];
      root = setupDOM(...buttons);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      mgr.moveFocus('right');
      mgr.moveFocus('right');
      mgr.moveFocus('left');

      const focused = root.querySelectorAll('[data-spatial-focus]');
      expect(focused.length).toBe(1);
      expect(focused[0]).toBe(buttons[1]); // b

      mgr.dispose();
    });
  });

  describe('selectFocused', () => {
    it('clicks the focused element', () => {
      const a = makeButton('a', { top: 0, left: 0, width: 80, height: 40 });
      const onClick = vi.fn();
      a.addEventListener('click', onClick);
      root = setupDOM(a);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      mgr.selectFocused();
      expect(onClick).toHaveBeenCalledOnce();

      mgr.dispose();
    });
  });

  describe('iframe safety', () => {
    it('uses focus({ preventScroll: true }) and never calls scrollIntoView', () => {
      // jsdom may not define scrollIntoView — polyfill it so we can spy.
      if (!Element.prototype.scrollIntoView) {
        Element.prototype.scrollIntoView = function () { /* noop */ };
      }
      const scrollIntoViewSpy = vi.spyOn(Element.prototype, 'scrollIntoView');

      const a = makeButton('a', { top: 0, left: 0, width: 80, height: 40 });
      const b = makeButton('b', { top: 0, left: 200, width: 80, height: 40 });
      const focusSpy = vi.spyOn(b, 'focus');
      root = setupDOM(a, b);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      mgr.moveFocus('right');

      expect(focusSpy).toHaveBeenCalledWith({ preventScroll: true });
      expect(scrollIntoViewSpy).not.toHaveBeenCalled();

      scrollIntoViewSpy.mockRestore();
      mgr.dispose();
    });
  });

  describe('dispose', () => {
    it('cleans up all state and listeners', () => {
      const a = makeButton('a', { top: 0, left: 0, width: 80, height: 40 });
      root = setupDOM(a);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      mgr.enterSpatialMode();
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(true);

      mgr.dispose();
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);

      // Should not throw on subsequent calls.
      mgr.dispose();
    });
  });
});
