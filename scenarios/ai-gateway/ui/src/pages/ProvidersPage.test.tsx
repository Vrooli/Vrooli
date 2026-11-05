import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { listProviderRoles, smokeProvider } from "../api/gateway";
import { ProviderRoleSchema } from "@vrooli/proto-types/ai-gateway/v1/inventory/inventory_pb";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { providerRolesFixture } from "../test-utils/mocks/gateway";
import { ProvidersPage } from "./ProvidersPage";

vi.mock("../api/gateway", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/gateway")>();
  const { makeGatewayApiMocks } = await import("../test-utils/mocks/gateway");
  return { ...actual, ...makeGatewayApiMocks() };
});

describe("ProvidersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("[REQ:AIGW-UI-DASHBOARD] renders provider role inventory", async () => {
    renderWithProviders(<ProvidersPage />);

    expect(await screen.findByTestId(selectors.providers.table)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.providers.roleRow({ provider: "ollama", role: "chat.default" }))).toBeInTheDocument();
  });

  it("shows the empty state when no roles are returned", async () => {
    vi.mocked(listProviderRoles).mockResolvedValueOnce({
      ...providerRolesFixture,
      roles: [],
      warnings: [],
    });

    renderWithProviders(<ProvidersPage />);

    expect(await screen.findByTestId(selectors.providers.empty)).toBeInTheDocument();
  });

  it("renders warning and neutral provider statuses", async () => {
    vi.mocked(listProviderRoles).mockResolvedValueOnce({
      ...providerRolesFixture,
      roles: [
        create(ProviderRoleSchema, {
          provider: "ollama",
          role: "chat.default",
          capabilities: [],
          locality: "local",
          status: "stale",
          policySchemaVersion: "",
        }),
        create(ProviderRoleSchema, {
          provider: "openrouter",
          role: "chat.default",
          capabilities: [],
          locality: "remote",
          status: "unknown",
          policySchemaVersion: "",
        }),
      ],
      warnings: [],
    });

    renderWithProviders(<ProvidersPage />);

    expect(await screen.findByTestId(selectors.providers.roleRow({ provider: "ollama", role: "chat.default" }))).toHaveTextContent("stale");
    expect(screen.getByTestId(selectors.providers.roleRow({ provider: "openrouter", role: "chat.default" }))).toHaveTextContent("unknown");
  });

  it("renders provider inventory errors", async () => {
    vi.mocked(listProviderRoles).mockRejectedValueOnce(new Error("resource policy unavailable"));

    renderWithProviders(<ProvidersPage />);

    expect(await screen.findByTestId(selectors.providers.error)).toHaveTextContent("resource policy unavailable");
  });

  it("[REQ:AIGW-UI-DASHBOARD] runs provider smoke checks from the inventory page", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProvidersPage />);

    const smokeButtons = await screen.findAllByRole("button", { name: "pages.providers.smoke" });
    const firstSmokeButton = smokeButtons[0];
    if (!firstSmokeButton) {
      throw new Error("expected at least one provider smoke button");
    }
    await user.click(firstSmokeButton);

    expect(vi.mocked(smokeProvider).mock.calls[0]?.[0]).toBe("ollama");
  });

  it("renders smoke check failures inline", async () => {
    vi.mocked(smokeProvider).mockRejectedValueOnce(new Error("smoke failed"));
    const user = userEvent.setup();
    renderWithProviders(<ProvidersPage />);

    const smokeButtons = await screen.findAllByRole("button", { name: "pages.providers.smoke" });
    await user.click(smokeButtons[0]!);

    expect(screen.getByTestId(selectors.pages.providers)).toHaveTextContent("smoke failed");
  });
});
