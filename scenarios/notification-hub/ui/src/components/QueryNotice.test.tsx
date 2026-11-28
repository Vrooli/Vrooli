import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { EmptyNotice, QueryError } from "./QueryNotice";

vi.mock("../api/notifications", () => ({
  identityClient: { login: vi.fn(), getSession: vi.fn() },
}));

const TITLE = "Couldn’t load things.";

describe("QueryError", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("names the failure, says nothing was lost, and retries without exposing the cause", async () => {
    const retry = vi.fn();
    renderWithProviders(<QueryError error={new Error("secret endpoint")} title="Couldn’t load things." detail="Nothing was dismissed." onRetry={retry} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn’t load things.");
    expect(screen.getByRole("alert")).toHaveTextContent("Nothing was dismissed.");
    expect(screen.queryByText(/secret endpoint/)).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("shows the way in, not a failure, when the Hub refused the read for lack of a session", () => {
    renderWithProviders(<QueryError error={new ConnectError("no session", Code.Unauthenticated)} title="Couldn’t load things." onRetry={vi.fn()} />);
    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByText(TITLE)).not.toBeInTheDocument();
  });
});

describe("EmptyNotice", () => {
  it("states a fact and offers its next step", () => {
    renderWithProviders(<EmptyNotice icon={<span />} title="All caught up" detail="Nothing is waiting."><a href="/history">See past decisions</a></EmptyNotice>);
    expect(screen.getByRole("heading", { name: "All caught up" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "See past decisions" })).toBeInTheDocument();
  });
});
