// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { acceptFinalTranscript, VoiceComposerButton } from "./VoiceComposerButton";

describe("VoiceComposerButton", () => {
  it("keeps the text composer usable when Audio Tools is not configured", () => { // OPT-02
    render(<VoiceComposerButton onTranscript={vi.fn()} />);
    const button = screen.getByRole("button", { name: "chat.composer.voiceUnavailable" });
    expect(button).toBeDisabled();
  });

  it("deduplicates repeated final transcript events without hiding later speech", () => { // UI-11
    const first = acceptFinalTranscript("  Export   the report ", undefined, 1_000);
    expect(first?.value).toBe("Export   the report");
    expect(acceptFinalTranscript("export the report", first?.state, 1_500)).toBeUndefined();
    const later = acceptFinalTranscript("export the report", first?.state, 3_100);
    expect(later?.value).toBe("export the report");
  });
});
