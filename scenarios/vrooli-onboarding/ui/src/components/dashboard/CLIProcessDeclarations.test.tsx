import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { CLIProcessDeclarations } from "./CLIProcessDeclarations";
import { listProcessSpawnDeclarations } from "../../api/cliManifestDeclarations";

vi.mock("../../api/cliManifestDeclarations", () => ({ listProcessSpawnDeclarations: vi.fn() }));

describe("CLIProcessDeclarations", () => {
  beforeEach(() => {
    vi.mocked(listProcessSpawnDeclarations).mockResolvedValue({
      schema_id: "cli-manifest/v1", source_digest: "sha256:0123456789abcdef0123456789abcdef", authorization_note: "Declarations are not grants.",
      declarations: [{ manifest: "cli/manifest.json", declaring_owner: "vrooli", command: "process/start", declared_effect: "write", declared_run_eligible: true, declared_permissions: ["process:spawn"], process_spawn_declared: true, derived_scope: "vrooli:write", dispatch_eligible: true, authorization_source: "caller and node scopes" }],
    });
  });

  it("labels source declarations separately from grants", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<CLIProcessDeclarations />, { queryClient: client });
    expect(await screen.findByText("vrooli · process/start")).toBeInTheDocument();
    expect(screen.getByText("Declarations are not grants.")).toBeInTheDocument();
  });
});
