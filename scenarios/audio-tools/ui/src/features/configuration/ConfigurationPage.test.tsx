import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";
import { i18n } from "../../i18n";

vi.mock("../../services/settings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../services/settings")>();
  return {
    ...actual,
    getProviderConfig: vi.fn(),
    listByokCredentials: vi.fn(),
    updateProviderConfig: vi.fn(),
    upsertByokCredential: vi.fn(),
    deleteByokCredential: vi.fn(),
  };
});

type TestProvider = {
  id: string;
  name: string;
  kind: "byok_api" | "gateway" | "full_duplex" | "local_resource";
  capabilities: string[];
};

type CatalogProvider = TestProvider & {
  status: "active" | "planned";
  description: string;
};

/**
 * Mutable catalogue backing the generated-module mock. Tests seed it per
 * case so option filtering and the empty state are exercised without
 * depending on toggling real catalogue entries.
 */
const catalog = vi.hoisted(() => ({
  providers: Object.create(null) as Record<string, CatalogProvider>,
}));

vi.mock("../providers/providerCatalog.generated", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../providers/providerCatalog.generated")>();
  return {
    ...actual,
    PROVIDERS: catalog.providers,
  };
});

import { ConfigurationPage, DUPLEX_CREDENTIAL_PROVISION_COMMAND } from "./ConfigurationPage";
import {
  getProviderConfig,
  listByokCredentials,
  upsertByokCredential,
  updateProviderConfig,
  deleteByokCredential,
} from "../../services/settings";

const TEST_PROVIDERS: TestProvider[] = [
  { id: "openai-whisper", name: "OpenAI Whisper (BYOK)", kind: "byok_api", capabilities: ["stt"] },
  { id: "deepgram", name: "Deepgram (BYOK)", kind: "byok_api", capabilities: ["stt"] },
  { id: "openai-tts", name: "OpenAI TTS (BYOK)", kind: "byok_api", capabilities: ["tts"] },
  { id: "elevenlabs", name: "ElevenLabs (BYOK)", kind: "byok_api", capabilities: ["tts"] },
  { id: "openrouter", name: "OpenRouter", kind: "byok_api", capabilities: ["summarize"] },
  { id: "openai-gpt-live", name: "OpenAI GPT-Live 1.1", kind: "full_duplex", capabilities: ["duplex"] },
  // Non-credential provider with TTS support must never appear in the select.
  { id: "kokoro-tts", name: "Kokoro TTS", kind: "local_resource", capabilities: ["tts"] },
];

const setProviders = (providers: TestProvider[]) => {
  for (const id of Object.keys(catalog.providers)) {
    Reflect.deleteProperty(catalog.providers, id);
  }
  for (const provider of providers) catalog.providers[provider.id] = {
    ...provider,
    status: provider.kind === "full_duplex" ? "planned" : "active",
    description: provider.kind === "full_duplex"
      ? "Planned hosted full-duplex audio provider; runtime composition is not yet available"
      : "Test provider",
  };
};

const happyProvider = {
  ok: true as const,
  data: { byokEnabled: true, vrooliEnabled: false, localEnabled: true },
};

const happyCreds = {
  ok: true as const,
  data: [
    {
      providerId: "openai-tts",
      capability: "tts",
      fingerprint: "fp_abc123",
      createdAt: "2026-05-16T00:00:00Z",
    },
  ],
};

const providerSelect = () => screen.getByLabelText<HTMLSelectElement>(strings.config.providerFieldLabel);
const capabilitySelect = () =>
  screen.getByLabelText<HTMLSelectElement>(strings.config.capabilityFieldLabel);
const optionValues = (select: HTMLSelectElement) =>
  Array.from(select.options).map((option) => option.value);
const addButton = () =>
  screen.getByRole("button", { name: strings.config.addCredential });

