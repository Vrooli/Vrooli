import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  CandidateOrigin,
  CandidateStatus,
} from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";
import { makeCandidate } from "./mocks/factories";
import { makeCandidatesMocks } from "./mocks/candidates";

vi.mock("../../api/candidates", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/candidates")>();
  return { ...actual, ...makeCandidatesMocks() };
});

import { LogoCandidateGallery } from "./LogoCandidateGallery";

const candidates = [
  makeCandidate({
    id: "candidate-1",
    concept: "constellation eagle line art",
    status: CandidateStatus.PROPOSED,
  }),
  makeCandidate({
    id: "candidate-2",
    concept: "monogram",
    status: CandidateStatus.REJECTED,
    origin: CandidateOrigin.EDITED,
    parentId: "candidate-1",
  }),
];

function renderGallery() {
  return renderWithProviders(
    <LogoCandidateGallery
      brandId="brand-1"
      candidates={candidates}
      isLoading={false}
      queryError={null}
      selectedForCompare={[]}
      onToggleCompare={() => {}}
      onSelectForRefine={() => {}}
    />,
  );
}

describe("LogoCandidateGallery", () => {
  beforeEach(() => {
    setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a card per candidate with its status chip and lineage", () => {
    renderGallery();
    expect(screen.getAllByTestId(selectors.logo.card)).toHaveLength(2);
    expect(screen.getByText("Proposed")).toBeInTheDocument();
    expect(screen.getByText("Rejected")).toBeInTheDocument();
    expect(screen.getByText(/Derived from/)).toBeInTheDocument();
  });

  it("picks a candidate and updates its chip immediately", async () => {
    const user = userEvent.setup();
    const api = await import("../../api/candidates");
    renderGallery();

    const firstCard = screen.getAllByTestId(selectors.logo.card)[0];
    await user.click(within(firstCard).getByTestId(selectors.logo.cardPick));

    expect(vi.mocked(api.pickCandidate)).toHaveBeenCalledWith("candidate-1");
    expect(within(firstCard).getByTestId(selectors.logo.cardStatus)).toHaveTextContent("Picked");
  });

  it("rejects a proposed candidate", async () => {
    const user = userEvent.setup();
    const api = await import("../../api/candidates");
    renderGallery();

    const firstCard = screen.getAllByTestId(selectors.logo.card)[0];
    await user.click(within(firstCard).getByTestId(selectors.logo.cardReject));

    expect(vi.mocked(api.rejectCandidate)).toHaveBeenCalledWith("candidate-1", "");
    expect(within(firstCard).getByTestId(selectors.logo.cardStatus)).toHaveTextContent("Rejected");
  });
});
