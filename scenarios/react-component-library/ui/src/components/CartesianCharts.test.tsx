import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { CartesianCharts } from "./CartesianCharts";

describe("CartesianCharts", () => {
  afterEach(() => cleanup());

  it("renders an accessible empty state", () => {
    render(
      <CartesianCharts title="Version progression" description="Chart description" data={[]} />,
    );

    expect(screen.getByRole("heading", { name: "Version progression" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("No progression data is available.");
    expect(screen.getByRole("table")).toBeInTheDocument();
  });

  it("renders a point without requiring detail text", () => {
    render(
      <CartesianCharts
        title="Version progression"
        data={[{ id: "1.0.0", label: "1.0.0", value: 80 }]}
      />,
    );

    expect(screen.getByRole("img", { name: "Version progression" })).toHaveTextContent("80");
    expect(screen.getByRole("table")).toHaveTextContent("1.0.0");
  });
});
