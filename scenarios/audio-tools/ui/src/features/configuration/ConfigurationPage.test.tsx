import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";

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

import { ConfigurationPage } from "./ConfigurationPage";
import {
  getProviderConfig,
  listByokCredentials,
  upsertByokCredential,
  updateProviderConfig,
  deleteByokCredential,
} from "../../services/settings";

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

beforeEach(() => {
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
    expect(await screen.findByText(/openai-tts/)).toBeInTheDocument();
    expect(screen.getByText(/fp_abc123/)).toBeInTheDocument();
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

  it("submits a new BYOK credential through upsertByokCredential exactly once", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ConfigurationPage />);
    await screen.findByText(/openai-tts/);

    await user.type(screen.getByPlaceholderText(strings.config.providerPlaceholder), "elevenlabs");
    await user.type(screen.getByPlaceholderText(strings.config.apiKeyPlaceholder), "sk-test");
    await user.click(screen.getByRole("button", { name: strings.config.addCredential }));

    await waitFor(() => {
      expect(vi.mocked(upsertByokCredential)).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(upsertByokCredential)).toHaveBeenCalledWith("elevenlabs", "tts", "sk-test");
  });
});
