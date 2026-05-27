import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";

// vi.hoisted so the value is available inside the hoisted vi.mock factory below.
const savedConfig = vi.hoisted(() => ({
  enabled: true,
  profileIds: ["p1"],
  threshold: 0.65,
  mode: "filter",
  rejectBehavior: "drop",
  fallbackWithoutVerification: false,
  extractionEnabled: true,
}));

vi.mock("../../services/speakerAdmin", () => ({
  getSpeakerStatus: vi.fn().mockResolvedValue({
    config: { ...savedConfig },
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
  updateSpeakerConfig: vi.fn().mockResolvedValue({ ...savedConfig }),
  unbindSpeakerProfile: vi.fn(),
  deleteSpeakerProfile: vi.fn(),
}));

import { SpeakerVerificationPage } from "./SpeakerVerificationPage";
import { updateSpeakerConfig } from "../../services/speakerAdmin";

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
    await waitFor(() =>
      expect(screen.getByTestId(selectors.speakerAdmin.profileName({ id: "p1" }))).toHaveTextContent("Alice"),
    );
    expect(screen.getByTestId(selectors.speakerAdmin.profileModel({ id: "p1" }))).toHaveTextContent("ecapa-tdnn");
    // Profile count chip
    expect(screen.getByText(/1 profile/i)).toBeInTheDocument();
  });

  it("reflects extraction state and sends it when saving", async () => {
    const { container } = renderWithClient();
    // Query by the form-field name (deterministic; independent of i18n label text).
    const extraction = await waitFor(() => {
      const el = container.querySelector<HTMLInputElement>('input[name="extraction"]');
      expect(el).not.toBeNull();
      return el!;
    });
    // Mirrors the persisted config (extractionEnabled: true).
    expect(extraction).toBeChecked();

    const form = container.querySelector("form")!;
    fireEvent.submit(form);

    // Assert on the first argument only — react-query passes a mutation context
    // as a second arg to the mutationFn.
    await waitFor(() =>
      expect(vi.mocked(updateSpeakerConfig).mock.calls[0]?.[0]).toEqual(
        expect.objectContaining({ extractionEnabled: true, mode: "filter" }),
      ),
    );
  });
});
