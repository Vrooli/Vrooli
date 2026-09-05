import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { RecentTargetsList } from "./RecentTargetsList";
import type { RecentTarget } from "./hooks/useRecentTargets";

afterEach(() => cleanup());

describe("RecentTargetsList", () => {
  it("renders the empty state when there are no recent targets", () => {
    renderWithProviders(<RecentTargetsList recent={[]} onRemove={vi.fn()} />);
    expect(
      screen.getByTestId(selectors.features.targets.recent.empty),
    ).toBeInTheDocument();
  });

  it("renders one row per recent target", () => {
    const recent: RecentTarget[] = [
      { scenario: "alpha", lastOpenedAt: "2026-01-01T00:00:00.000Z" },
      { scenario: "beta", lastOpenedAt: "2026-01-02T00:00:00.000Z" },
    ];
    renderWithProviders(<RecentTargetsList recent={recent} onRemove={vi.fn()} />);

    expect(screen.getByTestId(selectors.features.targets.recent.root)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.targets.recent.item({ scenario: "alpha" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.targets.recent.item({ scenario: "beta" })),
    ).toBeInTheDocument();
  });

  it("links the Open button to the workspace URL with an encoded path", () => {
    renderWithProviders(
      <RecentTargetsList
        recent={[{ scenario: "needs encoding/here", lastOpenedAt: "2026-01-01T00:00:00.000Z" }]}
        onRemove={vi.fn()}
      />,
    );
    const openLink = screen.getByTestId(
      selectors.features.targets.recent.openButton({ scenario: "needs encoding/here" }),
    );
    expect(openLink).toHaveAttribute("href", "/targets/needs%20encoding%2Fhere");
  });

  it("invokes onRemove when the remove button is pressed", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderWithProviders(
      <RecentTargetsList
        recent={[{ scenario: "alpha", lastOpenedAt: "2026-01-01T00:00:00.000Z" }]}
        onRemove={onRemove}
      />,
    );
    await user.click(
      screen.getByTestId(selectors.features.targets.recent.removeButton({ scenario: "alpha" })),
    );
    expect(onRemove).toHaveBeenCalledWith("alpha");
  });
});
