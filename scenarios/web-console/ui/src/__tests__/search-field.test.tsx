import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { SearchField } from "../components/SearchField";

/**
 * One search box, one shape. Every surface that adopts `SearchField` inherits
 * the governed InputGroup chrome, so these assertions are about the contract
 * the rest of the app relies on rather than about this component's internals.
 */
describe("SearchField", () => {
  it("renders the governed pill InputGroup and reports every value change", () => {
    const onValueChange = vi.fn();
    render(
      <SearchField
        testId="field"
        value="alpha"
        onValueChange={onValueChange}
        placeholder="Search messages"
        label="Search messages"
      />,
    );

    const group = screen.getByTestId("field-group");
    expect(group).toHaveAttribute("data-rcl-input-group", "true");
    expect(group).toHaveAttribute("data-shape", "pill");
    expect(group).toHaveAttribute("data-size", "md");

    fireEvent.change(screen.getByTestId("field"), { target: { value: "beta" } });
    expect(onValueChange).toHaveBeenLastCalledWith("beta");
  });

  it("clears through the shared trailing action", () => {
    const onValueChange = vi.fn();
    render(
      <SearchField
        testId="field"
        value="alpha"
        onValueChange={onValueChange}
        placeholder="Search"
        clearLabel="Clear search"
      />,
    );

    fireEvent.click(screen.getByTestId("field-clear"));
    expect(onValueChange).toHaveBeenLastCalledWith("");
  });

  it("hides the clear action when there is nothing to clear, and honours a compact size", () => {
    render(
      <SearchField
        testId="field"
        value=""
        onValueChange={() => undefined}
        placeholder="Search"
        size="sm"
      />,
    );

    expect(screen.queryByTestId("field-clear")).toBeNull();
    expect(screen.getByTestId("field-group")).toHaveAttribute("data-size", "sm");
  });
});
