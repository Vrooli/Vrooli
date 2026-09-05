/**
 * Unit tests for the SpatialGroup component.
 *
 * SpatialGroup wraps children in a div that registers a spatial-nav
 * focus group on mount and disposes on unmount. Three branches:
 *
 *   - mode="spatial" (or "passthrough" / "grid"): calls
 *     `controller.registerGroup(el, mode, options)` and runs the
 *     returned cleanup on unmount.
 *   - mode="modal": calls `controller.pushScope(el)` on mount and
 *     `controller.popScope()` on unmount (the cleanup is `popScope`,
 *     not the registerGroup return value).
 *   - mode change rerun: when `mode` flips, the previous-mode cleanup
 *     runs and the new-mode register fires — covered by mounting,
 *     re-rendering with a new mode, and asserting both effects.
 *
 * The test creates a controller mock and manually pokes
 * `controllerRef.current` to it (matching the shape `useSpatialNav`
 * exposes). No `vi.mock` of the spatial SDK needed — SpatialGroup only
 * touches the controller via the ref it's handed.
 * provider-free-exception: this isolated controller harness intentionally has no app providers.
 */
import { useEffect, useRef } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render } from "@testing-library/react";

import { SpatialGroup } from "./SpatialGroup";
import { makeMockSpatialNavController } from "../test-utils";

afterEach(() => {
  cleanup();
});

/**
 * Test harness that pre-populates a `controllerRef` with the supplied
 * controller mock before children render. SpatialGroup reads
 * `controllerRef.current` inside its useEffect, which fires after this
 * mount effect (effects run in child-first order).
 */
const Harness = (props: {
  controller: ReturnType<typeof makeMockSpatialNavController>;
  mode: "spatial" | "modal" | "passthrough" | "grid";
  options?: Record<string, unknown>;
}) => {
  const ref = useRef<ReturnType<typeof makeMockSpatialNavController> | null>(null);
  // Populate the ref synchronously on first render so SpatialGroup's
  // mount effect sees the controller.
  if (ref.current === null) ref.current = props.controller;
  // Keep it pointed at the latest prop in case the test swaps it.
  useEffect(() => {
    ref.current = props.controller;
  }, [props.controller]);
  return (
    <SpatialGroup
      controllerRef={ref as unknown as React.RefObject<never>}
      mode={props.mode}
      options={props.options}
    >
      <button type="button">child</button>
    </SpatialGroup>
  );
};

describe("SpatialGroup", () => {
  it("registers a spatial focus group on mount and runs the cleanup on unmount", () => {
    const ctrl = makeMockSpatialNavController();
    const { unmount } = render(<Harness controller={ctrl} mode="spatial" options={{ wrap: true }} />);

    expect(ctrl.registerGroup).toHaveBeenCalledTimes(1);
    const [el, mode, opts] = ctrl.registerGroup.mock.calls[0] as [HTMLElement, string, unknown];
    expect(el).toBeInstanceOf(HTMLDivElement);
    expect(mode).toBe("spatial");
    expect(opts).toEqual({ wrap: true });
    expect(ctrl.cleanup).not.toHaveBeenCalled();

    unmount();
    expect(ctrl.cleanup).toHaveBeenCalledTimes(1);
  });

  it("pushes a modal scope on mount and pops it on unmount", () => {
    const ctrl = makeMockSpatialNavController();
    const { unmount } = render(<Harness controller={ctrl} mode="modal" />);

    expect(ctrl.pushScope).toHaveBeenCalledTimes(1);
    expect(ctrl.popScope).not.toHaveBeenCalled();
    expect(ctrl.registerGroup).not.toHaveBeenCalled();

    unmount();
    expect(ctrl.popScope).toHaveBeenCalledTimes(1);
  });

  it("re-runs the effect when mode changes (spatial→modal cleans up registerGroup, registers modal scope)", () => {
    const ctrl = makeMockSpatialNavController();
    const { rerender, unmount } = render(<Harness controller={ctrl} mode="spatial" />);

    expect(ctrl.registerGroup).toHaveBeenCalledTimes(1);
    expect(ctrl.cleanup).not.toHaveBeenCalled();

    rerender(<Harness controller={ctrl} mode="modal" />);

    // Previous mode's cleanup fired before the new mode's effect set up.
    expect(ctrl.cleanup).toHaveBeenCalledTimes(1);
    expect(ctrl.pushScope).toHaveBeenCalledTimes(1);

    unmount();
    expect(ctrl.popScope).toHaveBeenCalledTimes(1);
  });
});
