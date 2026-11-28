import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { AdoptionFlow, type AdoptionFlowCopy } from "./AdoptionFlow";
import type { AdoptionRecordSnapshot } from "./model";

const copy: AdoptionFlowCopy = {
  draftForm: {
    intro: "Add connection facts. Enter the credential only in the secure credential step.",
    displayName: "Display name",
    host: "Server address",
    user: "SSH user",
    monthlyCostMinor: "Monthly cost",
    currency: "Currency",
    taxBasis: "Tax basis",
    continueLabel: "Save adoption details",
    errors: {
      address_required: "Address required",
      ssh_user_required: "SSH user required",
      monthly_cost_invalid: "Monthly cost required",
      currency_invalid: "Currency required",
      tax_basis_required: "Tax basis required",
    },
  },
  statusCard: {
    stateLabel: "Adoption state",
    states: {
      draft: "Draft",
      awaiting_secret: "Secure credential needed",
      credential_stored: "Credential stored",
      bridge_enrolling: "Verifying Bridge connection",
      ready: "Ready",
      failed: "Needs attention",
      released: "Released",
    },
    host: "Address",
    user: "SSH user",
    monthlyCost: "Declared monthly cost",
    taxBasis: "Tax basis",
    verificationPending: "Verification pending",
    minorUnitsLabel: "minor units",
    locale: "en-US",
    releaseSemantics: "Release removes the record and Bridge attachment only.",
  },
  createFailed: "Adoption details could not be saved.",
  credentialStepUnavailable: "The secure credential step is not available yet.",
  credentialStepAction: "Continue in secure credential step",
};

const validFields = () => {
  fireEvent.change(screen.getByLabelText(copy.draftForm.host), { target: { value: "vps.example.test" } });
  fireEvent.change(screen.getByLabelText(copy.draftForm.user), { target: { value: "deploy" } });
  fireEvent.change(screen.getByLabelText(copy.draftForm.monthlyCostMinor), { target: { value: "2500" } });
  fireEvent.change(screen.getByLabelText(copy.draftForm.currency), { target: { value: "USD" } });
  fireEvent.change(screen.getByLabelText(copy.draftForm.taxBasis), { target: { value: "pre-tax" } });
};

const awaitingCredential: AdoptionRecordSnapshot = {
  id: "adoption-1",
  host: "vps.example.test",
  user: "deploy",
  monthlyCostMinor: 2500,
  currency: "USD",
  taxBasis: "pre-tax",
  state: "awaiting_secret",
  adoptionEvidenceVerified: false,
  credentialStepRef: "opaque-non-secret-onboarding-ref",
};

describe("AdoptionFlow", () => {
  it("persists only non-secret facts then hands off by non-secret Onboarding reference", async () => {
    const createAdoption = vi.fn(() => Promise.resolve(awaitingCredential));
    const onOpenCredentialStep = vi.fn();
    render(<AdoptionFlow createAdoption={createAdoption} onOpenCredentialStep={onOpenCredentialStep} copy={copy} />);
    validFields();
    fireEvent.click(screen.getByTestId(selectors.adoption.submit));

    await waitFor(() => expect(createAdoption).toHaveBeenCalledTimes(1));
    expect(createAdoption).toHaveBeenCalledWith({
      host: "vps.example.test",
      user: "deploy",
      monthlyCostMinor: 2500,
      currency: "USD",
      taxBasis: "pre-tax",
    });
    expect(screen.getByTestId(selectors.adoption.status)).toHaveTextContent(copy.statusCard.states.awaiting_secret);
    fireEvent.click(screen.getByTestId(selectors.adoption.credentialStep));
    expect(onOpenCredentialStep).toHaveBeenCalledWith("opaque-non-secret-onboarding-ref");
    expect(JSON.stringify(createAdoption.mock.calls)).not.toMatch(/password|private.?key|secret/i);
  });

  it("does not expose provider error text in visible UI", async () => {
    const createAdoption = vi.fn(() => Promise.reject(new Error("secret=never-display")));
    render(<AdoptionFlow createAdoption={createAdoption} onOpenCredentialStep={vi.fn()} copy={copy} />);
    validFields();
    fireEvent.click(screen.getByTestId(selectors.adoption.submit));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(copy.createFailed));
    expect(screen.queryByText(/secret=never-display/)).not.toBeInTheDocument();
  });
});
