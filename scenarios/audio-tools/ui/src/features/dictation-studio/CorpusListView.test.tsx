import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const listClips = vi.fn();
const deleteClip = vi.fn();
const getClipAudio = vi.fn();
const reviewReference = vi.fn();
const reviewReferenceGroup = vi.fn();

vi.mock("../../services/corpus", () => ({
  ClipSource: { UNSPECIFIED: 0, FREE_FORM: 1, SCRIPTED: 2 },
  listClips: () => listClips(),
  deleteClip: (id: string) => deleteClip(id),
  getClipAudio: (id: string) => getClipAudio(id),
  reviewReference: (args: unknown) => reviewReference(args),
  reviewReferenceGroup: (args: unknown) => reviewReferenceGroup(args),
}));

const pushToast = vi.fn();
vi.mock("../../components/ui/toast", () => ({
  pushToast: (...args: unknown[]) => pushToast(...args),
}));

import { CorpusListView } from "./CorpusListView";

const CLIPS = [
  { id: "c1", referenceText: "first clip", tags: ["a"], durationMs: 1_200, sampleRateHz: 16_000, format: "pcm_s16le", source: 1, createdAt: "" },
  { id: "c2", referenceText: "second clip", tags: [], durationMs: 800, sampleRateHz: 16_000, format: "pcm_s16le", source: 2, createdAt: "" },
];

beforeEach(() => {
  vi.clearAllMocks();
  deleteClip.mockResolvedValue(undefined);
  getClipAudio.mockResolvedValue({ audio: new Uint8Array([0, 0, 255, 127]), clip: null });
  reviewReference.mockResolvedValue(undefined);
  reviewReferenceGroup.mockResolvedValue({ clips: [], affectedCount: 3 });
});
afterEach(cleanup);

