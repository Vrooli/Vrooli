import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

import { VoiceSettingsPanel } from "./VoiceSettingsPanel";

describe("VoiceSettingsPanel", () => {
  it("renders the default heading and aria-label so screen readers can find it", () => {
    render(<VoiceSettingsPanel />);
    expect(screen.getByRole("region", { name: "Voice settings" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Voice Input" })).toBeInTheDocument();
  });

  it("uses caller-supplied heading, body, and ariaLabel when provided", () => {
    render(
      <VoiceSettingsPanel
        ariaLabel="Custom voice region"
        heading="Custom Heading"
        body="Custom body text"
      />,
    );
    expect(screen.getByRole("region", { name: "Custom voice region" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Custom Heading" })).toBeInTheDocument();
    expect(screen.getByText("Custom body text")).toBeInTheDocument();
  });
});
