import { fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";

import { TemplateRenderer } from "./TemplateRenderer";
import { DEFAULT_TEMPLATE } from "./library";
import { node } from "./spec";

describe("TemplateRenderer canvas interactions", () => {
  it("selects the clicked layer and ignores canvas whitespace", () => {
    const spec = node("page", "page", {}, [node("headline", "text", { text: "Select me" })]);
    const onSelect = vi.fn();
    const { container, getByText } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        onSelect={onSelect}
      />,
    );
    const canvas = container.querySelector<HTMLElement>(".tpl");

    fireEvent.click(getByText("Select me"));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenLastCalledWith("headline");

    if (!canvas) throw new Error("Template canvas was not rendered.");
    fireEvent.click(canvas);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("moves layers before or after a target and leaves unrelated pointer gestures untouched", () => {
    const spec = node("page", "page", {}, [
      node("source", "text", { text: "Move me" }),
      node("target", "text", { text: "Target layer" }),
    ]);
    const onMoveLayer = vi.fn();
    const onSelect = vi.fn();
    const { container, getByText } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        onSelect={onSelect}
        onMoveLayer={onMoveLayer}
      />,
    );
    const root = container.querySelector<HTMLElement>(".tpl");
    const source = container.querySelector<HTMLElement>('[data-node-id="source"]');
    const target = container.querySelector<HTMLElement>('[data-node-id="target"]');
    if (!root || !source || !target) throw new Error("Canvas drag targets were not rendered.");

    vi.spyOn(target, "getBoundingClientRect").mockReturnValue({
      x: 20,
      y: 20,
      top: 20,
      right: 120,
      bottom: 100,
      left: 20,
      width: 100,
      height: 80,
      toJSON: () => ({}),
    });
    const previousElementFromPoint = Object.getOwnPropertyDescriptor(document, "elementFromPoint");
    Object.defineProperty(document, "elementFromPoint", {
      configurable: true,
      value: () => target,
    });

    try {
      fireEvent.pointerDown(source, { button: 0, pointerId: 1, clientX: 2, clientY: 2 });
      fireEvent.pointerMove(root, { pointerId: 1, clientX: 50, clientY: 25 });
      fireEvent.pointerUp(root, { pointerId: 1 });
      expect(onMoveLayer).toHaveBeenLastCalledWith("source", "target", "before");

      fireEvent.pointerDown(source, { button: 0, pointerId: 2, clientX: 2, clientY: 2 });
      fireEvent.pointerMove(root, { pointerId: 2, clientX: 50, clientY: 95 });
      fireEvent.pointerUp(root, { pointerId: 2 });
      expect(onMoveLayer).toHaveBeenLastCalledWith("source", "target", "after");

      fireEvent.pointerDown(source, { button: 2, pointerId: 3, pointerType: "mouse" });
      fireEvent.pointerMove(root, { pointerId: 3, clientX: 50, clientY: 60 });
      fireEvent.pointerUp(root, { pointerId: 3 });
      expect(onMoveLayer).toHaveBeenCalledTimes(2);

      fireEvent.pointerDown(source, { button: 0, pointerId: 4, clientX: 2, clientY: 2 });
      fireEvent.pointerMove(root, { pointerId: 5, clientX: 50, clientY: 60 });
      fireEvent.pointerMove(root, { pointerId: 4, clientX: 4, clientY: 3 });
      fireEvent.pointerUp(root, { pointerId: 5 });
      expect(onMoveLayer).toHaveBeenCalledTimes(2);
      fireEvent.pointerCancel(root, { pointerId: 4 });
      expect(root).not.toHaveAttribute("data-layer-dragging");

      fireEvent.click(getByText("Move me"));
      expect(onSelect).toHaveBeenLastCalledWith("source");
    } finally {
      if (previousElementFromPoint) {
        Object.defineProperty(document, "elementFromPoint", previousElementFromPoint);
      } else {
        Reflect.deleteProperty(document, "elementFromPoint");
      }
    }
  });
});
