import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  AuthSection,
  EntitlementErrorCard,
  PendingSyncBadge,
  PlanBadge,
  UsageMeter,
} from "./MonetizationAccount";

describe("MonetizationAccount v2", () => {
  it("keeps the account action keyboard reachable", () => {
    render(<AuthSection signedIn={false} onSignIn={() => undefined} onSignOut={() => undefined} />);
    const button = screen.getByRole("button", { name: "Sign in" });
    button.focus();
    expect(button).toHaveFocus();
  });

  it("exposes named usage semantics and a bounded progress value", () => {
    render(<UsageMeter used={7} limit={10} />);
    expect(screen.getByRole("region", { name: "credits usage" })).toBeVisible();
    expect(screen.getByRole("progressbar", { name: "credits usage percent" })).toHaveAttribute("value", "70");
  });

  it("provides a non-color contrast anchor for plan identity", () => {
    render(<PlanBadge plan="pro" />);
    const badge = screen.getByTestId("monetization.account-surface");
    expect(badge).toHaveTextContent("Pro");
    expect(badge).toHaveStyle({ color: "var(--color-primary)", border: expect.stringContaining("var(--color-primary)") });
  });

  it.each([
    "unauthorized",
    "subscription_required",
    "credits_required",
    "authority_unavailable",
    "rate_limited",
    "rank_required",
    "unknown",
  ])("renders the distinct %s error branch", (errorType) => {
    render(<EntitlementErrorCard errorType={errorType} />);
    expect(document.querySelector(`[data-error-type="${errorType}"]`)).toBeVisible();
  });

  it("does not announce an empty pending queue", () => {
    render(<PendingSyncBadge pending={0} />);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
