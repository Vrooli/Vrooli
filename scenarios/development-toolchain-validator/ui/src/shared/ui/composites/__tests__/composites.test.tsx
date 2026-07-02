/**
 * Composite smoke tests.
 *
 * Per the testing plan: assert each composite's prop contract — empty
 * → empty state, populated → expected element count, click → handler
 * fires.
 */
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { DiffViewer } from "../DiffViewer";
import { EmptyState } from "../EmptyState";
import { ErrorBoundary, ErrorBoundaryFallback } from "../ErrorBoundary";
import { LoadingSkeleton } from "../LoadingSkeleton";
import { MetadataList } from "../MetadataList";
import { MetricStat } from "../MetricStat";
import { PanelHeader } from "../PanelHeader";
import { StaleBadge } from "../StaleBadge";
import { VerdictCell } from "../VerdictCell";
import { VerdictGrid } from "../VerdictGrid";

describe("PanelHeader", () => {
  it("renders title + description + actions slots", () => {
    render(
      <PanelHeader
        title={<span data-testid="ph-title">t</span>}
        description={<span data-testid="ph-desc">d</span>}
        actions={<button type="button" data-testid="action">a</button>}
      />,
    );
    expect(screen.getByTestId("ph-title")).toBeInTheDocument();
    expect(screen.getByTestId("ph-desc")).toBeInTheDocument();
    expect(screen.getByTestId("action")).toBeInTheDocument();
  });
});

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(
      <EmptyState
        title={<span data-testid="es-title">t</span>}
        description={<span data-testid="es-desc">d</span>}
        testId="es"
      />,
    );
    expect(screen.getByTestId("es")).toBeInTheDocument();
    expect(screen.getByTestId("es-title")).toBeInTheDocument();
    expect(screen.getByTestId("es-desc")).toBeInTheDocument();
  });
});

describe("LoadingSkeleton", () => {
  it("renders count placeholders", () => {
    const { container } = render(<LoadingSkeleton variant="card" count={3} />);
    expect(container.querySelectorAll('[data-variant="card"]').length).toBe(3);
  });
});

describe("MetricStat", () => {
  it("renders label and value", () => {
    render(
      <MetricStat
        label={<span data-testid="ms-label">duration</span>}
        value={<span data-testid="ms-value">42ms</span>}
      />,
    );
    expect(screen.getByTestId("ms-label")).toBeInTheDocument();
    expect(screen.getByTestId("ms-value")).toBeInTheDocument();
  });

  it("renders an optional delta line", () => {
    render(
      <MetricStat
        label="coverage"
        value="94%"
        delta={<span data-testid="ms-delta">+3%</span>}
      />,
    );
    expect(screen.getByTestId("ms-delta")).toHaveTextContent("+3%");
  });
});

describe("MetadataList", () => {
  it("renders one row per item", () => {
    const { container } = render(
      <MetadataList items={[{ label: "k1", value: "v1" }, { label: "k2", value: "v2" }]} />,
    );
    expect(container.querySelectorAll("dt").length).toBe(2);
    expect(container.querySelectorAll("dd").length).toBe(2);
  });
});

describe("StaleBadge", () => {
  it("renders the verdict-stale variant", () => {
    render(<StaleBadge label="stale" reason="why" testId="sb" />);
    expect(screen.getByTestId("sb")).toHaveAttribute("data-variant", "verdict-stale");
  });

  it("renders directly when no reason is provided", () => {
    render(<StaleBadge label="stale" testId="sb-direct" />);
    expect(screen.getByTestId("sb-direct")).toHaveTextContent("stale");
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});

describe("VerdictCell", () => {
  it("fires onClick when clicked", () => {
    const onClick = vi.fn();
    render(<VerdictCell kind="pass" testId="vc" onClick={onClick} />);
    fireEvent.click(screen.getByTestId("vc"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("renders as a div (non-interactive) when no onClick", () => {
    render(<VerdictCell kind="neutral" testId="vc2" />);
    expect(screen.getByTestId("vc2").tagName.toLowerCase()).toBe("div");
  });
});

describe("VerdictGrid", () => {
  it("renders one row per data entry and fires the row click handler with the row id", () => {
    const onRowClick = vi.fn();
    render(
      <VerdictGrid
        testId="vg"
        rows={[
          { id: "r1", label: "Row 1", kind: "pass" },
          { id: "r2", label: "Row 2", kind: "stale" },
        ]}
        onRowClick={onRowClick}
      />,
    );
    expect(screen.getByTestId("vg-row-r1")).toBeInTheDocument();
    expect(screen.getByTestId("vg-row-r2")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("vg-row-r1"));
    expect(onRowClick).toHaveBeenCalledWith("r1");
  });

  it("renders the provided empty state when rows is empty", () => {
    render(
      <VerdictGrid
        testId="vg-empty"
        rows={[]}
        emptyState={<div data-testid="empty">none</div>}
      />,
    );
    expect(screen.getByTestId("empty")).toBeInTheDocument();
  });

  it("renders the default empty state when no empty state is provided", () => {
    render(<VerdictGrid testId="vg-default-empty" caption="No verdicts" rows={[]} />);
    expect(screen.getByTestId("vg-default-empty-empty")).toHaveTextContent("No verdicts");
  });

  it("renders optional row sub-labels and metrics without row click behavior", () => {
    render(
      <VerdictGrid
        testId="vg-static"
        rows={[
          {
            id: "r1",
            label: "Tuple",
            subLabel: "generated root",
            kind: "pass",
            metric: "12ms",
          },
        ]}
      />,
    );
    expect(screen.getByText("generated root")).toBeInTheDocument();
    expect(screen.getByText("12ms")).toBeInTheDocument();
    expect(screen.getByTestId("vg-static-row-r1").tagName.toLowerCase()).toBe("div");
  });
});

describe("DiffViewer", () => {
  it("classifies added and removed lines", () => {
    const diff = `--- a/foo.txt\n+++ b/foo.txt\n@@ -1,1 +1,1 @@\n-old\n+new`;
    const { container } = render(<DiffViewer diff={diff} testId="dv" />);
    const added = container.querySelectorAll('[data-kind="added"]');
    const removed = container.querySelectorAll('[data-kind="removed"]');
    expect(added.length).toBe(1);
    expect(removed.length).toBe(1);
  });

  it("renders an empty state when diff is empty", () => {
    render(<DiffViewer diff="" testId="dv-empty" />);
    expect(screen.getByTestId("dv-empty")).toBeInTheDocument();
  });
});

describe("ErrorBoundary", () => {
  it("catches a thrown render and shows the fallback", () => {
    const Bomb = () => {
      throw new Error("boom");
    };
    // Silence the expected React error during this test render.
    const spy = vi.spyOn(console, "error").mockImplementation(() => undefined);
    try {
      render(
        <ErrorBoundary fallback={<div data-testid="boundary-fallback">caught</div>}>
          <Bomb />
        </ErrorBoundary>,
      );
      expect(screen.getByTestId("boundary-fallback")).toBeInTheDocument();
    } finally {
      spy.mockRestore();
    }
  });

  it("renders ErrorBoundaryFallback standalone", () => {
    render(<ErrorBoundaryFallback onRetry={() => undefined} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});
