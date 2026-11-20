import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { strings } from "../../consts/strings";
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

    expect(screen.getByText(strings.apiError.unimplementedTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.apiError.unimplementedDescription)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.retry" }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("uses the supplied title and ordinary error message without a retry action", () => {
    const customTitle = "Custom failure";
    const networkError = "network down";
    render(<ApiErrorState error={new Error(networkError)} title={customTitle} />);

    expect(screen.getByText(customTitle)).toBeInTheDocument();
    expect(screen.getByText(networkError)).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
