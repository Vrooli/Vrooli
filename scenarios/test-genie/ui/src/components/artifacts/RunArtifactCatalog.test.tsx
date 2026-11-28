import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { RunArtifactCatalog } from "./RunArtifactCatalog";
import { useRunArtifacts } from "../../hooks/useRunArtifacts";

vi.mock("../../hooks/useRunArtifacts", () => ({ useRunArtifacts: vi.fn() }));
const mockedUseRunArtifacts = vi.mocked(useRunArtifacts);

describe("RunArtifactCatalog", () => {
  it("retains cached evidence during catalog loss and retries the same run", () => {
    const data = { artifacts: [{ id: "artifact:cached", kind: "log", label: "Last successful evidence" }] };
    const refetch = vi.fn();
    mockedUseRunArtifacts.mockReturnValue({ isLoading: false, isError: true, isFetching: false, data, refetch } as unknown as ReturnType<typeof useRunArtifacts>);
    const view = render(<RunArtifactCatalog scenarioName="demo" runId="run-1" />);

    expect(screen.getByText("Last successful evidence")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(/refresh failed/i);
    fireEvent.click(screen.getByRole("button", { name: /retry artifact catalog/i }));
    expect(refetch).toHaveBeenCalledOnce();

    mockedUseRunArtifacts.mockReturnValue({ isLoading: false, isError: false, isFetching: false, data } as unknown as ReturnType<typeof useRunArtifacts>);
    view.rerender(<RunArtifactCatalog scenarioName="demo" runId="run-1" />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByText("Last successful evidence")).toBeInTheDocument();
  });

  it("opens opaque catalog identifiers through the owner route and marks legacy evidence", () => {
    mockedUseRunArtifacts.mockReturnValue({ isLoading: false, isError: false, data: { artifacts: [{ id: "artifact:opaque", kind: "future-kind", label: "Future evidence", relationships: [{ type: "derived_from", targetArtifactId: "artifact:parent" }] }], legacyDiscovered: true, degradedReasons: ["run predates persisted artifact catalogs; evidence was discovered read-only"] } } as ReturnType<typeof useRunArtifacts>);
    render(<RunArtifactCatalog scenarioName="demo" runId="run-1" />);
    expect(screen.getByText("Future evidence")).toBeInTheDocument();
    expect(screen.getByText(/Evidence needs attention/i)).toBeInTheDocument();
    expect(screen.getByText(/Artifact ID: artifact:opaque/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open/i })).toHaveAttribute("href", expect.stringContaining("/api/v1/scenarios/demo/runs/run-1/artifacts/artifact%3Aopaque"));
  });
});
