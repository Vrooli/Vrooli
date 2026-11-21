import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../test-utils";
import { makeAsk } from "../mocks/askFactory";
import { emptyDraft } from "../model/draft";
import { MainOptions } from "./MainOptions";

const options = [
  { key: "approve", label: "Allow it" },
  { key: "decline", label: "Don't allow it" },
];

// A permission request is approved only with the operator's sign-in on the
// source's own page, so its approve option opens that page instead of being
// an answer the Hub would refuse.
describe("MainOptions approval page", () => {
  it("makes approve a link to the approval page and keeps the other answers", () => {
    const page = "https://am.example.test/api/v1/authorizations/a1/approve";
    renderWithProviders(
      <MainOptions ask={makeAsk({ options, recommended: "approve", approvalUrl: page })} draft={emptyDraft()} disabled={false} onChange={() => undefined} />,
    );

    const link = screen.getByRole("link", { name: /Allow it/ });
    expect(link).toHaveAttribute("href", page);
    expect(link).toHaveAttribute("target", "_blank");
    expect(screen.queryByRole("radio", { name: /Allow it/ })).toBeNull();
    expect(screen.getByRole("radio", { name: /Don't allow it/ })).toBeInTheDocument();
  });

  it("leaves approve an ordinary answer when the ask names no approval page", () => {
    renderWithProviders(<MainOptions ask={makeAsk({ options })} draft={emptyDraft()} disabled={false} onChange={() => undefined} />);

    expect(screen.getByRole("radio", { name: /Allow it/ })).toBeInTheDocument();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
