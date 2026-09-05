import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { AllowlistRulesManager } from "./AllowlistRulesManager";

const apiMocks = vi.hoisted(() => ({
  fetchAllowlistRules: vi.fn(),
  createAllowlistRule: vi.fn(),
  updateAllowlistRule: vi.fn(),
  deleteAllowlistRule: vi.fn()
}));

vi.mock("../lib/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../lib/api")>()),
  fetchAllowlistRules: apiMocks.fetchAllowlistRules,
  createAllowlistRule: apiMocks.createAllowlistRule,
  updateAllowlistRule: apiMocks.updateAllowlistRule,
  deleteAllowlistRule: apiMocks.deleteAllowlistRule
}));

function renderRules() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } }
  });
  return renderWithProviders(
    <QueryClientProvider client={queryClient}>
      <AllowlistRulesManager />
    </QueryClientProvider>
  );
}

describe("AllowlistRulesManager", () => {
  beforeEach(() => {
    apiMocks.fetchAllowlistRules.mockResolvedValue({ rules: [], count: 0 });
    apiMocks.createAllowlistRule.mockResolvedValue({ id: "rule-2" });
    apiMocks.updateAllowlistRule.mockResolvedValue({ id: "rule-1" });
    apiMocks.deleteAllowlistRule.mockResolvedValue(undefined);
  });

  afterEach(cleanup);

  it("validates and creates normalized allowlist rules", async () => {
    renderRules();
    expect(await screen.findByText("No allowlist rules configured.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByText("Path pattern is required")).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("e.g. *_test.go"), { target: { value: "  *_test.go " } });
    fireEvent.change(screen.getByPlaceholderText("pii_email, pii_phone_us"), { target: { value: " pii_email, pii_phone_us " } });
    fireEvent.change(screen.getByPlaceholderText("(optional)"), { target: { value: "  Test fixtures " } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));

    await waitFor(() => expect(apiMocks.createAllowlistRule).toHaveBeenCalledWith({
      path_pattern: "*_test.go",
      excluded_types: ["pii_email", "pii_phone_us"],
      description: "Test fixtures",
      enabled: true
    }));
  });

  it("edits, toggles, and deletes existing rules", async () => {
    apiMocks.fetchAllowlistRules.mockResolvedValue({
      rules: [{
        id: "rule-1",
        path_pattern: "fixtures/**",
        excluded_types: ["*"],
        description: "Fixtures",
        enabled: true
      }],
      count: 1
    });
    renderRules();
    expect(await screen.findByText("fixtures/**")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit allowlist rule fixtures/**" }));
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("(optional)"), { target: { value: "Changed" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(apiMocks.updateAllowlistRule).toHaveBeenCalledWith("rule-1", {
      path_pattern: "fixtures/**",
      excluded_types: ["*"],
      description: "Changed",
      enabled: true
    }));

    fireEvent.click(screen.getByRole("checkbox", { name: "Enabled" }));
    await waitFor(() => expect(apiMocks.updateAllowlistRule).toHaveBeenCalledWith("rule-1", {
      path_pattern: "fixtures/**",
      excluded_types: ["*"],
      description: "Fixtures",
      enabled: false
    }));

    fireEvent.click(screen.getByRole("button", { name: "Delete allowlist rule fixtures/**" }));
    expect(screen.getByText("Delete allowlist rule?")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(apiMocks.deleteAllowlistRule).toHaveBeenCalledWith("rule-1"));
  });
});
