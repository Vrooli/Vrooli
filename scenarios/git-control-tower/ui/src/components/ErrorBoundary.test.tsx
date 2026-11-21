import { fireEvent, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { ErrorBoundary } from "./ErrorBoundary";

it("offers deliberate stale-module recovery without clearing drafts or reloading automatically", () => {
  localStorage.setItem("gct.commitMessage", "Operator draft");
  sessionStorage.setItem(
    "gct.proposals.draft.example",
    "Retained proposal wording",
  );
  const reload = vi.fn();
  renderWithProviders(
    <ErrorBoundary onReload={reload}>
      <div>Workspace</div>
    </ErrorBoundary>,
  );
  const failure = new Event("vite:preloadError", { cancelable: true });
  fireEvent(window, failure);
  expect(failure.defaultPrevented).toBe(true);
  expect(screen.getByRole("heading")).toHaveTextContent(
    "An updated UI is available",
  );
  expect(reload).not.toHaveBeenCalled();
  expect(
    screen.queryByRole("button", { name: /reset/i }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Reload updated UI" }));
  expect(reload).toHaveBeenCalledOnce();
  expect(localStorage.getItem("gct.commitMessage")).toBe("Operator draft");
  expect(sessionStorage.getItem("gct.proposals.draft.example")).toBe(
    "Retained proposal wording",
  );
});
