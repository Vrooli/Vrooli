import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(<EmptyState title="t" description="d" />);
    expect(screen.getByTestId(selectors.shared.emptyState.title)).toHaveTextContent("t");
    expect(screen.getByTestId(selectors.shared.emptyState.description)).toHaveTextContent("d");
  });

  it("renders only title when description omitted", () => {
    render(<EmptyState title="t" />);
    expect(screen.getByTestId(selectors.shared.emptyState.title)).toHaveTextContent("t");
    expect(screen.queryByTestId(selectors.shared.emptyState.description)).toBeNull();
  });

  it("renders the action slot when provided", () => {
    render(<EmptyState title="t" action={<button>act</button>} />);
    expect(screen.getByTestId(selectors.shared.emptyState.action)).toBeInTheDocument();
  });

  it("uses role=status so screen readers announce state changes", () => {
    render(<EmptyState title="t" />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });
});
