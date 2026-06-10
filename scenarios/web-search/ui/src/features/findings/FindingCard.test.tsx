import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  FindingSchema,
  FindingSource,
  FindingStatus,
} from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: {
    editFinding: vi.fn(),
    supersedeFinding: vi.fn(),
    flagFinding: vi.fn(),
  },
  liveSearchClient: { search: vi.fn() },
}));

import { findingsClient } from "../../api/clients";
import { setLocale } from "../../i18n";
import { interp } from "../../test-utils";
import en from "../../i18n/locales/en.json";
import { FindingCard } from "./FindingCard";

const baseFinding = create(FindingSchema, {
  id: "f1",
  claim: "A claim",
  confidence: 0.8,
  status: FindingStatus.ACTIVE,
  query: "q",
  source: FindingSource.MANUAL,
});

const findingsKey = ["findings", "all", false] as const;

describe("FindingCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows a dispute note for disputed findings", () => {
    // cimode returns the key (no `{{note}}` interpolation), so assert the
    // disputed branch rendered its dispute-note line via the strings key.
    renderWithProviders(
      <FindingCard
        finding={{ ...baseFinding, status: FindingStatus.DISPUTED, disputeNote: "conflicting sources" }}
        findingsKey={findingsKey}
      />,
    );
    expect(screen.getByText(strings.findings.disputeNoteLabel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.findings.statusBadge)).toHaveTextContent(
      strings.findings.statusDisputed,
    );
  });

  it("submits an edit with the new claim and confidence", async () => {
    vi.mocked(findingsClient.editFinding).mockResolvedValue({ finding: baseFinding } as never);

    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    fireEvent.click(screen.getByTestId(selectors.findings.editButton));
    fireEvent.change(screen.getByTestId(selectors.findings.editClaim), {
      target: { value: "A better claim" },
    });
    fireEvent.change(screen.getByTestId(selectors.findings.editConfidence), {
      target: { value: "0.95" },
    });
    fireEvent.click(screen.getByTestId(selectors.findings.editSave));

    await waitFor(() => {
      expect(findingsClient.editFinding).toHaveBeenCalledWith({
        id: "f1",
        claim: "A better claim",
        confidence: 0.95,
      });
    });
  });

  it("submits a supersede with replacement id and reason", async () => {
    vi.mocked(findingsClient.supersedeFinding).mockResolvedValue({ finding: baseFinding } as never);

    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    fireEvent.click(screen.getByTestId(selectors.findings.supersedeButton));
    const form = screen.getByTestId(selectors.findings.supersedeForm);
    fireEvent.change(form.querySelector("input") as HTMLInputElement, {
      target: { value: "f2" },
    });
    fireEvent.change(form.querySelector("textarea") as HTMLTextAreaElement, {
      target: { value: "newer data" },
    });
    fireEvent.submit(form);

    await waitFor(() => {
      expect(findingsClient.supersedeFinding).toHaveBeenCalledWith({
        id: "f1",
        replacement: "f2",
        reason: "newer data",
      });
    });
  });

  it("submits a flag with a reason", async () => {
    vi.mocked(findingsClient.flagFinding).mockResolvedValue({ finding: baseFinding } as never);

    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    fireEvent.click(screen.getByTestId(selectors.findings.flagButton));
    const form = screen.getByTestId(selectors.findings.flagForm);
    fireEvent.change(form.querySelector("textarea") as HTMLTextAreaElement, {
      target: { value: "looks wrong" },
    });
    fireEvent.submit(form);

    await waitFor(() => {
      expect(findingsClient.flagFinding).toHaveBeenCalledWith({ id: "f1", reason: "looks wrong" });
    });
  });

  it("cancels the edit form without calling the client", () => {
    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    fireEvent.click(screen.getByTestId(selectors.findings.editButton));
    expect(screen.getByTestId(selectors.findings.editForm)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId(selectors.findings.editCancel));
    expect(screen.queryByTestId(selectors.findings.editForm)).not.toBeInTheDocument();
    expect(findingsClient.editFinding).not.toHaveBeenCalled();
  });

  it("uses strings registry for the status label (cimode)", () => {
    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    expect(screen.getByTestId(selectors.findings.statusBadge)).toHaveTextContent(
      strings.findings.statusActive,
    );
  });

  it("displays a human-readable age measured from the retrieval date (OT-P1-006)", async () => {
    // Real English so the interpolated relative-time string is visible
    // (cimode renders the bare key path without interpolation).
    await setLocale("en");
    const retrieved = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000);
    const aged = create(FindingSchema, {
      ...baseFinding,
      retrievalDate: timestampFromDate(retrieved),
    });

    renderWithProviders(<FindingCard finding={aged} findingsKey={findingsKey} />);

    const age = screen.getByTestId(selectors.findings.age);
    expect(age).toHaveTextContent(interp(en.findings.ageLabel, { age: "3 months ago" }));
    // The precise stamp stays inspectable via the tooltip.
    expect(age).toHaveAttribute("title", retrieved.toISOString());
  });

  it("omits the age line when the finding carries no usable timestamp", () => {
    renderWithProviders(<FindingCard finding={baseFinding} findingsKey={findingsKey} />);
    expect(screen.queryByTestId(selectors.findings.age)).not.toBeInTheDocument();
  });
});
