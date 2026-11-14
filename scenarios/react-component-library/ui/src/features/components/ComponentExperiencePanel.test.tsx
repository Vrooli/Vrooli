import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import type { ComponentExperience } from "../../api/components";
import { ComponentExperiencePanel } from "./ComponentExperiencePanel";

function experience(overrides: Partial<ComponentExperience> = {}): ComponentExperience {
  return {
    componentId: "button",
    libraryId: "react-component-library:Button",
    version: "1.2.0",
    contractId: "button",
    title: "Button",
    purpose: "Provide an accessible action.",
    evidenceStatus: "available",
    evidenceMessage: "Evidence is measured from the isolated story harness.",
    states: [{ id: "primary", exampleName: "primary", description: "Primary action." }],
    claims: [
      {
        id: "machine",
        type: "element-present",
        statement: "The action is present.",
        tier: "machine",
        states: ["primary"],
      },
      {
        id: "manual",
        type: "manual-review",
        statement: "The action is understandable.",
        tier: "manual",
        states: [],
      },
      {
        id: "aspirational",
        type: "future",
        statement: "The action can be improved.",
        tier: "aspirational",
        states: [],
      },
    ],
    evidence: [
      {
        claimId: "machine",
        verdict: "passed",
        stateId: "primary",
        exampleName: "primary",
        checkedAt: "2026-08-24T10:00:00.000Z",
        message: "Measured successfully.",
        viewport: "desktop",
        viewportWidth: 1280,
        viewportHeight: 720,
        captureRef: "https://example.test/captures/button.png",
      },
      {
        claimId: "manual",
        verdict: "failed",
        stateId: "primary",
        exampleName: "primary",
        checkedAt: "2026-08-24T10:00:00.000Z",
        message: "Requires review.",
        viewport: "desktop",
        viewportWidth: 1280,
        viewportHeight: 720,
        captureRef: "artifact://button-dom.json",
      },
      {
        claimId: "aspirational",
        verdict: "skipped",
        stateId: "primary",
        exampleName: "primary",
        checkedAt: "",
        message: "Not measured.",
        viewport: "desktop",
        viewportWidth: 1280,
        viewportHeight: 720,
        captureRef: "",
      },
    ],
    ...overrides,
  };
}

describe("ComponentExperiencePanel", () => {
  afterEach(() => vi.useRealTimers());

  it("renders measured, failed, skipped, stale, URL, and artifact evidence states", () => {
    vi.setSystemTime(new Date("2026-08-24T11:00:00.000Z"));
    const { getByText, getAllByText, getByRole } = renderWithProviders(
      <ComponentExperiencePanel isLoading={false} experience={experience()} />,
    );

    expect(getByText("Evidence is measured from the isolated story harness.")).toBeInTheDocument();
    expect(getByText("passed")).toBeInTheDocument();
    expect(getByText("failed")).toBeInTheDocument();
    expect(getByText("skipped")).toBeInTheDocument();
    expect(getAllByText("componentDetail.experience.stale")).toHaveLength(2);
    expect(getByRole("link", { name: "componentDetail.experience.openCapture" })).toHaveAttribute(
      "href",
      "https://example.test/captures/button.png",
    );
    expect(getByText("artifact://button-dom.json")).toBeInTheDocument();
  });

  it("renders a warning summary when evidence exists but is unavailable", () => {
    const { getByText } = renderWithProviders(
      <ComponentExperiencePanel
        isLoading={false}
        experience={experience({
          evidenceStatus: "unavailable",
          version: "",
          evidence: [],
          states: [],
          claims: [],
        })}
      />,
    );

    expect(getByText("componentDetail.experience.unavailable")).toBeInTheDocument();
    expect(getByText(/evidence is measured/i)).toBeInTheDocument();
  });

  it("uses the server message for a not-configured contract", () => {
    const { getByText } = renderWithProviders(
      <ComponentExperiencePanel
        isLoading={false}
        experience={experience({
          evidenceStatus: "not-configured",
          evidenceMessage: "No contract is registered yet.",
        })}
      />,
    );

    expect(getByText("No contract is registered yet.")).toBeInTheDocument();
  });

  it("renders the translated not-configured state when no experience is returned", () => {
    const { getByText } = renderWithProviders(<ComponentExperiencePanel isLoading={false} />);
    expect(getByText("componentDetail.experience.notConfigured")).toBeInTheDocument();
  });
});