beforeEach(() => {
  setProviders(TEST_PROVIDERS);
  vi.mocked(getProviderConfig).mockResolvedValue(happyProvider);
  vi.mocked(listByokCredentials).mockResolvedValue(happyCreds);
  vi.mocked(updateProviderConfig).mockResolvedValue({ ok: true, data: undefined as never });
  vi.mocked(upsertByokCredential).mockResolvedValue({ ok: true, data: undefined as never });
  vi.mocked(deleteByokCredential).mockResolvedValue({ ok: true, data: undefined as never });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ConfigurationPage", () => {
  it("renders happy data with tier flags and credentials table", async () => {
    renderWithProviders(<ConfigurationPage />);
    expect(await screen.findByText(strings.config.title)).toBeInTheDocument();
    expect(await screen.findByText(/fp_abc123/)).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "openai-tts" })).toBeInTheDocument();
  });

  it("renders empty state for BYOK credentials", async () => {
    vi.mocked(listByokCredentials).mockResolvedValue({ ok: true, data: [] });
    renderWithProviders(<ConfigurationPage />);
    expect(await screen.findByText(strings.config.byokEmpty)).toBeInTheDocument();
  });

  it("renders error state when provider config fails", async () => {
    vi.mocked(getProviderConfig).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "boom", 500),
    });
    renderWithProviders(<ConfigurationPage />);
    await waitFor(() => expect(screen.getByText(/boom/)).toBeInTheDocument());
  });

  it("filters provider options by the selected capability and credential kind", async () => {
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/fp_abc123/);

    expect(optionValues(providerSelect())).toEqual(["", "openai-tts", "elevenlabs"]);
    expect(
      screen.getByRole("option", { name: "ElevenLabs (BYOK) (elevenlabs)" }),
    ).toBeInTheDocument();
    // Local-resource TTS provider is excluded; other capabilities are excluded.
    expect(screen.queryByRole("option", { name: /Kokoro/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /OpenAI Whisper/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /OpenRouter/ })).not.toBeInTheDocument();
  });

  it("resets an invalid provider when the capability changes", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/fp_abc123/);

    await user.selectOptions(providerSelect(), "elevenlabs");
    expect(providerSelect().value).toBe("elevenlabs");

    await user.selectOptions(capabilitySelect(), "stt");

    expect(providerSelect().value).toBe("");
    expect(optionValues(providerSelect())).toEqual(["", "openai-whisper", "deepgram"]);
    await user.type(
      screen.getByPlaceholderText(strings.config.apiKeyPlaceholder),
      "sk-test",
    );
    expect(addButton()).toBeDisabled();
  });

  it("submits a new BYOK credential with the exact provider id", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/fp_abc123/);

    await user.selectOptions(providerSelect(), "elevenlabs");
    await user.type(
      screen.getByPlaceholderText(strings.config.apiKeyPlaceholder),
      "sk-test",
    );
    await user.click(addButton());

    await waitFor(() => {
      expect(vi.mocked(upsertByokCredential)).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(upsertByokCredential)).toHaveBeenCalledWith("elevenlabs", "tts", "sk-test");
  });

  it("routes full-duplex credential setup through Vrooli's canonical credential authority", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/fp_abc123/);

    await user.selectOptions(capabilitySelect(), "duplex");
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: i18n.t(strings.config.duplexCredentialOpenPlatform) }))
      .toHaveAttribute("href", "https://platform.openai.com/api-keys");
    expect(screen.getByRole("link", { name: i18n.t(strings.config.duplexCredentialOpenPlatform) }))
      .toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByRole("status").querySelector("code")?.textContent)
      .toBe(DUPLEX_CREDENTIAL_PROVISION_COMMAND);
    expect(screen.getByRole("button", { name: i18n.t(strings.streamConfigAdmin.copyCommand) }))
      .toBeInTheDocument();
    expect(screen.queryByPlaceholderText(strings.config.apiKeyPlaceholder)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(strings.config.providerFieldLabel)).not.toBeInTheDocument();
    expect(vi.mocked(upsertByokCredential)).not.toHaveBeenCalled();
  });

  it("shows an announced empty state when no credential provider supports the capability", async () => {
    setProviders(TEST_PROVIDERS.filter((provider) => !provider.capabilities.includes("summarize")));
    const user = userEvent.setup();
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/fp_abc123/);

    await user.selectOptions(capabilitySelect(), "summarize");

    const emptyHint = await screen.findByText(
      i18n.t(strings.config.providerNoneForCapability, { capability: "summarize" }),
    );
    expect(emptyHint).toHaveAttribute("role", "status");
    expect(screen.queryByLabelText(strings.config.providerFieldLabel)).not.toBeInTheDocument();

    await user.type(
      screen.getByPlaceholderText(strings.config.apiKeyPlaceholder),
      "sk-test",
    );
    expect(addButton()).toBeDisabled();
  });
});
