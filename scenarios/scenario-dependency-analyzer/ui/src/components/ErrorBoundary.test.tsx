import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { ErrorBoundary } from "./ErrorBoundary";

function BrokenComponent(): React.ReactElement {
  throw new Error("broken component");
}

describe("ErrorBoundary", () => {
  it("renders recovery UI and reports caught errors", () => {
    const onError = vi.fn();
    const preventExpectedError = (event: ErrorEvent) => {
      event.preventDefault();
    };
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    window.addEventListener("error", preventExpectedError);

    try {
      renderWithProviders(
        <ErrorBoundary onError={onError}>
          <BrokenComponent />
        </ErrorBoundary>
      );
    } finally {
      window.removeEventListener("error", preventExpectedError);
    }

    expect(screen.getByTestId("sda-error-boundary")).toBeInTheDocument();
    expect(onError).toHaveBeenCalledTimes(1);
  });
});
