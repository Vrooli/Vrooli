import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { SettingsPolicyProjection } from "../../types";
import { WorkshopTab } from "./WorkshopTab";

const projection: SettingsPolicyProjection = {
  effectiveControls: {
    defaultMode: "yolo",
    autoInitialize: true,
    autoAdvanceEnabled: true,
    cascadeEnabled: true,
    autoAdvanceDelaySeconds: 10,
    maxAutoRounds: 10,
    autoFixup: false,
    maxFixupAttempts: 2,
    reviewAgentEnabled: true,
    reviewCodeQualityMinScore: 60,
    reviewTestMinPassRate: 1,
    reviewMaxBlockingViolations: 0,
    reviewMaxWarnings: -1,
    reviewRequireScreenshots: true,
    reviewRequireTests: true,
    agentMaxTurns: 600,
    agentTimeoutSeconds: 3600,
  },
  classifications: [
    {
      field: "auto_advance_workshop",
      role: "policy_control",
      control: "auto_advance.enabled",
      note: "Retained user preference.",
    },
    {
      field: "auto_advance_delay_seconds",
      role: "policy_control",
      control: "auto_advance.delay_seconds",
      note: "Retained user preference.",
    },
  ],
};

describe("WorkshopTab policy-controls labeling", () => {
  it("labels the workshop card as policy controls and lists destinations from the projection", () => {
    renderWithProviders(
      <WorkshopTab form={DEFAULT_SETTINGS} patch={() => {}} policyProjection={projection} />,
    );

    expect(screen.getByTestId("policy-controls-badge")).toBeInTheDocument();
    const note = screen.getByTestId("policy-controls-note");
    expect(note).toHaveTextContent("transition policies");
    expect(note).toHaveTextContent("auto_advance.enabled");
    expect(note).toHaveTextContent("auto_advance.delay_seconds");

    // Existing user-facing controls stay untouched.
    expect(screen.getByText("Auto-Advance Workshop")).toBeInTheDocument();
    expect(screen.getByText("Auto-Advance Delay")).toBeInTheDocument();
  });

  it("keeps the static labeling when the projection is unavailable", () => {
    renderWithProviders(
      <WorkshopTab form={DEFAULT_SETTINGS} patch={() => {}} policyProjection={null} />,
    );

    expect(screen.getByTestId("policy-controls-badge")).toBeInTheDocument();
    const note = screen.getByTestId("policy-controls-note");
    expect(note).toHaveTextContent("transition policies");
    expect(note).not.toHaveTextContent("auto_advance.enabled");
  });
});
