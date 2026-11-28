import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SearchInput } from "@vrooli/react-component-library/SearchInput/1.2.1";

afterEach(cleanup);

const suggestions = [
  { value: "Search inputs", description: "Fields that collect a query." },
  { value: "Search results", description: "The collection that answers it." },
];

describe("SearchInput labelled form field", () => {
  it("names the input from its visible label and reports value-first changes", () => {
    const onValueChange = vi.fn();
    render(
      <SearchInput
        label="Search pages"
        placeholder="Search pages"
        onValueChange={onValueChange}
      />,
    );

    const input = screen.getByRole("searchbox", { name: "Search pages" });
    fireEvent.change(input, { target: { value: "continuity" } });

    expect(onValueChange).toHaveBeenCalledWith("continuity");
    expect(input).toHaveValue("continuity");
  });

  it("keeps a hidden label in the accessibility tree", () => {
    render(
      <SearchInput
        label="Search the library"
        labelVisibility="hidden"
        appearance="toolbar"
        placeholder="Search"
      />,
    );

    expect(screen.getByRole("searchbox", { name: "Search the library" })).toBeInTheDocument();
    expect(screen.getByText("Search the library")).toBeInTheDocument();
  });

  it("keeps the legacy native change contract working through the clear action", () => {
    const onChange = vi.fn();
    render(
      <SearchInput
        label="Search"
        defaultValue="release notes"
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByTestId("forms.search-input-clear"));

    expect(screen.getByRole("searchbox")).toHaveValue("");
    expect(onChange).toHaveBeenCalledTimes(1);
    expect((onChange.mock.calls[0]?.[0] as { currentTarget: { value: string } }).currentTarget.value).toBe("");
  });
});

describe("SearchInput suggestions", () => {
  it("filters static suggestions and commits the active option with Enter", () => {
    const onValueChange = vi.fn();
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        defaultValue="sea"
        suggestions={suggestions}
        defaultOpen
        onValueChange={onValueChange}
      />,
    );

    const input = screen.getByRole("combobox", { name: "Search the library" });
    const listbox = screen.getByRole("listbox", { name: "Search suggestions" });
    expect(within(listbox).getAllByRole("option")).toHaveLength(2);

    fireEvent.keyDown(input, { key: "Enter" });

    expect(onValueChange).toHaveBeenLastCalledWith("Search inputs");
    expect(screen.queryByRole("listbox")).toBeNull();
  });

  it("moves the active option with ArrowDown and announces it through aria-activedescendant", () => {
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        defaultValue="search"
        suggestions={suggestions}
        defaultOpen
      />,
    );

    const input = screen.getByRole("combobox");
    expect(input).toHaveAttribute("aria-activedescendant", expect.stringContaining("option-0"));

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(input).toHaveAttribute("aria-activedescendant", expect.stringContaining("option-1"));
  });

  it("offers recent queries when the field is empty", () => {
    const onValueChange = vi.fn();
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        recentQueries={["deployments", "continuity"]}
        defaultOpen
        onValueChange={onValueChange}
      />,
    );

    expect(screen.getByText("Recent searches")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("option", { name: "deployments" }));

    expect(onValueChange).toHaveBeenCalledWith("deployments");
    expect(screen.queryByRole("listbox")).toBeNull();
  });

  it("says a real query has no matches instead of showing an empty panel", () => {
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        defaultValue="zzzz"
        suggestions={suggestions}
        defaultOpen
      />,
    );

    expect(screen.getByText(/No results for/)).toBeInTheDocument();
    expect(screen.getByText(/zzzz/)).toBeInTheDocument();
    expect(screen.queryByRole("option")).toBeNull();
  });
});

describe("SearchInput async search", () => {
  it("debounces rapid edits into one request for the final query", async () => {
    const onSearch = vi.fn().mockResolvedValue(["continuity"]);
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        debounceMs={5}
        onSearch={onSearch}
        defaultOpen
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.change(input, { target: { value: "c" } });
    fireEvent.change(input, { target: { value: "co" } });
    fireEvent.change(input, { target: { value: "continuity" } });

    await waitFor(() => expect(onSearch).toHaveBeenCalledTimes(1));
    expect(onSearch).toHaveBeenCalledWith("continuity", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    await waitFor(() => expect(screen.getByRole("option", { name: "continuity" })).toBeInTheDocument());
  });

  it("aborts an in-flight search when the query changes", async () => {
    let firstSignal: AbortSignal | undefined;
    const onSearch = vi.fn((query: string, { signal }: { signal: AbortSignal }) => {
      if (query === "a") firstSignal = signal;
      return new Promise<readonly string[]>((resolve) => {
        setTimeout(() => resolve([query]), 40);
      });
    });
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        debounceMs={1}
        onSearch={onSearch}
        defaultOpen
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.change(input, { target: { value: "a" } });
    await waitFor(() => expect(onSearch).toHaveBeenCalledWith("a", expect.anything()));

    fireEvent.change(input, { target: { value: "ab" } });
    await waitFor(() => expect(firstSignal?.aborted).toBe(true));
  });

  it("shows the search failure and retries through the host callback", () => {
    const onRetry = vi.fn();
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        defaultOpen
        searchError="The index is unavailable."
        onRetry={onRetry}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("The index is unavailable.");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});

describe("SearchInput state contracts", () => {
  it("marks loading on the input and shows one status line", () => {
    render(
      <SearchInput
        label="Search the library"
        appearance="toolbar"
        defaultOpen
        loading
        suggestions={suggestions}
      />,
    );

    expect(screen.getByRole("combobox", { name: "Search the library" })).toHaveAttribute(
      "aria-busy",
      "true",
    );
    expect(screen.getByText("Searching…")).toBeInTheDocument();
  });

  it("disables the field and withholds the clear action", () => {
    render(
      <SearchInput label="Search" defaultValue="locked" disabled />,
    );

    expect(screen.getByRole("searchbox")).toBeDisabled();
    expect(screen.queryByTestId("forms.search-input-clear")).toBeNull();
  });

  it("associates description and validation with the input", () => {
    render(
      <SearchInput
        label="Search"
        description="Searches titles and descriptions."
        error="Enter a query first."
      />,
    );

    const input = screen.getByRole("searchbox");
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input.getAttribute("aria-describedby")).toContain("search-input-description");
    expect(input.getAttribute("aria-describedby")).toContain("search-input-error");
    expect(screen.getByRole("alert")).toHaveTextContent("Enter a query first.");
  });
});
