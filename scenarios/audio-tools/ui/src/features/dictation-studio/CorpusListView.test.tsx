import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const listClips = vi.fn();
const deleteClip = vi.fn();

vi.mock("../../services/corpus", () => ({
  ClipSource: { UNSPECIFIED: 0, FREE_FORM: 1, SCRIPTED: 2 },
  listClips: () => listClips(),
  deleteClip: (id: string) => deleteClip(id),
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
});
