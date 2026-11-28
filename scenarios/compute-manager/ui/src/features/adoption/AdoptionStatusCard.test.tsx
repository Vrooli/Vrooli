import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { selectors } from "../../consts/selectors";
import { AdoptionStatusCard } from "./AdoptionStatusCard";
import type { AdoptionStatusCardCopy } from "./AdoptionStatusCard";
import { toAdoptionViewModel, type AdoptionRecordSnapshot } from "./model";

const copy: AdoptionStatusCardCopy = {
  stateLabel: "Adoption state",
  states: {
    draft: "Draft",
    awaiting_secret: "Secure credential needed",
    credential_stored: "Credential stored securely",
    bridge_enrolling: "Verifying Bridge connection",
    ready: "Ready",
    failed: "Needs attention",
    released: "Released",
  },
  host: "Address",
  user: "SSH user",
  monthlyCost: "Declared monthly cost",
  taxBasis: "Tax basis",
  verificationPending: "Verification remains pending.",
  minorUnitsLabel: "minor units",
  locale: "en-US",
  releaseSemantics: "Release removes only the record and Bridge attachment; it does not stop or delete your VPS.",
};

const readyRecord: AdoptionRecordSnapshot = {
  id: "adoption-1",
  host: "host.example.test",
  user: "runner",
  monthlyCostMinor: 1250,
  currency: "USD",
  taxBasis: "operator-declared, pre-tax",
  state: "ready",
  adoptionEvidenceVerified: true,
};

describe("AdoptionStatusCard", () => {
  it("shows persisted ready status and safe release semantics", () => {
    render(<AdoptionStatusCard adoption={toAdoptionViewModel(readyRecord)} copy={copy} />);
    expect(screen.getByTestId(selectors.adoption.status)).toHaveTextContent(copy.states.ready);
    expect(screen.getByTestId(selectors.adoption.statusCard)).toHaveTextContent(copy.releaseSemantics);
  });

  it("does not show ready when verification evidence is absent", () => {
    render(<AdoptionStatusCard adoption={toAdoptionViewModel({ ...readyRecord, adoptionEvidenceVerified: false })} copy={copy} />);
    expect(screen.getByTestId(selectors.adoption.status)).toHaveTextContent(copy.states.bridge_enrolling);
    expect(screen.getByTestId(selectors.adoption.statusCard)).not.toHaveTextContent(copy.states.ready);
  });
});
