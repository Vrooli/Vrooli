import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { GraphNavControls, PAN_AMOUNT } from "./GraphNavControls";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { selectors } from "../../../consts/selectors";

const sel = selectors.graphNavControls;

function createMockFlowInstance() {
  return {
    getViewport: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
    setViewport: vi.fn(),
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    fitView: vi.fn(),
  };
}

describe("GraphNavControls", () => {
  let mockInstance: ReturnType<typeof createMockFlowInstance>;

  beforeEach(() => {
    mockInstance = createMockFlowInstance();
    useGraphUIStore.setState({ flowInstance: mockInstance as never });
  });

  it("renders all 7 navigation buttons", () => {
    render(<GraphNavControls />);
    expect(screen.getByTestId(sel.panUp)).toBeInTheDocument();
    expect(screen.getByTestId(sel.panDown)).toBeInTheDocument();
    expect(screen.getByTestId(sel.panLeft)).toBeInTheDocument();
    expect(screen.getByTestId(sel.panRight)).toBeInTheDocument();
    expect(screen.getByTestId(sel.zoomIn)).toBeInTheDocument();
    expect(screen.getByTestId(sel.zoomOut)).toBeInTheDocument();
    expect(screen.getByTestId(sel.fitView)).toBeInTheDocument();
  });

  it("pan-left shifts viewport x positively", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.panLeft));
    expect(mockInstance.setViewport).toHaveBeenCalledWith(
      { x: PAN_AMOUNT, y: 0, zoom: 1 },
      { duration: 200 },
    );
  });

  it("pan-right shifts viewport x negatively", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.panRight));
    expect(mockInstance.setViewport).toHaveBeenCalledWith(
      { x: -PAN_AMOUNT, y: 0, zoom: 1 },
      { duration: 200 },
    );
  });

  it("pan-up shifts viewport y positively", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.panUp));
    expect(mockInstance.setViewport).toHaveBeenCalledWith(
      { x: 0, y: PAN_AMOUNT, zoom: 1 },
      { duration: 200 },
    );
  });

  it("pan-down shifts viewport y negatively", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.panDown));
    expect(mockInstance.setViewport).toHaveBeenCalledWith(
      { x: 0, y: -PAN_AMOUNT, zoom: 1 },
      { duration: 200 },
    );
  });

  it("zoom-in delegates to flowInstance.zoomIn", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.zoomIn));
    expect(mockInstance.zoomIn).toHaveBeenCalledWith({ duration: 200 });
  });

  it("zoom-out delegates to flowInstance.zoomOut", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.zoomOut));
    expect(mockInstance.zoomOut).toHaveBeenCalledWith({ duration: 200 });
  });

  it("fit-to-view delegates to flowInstance.fitView", () => {
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.fitView));
    expect(mockInstance.fitView).toHaveBeenCalledWith({
      padding: 0.2,
      maxZoom: 1.2,
      duration: 300,
    });
  });

  it("does not throw when flowInstance is null", () => {
    useGraphUIStore.setState({ flowInstance: null });
    render(<GraphNavControls />);

    // Click every button — none should throw
    fireEvent.click(screen.getByTestId(sel.panLeft));
    fireEvent.click(screen.getByTestId(sel.panRight));
    fireEvent.click(screen.getByTestId(sel.panUp));
    fireEvent.click(screen.getByTestId(sel.panDown));
    fireEvent.click(screen.getByTestId(sel.zoomIn));
    fireEvent.click(screen.getByTestId(sel.zoomOut));
    fireEvent.click(screen.getByTestId(sel.fitView));

    // If we got here without errors, the test passes
    expect(screen.getByTestId(sel.container)).toBeInTheDocument();
  });

  it("uses current viewport position for pan calculations", () => {
    mockInstance.getViewport.mockReturnValue({ x: 50, y: -30, zoom: 1.5 });
    render(<GraphNavControls />);
    fireEvent.click(screen.getByTestId(sel.panLeft));
    expect(mockInstance.setViewport).toHaveBeenCalledWith(
      { x: 50 + PAN_AMOUNT, y: -30, zoom: 1.5 },
      { duration: 200 },
    );
  });
});
