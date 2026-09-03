import { screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { RunArtifactCatalog } from "./RunArtifactCatalog";
import { useRunArtifacts } from "../../hooks/useRunArtifacts";

vi.mock("../../hooks/useRunArtifacts", () => ({ useRunArtifacts: vi.fn() }));
const mockedUseRunArtifacts = vi.mocked(useRunArtifacts);

describe("RunArtifactCatalog", () => {
  it("uses only catalog-provided opaque artifact access and marks legacy evidence", () => {
    mockedUseRunArtifacts.mockReturnValue({ isLoading: false, isError: false, data: { artifacts: [{ id: "artifact:opaque", kind: "future-kind", label: "Future evidence", accessPath: "/api/v1/scenarios/demo/runs/run-1/artifacts/artifact%3Aopaque", relationships: [{ type: "derived_from", targetArtifactId: "artifact:parent" }] }], legacyDiscovered: true, degradedReasons: ["run predates persisted artifact catalogs; evidence was discovered read-only"] } } as ReturnType<typeof useRunArtifacts>);
    render(<RunArtifactCatalog scenarioName="demo" runId="run-1" />);
    expect(screen.getByText("Future evidence")).toBeInTheDocument();
    expect(screen.getByText(/Evidence needs attention/i)).toBeInTheDocument();
    expect(screen.getByText(/Artifact ID: artifact:opaque/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open/i })).toHaveAttribute("href", expect.stringContaining("artifact%3Aopaque"));
  });
});
