import { afterEach, describe, expect, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { COMPONENT_STYLE_ID_ATTRIBUTE, useComponentStyles } from "./useComponentStyles";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.2.1";
import { Card } from "@vrooli/react-component-library/Card/1.2.1";
import { renderWithProviders as render } from "../test-utils";

const CSS_A = "[data-probe-a] { color: red; }";
const CSS_B = "[data-probe-b] { color: blue; }";

function Probe({ id, css }: { id: string; css: string }) {
  useComponentStyles(id, css);
  return <span data-probe />;
}

function sheets(id: string): HTMLStyleElement[] {
  return Array.from(
    document.head.querySelectorAll<HTMLStyleElement>(
      `style[${COMPONENT_STYLE_ID_ATTRIBUTE}="${id}"]`,
    ),
  );
}

function librarySheets(name: string): HTMLStyleElement[] {
  return Array.from(
    document.head.querySelectorAll<HTMLStyleElement>(`style[data-rcl-sheet="${name}"]`),
  );
}

describe("useComponentStyles", () => {
  afterEach(() => cleanup());

  it("injects one stylesheet however many instances ask for it", () => {
    render(
      <>
        <Probe id="probe-many" css={CSS_A} />
        <Probe id="probe-many" css={CSS_A} />
        <Probe id="probe-many" css={CSS_A} />
        <Probe id="probe-many" css={CSS_A} />
        <Probe id="probe-many" css={CSS_A} />
      </>,
    );

    const injected = sheets("probe-many");
    expect(injected).toHaveLength(1);
    expect(injected[0]?.textContent).toBe(CSS_A);
  });

  it.each([1, 3, 10])("keeps the stylesheet count constant for %i instances", (count) => {
    const { container } = render(
      <>
        {Array.from({ length: count }, (_, index) => (
          <Probe key={index} id="probe-cardinality" css={CSS_A} />
        ))}
      </>,
    );

    expect(container.querySelectorAll("style")).toHaveLength(0);
    expect(sheets("probe-cardinality")).toHaveLength(1);
  });

  it("uses a real stylesheet element in document.head, not an inline JSX style tag", () => {
    const { container } = render(<Probe id="probe-head" css={CSS_A} />);

    expect(container.querySelector("style")).toBeNull();
    const injected = sheets("probe-head")[0];
    expect(injected?.parentElement).toBe(document.head);
    expect(injected?.tagName).toBe("STYLE");
  });

  it("places library rules before consumer head styles", () => {
    const consumer = document.createElement("style");
    consumer.textContent = ".consumer-override { color: blue; }";
    document.head.appendChild(consumer);
    try {
      render(<Probe id="probe-order" css={CSS_A} />);

      const injected = sheets("probe-order")[0];
      expect(injected).toBeDefined();
      expect(injected!.compareDocumentPosition(consumer) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
        Node.DOCUMENT_POSITION_FOLLOWING,
      );
    } finally {
      consumer.remove();
    }
  });

  it("keeps the sheet while any instance is still mounted", () => {
    const first = render(<Probe id="probe-refcount" css={CSS_A} />);
    const second = render(<Probe id="probe-refcount" css={CSS_A} />);
    expect(sheets("probe-refcount")).toHaveLength(1);

    first.unmount();
    // The second instance still needs it — over-removal would unstyle a live component.
    expect(sheets("probe-refcount")).toHaveLength(1);

    second.unmount();
    expect(sheets("probe-refcount")).toHaveLength(0);
  });

  it("does not leak across repeated mount/unmount cycles", () => {
    for (let cycle = 0; cycle < 5; cycle += 1) {
      const view = render(<Probe id="probe-cycle" css={CSS_A} />);
      expect(sheets("probe-cycle")).toHaveLength(1);
      view.unmount();
      expect(sheets("probe-cycle")).toHaveLength(0);
    }
    expect(document.head.querySelectorAll(`style[${COMPONENT_STYLE_ID_ATTRIBUTE}]`)).toHaveLength(
      0,
    );
  });

  it("keeps distinct ids as distinct sheets", () => {
    render(
      <>
        <Probe id="probe-distinct-a" css={CSS_A} />
        <Probe id="probe-distinct-b" css={CSS_B} />
      </>,
    );

    expect(sheets("probe-distinct-a")[0]?.textContent).toBe(CSS_A);
    expect(sheets("probe-distinct-b")[0]?.textContent).toBe(CSS_B);
  });

  it("re-attaches a sheet whose element was detached behind its back", () => {
    const view = render(<Probe id="probe-detached" css={CSS_A} />);
    sheets("probe-detached")[0]?.remove();
    expect(sheets("probe-detached")).toHaveLength(0);

    render(<Probe id="probe-detached" css={CSS_A} />);
    expect(sheets("probe-detached")).toHaveLength(1);

    view.unmount();
  });

  describe("real components", () => {
    it("emits one sheet for five StatusBadge instances", () => {
      const { container } = render(
        <div>
          <StatusBadge>one</StatusBadge>
          <StatusBadge>two</StatusBadge>
          <StatusBadge>three</StatusBadge>
          <StatusBadge>four</StatusBadge>
          <StatusBadge>five</StatusBadge>
        </div>,
      );

      // Before the shared injector this rendered five byte-identical <style>
      // blocks inline in the tree; now it is one sheet in <head>.
      expect(container.querySelectorAll("style")).toHaveLength(0);
      expect(librarySheets("status-badge")).toHaveLength(1);
    });

    it("shares nothing between different components", () => {
      render(
        <div>
          <StatusBadge>badge</StatusBadge>
          <Card>card</Card>
        </div>,
      );

      expect(librarySheets("status-badge")).toHaveLength(1);
      expect(librarySheets("card-1-2-1")).toHaveLength(1);
    });
  });
});