describe("CorpusListView", () => {
  it("renders a row per clip", async () => {
    listClips.mockResolvedValue(CLIPS);
    renderWithProviders(<CorpusListView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.clipRow({ id: "c1" }))).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.dictationStudio.clipRow({ id: "c2" }))).toBeInTheDocument();
  });

  it("shows the empty state when there are no clips", async () => {
    listClips.mockResolvedValue([]);
    renderWithProviders(<CorpusListView />);
    expect(await screen.findByText(strings.dictationStudio.corpusEmpty)).toBeInTheDocument();
  });

  it("deletes a clip and toasts", async () => {
    listClips.mockResolvedValue(CLIPS);
    const user = userEvent.setup();
    renderWithProviders(<CorpusListView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.clipDelete({ id: "c1" }))).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId(selectors.dictationStudio.clipDelete({ id: "c1" })));
    await waitFor(() => expect(deleteClip).toHaveBeenCalledWith("c1"));
    await waitFor(() => expect(pushToast).toHaveBeenCalled());
  });

  it("plays stored PCM audio and submits the edited text against the original digest", async () => {
    listClips.mockResolvedValue(CLIPS);
    const user = userEvent.setup();
    renderWithProviders(<CorpusListView />);
    const reviewButtons = await screen.findAllByRole("button", { name: strings.dictationStudio.reviewButton });
    await user.click(reviewButtons[0]!);
    const player = await screen.findByTestId(selectors.dictationStudio.reviewAudio);
    expect(player).toHaveAttribute("src", expect.stringContaining("blob:"));
    expect(screen.getByRole("button", { name: strings.dictationStudio.submitReview })).toBeDisabled();
    fireEvent.ended(player);
    const transcript = screen.getByLabelText(strings.dictationStudio.referenceTranscriptLabel);
    await user.clear(transcript);
    await user.type(transcript, "corrected phrase");
    await user.type(screen.getByLabelText(strings.dictationStudio.reviewerLabel), "human-1");
    await user.click(screen.getByRole("button", { name: strings.dictationStudio.submitReview }));
    await waitFor(() => expect(reviewReference).toHaveBeenCalledWith({
      id: "c1",
      expectedReferenceSha256: "1b14fa5ba703317eb4fe0227ce9e4773d789058e8ecce8921e1ed4d0ea480a57",
      referenceText: "corrected phrase",
      reviewer: "human-1",
      protocol: "listen-and-correct-v1",
    }));
  });

  it("keeps reviewer submission disabled until all explicit review steps are complete and surfaces a stale digest", async () => {
    listClips.mockResolvedValue(CLIPS);
    const user = userEvent.setup();
    renderWithProviders(<CorpusListView />);
    const reviewButtons = await screen.findAllByRole("button", { name: strings.dictationStudio.reviewButton });
    await user.click(reviewButtons[0]!);
    const player = await screen.findByTestId(selectors.dictationStudio.reviewAudio);
    const submit = screen.getByRole("button", { name: strings.dictationStudio.submitReview });
    expect(submit).toBeDisabled();
    await user.type(screen.getByLabelText(strings.dictationStudio.reviewerLabel), "human-1");
    expect(submit).toBeDisabled();
    fireEvent.ended(player);
    expect(submit).toBeEnabled();
    reviewReference.mockRejectedValueOnce(new Error("aborted: stale reference digest"));
    await user.click(submit);
    expect(await screen.findByText(strings.dictationStudio.staleReviewError)).toBeInTheDocument();
  });

  it("previews the exact source group and requires full playback of every variant", async () => {
    listClips.mockResolvedValue(["clean", "20db", "10db"].map((variant, index) => ({
      id: `v${index}`, referenceText: "same source", tags: [], durationMs: 1000, sampleRateHz: 16000, format: "pcm_s16le", source: 2, createdAt: "",
      corpusRevision: "vctk-v1", sourceId: "p230_001", accentGroup: "uk", speakerPseudonym: "vctk-p230", noiseCondition: variant, snrDb: index === 1 ? 20 : index === 2 ? 10 : 0,
      manifestTranscriptSha256: "4343d14af5d0df58895df410237461f6f143112ba856709e1a497e6500e22a2c", lineageManifestSha256: "f".repeat(64),
    })));
    const user = userEvent.setup();
    renderWithProviders(<CorpusListView />);
    await user.click((await screen.findAllByRole("button", { name: strings.dictationStudio.reviewButton }))[0]!);
    await user.click(await screen.findByLabelText(strings.dictationStudio.reviewAllVariants.replace("{{count}}", "3")));
    await user.type(screen.getByLabelText(strings.dictationStudio.reviewerLabel), "human-1");
    const submit = screen.getByRole("button", { name: strings.dictationStudio.submitGroupReview.replace("{{count}}", "3") });
    expect(submit).toBeDisabled();
    for (const id of ["v0", "v1", "v2"]) expect(await screen.findByTestId(selectors.dictationStudio.variantReviewAudio({ id }))).toBeInTheDocument();
    await waitFor(() => expect(getClipAudio).toHaveBeenCalledTimes(3));
    fireEvent.ended(screen.getByTestId(selectors.dictationStudio.variantReviewAudio({ id: "v0" })));
    fireEvent.ended(screen.getByTestId(selectors.dictationStudio.variantReviewAudio({ id: "v1" })));
    expect(submit).toBeDisabled();
    fireEvent.ended(screen.getByTestId(selectors.dictationStudio.variantReviewAudio({ id: "v2" })));
    expect(submit).toBeEnabled();
    await user.click(submit);
    await waitFor(() => expect(reviewReferenceGroup).toHaveBeenCalledWith({
      members: ["v0", "v1", "v2"].map((id) => ({ id, expectedReferenceSha256: "4343d14af5d0df58895df410237461f6f143112ba856709e1a497e6500e22a2c" })),
      referenceText: "same source", reviewer: "human-1", protocol: "listen-and-correct-each-source-variant-v1",
    }));
  });
});
