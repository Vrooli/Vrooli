import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  AuthSection,
  EntitlementErrorCard,
  PendingSyncBadge,
  MonetizationAccountPanel,
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
    expect(screen.getByRole("progressbar", { name: "credits usage percent" })).toHaveAttribute(
      "value",
      "70",
    );
  });

  it("provides a non-color contrast anchor for plan identity", () => {
    render(<PlanBadge plan="pro" />);
    const badge = screen.getByTestId("monetization.account-surface");
    expect(badge).toHaveTextContent("Pro");
    expect(badge).toHaveStyle({
      color: "var(--color-primary)",
      border: expect.stringContaining("var(--color-primary)"),
    });
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

describe("MonetizationAccountPanel", () => {
  const actions = {
    onSignIn: () => undefined,
    onSignOut: () => undefined,
    onRefresh: () => undefined,
    onManageAccount: () => undefined,
    onComparePlans: () => undefined,
  };

  it("presents the shared signed-out journey and accepts app-owned sign-in controls", () => {
    render(
      <MonetizationAccountPanel
        configured={false}
        plan="free"
        status="signed_out"
        statusLabel="Not connected"
        creditsLabel="Unavailable"
        signInDetails={
          <label>
            Optional refresh token
            <input aria-label="Optional refresh token" />
          </label>
        }
        {...actions}
      >
        <div data-testid="app-owned-key-management">Bring your own key</div>
      </MonetizationAccountPanel>,
    );
    expect(screen.getByRole("region", { name: "Subscription and access" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeVisible();
    expect(screen.getByLabelText("Optional refresh token")).toBeVisible();
    expect(screen.getByRole("button", { name: "Manage billing" })).toBeVisible();
    expect(screen.getByTestId("app-owned-key-management")).toBeVisible();
  });

  it("shows plan, status, pending sync and an accessible account error", () => {
    render(
      <MonetizationAccountPanel
        configured
        plan="pro"
        status="past_due"
        statusLabel="Payment needs attention"
        creditsLabel="72"
        pendingSync={2}
        error="Account service is unavailable."
        {...actions}
      />,
    );
    expect(screen.getByTestId("account-status")).toHaveAccessibleName(
      "Subscription status: Payment needs attention",
    );
    expect(screen.getByTestId("account-pending-sync")).toHaveTextContent(
      "2 usage items waiting to sync",
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Account service is unavailable.");
    expect(screen.getByText("72")).toBeVisible();
  });
});
