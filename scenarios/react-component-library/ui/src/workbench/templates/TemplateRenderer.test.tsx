import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { i18n } from "../../i18n";
import { strings } from "../../consts/strings";

import { TemplateRenderer } from "./TemplateRenderer";
import { DEFAULT_TEMPLATE } from "./library";
import { node, slotNode } from "./spec";

describe("TemplateRenderer", () => {
  it("renders each bound catalog slot at its authored position", () => {
    const spec = node("page", "page", {}, [
      node("first-slot", "slot", { componentId: "asset-one", storyId: "one" }),
      node("second-slot", "slot", { componentId: "asset-two", storyId: "two" }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        renderComponentSlot={(slot) => (
          <span data-bound-component={slot.props.componentId as string} />
        )}
      />,
    );

    expect(
      container.querySelector('[data-node-id="first-slot"] [data-bound-component="asset-one"]'),
    ).not.toBeNull();
    expect(
      container.querySelector('[data-node-id="second-slot"] [data-bound-component="asset-two"]'),
    ).not.toBeNull();
  });

  it("renders bounded imported CSS on selectable template layers", () => {
    const spec = node(
      "page",
      "page",
      {
        css: '{"--accent":"#ff0","background":"#101010","backgroundImage":"url(https://bad.test)"}',
      },
      [
        node("heading", "text", {
          text: "Imported heading",
          variant: "display",
          css: '{"color":"#abc","fontSize":"22px"}',
        }),
        node("action", "button", {
          text: "Go",
          variant: "secondary",
          css: '{"background":"#203040","color":"#fff","borderRadius":"24px"}',
        }),
      ],
    );
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    const page = container.querySelector<HTMLElement>(".tpl-page");
    expect(page?.style.getPropertyValue("--accent")).toBe("#ff0");
    expect(page?.style.background).toBe("rgb(16, 16, 16)");
    expect(container.querySelector('[data-node-id="heading"]')?.getAttribute("style")).toContain(
      "color: rgb(170, 187, 204)",
    );
    expect(container.querySelector(".tpl-control button")?.getAttribute("style")).toContain(
      "background: rgb(32, 48, 64)",
    );
    expect(container.innerHTML).not.toContain("bad.test");
  });

  it("keeps rich source button layers inside the mapped RCL Button", () => {
    const spec = node("page", "page", {}, [
      node(
        "action",
        "button",
        {
          pressed: false,
          css: '{"display":"flex","flex-direction":"column"}',
        },
        [
          node("label", "text", { text: "Groceries" }),
          node("detail", "text", { text: "€96 of €140" }),
        ],
      ),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    const control = container.querySelector<HTMLButtonElement>(".tpl-control button");

    expect(control).not.toBeNull();
    expect(control?.textContent).toContain("Groceries€96 of €140");
    expect(control).toHaveAttribute("aria-pressed", "false");
    expect(container.querySelector<HTMLElement>(".tpl-control")?.style.flexDirection).toBe(
      "column",
    );
    expect(container.querySelector<HTMLElement>(".tpl-control-content")?.style.flexDirection).toBe(
      "column",
    );
    expect(container.querySelector('[data-node-id="label"]')?.closest("button")).toBe(control);
  });

  it("preserves compact source button geometry on touch screens", () => {
    const spec = node("page", "page", {}, [
      node("period-week", "button", {
        text: "Week",
        css: '{"height":"26px","font-size":"11px"}',
      }),
      node("primary-action", "button", {
        text: "Continue",
        css: '{"height":"48px"}',
      }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    const compact = container.querySelector<HTMLButtonElement>(
      '[data-node-id="period-week"] button',
    );
    const primary = container.querySelector<HTMLButtonElement>(
      '[data-node-id="primary-action"] button',
    );

    expect(compact).toHaveAttribute("data-rcl-tap-target", "dense");
    expect(compact).toHaveStyle({ height: "26px", minHeight: "26px" });
    expect(primary).toHaveAttribute("data-rcl-tap-target", "comfortable");
    expect(primary).not.toHaveStyle({ minHeight: "48px" });
  });

  it("keeps imported transparent artwork free of the reference-image card", () => {
    const spec = node("page", "page", {}, [
      node("leaf", "image", {
        src: "data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%3E%3C/svg%3E",
        css: '{"position":"absolute","width":"220px","height":"260px"}',
      }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    const artwork = container.querySelector<HTMLElement>('[data-node-id="leaf"]');
    expect(artwork?.style.background).toBe("transparent");
    expect(artwork?.style.borderRadius).toBe("0");
    expect(artwork?.style.position).toBe("absolute");
  });

  it("renders the neutral workspace starter and component slot", () => {
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={DEFAULT_TEMPLATE.spec} mode="preview" />,
    );
    expect(container.querySelector(".tpl-page")).not.toBeNull();
    expect(container.querySelector(".tpl-slot")).not.toBeNull();
  });

  it("renders a live slot without the placeholder hint", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = renderWithProviders(
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

  it("keeps secondary bound slots visible when the primary slot is placed", () => {
    const spec = node("page", "page", {}, [
      node("primary", "slot", { componentId: "asset-primary" }),
      node("secondary", "slot", { componentId: "asset-secondary" }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        slot={<span>Primary component</span>}
        slotPlacement={{ x: 20, y: 20, w: 220, h: 64 }}
        slotEditable
        renderComponentSlot={(slot) =>
          slot.id === "secondary" ? <span>Secondary component</span> : undefined
        }
      />,
    );

    const secondary = container.querySelector<HTMLElement>('[data-node-id="secondary"]');
    expect(secondary?.textContent).toContain("Secondary component");
    expect(secondary).not.toHaveClass("is-placed-anchor");
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
    const { container, rerender } = renderWithProviders(
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

  it("fills the chosen slot and leaves the other slots on their fallbacks", () => {
    const spec = node("page", "page", {}, [
      slotNode("action", {
        role: "action",
        accepts: ["button"],
        children: [node("action.fallback", "text", { text: "Action fallback" })],
      }),
      slotNode("metric", {
        role: "metric",
        accepts: ["metric"],
        children: [node("metric.fallback", "text", { text: "Metric fallback" })],
      }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        activeSlotId="metric"
        slot={<span>Live metric</span>}
      />,
    );
    const action = container.querySelector<HTMLElement>('[data-node-id="action"]');
    const metric = container.querySelector<HTMLElement>('[data-node-id="metric"]');
    expect(action?.textContent).toContain("Action fallback");
    expect(action).not.toHaveClass("is-filled");
    expect(metric?.textContent).toContain("Live metric");
    expect(metric).toHaveClass("is-filled");
  });

  it("floats a placed component above the authored page and keeps the anchor", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = renderWithProviders(
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
    renderWithProviders(
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

  it("moves and resizes the placed component with pointer gestures", () => {
    const template = DEFAULT_TEMPLATE;
    const onChange = vi.fn();
    const placement = { x: 42, y: 96, w: 240, h: 72 };
    const { rerender } = renderWithProviders(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="preview"
        slot={<button type="button">Live action</button>}
        slotEditable
        slotPlacement={placement}
        onSlotPlacementChange={onChange}
      />,
    );

    const moveGrip = screen.getByRole("button", { name: "workbench.template.moveComponent" });
    fireEvent.pointerDown(moveGrip, {
      button: 0,
      pointerId: 1,
      pointerType: "touch",
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerMove(window, {
      pointerId: 1,
      clientX: 101,
      clientY: 100,
    });
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.pointerMove(window, {
      pointerId: 1,
      clientX: 116,
      clientY: 0,
    });
    fireEvent.pointerUp(window, { pointerId: 1 });

    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 58,
      y: 0,
      w: 240,
      h: 72,
    });

    const movedPlacement = { x: 58, y: 0, w: 240, h: 72 };
    rerender(
      <TemplateRenderer
        template={template}
        spec={template.spec}
        mode="preview"
        slot={<button type="button">Live action</button>}
        slotEditable
        slotPlacement={movedPlacement}
        onSlotPlacementChange={onChange}
      />,
    );

    const resizeGrip = screen.getByRole("button", { name: "workbench.template.resizeComponent" });
    fireEvent.pointerDown(resizeGrip, {
      button: 0,
      pointerId: 2,
      pointerType: "touch",
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerMove(window, {
      pointerId: 2,
      clientX: 124,
      clientY: -100,
    });
    fireEvent.pointerUp(window, { pointerId: 2 });

    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 58,
      y: 0,
      w: 264,
      h: 24,
    });
  });

  it("marks the selected node for the studio outline", () => {
    const template = DEFAULT_TEMPLATE;
    const { container } = renderWithProviders(
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

  it("reparents a layer dropped onto a container in the canvas", () => {
    const spec = node("page", "page", {}, [
      node("source", "text", { text: "Move me" }),
      node("target", "card", {}, [node("target-copy", "text", { text: "Drop here" })]),
    ]);
    const onMoveLayer = vi.fn();
    const onSelect = vi.fn();
    const { container } = renderWithProviders(
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
      fireEvent.pointerDown(source, {
        button: 0,
        pointerId: 1,
        clientX: 2,
        clientY: 2,
      });
      fireEvent.pointerMove(root, {
        pointerId: 1,
        clientX: 60,
        clientY: 60,
      });
      expect(root).toHaveAttribute("data-layer-dragging", "true");
      expect(target).toHaveAttribute("data-canvas-drop", "inside");
      fireEvent.pointerUp(root, { pointerId: 1, clientX: 60, clientY: 60 });
    } finally {
      if (previousElementFromPoint) {
        Object.defineProperty(document, "elementFromPoint", previousElementFromPoint);
      } else {
        Reflect.deleteProperty(document, "elementFromPoint");
      }
    }

    expect(onMoveLayer).toHaveBeenCalledWith("source", "target", "inside");
    expect(onSelect).toHaveBeenCalledWith("source");
    expect(root).not.toHaveAttribute("data-layer-dragging");
  });

  it("shows published primitives and custom layers in the RCL map", () => {
    const spec = node("page", "page", {}, [
      node("action", "button", { text: "Continue" }),
      node("headline", "text", { text: "Welcome" }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        coverageMode
        onSelect={() => undefined}
      />,
    );

    expect(container.querySelector(".tpl")).toHaveAttribute("data-coverage-mode", "true");
    expect(container.querySelector('[data-node-id="page"]')).not.toHaveAttribute(
      "data-rcl-coverage",
    );
    expect(container.querySelector('[data-node-id="action"]')).toHaveAttribute(
      "data-rcl-coverage",
      "rcl",
    );
    expect(container.querySelector('[data-node-id="headline"]')).toHaveAttribute(
      "data-rcl-coverage",
      "custom",
    );
  });

  it("switches an imported page between editable layers and its isolated source view", () => {
    const spec = node("page", "page", { sourceHtml: "<!doctype html><h1>Original</h1>" }, [
      node("headline", "text", { text: "Editable" }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="canvas" />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.template.sourceViewOriginal) }),
    );
    const source = container.querySelector("iframe.tpl-source-frame");
    expect(source).toHaveAttribute("srcdoc", "<!doctype html><h1>Original</h1>");
    expect(source).toHaveAttribute("sandbox", "");
    expect(container.querySelector('[data-node-id="headline"]')).toBeNull();

    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.template.sourceViewLayers) }),
    );
    expect(container.querySelector('[data-node-id="headline"]')).not.toBeNull();
    expect(container.querySelector("iframe.tpl-source-frame")).toBeNull();
  });

  it("selects the closest layer in canvas mode and stays inert in preview", () => {
    const spec = node("page", "page", {}, [
      node("card", "card", {}, [
        node("copy", "text", { text: i18n.t(strings.workbench.template.move) }),
      ]),
    ]);
    const onSelect = vi.fn();
    const { container, rerender } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        onSelect={onSelect}
      />,
    );

    const copy = container.querySelector<HTMLElement>('[data-node-id="copy"]');
    if (!copy) throw new Error("Selectable text layer was not rendered.");
    fireEvent.click(copy);
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith("copy");

    onSelect.mockClear();
    rerender(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        onSelect={onSelect}
      />,
    );
    const previewCopy = container.querySelector<HTMLElement>('[data-node-id="copy"]');
    if (!previewCopy) throw new Error("Preview text layer was not rendered.");
    fireEvent.click(previewCopy);
    expect(onSelect).not.toHaveBeenCalled();
    expect(container.querySelector(".tpl")).toHaveClass("is-preview");
  });

  it("renders imported source children in preview without exposing the source switch", () => {
    const spec = node("page", "page", { sourceHtml: "<h1>Original</h1>" }, [
      node("imported-title", "text", {
        text: i18n.t(strings.workbench.template.componentSlot),
      }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    expect(container.querySelector('[data-node-id="imported-title"]')).toBeInTheDocument();
    expect(container.querySelector("iframe.tpl-source-frame")).toBeNull();
    expect(screen.queryByRole("button", { name: /original source/i })).toBeNull();
  });

  it("uses the thumbnail presentation without enabling canvas source controls", () => {
    const spec = node("page", "page", { sourceHtml: "<h1>Original</h1>" }, [
      node("headline", "text", { text: i18n.t(strings.workbench.template.move) }),
    ]);
    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="thumb" />,
    );

    expect(container.querySelector(".tpl")).toHaveClass("is-thumb");
    expect(container.querySelector('[data-node-id="headline"]')).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /original source/i })).toBeNull();
  });

  it("ignores canvas pointer gestures that do not produce a valid layer drop", () => {
    const spec = node("page", "page", {}, [node("source", "text", { text: "Move me" })]);
    const onMoveLayer = vi.fn();
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        onSelect={() => undefined}
        onMoveLayer={onMoveLayer}
      />,
    );
    const root = container.querySelector<HTMLElement>(".tpl");
    const source = container.querySelector<HTMLElement>('[data-node-id="source"]');
    if (!root || !source) throw new Error("Canvas source layer was not rendered.");

    const previousElementFromPoint = Object.getOwnPropertyDescriptor(document, "elementFromPoint");
    Object.defineProperty(document, "elementFromPoint", {
      configurable: true,
      value: () => source,
    });

    try {
      fireEvent.pointerDown(source, {
        button: 0,
        pointerId: 4,
        pointerType: "touch",
        clientX: 4,
        clientY: 4,
      });
      fireEvent.pointerMove(root, { pointerId: 5, clientX: 20, clientY: 20 });
      fireEvent.pointerMove(root, { pointerId: 4, clientX: 4, clientY: 4 });
      expect(root).not.toHaveAttribute("data-layer-dragging");

      fireEvent.pointerMove(root, { pointerId: 4, clientX: 20, clientY: 20 });
      expect(root).toHaveAttribute("data-layer-dragging", "true");
      fireEvent.pointerUp(root, { pointerId: 4 });
    } finally {
      if (previousElementFromPoint) {
        Object.defineProperty(document, "elementFromPoint", previousElementFromPoint);
      } else {
        Reflect.deleteProperty(document, "elementFromPoint");
      }
    }

    expect(onMoveLayer).not.toHaveBeenCalled();
    expect(root).not.toHaveAttribute("data-layer-dragging");
  });

  it("releases a controlled placement when its owner clears it", () => {
    const placement = { x: 32, y: 48, w: 200, h: 64 };
    const spec = node("page", "page", {}, [slotNode("metric")]);
    const { container, rerender } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        slot={<span>Live metric</span>}
        slotPlacement={placement}
        slotEditable
      />,
    );
    const move = screen.getByRole("button", { name: "workbench.template.moveComponent" });
    fireEvent.keyDown(move, { key: "ArrowRight" });
    expect(container.querySelector(".tpl-slot-float")).toHaveStyle({
      left: "min(32px, calc(100% - 200px))",
    });

    rerender(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="preview"
        slot={<span>Live metric</span>}
        slotEditable
      />,
    );

    expect(container.querySelector(".tpl-slot-float")).toBeNull();
  });

  it("uses before and after positions for drops outside a container's inner area", () => {
    const spec = node("page", "page", {}, [
      node("source", "text", { text: "Move me" }),
      node("target", "card", {}, [node("target-copy", "text", { text: "Drop here" })]),
    ]);
    const onMoveLayer = vi.fn();
    const { container } = renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={spec}
        mode="canvas"
        onSelect={() => undefined}
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
    const original = Object.getOwnPropertyDescriptor(document, "elementFromPoint");
    Object.defineProperty(document, "elementFromPoint", {
      configurable: true,
      value: () => target,
    });

    try {
      for (const [pointerId, clientY, expected] of [
        [1, 25, "before"],
        [2, 95, "after"],
      ] as const) {
        fireEvent.pointerDown(source, { button: 0, pointerId, clientX: 2, clientY: 2 });
        fireEvent.pointerMove(root, { pointerId, clientX: 40, clientY });
        fireEvent.pointerUp(root, { pointerId, clientX: 40, clientY });
        expect(onMoveLayer).toHaveBeenLastCalledWith("source", "target", expected);
      }
    } finally {
      if (original) Object.defineProperty(document, "elementFromPoint", original);
      else Reflect.deleteProperty(document, "elementFromPoint");
    }
  });

  it("ignores non-arrow move keys and resize keys that do not change width", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={DEFAULT_TEMPLATE.spec}
        mode="preview"
        slot={<span>Live metric</span>}
        slotEditable
        slotPlacement={{ x: 10, y: 20, w: 120, h: 48 }}
        onSlotPlacementChange={onChange}
      />,
    );

    fireEvent.keyDown(screen.getByRole("button", { name: "workbench.template.moveComponent" }), {
      key: "Enter",
    });
    fireEvent.keyDown(screen.getByRole("button", { name: "workbench.template.resizeComponent" }), {
      key: "ArrowUp",
    });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("uses shifted keyboard steps and clamps component placement to usable bounds", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <TemplateRenderer
        template={DEFAULT_TEMPLATE}
        spec={DEFAULT_TEMPLATE.spec}
        mode="preview"
        slot={<span>Live metric</span>}
        slotEditable
        slotPlacement={{ x: 0, y: 0, w: 80, h: 24 }}
        onSlotPlacementChange={onChange}
      />,
    );

    const move = screen.getByRole("button", { name: "workbench.template.moveComponent" });
    fireEvent.keyDown(move, { key: "ArrowLeft", shiftKey: true });
    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 0,
      y: 0,
      w: 80,
      h: 24,
    });
    fireEvent.keyDown(move, { key: "ArrowDown", shiftKey: true });
    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 0,
      y: 24,
      w: 80,
      h: 24,
    });

    const resize = screen.getByRole("button", { name: "workbench.template.resizeComponent" });
    fireEvent.keyDown(resize, { key: "ArrowLeft" });
    expect(onChange).toHaveBeenLastCalledWith(expect.any(String), {
      x: 0,
      y: 0,
      w: 80,
      h: 24,
    });
    fireEvent.keyDown(resize, { key: "ArrowUp" });
    expect(onChange).toHaveBeenCalledTimes(3);
  });
});
