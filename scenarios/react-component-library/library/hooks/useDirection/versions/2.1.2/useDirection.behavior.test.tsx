import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { act, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { useDirection } from "./useDirection";

function Readout({ testId }: { testId: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const direction = useDirection(ref);
  return <div ref={ref} data-testid={testId} data-direction={direction} />;
}

function Unscoped() {
  return <div data-testid="unscoped" data-direction={useDirection()} />;
}

describe("useDirection", () => {
  it("reports left to right by default", () => {
    renderWithProviders(<Unscoped />);
    expect(screen.getByTestId("unscoped").getAttribute("data-direction")).toBe("ltr");
  });

  it("reads the document direction when given no element", async () => {
    document.documentElement.setAttribute("dir", "rtl");
    try {
      renderWithProviders(<Unscoped />);
      expect(screen.getByTestId("unscoped").getAttribute("data-direction")).toBe("rtl");
    } finally {
      // The observer callback is a microtask, so the resulting state update
      // lands outside the synchronous body unless it is awaited inside act.
      await act(async () => {
        document.documentElement.removeAttribute("dir");
      });
    }
  });

  // Refs attach after the first render, so a render-time read resolves against
  // the document and reports the wrong side for a mirrored region.
  it("resolves a mirrored ancestor on the first paint", () => {
    renderWithProviders(
      <div dir="rtl">
        <Readout testId="scoped" />
      </div>,
    );
    expect(screen.getByTestId("scoped").getAttribute("data-direction")).toBe("rtl");
  });

  it("does not let a mirrored sibling leak into an unmirrored subtree", () => {
    renderWithProviders(
      <>
        <div dir="rtl" />
        <Readout testId="scoped" />
      </>,
    );
    expect(screen.getByTestId("scoped").getAttribute("data-direction")).toBe("ltr");
  });

  it("re-renders when the direction changes after mount", async () => {
    renderWithProviders(
      <div dir="ltr" data-testid="region">
        <Readout testId="scoped" />
      </div>,
    );
    expect(screen.getByTestId("scoped").getAttribute("data-direction")).toBe("ltr");
    await act(async () => {
      screen.getByTestId("region").setAttribute("dir", "rtl");
    });
    expect(screen.getByTestId("scoped").getAttribute("data-direction")).toBe("rtl");
  });
});

// Renders the published story specimen through the same path the story-contract
// runner uses, so a disagreement between the two is visible here rather than
// only in the aggregate suite.
describe("story specimens", () => {
  it("resolves the mirrored specimen the contract asserts", async () => {
    const { RightToLeft } = await import("./story");
    await act(async () => {
      renderWithProviders(<RightToLeft />);
    });
    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 60));
    });
    const node = document.querySelector("[data-testid='hooks.use-direction.rtl']");
    expect(node?.getAttribute("data-direction")).toBe("rtl");
  });
});
