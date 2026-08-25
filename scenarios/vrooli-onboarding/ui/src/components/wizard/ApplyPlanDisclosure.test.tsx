import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ApplyPlanDisclosure } from "./ApplyPlanDisclosure";
import type { ApplyPlanItem } from "../../lib/applyPlan";

const gitItem: ApplyPlanItem = { id: "tool:git", kind: "tool", name: "git", required: true, state: "satisfied" };
const jqItem: ApplyPlanItem = { id: "tool:jq", kind: "tool", name: "jq", required: false, state: "satisfied" };
const hardeningItem: ApplyPlanItem = { id: "safeguard:host_hardening", kind: "safeguard", name: "host_hardening", required: true, privileged: true, state: "pending" };
const clockItem: ApplyPlanItem = { id: "safeguard:clock", kind: "safeguard", name: "clock", required: true, state: "pending" };
const postgresItem: ApplyPlanItem = { id: "resource:postgres", kind: "resource", name: "postgres", required: true };

const allItems = [gitItem, jqItem, hardeningItem, clockItem, postgresItem];

describe("ApplyPlanDisclosure", () => {
  it("separates what changes the host from what is already in place", () => {
    // Before this the UI rendered one flat list, so an operator on a fully
    // configured host saw every item as a pending change.
    render(<ApplyPlanDisclosure items={allItems} />);
    const pending = screen.getByTestId("apply-plan-pending");
    expect(within(pending).getByText(/host_hardening/)).toBeInTheDocument();
    expect(within(pending).queryByText(/\bgit\b/)).toBeNull();

    const satisfied = screen.getByTestId("apply-plan-satisfied");
    expect(within(satisfied).getByText(/git, jq/)).toBeInTheDocument();
  });

  it("marks every elevated item and counts them, instead of hedging", () => {
    render(<ApplyPlanDisclosure items={allItems} />);
    const elevated = screen.getByTestId("apply-plan-item-safeguard:host_hardening");
    expect(within(elevated).getByLabelText("elevated")).toBeInTheDocument();
    expect(elevated.textContent).toContain("required, elevated");

    const notElevated = screen.getByTestId("apply-plan-item-safeguard:clock");
    expect(within(notElevated).queryByLabelText("elevated")).toBeNull();

    expect(screen.getByTestId("privilege-warning").textContent).toContain("1 item run");
  });

  it("says plainly when nothing needs elevation", () => {
    render(<ApplyPlanDisclosure items={[gitItem]} />);
    expect(screen.getByTestId("privilege-warning").textContent).toContain("No item in this plan requires elevated privilege");
  });

  it("discloses what apply does and that it removes nothing", () => {
    render(<ApplyPlanDisclosure items={allItems} />);
    const effects = screen.getByTestId("apply-plan-effects");
    expect(effects.textContent).toContain("vrooli host safeguard <name>");
    expect(effects.textContent).toContain("Nothing is removed");
  });

  it("never claims an item with no reported state is already in place", () => {
    render(<ApplyPlanDisclosure items={[postgresItem]} />);
    expect(screen.getByTestId("apply-plan-unknown")).toBeInTheDocument();
    expect(screen.queryByTestId("apply-plan-satisfied")).toBeNull();
  });

  it("keeps the empty-plan message", () => {
    render(<ApplyPlanDisclosure items={[]} />);
    expect(screen.getByTestId("apply-plan").textContent).toContain("no consented host changes");
  });
});
