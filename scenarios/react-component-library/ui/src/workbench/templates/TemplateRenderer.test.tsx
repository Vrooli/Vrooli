import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TemplateRenderer } from "./TemplateRenderer";
import { DEFAULT_TEMPLATE, TEMPLATES } from "./library";
import { node, slotNode } from "./spec";

describe("TemplateRenderer", () => {
  it("renders every template's page root and component slot", () => {
    for (const template of TEMPLATES) {
      const { container, unmount } = render(
        <TemplateRenderer template={template} spec={template.spec} mode="preview" />,
      );
      expect(container.querySelector(".tpl-page")).not.toBeNull();
      expect(container.querySelector(".tpl-slot")).not.toBeNull();
      unmount();
    }
  });

  it("renders a live slot without the placeholder hint", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = render(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="preview"
        slot={<button type="button">Live action</button>}
      />,
    );
    expect(screen.getByRole("button", { name: "Live action" })).toBeInTheDocument();
    expect(container.querySelector(".tpl-slot")?.className).toContain("is-filled");
  });

  it("shows a slot's authored fallback until a live component replaces it", () => {
    const fallbackCopy = "Authored fallback";
    const liveCopy = "Live metric";
    const spec = node("page", "page", {}, [
      slotNode("metric", {
        hint: "KPI card",
        minH: 96,
        role: "metric",
        children: [node("metric.fallback", "text", { text: fallbackCopy })],
      }),
    ]);
    const { container, rerender } = render(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    expect(screen.getByText(fallbackCopy)).toBeInTheDocument();
    expect(container.querySelector(".tpl-slot")?.className).toContain("is-fallback");

    rerender(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        slot={<span>{liveCopy}</span>}
      />,
    );
    expect(screen.queryByText(fallbackCopy)).toBeNull();
    expect(screen.getByText(liveCopy)).toBeInTheDocument();
  });

  it("floats a placed component above the authored page and keeps the anchor", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = render(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="preview"
        slot={<button type="button">Live action</button>}
        slotPlacement={{ x: 42, y: 96, w: 240, h: 72 }}
      />,
    );

    const floating = container.querySelector<HTMLElement>(".tpl-slot-float");
    expect(floating).not.toBeNull();
    expect(floating?.style.top).toBe("96px");
    expect(floating?.style.left).toContain("42px");
    expect(floating?.style.width).toContain("240px");
    expect(floating?.querySelector<HTMLElement>(".tpl-slot-float-body")?.style.height).toBe("72px");
    expect(container.querySelector(".tpl-slot")?.className).toContain("is-placed-anchor");
    expect(screen.getByRole("button", { name: "Live action" })).toBeInTheDocument();
  });

  it("moves and resizes the placed component from the keyboard", () => {
    const template = DEFAULT_TEMPLATE;
    const onChange = vi.fn();
    render(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="preview"
        slot={<button type="button">Live action</button>}
        slotEditable
        slotPlacement={{ x: 42, y: 96, w: 240, h: 72 }}
        onSlotPlacementChange={onChange}
      />,
    );

    fireEvent.keyDown(screen.getByRole("button", { name: "workbench.template.moveComponent" }), {
      key: "ArrowRight",
    });
    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 50,
      y: 96,
      w: 240,
      h: 72,
    });

    fireEvent.keyDown(screen.getByRole("button", { name: "workbench.template.resizeComponent" }), {
      key: "ArrowRight",
    });
    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 42,
      y: 96,
      w: 248,
      h: 72,
    });
  });

  it("marks the selected node for the studio outline", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = render(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="canvas"
        selectedId={template.spec.id}
        onSelect={() => undefined}
      />,
    );
    expect(container.querySelector(`[data-node-id="${template.spec.id}"]`)).toHaveAttribute(
      "data-selected",
      "true",
    );
  });
});
