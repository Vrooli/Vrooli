import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { strings } from "../../consts/strings";

const listClipsMock = vi.hoisted(() => vi.fn());
const deleteClipMock = vi.hoisted(() => vi.fn());

vi.mock("../../services/speakerAdmin", () => ({
  listSpeakerProfileClips: listClipsMock,
  deleteSpeakerProfileClip: deleteClipMock,
}));

import { SpeakerProfileClips } from "./SpeakerProfileClips";

function renderClips() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<SpeakerProfileClips profileId="profile-1" />, { queryClient });
}

describe("SpeakerProfileClips", () => {
  it("renders the explicit empty state", async () => {
    listClipsMock.mockResolvedValueOnce([]);
    renderClips();

    await waitFor(() => expect(screen.getByText(strings.speakerAdmin.clipsEmpty)).toBeInTheDocument());
  });

  it("renders clip metadata and deletes the selected clip", async () => {
    const clipLabel = "desk mic";
    // voicedSeconds 2.34 renders at one decimal place with a unit suffix.
    const formattedVoicedSeconds = "2.3s";
    listClipsMock
      .mockResolvedValueOnce([{ clipId: "clip-1", label: clipLabel, voicedSeconds: 2.34 }])
      .mockResolvedValueOnce([]);
    deleteClipMock.mockResolvedValueOnce(undefined);
    renderClips();

    await waitFor(() => expect(screen.getByText(clipLabel)).toBeInTheDocument());
    expect(screen.getByText(formattedVoicedSeconds)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "speakerAdmin.deleteClipButton" }));
    await waitFor(() => expect(deleteClipMock).toHaveBeenCalledWith("profile-1", "clip-1"));
  });

  it("falls back to the clip id when no label is supplied", async () => {
    const clipId = "clip-unlabeled";
    listClipsMock.mockResolvedValueOnce([{ clipId, label: "", voicedSeconds: 1 }]);
    renderClips();

    await waitFor(() => expect(screen.getByText(clipId)).toBeInTheDocument());
  });
});
