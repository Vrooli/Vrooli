import { describe, it, expect, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { Capability } from "@vrooli/proto-types/audio-tools/v1/diagnostics/diagnostics_pb";
import { State, type CapabilityHealth } from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

import { CapabilityRow } from "./CapabilityRow";
import { strings } from "../../consts/strings";

function makeCap(overrides: Partial<CapabilityHealth> = {}): CapabilityHealth {
  return {
    $typeName: "audio_tools.v1.health_status.CapabilityHealth",
    capability: Capability.STT,
    effectiveState: State.AVAILABLE,
    providers: [],
    ...overrides,
  } as unknown as CapabilityHealth;
}

afterEach(() => {
  cleanup();
});

describe("CapabilityRow", () => {
  it("renders the capability label and effective state badge", () => {
    render(
      <CapabilityRow
        capability={makeCap({
          capability: Capability.SUMMARIZE,
          effectiveState: State.AVAILABLE,
          providers: [],
        })}
      />,
    );
    expect(screen.getByText(strings.status.capabilityLabelSummarize)).toBeInTheDocument();
    expect(screen.getAllByText(strings.status.stateAvailable).length).toBeGreaterThan(0);
  });

  it("renders empty-providers copy when the capability has none registered", () => {
    render(<CapabilityRow capability={makeCap()} />);
    expect(screen.getByText(strings.status.noProviders)).toBeInTheDocument();
  });

  it("renders each provider's id, state, tier badge, and error message", () => {
    render(
      <CapabilityRow
        capability={makeCap({
          effectiveState: State.UNAVAILABLE,
          providers: [
            {
              $typeName: "audio_tools.v1.health_status.ProviderHealth",
              capability: Capability.STT,
              tier: ProviderTier.LOCAL,
              providerId: "whisper-stt",
              state: State.UNAVAILABLE,
              lastCheckedAt: "2026-05-17T00:00:00Z",
              errorMessage: "whisper down",
              // proto-message field defaults
            } as never,
          ],
        })}
      />,
    );
    expect(screen.getByText(/^whisper-stt$/)).toBeInTheDocument();
    expect(screen.getAllByText(strings.status.stateUnavailable).length).toBeGreaterThan(0);
    expect(screen.getByText(/^Local$/)).toBeInTheDocument();
    expect(screen.getByText(/^whisper down$/)).toBeInTheDocument();
  });

  it("invokes renderProviderActions for each provider", () => {
    render(
      <CapabilityRow
        capability={makeCap({
          providers: [
            { $typeName: "audio_tools.v1.health_status.ProviderHealth", capability: Capability.STT, tier: ProviderTier.LOCAL, providerId: "p1", state: State.AVAILABLE, lastCheckedAt: "", errorMessage: "" } as never,
            { $typeName: "audio_tools.v1.health_status.ProviderHealth", capability: Capability.STT, tier: ProviderTier.LOCAL, providerId: "p2", state: State.AVAILABLE, lastCheckedAt: "", errorMessage: "" } as never,
          ],
        })}
        renderProviderActions={(id) => <button type="button">act-{id}</button>}
      />,
    );
    expect(screen.getByRole("button", { name: "act-p1" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "act-p2" })).toBeInTheDocument();
  });
});
