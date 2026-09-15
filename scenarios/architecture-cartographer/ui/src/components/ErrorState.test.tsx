import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { ErrorState } from "./ErrorState";

describe("ErrorState", () => {
  it("renders title and message with role=alert", () => {
    render(<ErrorState title="t" message="m" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.shared.errorState.title)).toHaveTextContent("t");
    expect(screen.getByTestId(selectors.shared.errorState.message)).toHaveTextContent("m");
  });

  it("omits the retry button when no retry handler", () => {
    render(<ErrorState title="t" message="m" />);
    expect(screen.queryByTestId(selectors.shared.errorState.retryButton)).toBeNull();
  });

  it("renders retry button and fires the handler", () => {
    const onRetry = vi.fn();
    render(<ErrorState title="t" message="m" retryLabel="retry" onRetry={onRetry} />);
    fireEvent.click(screen.getByTestId(selectors.shared.errorState.retryButton));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
