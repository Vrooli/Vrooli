import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { AsyncSection } from "./AsyncSection";

afterEach(cleanup);

it("uses the shared experience surface for loading while preserving skeleton rows", () => {
  renderWithProviders(
    <AsyncSection isLoading isError={false} skeletonRows={2}>
      <p>Loaded</p>
    </AsyncSection>,
  );

  const surface = screen.getByTestId(selectors.async.loading);
  expect(surface).toHaveAttribute("data-experience-state", "loading");
  expect(surface).toHaveAttribute("aria-busy", "true");
  expect(screen.getByRole("status", { name: strings.common.loading })).toBeInTheDocument();
  expect(surface.querySelectorAll(".animate-pulse")).toHaveLength(2);
});

it("keeps the localized retry alert and adds shared error state semantics", () => {
  const onRetry = vi.fn();
  renderWithProviders(
    <AsyncSection isLoading={false} isError onRetry={onRetry}>
      <p>Loaded</p>
    </AsyncSection>,
  );

  const surface = screen.getByTestId(selectors.async.error);
  expect(surface).toHaveAttribute("data-experience-state", "error");
  expect(screen.getByRole("status", { name: strings.common.loadError })).toBeInTheDocument();
  expect(screen.getByRole("alert")).toHaveTextContent(strings.common.loadError);
  fireEvent.click(screen.getByTestId(selectors.async.retry));
  expect(onRetry).toHaveBeenCalledOnce();
});

it("marks the empty state without changing its caller-provided content", () => {
  renderWithProviders(
    <AsyncSection isLoading={false} isError={false} isEmpty emptyState={<p>No runs</p>}>
      <p>Loaded</p>
    </AsyncSection>,
  );

  expect(screen.getByTestId(selectors.async.empty)).toHaveAttribute("data-experience-state", "empty");
  expect(screen.getByText("No runs")).toBeInTheDocument();
});

it("leaves the ready branch unwrapped to preserve page layout", () => {
  const { container } = renderWithProviders(
    <AsyncSection isLoading={false} isError={false}>
      <p>Loaded</p>
    </AsyncSection>,
  );

  expect(screen.getByText("Loaded")).toBeInTheDocument();
  expect(container.querySelector("[data-experience-surface]")).toBeNull();
});
