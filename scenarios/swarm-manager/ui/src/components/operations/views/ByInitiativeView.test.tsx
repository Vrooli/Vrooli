import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { ByInitiativeView } from "./ByInitiativeView";
import type { ActivityRow } from "../../../types/operations";
import { selectors } from "../../../consts/selectors";

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: "a-x",
    runId: "run-x",
    ownerType: "backlog",
    ownerName: "fix-foo",
    purpose: "process",
    status: "running",
    requestedAt: "2026-05-02T01:00:00Z",
    ...overrides,
  };
}

function renderView(rows: ActivityRow[]) {
  return render(
    <MemoryRouter>
      <ByInitiativeView activities={rows} />
    </MemoryRouter>,
  );
}

describe("ByInitiativeView", () => {
  it("renders an initiative card per group", () => {
    renderView([
      row({
        activityId: "a-1",
        runId: "run-1",
        ownerType: "initiative",
        ownerName: "auth-rewrite",
        ownerTitle: "Auth Rewrite",
        initiativeName: "auth-rewrite",
        mode: "holistic-loop",
        phase: "execute",
        round: 4,
      }),
      row({
        activityId: "a-2",
        runId: "run-2",
        ownerType: "initiative",
        ownerName: "mobile-polish",
        ownerTitle: "Mobile Polish",
        initiativeName: "mobile-polish",
        mode: "phased-plan-drain",
        phase: "execute_next",
        round: 2,
      }),
    ]);
    const cards = screen.getAllByTestId(selectors.operationsCenter.initiativeCard);
    expect(cards).toHaveLength(2);
    expect(cards[0]?.textContent).toContain("auth-rewrite");
    expect(cards[1]?.textContent).toContain("mobile-polish");
  });

  it("places non-initiative activities under a standalone bucket", () => {
    renderView([
      row({
        activityId: "a-3",
        runId: "run-3",
        ownerType: "backlog",
        ownerKind: "fix",
        ownerName: "fix-login-flicker",
        ownerTitle: "Fix login flicker",
        purpose: "process",
      }),
    ]);
    expect(screen.queryByTestId(selectors.operationsCenter.initiativeCard)).toBeNull();
    const standalone = screen.getByTestId(selectors.operationsCenter.standaloneBucket);
    expect(standalone).toBeInTheDocument();
    expect(within(standalone).getByText(/Fix login flicker/)).toBeInTheDocument();
  });

  it("renders both an initiative card and the standalone bucket together", () => {
    renderView([
      row({
        activityId: "a-1",
        ownerType: "initiative",
        ownerName: "auth-rewrite",
        ownerTitle: "Auth Rewrite",
        initiativeName: "auth-rewrite",
      }),
      row({
        activityId: "a-2",
        ownerType: "backlog",
        ownerName: "fix-foo",
      }),
    ]);
    expect(
      screen.getByTestId(selectors.operationsCenter.initiativeCard),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.operationsCenter.standaloneBucket),
    ).toBeInTheDocument();
  });

  it("nests one row per activity inside a group", () => {
    renderView([
      row({
        activityId: "a-1",
        runId: "run-1",
        ownerType: "initiative",
        ownerName: "auth-rewrite",
        initiativeName: "auth-rewrite",
        ownerTitle: "Round 1",
      }),
      row({
        activityId: "a-2",
        runId: "run-2",
        ownerType: "initiative",
        ownerName: "auth-rewrite",
        initiativeName: "auth-rewrite",
        ownerTitle: "Round 2",
      }),
    ]);
    const card = screen.getByTestId(selectors.operationsCenter.initiativeCard);
    const rows = within(card).getAllByTestId(selectors.operationsCenter.activityRow);
    expect(rows).toHaveLength(2);
  });

  it("renders nothing when there are no activities", () => {
    const { container } = renderView([]);
    expect(container.firstChild).toBeNull();
  });
});
