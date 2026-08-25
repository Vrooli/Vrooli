import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { Chart, type ChartDatum } from "@vrooli/react-component-library/Chart/1.0.1";

const points: ChartDatum[] = [
  { id: "first", label: "January", value: 12, detail: "authoritative" },
  { id: "last", label: "February", value: 18 },
];

describe("Chart", () => {
  it("renders a named plot, exact table, and selectable legend values", async () => {
    const user = userEvent.setup();

    renderWithProviders(<Chart data={points} title="Cash trend" />);

    expect(screen.getAllByRole("img", { name: "Cash trend" })[0]).toBeInTheDocument();
    expect(screen.getByRole("table")).toHaveTextContent("January");
    expect(screen.getByRole("table")).toHaveTextContent("18");
    expect(screen.getByRole("status")).toHaveTextContent("February");

    await user.click(screen.getByRole("button", { name: /January/ }));
    expect(screen.getByRole("status")).toHaveTextContent("January");
    expect(screen.getByRole("status")).toHaveTextContent("authoritative");
  });

  it.each(["success", "refreshing", "stale", "partial-error"] as const)(
    "keeps the data table available in %s state",
    (status) => {
      renderWithProviders(<Chart data={points} title="Trend" status={status} />);

      expect(screen.getByRole("table")).toBeInTheDocument();
      expect(screen.getAllByRole("img", { name: "Trend" })[0]).toBeInTheDocument();
    },
  );

  it("renders empty and failure states with retry actions", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn().mockResolvedValue(undefined);

    const { rerender } = renderWithProviders(
      <Chart data={[]} title="Empty trend" status="empty" emptyMessage="No observations" />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("No observations");

    rerender(
      <Chart data={[]} title="Offline trend" status="offline" onRetry={onRetry} />,
    );
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledOnce();

    rerender(
      <Chart
        data={[]}
        title="Broken trend"
        status="error"
        onRetry={onRetry}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Chart unavailable");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledTimes(2);
  });
});
