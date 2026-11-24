import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchAmbientSeams } from "../../api/ambientSeams";
import { AmbientSeamPanel } from "./AmbientSeamPanel";

vi.mock("../../api/ambientSeams", () => ({ searchAmbientSeams: vi.fn() }));

describe("AmbientSeamPanel", () => {
  beforeEach(() => vi.clearAllMocks());
  it("shows source location and static-vs-runtime limits", async () => {
    vi.mocked(searchAmbientSeams).mockResolvedValue({
      query: "HTTP default client", scenario: "code-facts", scope: "one bounded Go scenario", match_status: "MATCHED",
      results: [{
        id: "code-facts:ambient:api/handlers/facts/module.go:28:http_client", title: "http_client seam: http.DefaultClient",
        summary: "http_client in NON_TEST_SOURCE_CANDIDATE at api/handlers/facts/module.go:28 within Module; passed as an argument to internalfacts.NewTypeScriptGraphProvider",
        scenario: "code-facts", kind: "http_client", expression: "http.DefaultClient", context_symbol: "Module", source_layer: "NON_TEST_SOURCE_CANDIDATE",
        source_path: "api/handlers/facts/module.go", line: 28, source_sha256: "deadbeef", argument_context_status: "STATIC_ARGUMENT_RELATIONSHIP",
        argument_callee: "internalfacts.NewTypeScriptGraphProvider", argument_line: 28, matched_terms: ["client", "default", "http"], matched_fields: ["kind", "expression"],
        match_status: "MATCHED", match_evidence: [{ term: "client", field: "expression", value: "http.DefaultClient", source_path: "api/handlers/facts/module.go", line: 28, source_sha256: "deadbeef" }],
        basis: "go_ast_call_or_provider_expression", limit: "Source location and file suffix do not prove build inclusion, production/test dispatch, runtime use, or replacement of this dependency.", score: 1,
      }], degradations: [],
    });
    render(<AmbientSeamPanel />);
    fireEvent.click(screen.getByRole("button", { name: "Search seams" }));
    await waitFor(() => expect(screen.getByText(/http_client seam: http.DefaultClient/)).toBeInTheDocument());
    expect(screen.getByText(/Lexical match status: MATCHED/)).toBeInTheDocument();
    expect(screen.getByText(/client matched expression=http.DefaultClient at api\/handlers\/facts\/module.go:28/)).toBeInTheDocument();
    expect(screen.getByText("Lexical query matches: client, default, http · evidence fields: kind, expression")).toBeInTheDocument();
    expect(screen.getByText(/module\.go:28 · sha256:deadbeef/)).toBeInTheDocument();
    expect(screen.getByText(/NON_TEST_SOURCE_CANDIDATE/)).toBeInTheDocument();
    expect(screen.getByText(/AST argument context STATIC_ARGUMENT_RELATIONSHIP: http.DefaultClient appears within arguments to internalfacts.NewTypeScriptGraphProvider at line 28/)).toBeInTheDocument();
    expect(screen.getByText(/Evidence basis: go_ast_call_or_provider_expression/)).toBeInTheDocument();
    expect(screen.getByText(/do not prove build inclusion/)).toBeInTheDocument();
  });

  it("preserves no-match degradation and the static-only argument boundary", async () => {
    vi.mocked(searchAmbientSeams).mockResolvedValue({
      query: "underwater penguin", scenario: "code-facts", scope: "one bounded Go scenario", match_status: "UNKNOWN", results: [],
      degradations: [{ code: "no_seam_match", message: "No recognized ambient seam matched within the bounded Go source scope." }],
    });
    render(<AmbientSeamPanel />);
    fireEvent.change(screen.getByLabelText("Ambient seam query"), { target: { value: "underwater penguin" } });
    fireEvent.click(screen.getByRole("button", { name: "Search seams" }));
    expect(await screen.findByText("No ambient seam matched in this bounded source scope.")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("no_seam_match: No recognized ambient seam matched within the bounded Go source scope.");
    expect(screen.getByText(/direct AST argument containment is source syntax, not data flow or runtime wiring/)).toBeInTheDocument();
  });
});
