import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { ApiErrorState } from "./ApiErrorState";

describe("ApiErrorState", () => {
  it("renders the unimplemented explanation and an optional retry action", () => {
    const retry = vi.fn();
    render(
      <ApiErrorState
        error={{ code: "unimplemented", message: "not used" } as never}
        onRetry={retry}
      />,
    );

    expect(screen.getByText("apiError.unimplementedTitle")).toBeInTheDocument();
    expect(screen.getByText("apiError.unimplementedDescription")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.retry" }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("uses the supplied title and ordinary error message without a retry action", () => {
    render(<ApiErrorState error={new Error("network down")} title="Custom failure" />);

    expect(screen.getByText("Custom failure")).toBeInTheDocument();
    expect(screen.getByText("network down")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
