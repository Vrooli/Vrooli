import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PreflightCheckList } from "./PreflightCheckList";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("PreflightCheckList", () => {
  it("omits empty checks", () => {
    renderWithProviders(<PreflightCheckList checks={[] as never} />);
    expect(screen.queryByText(/Test cases/)).not.toBeInTheDocument();
  });

  it("presents all check states and an inspectable listening URL", () => {
    renderWithProviders(
      <PreflightCheckList
        checks={
          [
            {
              id: "api",
              name: "API health",
              status: "pass",
              detail: "listening on 19925",
            },
            {
              id: "cache",
              name: "Cache",
              status: "warning",
              detail: "slow start",
            },
            { id: "bundle", name: "Bundle", status: "fail" },
            { id: "legacy", name: "Legacy", status: "skipped" },
          ] as never
        }
      />,
    );
    expect(screen.getByText("Test cases (4)")).toBeInTheDocument();
    expect(screen.getByText("PASS")).toBeInTheDocument();
    expect(screen.getByText("WARN")).toBeInTheDocument();
    expect(screen.getByText("FAIL")).toBeInTheDocument();
    expect(screen.getByText("SKIP")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open" })).toHaveAttribute(
      "href",
      "http://localhost:19925",
    );
  });
});
