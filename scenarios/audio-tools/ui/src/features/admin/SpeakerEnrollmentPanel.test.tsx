import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { renderWithProviders as render } from "@vrooli/api-base/testing";

const enrollMock = vi.hoisted(() => vi.fn());

vi.mock("../../services/speakerAdmin", () => ({
  enrollSpeakerProfile: enrollMock,
}));

import { SpeakerEnrollmentPanel } from "./SpeakerEnrollmentPanel";
import type { EnrollRecordHandle } from "./speakerEnrollmentRecorder";

function renderPanel(recordClip: () => Promise<EnrollRecordHandle>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<SpeakerEnrollmentPanel recordClip={recordClip} />, { queryClient: qc });
}

describe("SpeakerEnrollmentPanel", () => {
  it("records a clip, enrolls it, and surfaces the result", async () => {
    enrollMock.mockResolvedValue({
      profileId: "p-gen",
      clipId: "c1",
      label: "laptop-normal",
      voicedSeconds: 3.4,
      clipCount: 1,
      totalVoicedSeconds: 3.4,
    });
    // A recorder whose clip is already captured (done resolves immediately).
    const handle: EnrollRecordHandle = {
      done: Promise.resolve({ audio: new Uint8Array([1, 2, 3]), format: AudioFormat.WEBM }),
      stop: vi.fn(),
    };
    const recordClip = vi.fn().mockResolvedValue(handle);

    renderPanel(recordClip);

    fireEvent.change(screen.getByTestId("speaker-enroll-clip-label"), {
      target: { value: "laptop-normal" },
    });
    fireEvent.click(screen.getByTestId("speaker-enroll-record"));

    await waitFor(() => expect(enrollMock).toHaveBeenCalledTimes(1));
    expect(enrollMock.mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({ label: "laptop-normal", format: AudioFormat.WEBM }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("speaker-enroll-last-clip")).toHaveTextContent("laptop-normal"),
    );
  });
});
