import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { ArtifactStatusPill } from "./ArtifactStatusPill";

describe("ArtifactStatusPill", () => {
  afterEach(() => cleanup());

  // cimode is enabled in tests; useTranslation echoes the key path verbatim.
  // The integration test (Inventory row → pill) covers the live-locale
  // rendering separately.
  it("renders the fresh state with a green tone", () => {
    renderWithProviders(<ArtifactStatusPill status="fresh" />);
    const pill = screen.getByTestId("artifact-status-fresh");
    expect(pill).toBeInTheDocument();
    expect(pill).toHaveTextContent("artifacts.statusFresh");
  });

  it("renders the missing state as Needs generate", () => {
    renderWithProviders(<ArtifactStatusPill status="missing" />);
    expect(screen.getByTestId("artifact-status-missing")).toHaveTextContent(
      "artifacts.statusMissing",
    );
  });

  it("renders the needs_generate alias for failureReason-derived state", () => {
    renderWithProviders(<ArtifactStatusPill status="needs_generate" />);
    expect(screen.getByTestId("artifact-status-needs_generate")).toHaveTextContent(
      "artifacts.statusNeedsGenerate",
    );
  });

  it("honours an explicit testId", () => {
    renderWithProviders(<ArtifactStatusPill status="fresh" testId="custom-id" />);
    expect(screen.getByTestId("custom-id")).toBeInTheDocument();
  });
});
