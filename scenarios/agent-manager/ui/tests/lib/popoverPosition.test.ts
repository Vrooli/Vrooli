import test from "node:test";
import assert from "node:assert/strict";
import { getPopoverPosition } from "../../src/lib/popoverPosition.js";

test("popover stays within the right viewport edge", () => {
  const position = getPopoverPosition({
    anchorRect: { top: 24, left: 360, width: 32, height: 32, bottom: 56, right: 392 },
    viewportWidth: 400,
    viewportHeight: 900,
    preferredWidth: 320,
    panelHeight: 300,
  });

  assert.equal(position.left, 72);
  assert.equal(position.width, 320);
  assert.equal(position.placement, "bottom");
});

test("popover flips above when bottom space is too small", () => {
  const position = getPopoverPosition({
    anchorRect: { top: 540, left: 280, width: 32, height: 32, bottom: 572, right: 312 },
    viewportWidth: 640,
    viewportHeight: 640,
    preferredWidth: 320,
    panelHeight: 260,
  });

  assert.equal(position.placement, "top");
  assert.equal(position.top, 272);
  assert.equal(position.maxHeight, 524);
});

test("popover narrows on small viewports and clamps to padding", () => {
  const position = getPopoverPosition({
    anchorRect: { top: 16, left: 120, width: 32, height: 32, bottom: 48, right: 152 },
    viewportWidth: 260,
    viewportHeight: 500,
    preferredWidth: 320,
    panelHeight: 220,
  });

  assert.equal(position.width, 244);
  assert.equal(position.left, 8);
  assert.equal(position.top, 56);
});
