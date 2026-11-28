import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import type { AdoptionDraftFormCopy } from "./AdoptionDraftForm";
import { AdoptionDraftForm } from "./AdoptionDraftForm";

const copy: AdoptionDraftFormCopy = {
  intro: "Add a server you own. Enter credentials only in the secure credential step.",
  displayName: "Display name (optional)",
  host: "Server address",
  user: "SSH user",
  monthlyCostMinor: "Monthly cost (minor currency units)",
  currency: "Currency (ISO 4217)",
  taxBasis: "Tax basis",
  continueLabel: "Continue to secure credential step",
  errors: {
    address_required: "Enter the server host name or IP address.",
    ssh_user_required: "Enter the SSH user name used by this server.",
    monthly_cost_invalid: "Enter an explicit non-negative monthly cost in minor currency units.",
    currency_invalid: "Enter a three-letter ISO currency code.",
    tax_basis_required: "State whether the declared amount includes tax or is pre-tax.",
  },
};

describe("AdoptionDraftForm", () => {
  it("submits only explicit non-secret adoption facts", () => {
    const onSubmit = vi.fn();
    render(<AdoptionDraftForm onSubmit={onSubmit} copy={copy} />);
    fireEvent.change(screen.getByLabelText(copy.displayName), { target: { value: "Compile host" } });
    fireEvent.change(screen.getByLabelText(copy.host), { target: { value: "compile.example.test" } });
    fireEvent.change(screen.getByLabelText(copy.user), { target: { value: "builder" } });
    fireEvent.change(screen.getByLabelText(copy.monthlyCostMinor), { target: { value: "0" } });
    fireEvent.change(screen.getByLabelText(copy.currency), { target: { value: "usd" } });
    fireEvent.change(screen.getByLabelText(copy.taxBasis), { target: { value: "pre-tax" } });
    fireEvent.click(screen.getByTestId(selectors.adoption.submit));

    expect(onSubmit).toHaveBeenCalledWith({
      displayName: "Compile host",
      host: "compile.example.test",
      user: "builder",
      monthlyCostMinor: 0,
      currency: "USD",
      taxBasis: "pre-tax",
    });
    expect(JSON.stringify(onSubmit.mock.calls)).not.toMatch(/password|private.?key|secret/i);
  });

  it("requires explicit cost facts and never reports successful submission when incomplete", () => {
    const onSubmit = vi.fn();
    render(<AdoptionDraftForm onSubmit={onSubmit} copy={copy} />);
    fireEvent.click(screen.getByTestId(selectors.adoption.submit));
    expect(screen.getByRole("alert")).toHaveTextContent("server host name or IP address");
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
