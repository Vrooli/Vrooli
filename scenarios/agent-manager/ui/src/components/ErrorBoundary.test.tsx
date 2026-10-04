import { createElement } from "react";
import { fireEvent, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ErrorBoundary } from "./ErrorBoundary";
import { renderWithProviders } from "../test-utils";

afterEach(() => vi.restoreAllMocks());

test("section failure retries successfully after its dependency recovers", () => {
  vi.spyOn(console, "error").mockImplementation(() => undefined);
  let failing = true;
  function Child() {
    if (failing) throw new Error("History unavailable");
    return createElement("p", null, "Recovered history");
  }
  renderWithProviders(createElement(ErrorBoundary, { section: "History" }, createElement(Child)));
  expect(screen.getByText("History encountered an error")).toBeInTheDocument();
  expect(screen.getByText("History unavailable")).toBeInTheDocument();
  failing = false;
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(screen.getByText("Recovered history")).toBeInTheDocument();
  expect(screen.queryByText("History unavailable")).not.toBeInTheDocument();
});

test("default section and custom fallback handle unknown rendering failures", () => {
  vi.spyOn(console, "error").mockImplementation(() => undefined);
  function Child() { throw new Error("Dependency unavailable"); }
  const { unmount } = renderWithProviders(createElement(ErrorBoundary, null, createElement(Child)));
  expect(screen.getByText("This section encountered an error")).toBeInTheDocument();
  expect(screen.getByText("Dependency unavailable")).toBeInTheDocument();
  unmount();
  renderWithProviders(createElement(ErrorBoundary, { fallback: createElement("p", null, "Custom recovery") }, createElement(Child)));
  expect(screen.getByText("Custom recovery")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
});
