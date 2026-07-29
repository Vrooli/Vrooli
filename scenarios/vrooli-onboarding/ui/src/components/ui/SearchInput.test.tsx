import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
// provider-free-exception: SearchInput is a controlled presentation primitive with no provider dependency.
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { SearchInput, type SearchInputHandle } from "./SearchInput";

// [REQ:ONBOARD-SMART-FLOW-001] Search input component tests

const baseProps = {
  value: "",
  onChange: vi.fn(),
  ariaLabel: "Search items",
  testId: "search-input",
  clearTestId: "clear-search",
};

describe("SearchInput", () => {
  it("renders the search input with placeholder", () => {
    render(<SearchInput {...baseProps} placeholder="Find something" />);
    expect(screen.getByPlaceholderText("Find something")).toBeInTheDocument();
  });

  it("uses default placeholder when none provided", () => {
    render(<SearchInput {...baseProps} />);
    expect(screen.getByPlaceholderText("Search...")).toBeInTheDocument();
  });

  it("sets the aria-label", () => {
    render(<SearchInput {...baseProps} />);
    expect(screen.getByRole("searchbox")).toHaveAttribute("aria-label", "Search items");
  });

  it("sets data-testid on the input", () => {
    render(<SearchInput {...baseProps} />);
    expect(screen.getByTestId("search-input")).toBeInTheDocument();
  });

  it("calls onChange when user types", async () => {
    const onChange = vi.fn();
    render(<SearchInput {...baseProps} onChange={onChange} />);
    const input = screen.getByTestId("search-input");
    await userEvent.type(input, "a");
    expect(onChange).toHaveBeenCalledWith("a");
  });

  it("does not show clear button when value is empty", () => {
    render(<SearchInput {...baseProps} value="" />);
    expect(screen.queryByTestId("clear-search")).not.toBeInTheDocument();
  });

  it("shows clear button when value is non-empty and not busy", () => {
    render(<SearchInput {...baseProps} value="hello" />);
    expect(screen.getByTestId("clear-search")).toBeInTheDocument();
  });

  it("calls onChange with empty string when clear is clicked", async () => {
    const onChange = vi.fn();
    render(<SearchInput {...baseProps} value="hello" onChange={onChange} />);
    await userEvent.click(screen.getByTestId("clear-search"));
    expect(onChange).toHaveBeenCalledWith("");
  });

  it("shows busy indicator instead of clear button when busy", () => {
    render(<SearchInput {...baseProps} value="hello" busy busyTestId="busy-spinner" />);
    expect(screen.getByTestId("busy-spinner")).toBeInTheDocument();
    expect(screen.queryByTestId("clear-search")).not.toBeInTheDocument();
  });

  it("does not show busy indicator when not busy", () => {
    render(<SearchInput {...baseProps} value="hello" busyTestId="busy-spinner" />);
    expect(screen.queryByTestId("busy-spinner")).not.toBeInTheDocument();
  });

  it("exposes focus via ref", () => {
    const ref = createRef<SearchInputHandle>();
    render(<SearchInput {...baseProps} ref={ref} />);
    const input = screen.getByTestId("search-input");
    input.blur();
    ref.current?.focus();
    expect(document.activeElement).toBe(input);
  });
});
