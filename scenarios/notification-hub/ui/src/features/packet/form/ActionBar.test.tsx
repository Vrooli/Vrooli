import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../test-utils";
import { ActionBar } from "./ActionBar";

const noop = () => undefined;

// The bar is pinned over the page on a phone, so every pixel it takes is a pixel
// the decision loses. One row, no second safe-area inset (the tab bar owns it),
// and the reason sitting beside the button rather than above it.
describe("ActionBar", () => {
  it("is one row: the reason beside a compact button while the action is unavailable", () => {
    renderWithProviders(<ActionBar primary={{ label: "Send answer", onClick: noop, disabled: true }} hint="Choose an answer to send." disabled={false} />);

    const hint = screen.getByRole("status");
    const send = screen.getByRole("button", { name: "Send answer" });
    expect(hint).toHaveTextContent("Choose an answer to send.");
    expect(hint.parentElement).toBe(send.parentElement);
    expect(send).toHaveAttribute("aria-describedby", hint.id);
    expect(send).not.toHaveClass("flex-1");
    expect(send).toBeDisabled();
  });

  it("gives the button the whole row once it can be pressed", () => {
    renderWithProviders(<ActionBar primary={{ label: "Send answer", onClick: noop }} disabled={false} />);

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send answer" })).toHaveClass("flex-1");
  });

  it("keeps a step back on the same row, before the reason and the action", () => {
    renderWithProviders(<ActionBar primary={{ label: "Next", onClick: noop }} back={{ label: "Previous", onClick: noop }} disabled={false} />);

    const [back, next] = screen.getAllByRole("button");
    expect(back).toHaveTextContent("Previous");
    expect(next).toHaveTextContent("Next");
    expect(back?.parentElement).toBe(next?.parentElement);
  });

  it("is a pinned dock that adds no safe-area inset of its own", () => {
    renderWithProviders(<ActionBar primary={{ label: "Send answer", onClick: noop }} disabled={false} />);

    const dock = screen.getByRole("contentinfo");
    expect(dock).toHaveClass("packet-dock");
    expect(dock.className).not.toContain("safe-area");
  });
});
