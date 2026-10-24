/**
 * EvidenceSummary tests — every evidence state (none / live tones / stale /
 * manual / expired) plus the verbose timestamp block. Rendered directly with
 * providers (cimode) so assertions bind to string keys, not copy.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { EvidenceSummary } from "./EvidenceSummary";
import { makeEvidenceCell, makeManualAttestation } from "./mocks/factories";

describe("EvidenceSummary", () => {
  afterEach(() => cleanup());

  it("renders the no-evidence state when there is neither live status nor manual", () => {
    renderWithProviders(<EvidenceSummary evidence={undefined} />);
    expect(screen.getByText(strings.matrix.evidence.none)).toBeInTheDocument();
  });

  it("renders a live status chip for a passing suite", () => {
    renderWithProviders(<EvidenceSummary evidence={makeEvidenceCell({ liveStatus: "passed" })} />);
    expect(screen.getByText(strings.matrix.evidence.live)).toBeInTheDocument();
  });

  it("renders a failing live status", () => {
    renderWithProviders(<EvidenceSummary evidence={makeEvidenceCell({ liveStatus: "failed" })} />);
    expect(screen.getByText(strings.matrix.evidence.live)).toBeInTheDocument();
  });

  it("flags stale evidence", () => {
    renderWithProviders(
      <EvidenceSummary evidence={makeEvidenceCell({ liveStatus: "running", stale: true })} />,
    );
    expect(screen.getByText(strings.matrix.evidence.stale)).toBeInTheDocument();
  });

  it("renders manual + verbose timestamps for an unexpired attestation", () => {
    renderWithProviders(
      <EvidenceSummary
        verbose
        evidence={makeEvidenceCell({
          liveStatus: "missing",
          lastSyncedAt: timestampFromDate(new Date("2026-06-01T00:00:00Z")),
          manual: makeManualAttestation({
            attestedBy: "agent:qa",
            expired: false,
            expiresAt: timestampFromDate(new Date("2026-12-01T00:00:00Z")),
          }),
        })}
      />,
    );
    expect(screen.getByText(strings.matrix.evidence.manual)).toBeInTheDocument();
    expect(screen.getByText(strings.matrix.evidence.syncedAt)).toBeInTheDocument();
    expect(screen.getByText(strings.matrix.evidence.attestedBy)).toBeInTheDocument();
    expect(screen.getByText(strings.matrix.evidence.expires)).toBeInTheDocument();
  });

  it("renders an expired-attestation chip", () => {
    renderWithProviders(
      <EvidenceSummary
        verbose
        evidence={makeEvidenceCell({
          liveStatus: "",
          manual: makeManualAttestation({ expired: true }),
        })}
      />,
    );
    expect(screen.getByText(strings.matrix.evidence.expired)).toBeInTheDocument();
  });
});
