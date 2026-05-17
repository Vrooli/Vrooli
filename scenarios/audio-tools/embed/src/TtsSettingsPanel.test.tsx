import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

import { TtsSettingsPanel } from "./TtsSettingsPanel";

describe("TtsSettingsPanel", () => {
  it("renders the default heading and aria-label so screen readers can find it", () => {
    render(<TtsSettingsPanel />);
    expect(screen.getByRole("region", { name: "TTS settings" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Voice Output" })).toBeInTheDocument();
  });

  it("uses caller-supplied heading, body, and ariaLabel when provided", () => {
    render(
      <TtsSettingsPanel ariaLabel="Custom TTS region" heading="Output Heading" body="Output body text" />,
    );
    expect(screen.getByRole("region", { name: "Custom TTS region" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Output Heading" })).toBeInTheDocument();
    expect(screen.getByText("Output body text")).toBeInTheDocument();
  });
});
