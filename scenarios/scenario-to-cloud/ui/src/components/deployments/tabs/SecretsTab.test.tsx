import "@testing-library/jest-dom";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

const hooks = vi.hoisted(() => ({
  useVPSSecrets: vi.fn(),
  useExpectedSecrets: vi.fn(),
}));

vi.mock("../../../hooks/useVPSSecrets", () => ({ useVPSSecrets: hooks.useVPSSecrets }));
vi.mock("../../../hooks/useExpectedSecrets", () => ({ useExpectedSecrets: hooks.useExpectedSecrets }));

import { SecretsTab } from "./SecretsTab";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";

const configured = { key: "API_TOKEN" };
const extra = { key: "UNDECLARED" };
const expected = [
  { secret_id: "API_TOKEN", label: "API token", description: "Required for API access", classification: "service", required: true },
  { secret_id: "OPTIONAL_KEY", label: "Optional", description: "Optional integration", classification: "user", required: false, default_hint: "Only needed for integration" },
];

function first<T>(items: T[], description: string): T {
  const item = items[0];
  if (!item) throw new Error(`expected ${description}`);
  return item;
}

function last<T>(items: T[], description: string): T {
  const item = items[items.length - 1];
  if (!item) throw new Error(`expected ${description}`);
  return item;
}

