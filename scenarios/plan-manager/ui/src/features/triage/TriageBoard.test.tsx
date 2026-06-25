/**
 * TriageBoard tests — candidate list, empty state, promote/dismiss, and
 * axe-clean structure. api/execution is mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { FindingSchema, FindingTriage } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const listCandidateFindings = vi.fn();
const triageFinding = vi.fn();

vi.mock("../../api/execution", () => ({
  listCandidateFindings: (...a: unknown[]) => listCandidateFindings(...a),
  triageFinding: (...a: unknown[]) => triageFinding(...a),
}));

import { TriageBoard } from "./TriageBoard";

const finding = create(FindingSchema, {
  id: "f1",
  title: "Possible nil deref",
  detail: "in handler",
  phaseId: "p1",
  recordedAt: "2026-06-25T10:00:00Z",
});

describe("TriageBoard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state with no candidates", async () => {
    listCandidateFindings.mockResolvedValue([]);
    renderWithProviders(<TriageBoard />);
    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.triage.list}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("lists candidate findings", async () => {
    listCandidateFindings.mockResolvedValue([finding]);
    renderWithProviders(<TriageBoard />);
    expect(await screen.findByTestId(selectors.triage.row({ id: "f1" }))).toBeInTheDocument();
  });

  it("promotes a finding to a bug", async () => {
    const user = userEvent.setup();
    listCandidateFindings.mockResolvedValue([finding]);
    triageFinding.mockResolvedValue(create(FindingSchema, { id: "f1", triage: FindingTriage.PROMOTED }));

    renderWithProviders(<TriageBoard />);
    await user.click(await screen.findByTestId(selectors.triage.promote({ id: "f1" })));
    await waitFor(() => {
      expect(triageFinding).toHaveBeenCalledWith("f1", FindingTriage.PROMOTED);
    });
  });

  it("dismisses a finding", async () => {
    const user = userEvent.setup();
    listCandidateFindings.mockResolvedValue([finding]);
    triageFinding.mockResolvedValue(create(FindingSchema, { id: "f1", triage: FindingTriage.DISMISSED }));

    renderWithProviders(<TriageBoard />);
    await user.click(await screen.findByTestId(selectors.triage.dismiss({ id: "f1" })));
    await waitFor(() => {
      expect(triageFinding).toHaveBeenCalledWith("f1", FindingTriage.DISMISSED);
    });
  });

  it("renders the list without axe violations", async () => {
    listCandidateFindings.mockResolvedValue([finding]);
    const { container } = renderWithProviders(<TriageBoard />);
    await screen.findByTestId(selectors.triage.row({ id: "f1" }));
    await expectNoA11yViolations(container);
  });
});
