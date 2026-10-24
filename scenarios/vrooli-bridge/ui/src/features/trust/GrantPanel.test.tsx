import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { renderWithProviders } from "../../test-utils";

const { listGrants, revokeGrant } = vi.hoisted(() => ({ listGrants: vi.fn(), revokeGrant: vi.fn() }));

vi.mock("../../api/grants", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/grants")>();
  return { ...actual, grantsClient: { ...actual.grantsClient, listGrants, revokeGrant } };
});

import { GrantPanel } from "./GrantPanel";

describe("GrantPanel", () => {
  beforeEach(() => {
    listGrants.mockResolvedValue({ grants: [{ id: "grant-1", nodeId: "node-1", logicalId: "vrooli/test", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1n, ackedGeneration: 1n, receiptAccepted: true }] });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("reports local revocation while remote purge is pending", async () => {
    revokeGrant.mockResolvedValue({ id: "grant-1", purgeState: "pending" });
    const user = userEvent.setup();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    renderWithProviders(<QueryClientProvider client={queryClient}><GrantPanel /></QueryClientProvider>);

    await user.click(await screen.findByRole("button", { name: "Revoke" }));

    expect(await screen.findByRole("status")).toHaveTextContent("Grant revoked locally; remote purge is pending until the node reconnects.");
  });
});