describe("SecretsTab", () => {
  let create: ReturnType<typeof vi.fn>;
  let update: ReturnType<typeof vi.fn>;
  let deleteSecret: ReturnType<typeof vi.fn>;
  let revealSecret: ReturnType<typeof vi.fn>;
  let hideSecret: ReturnType<typeof vi.fn>;
  let refetch: ReturnType<typeof vi.fn>;
  let refetchExpected: ReturnType<typeof vi.fn>;
  let clipboardWriteText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    create = vi.fn().mockResolvedValue({ ok: true });
    update = vi.fn().mockResolvedValue({ ok: true });
    deleteSecret = vi.fn().mockResolvedValue({ ok: true });
    revealSecret = vi.fn().mockResolvedValue("revealed-value");
    hideSecret = vi.fn();
    refetch = vi.fn();
    refetchExpected = vi.fn();
    clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: clipboardWriteText } });
    hooks.useVPSSecrets.mockReturnValue({
      secrets: [configured, extra],
      metadata: { last_updated: "2026-08-14T12:00:00.000Z" },
      isLoading: false,
      error: null,
      refetch,
      revealSecret,
      hideSecret,
      getSecretValue: vi.fn(() => ({ value: "********", masked: true })),
      isRevealed: vi.fn(() => false),
      create,
      update,
      delete: deleteSecret,
      isCreating: false,
      isUpdating: false,
      isDeleting: false,
    });
    hooks.useExpectedSecrets.mockReturnValue({
      expectedSecrets: expected,
      summary: { total: 2, configured: 1, missing: 1, required: 0 },
      isLoading: false,
      error: null,
      refetch: refetchExpected,
    });
  });

  it("shows expected/configured/additional secrets and handles reveal, copy, and refresh", async () => {
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    expect(screen.getByText((_, element) =>
      element !== null && element.tagName === "P" && element.textContent.includes("expected secrets configured"),
    )).toBeInTheDocument();
    expect(screen.getAllByText("API_TOKEN")).toHaveLength(2);
    expect(screen.getByText("OPTIONAL_KEY")).toBeInTheDocument();
    expect(screen.getByText("UNDECLARED")).toBeInTheDocument();

    fireEvent.click(first(screen.getAllByTitle("Reveal value"), "reveal-secret button"));
    await waitFor(() => expect(revealSecret).toHaveBeenCalledWith("API_TOKEN"));
    fireEvent.click(first(screen.getAllByTitle("Copy value"), "copy-secret button"));
    await waitFor(() => expect(revealSecret).toHaveBeenCalledWith("API_TOKEN"));
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(refetch).toHaveBeenCalledOnce();
    expect(refetchExpected).toHaveBeenCalledOnce();
  });

  it("validates and creates a secret, including restart preference", async () => {
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    fireEvent.click(last(screen.getAllByRole("button", { name: "Add Secret" }), "modal add-secret button"));
    fireEvent.click(last(screen.getAllByRole("button", { name: "Add Secret" }), "add-secret modal button"));
    expect(screen.getByText("Key is required")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("MY_API_KEY"), { target: { value: "bad-key" } });
    fireEvent.change(screen.getByPlaceholderText("Enter secret value..."), { target: { value: "value" } });
    fireEvent.click(last(screen.getAllByRole("button", { name: "Add Secret" }), "add-secret modal button"));
    expect(screen.getByText("Key must be uppercase letters, numbers, and underscores")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("MY_API_KEY"), { target: { value: "NEW_SECRET" } });
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(last(screen.getAllByRole("button", { name: "Add Secret" }), "add-secret modal button"));
    await waitFor(() => expect(create).toHaveBeenCalledWith({ key: "NEW_SECRET", value: "value", restartScenario: true }));
  });

  it("edits and deletes secrets only after their validation requirements pass", async () => {
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    fireEvent.click(first(screen.getAllByTitle("Edit secret"), "edit-secret button"));
    fireEvent.click(screen.getByRole("button", { name: "Update Secret" }));
    expect(screen.getByText("Value is required")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Enter new value..."), { target: { value: "updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Update Secret" }));
    await waitFor(() => expect(update).toHaveBeenCalledWith({ key: "API_TOKEN", value: "updated", restartScenario: false }));

    fireEvent.click(first(screen.getAllByTitle("Delete secret"), "delete-secret button"));
    expect(screen.getByRole("button", { name: "Delete Secret" })).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("DELETE"), { target: { value: "NO" } });
    expect(screen.getByRole("button", { name: "Delete Secret" })).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("DELETE"), { target: { value: "DELETE" } });
    fireEvent.click(screen.getByRole("button", { name: "Delete Secret" }));
    await waitFor(() => expect(deleteSecret).toHaveBeenCalledWith({ key: "API_TOKEN", restartScenario: false }));

    cleanup();
    hooks.useVPSSecrets.mockReturnValue({
      secrets: [configured], metadata: null, isLoading: false, error: null, refetch,
      revealSecret, hideSecret, getSecretValue: vi.fn(() => ({ value: "plain-value", masked: false })),
      isRevealed: vi.fn(() => true), create, update, delete: deleteSecret,
      isCreating: false, isUpdating: false, isDeleting: false,
    });
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    fireEvent.click(screen.getByTitle("Hide value"));
    expect(hideSecret).toHaveBeenCalledWith("API_TOKEN");
    fireEvent.click(screen.getByTitle("Copy value"));
    await waitFor(() => expect(clipboardWriteText).toHaveBeenCalledWith("plain-value"));
  });

  it("renders loading and load-error states without exposing secret content", () => {
    hooks.useVPSSecrets.mockReturnValue({ isLoading: true, secrets: [] });
    hooks.useExpectedSecrets.mockReturnValue({ isLoading: false, expectedSecrets: [] });
    const { rerender } = renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();

    hooks.useVPSSecrets.mockReturnValue({ isLoading: false, error: new Error("secrets unavailable"), secrets: [] });
    rerender(<SecretsTab deploymentId="deployment-1" />);
    expect(screen.getByText("Failed to load secrets: secrets unavailable")).toBeInTheDocument();
  });

  it("handles expected-secret warnings and mutation/reveal failures safely", async () => {
    hooks.useVPSSecrets.mockReturnValue({
      secrets: [], metadata: null, isLoading: false, error: null, refetch, revealSecret, hideSecret,
      getSecretValue: vi.fn(() => ({ value: "********", masked: true })), isRevealed: vi.fn(() => false),
      create: vi.fn().mockRejectedValue(new Error("create denied")), update, delete: deleteSecret,
      isCreating: false, isUpdating: false, isDeleting: false,
    });
    hooks.useExpectedSecrets.mockReturnValue({ expectedSecrets: [expected[1]], summary: null, isLoading: false, error: new Error("metadata unavailable"), refetch: refetchExpected });
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    expect(screen.getByText("metadata unavailable. Showing VPS secrets only.")).toBeInTheDocument();
    expect(screen.getByText("0 secrets on VPS")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^Add$/ }));
    fireEvent.change(screen.getByPlaceholderText("Enter secret value..."), { target: { value: "value" } });
    fireEvent.click(last(screen.getAllByRole("button", { name: "Add Secret" }), "modal add-secret button"));
    expect(await screen.findByText("create denied")).toBeInTheDocument();

    cleanup();
    deleteSecret.mockRejectedValueOnce(new Error("delete denied"));
    hooks.useVPSSecrets.mockReturnValue({
      secrets: [configured], metadata: null, isLoading: false, error: null, refetch, hideSecret,
      revealSecret: vi.fn().mockRejectedValue(new Error("reveal denied")),
      getSecretValue: vi.fn(() => ({ value: "********", masked: true })), isRevealed: vi.fn(() => false),
      create, update: vi.fn().mockRejectedValue(new Error("update denied")), delete: deleteSecret,
      isCreating: false, isUpdating: false, isDeleting: false,
    });
    hooks.useExpectedSecrets.mockReturnValue({ expectedSecrets: [], summary: null, isLoading: false, error: null, refetch: refetchExpected });
    renderWithProviders(<SecretsTab deploymentId="deployment-1" />);
    fireEvent.click(screen.getByTitle("Reveal value"));
    await waitFor(() => expect(screen.getByTitle("Reveal value")).toBeInTheDocument());
    fireEvent.click(screen.getByTitle("Copy value"));
    await waitFor(() => expect(screen.getByTitle("Copy value")).toBeInTheDocument());
    fireEvent.click(screen.getByTitle("Edit secret"));
    fireEvent.change(screen.getByPlaceholderText("Enter new value..."), { target: { value: "new" } });
    fireEvent.click(screen.getByRole("button", { name: "Update Secret" }));
    expect(await screen.findByText("update denied")).toBeInTheDocument();
    fireEvent.click(screen.getByTitle("Delete secret"));
    fireEvent.change(screen.getByPlaceholderText("DELETE"), { target: { value: "DELETE" } });
    fireEvent.click(screen.getByRole("button", { name: "Delete Secret" }));
    await waitFor(() => expect(deleteSecret).toHaveBeenCalledWith({ key: "API_TOKEN", restartScenario: false }));
  });
});
