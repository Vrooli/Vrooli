import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

vi.mock("../../services/speakerAdmin", () => ({
  getSpeakerStatus: vi.fn().mockResolvedValue({
    config: {
      enabled: true,
      profileIds: ["p1"],
      threshold: 0.65,
      mode: "filter",
      rejectBehavior: "drop",
      fallbackWithoutVerification: false,
    },
    capability: "available",
    capabilityLabel: "Available",
    resourceReady: true,
    profileConfigured: true,
    profileExists: true,
    profileCount: 1,
    profiles: [
      {
        id: "p1",
        displayName: "Alice",
        createdAt: "2026-05-17T00:00:00.000Z",
        modelName: "ecapa-tdnn",
        sampleRate: 16000,
        enrollmentAudioSeconds: 7.5,
      },
    ],
  }),
  updateSpeakerConfig: vi.fn(),
  unbindSpeakerProfile: vi.fn(),
  deleteSpeakerProfile: vi.fn(),
}));

import { SpeakerVerificationPage } from "./SpeakerVerificationPage";

function renderWithClient() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <SpeakerVerificationPage />
    </QueryClientProvider>,
  );
}

describe("SpeakerVerificationPage", () => {
  it("renders status + enrolled profile from the speaker service", async () => {
    renderWithClient();
    await waitFor(() => expect(screen.getByText("Alice")).toBeInTheDocument());
    expect(screen.getByText("ecapa-tdnn")).toBeInTheDocument();
    // Profile count chip
    expect(screen.getByText(/1 profile/i)).toBeInTheDocument();
  });
});
