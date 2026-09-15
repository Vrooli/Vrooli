import { describe, it, expect, afterEach, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { ProviderTraceCard, type TraceEntry } from "./ProviderTraceCard";

afterEach(() => {
  cleanup();
});

const sttEntry: TraceEntry = {
  capability: "stt",
  emittedAt: 1,
  providerTier: "local",
  providerId: "whisper",
  modelId: "v3",
  latencyMs: 42,
};
const ttsEntry: TraceEntry = {
  capability: "tts",
  emittedAt: 2,
  providerTier: "local",
  providerId: "kokoro",
  modelId: "v1",
  latencyMs: 8,
};

describe("ProviderTraceCard", () => {
  it("renders the empty-state copy key when there are no entries", () => {
    renderWithProviders(<ProviderTraceCard entries={[]} />);
    expect(screen.getByText(strings.diagnostics.traceEmpty)).toBeInTheDocument();
  });

  it("renders one row per entry under the default 'all' filter", () => {
    renderWithProviders(<ProviderTraceCard entries={[sttEntry, ttsEntry]} />);
    expect(screen.getByText(/^whisper$/)).toBeInTheDocument();
    expect(screen.getByText(/^kokoro$/)).toBeInTheDocument();
  });

  it("filter buttons are disabled when no onFilterChange is supplied", () => {
    renderWithProviders(<ProviderTraceCard entries={[sttEntry]} />);
    expect(screen.getByTestId("trace-filter-stt")).toBeDisabled();
  });

  it("invokes onFilterChange when the user picks a filter", async () => {
    const onFilterChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ProviderTraceCard entries={[sttEntry, ttsEntry]} onFilterChange={onFilterChange} />,
    );
    await user.click(screen.getByTestId("trace-filter-tts"));
    expect(onFilterChange).toHaveBeenCalledWith("tts");
  });

  it("filters entries to a single capability when filter='stt'", () => {
    renderWithProviders(
      <ProviderTraceCard entries={[sttEntry, ttsEntry]} filter="stt" />,
    );
    expect(screen.getByText(/^whisper$/)).toBeInTheDocument();
    expect(screen.queryByText(/^kokoro$/)).not.toBeInTheDocument();
  });
});
