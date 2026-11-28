import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { Readiness } from "@vrooli/proto-types/audio-tools/v1/models/models_pb";

const rpc = vi.hoisted(() => ({
  listModels: vi.fn(),
  resolveModel: vi.fn(),
}));

const WHISPER_LOCAL_ENGINE_ID = "whisper-local";

vi.mock("@connectrpc/connect", () => ({
  createClient: () => rpc,
}));

import { ModelsPage } from "./ModelsPage";
import { makeModelRecord } from "./mocks/factories";

function lastListRequest() {
  const calls = rpc.listModels.mock.calls;
  return calls[calls.length - 1]?.[0];
}

beforeEach(() => {
  rpc.listModels.mockResolvedValue({
    models: [
      makeModelRecord({ engineDefaults: [WHISPER_LOCAL_ENGINE_ID] }),
      makeModelRecord({
        id: "kokoro-82m",
        name: "Kokoro 82M",
        providerIds: ["kokoro-tts"],
        engineIds: [],
        capability: "tts",
        runtime: "onnx",
        languages: ["en", "ja"],
        sizeMbApprox: 82,
        license: "Apache-2.0",
        isDefault: false,
        roles: ["fast"],
      }),
    ],
  });
  rpc.resolveModel.mockResolvedValue({
    model: makeModelRecord(),
    explanation: "selected: readiness=available, roles=default,accurate",
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ModelsPage", () => {
  it("renders registry records, readiness facts, and the default badge", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ModelsPage />);

    expect(await screen.findByText(strings.models.title)).toBeInTheDocument();
    expect(await screen.findByText(/Whisper Large v3/)).toBeInTheDocument();
    const registry = screen.getByLabelText(strings.models.registryTitle);
    expect(within(registry).getByText(/whisper-large-v3/)).toBeInTheDocument();
    expect(screen.getByText(strings.models.defaultBadge)).toBeInTheDocument();

    const row = screen.getByText(/Whisper Large v3/).closest("tr");
    expect(row).not.toBeNull();
    if (row) {
      expect(
        within(row).getByText(strings.models.readiness.available),
      ).toBeInTheDocument();
      expect(
        within(row).getByText(strings.models.commercial.allowed),
      ).toBeInTheDocument();
      expect(
        within(row).getByText(strings.models.defaultBadge),
      ).toBeInTheDocument();
      await user.click(
        within(row).getByRole("button", { name: strings.models.detailsToggle }),
      );
    }
    expect(await screen.findByText(strings.models.detailsEngineDefaults)).toBeInTheDocument();
    const engineDefaultLabel = screen.getByText(
      strings.models.detailsEngineDefaults,
    );
    expect(engineDefaultLabel.parentElement?.querySelector("dd")).toHaveTextContent(
      WHISPER_LOCAL_ENGINE_ID,
    );
  });

  it("shows the promoted default and its size per capability", async () => {
    renderWithProviders(<ModelsPage />);

    const panel = await screen.findByLabelText(strings.models.footprintTitle);
    expect(within(panel).getByText(/whisper-large-v3/)).toBeInTheDocument();
    expect(within(panel).getByText(strings.models.sizeMb)).toBeInTheDocument();
  });

  it("distinguishes Kale's conditional product route from Kokoro's catalog default", async () => {
    const user = userEvent.setup();
    rpc.listModels.mockResolvedValue({
      models: [
        makeModelRecord(),
        makeModelRecord({
          id: "kokoro-82m",
          name: "Kokoro 82M",
          providerIds: ["kokoro-tts"],
          engineIds: [],
          capability: "tts",
          runtime: "kokoro",
          languages: ["en"],
          sizeMbApprox: 330,
          isDefault: true,
          roles: ["default"],
        }),
        makeModelRecord({
          id: "kale-tts-1-nano",
          name: "Kale TTS 1 Nano",
          providerIds: ["kale-tts"],
          engineIds: [],
          capability: "tts",
          runtime: "onnxruntime",
          languages: ["en"],
          sizeMbApprox: 14.8,
          readiness: Readiness.CANDIDATE,
          roles: ["fast", "edge"],
          defaultEligible: false,
          isDefault: false,
          notes: "Conditional product auto route: Kale Nano is tried before Kokoro when healthy; complete installed model bundle is about 20 MB including voice assets. Kale remains an unqualified candidate, not the catalog default.",
        }),
      ],
    });
    renderWithProviders(<ModelsPage />);

    const kale = await screen.findByText(/Kale TTS 1 Nano/);
    const row = kale.closest("tr");
    expect(row).not.toBeNull();
    if (!row) return;
    expect(within(row).getByText(strings.models.readiness.candidate)).toBeInTheDocument();
    expect(within(row).queryByText(strings.models.defaultBadge)).not.toBeInTheDocument();
    await user.click(within(row).getByRole("button", { name: strings.models.detailsToggle }));
    expect(await screen.findByText(/Conditional product auto route: Kale Nano is tried before Kokoro/)).toBeInTheDocument();
    expect(screen.getByText(/complete installed model bundle is about 20 MB including voice assets/)).toBeInTheDocument();

    const footprint = screen.getByLabelText(strings.models.footprintTitle);
    expect(within(footprint).getByText(/kokoro-82m/)).toBeInTheDocument();
    expect(within(footprint).queryByText(/kale-tts-1-nano/)).not.toBeInTheDocument();
  });

  it("sends the default (unfiltered) ListModelsRequest", async () => {
    renderWithProviders(<ModelsPage />);

    await screen.findByText(/Whisper Large v3/);
    expect(lastListRequest()).toEqual({
      capability: "",
      language: "",
      commercialOnly: false,
      onlyReady: false,
    });
  });

  it("forwards capability, language, commercial-only, and only-ready to the request", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ModelsPage />);
    await screen.findByText(/Whisper Large v3/);

    const filters = screen.getByRole("region", {
      name: strings.models.filtersTitle,
    });

    await user.selectOptions(
      within(filters).getByLabelText(strings.models.capabilityLabel),
      "tts",
    );
    await waitFor(() => {
      expect(lastListRequest().capability).toBe("tts");
    });

    await user.type(
      within(filters).getByLabelText(strings.models.languageLabel),
      "en-US",
    );
    await waitFor(() => {
      expect(lastListRequest().language).toBe("en");
    });

    await user.click(
      within(filters).getByLabelText(strings.models.commercialOnlyLabel),
    );
    await user.click(
      within(filters).getByLabelText(strings.models.onlyReadyLabel),
    );
    await waitFor(() => {
      expect(lastListRequest()).toMatchObject({
        commercialOnly: true,
        onlyReady: true,
      });
    });
  });

  it("renders the empty state when the server returns no models", async () => {
    rpc.listModels.mockResolvedValue({ models: [] });
    renderWithProviders(<ModelsPage />);

    expect(
      await screen.findByText(strings.models.empty),
    ).toBeInTheDocument();
  });

  it("resolves a model and renders the server explanation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ModelsPage />);
    await screen.findByText(/Whisper Large v3/);

    const resolve = screen.getByRole("region", {
      name: strings.models.resolveTitle,
    });

    await user.selectOptions(
      within(resolve).getByLabelText(strings.models.resolveCapabilityLabel),
      "tts",
    );
    await user.type(
      within(resolve).getByLabelText(strings.models.resolveLanguageLabel),
      "fr-CA",
    );
    await user.click(
      within(resolve).getByLabelText(strings.models.resolveCommercialLabel),
    );
    await user.click(
      within(resolve).getByRole("button", {
        name: strings.models.resolveButton,
      }),
    );

    expect(
      await within(resolve).findByText(
        /selected: readiness=available, roles=default,accurate/,
      ),
    ).toBeInTheDocument();
    expect(
      within(resolve).getByText(strings.models.resolveResultTitle),
    ).toBeInTheDocument();
    expect(within(resolve).getByText(/Whisper Large v3/)).toBeInTheDocument();

    const calls = rpc.resolveModel.mock.calls;
    expect(calls[calls.length - 1]?.[0]).toEqual({
      capability: "tts",
      language: "fr",
      engineId: "",
      commercial: true,
    });
  });

  it("resolves an STT model within the entered engine", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ModelsPage />);
    await screen.findByText(/Whisper Large v3/);
    const resolve = screen.getByRole("region", {
      name: strings.models.resolveTitle,
    });

    await user.type(
      within(resolve).getByLabelText(strings.models.resolveEngineLabel),
      "sherpa-streaming",
    );
    await user.click(
      within(resolve).getByRole("button", { name: strings.models.resolveButton }),
    );
    await waitFor(() => {
      const calls = rpc.resolveModel.mock.calls;
      expect(calls[calls.length - 1]?.[0]).toMatchObject({
        capability: "stt",
        engineId: "sherpa-streaming",
      });
    });
  });

  it("renders a friendly message when resolve reports NOT_FOUND", async () => {
    const user = userEvent.setup();
    rpc.resolveModel.mockRejectedValue(
      Object.assign(new Error("no eligible model"), { code: "not_found" }),
    );
    renderWithProviders(<ModelsPage />);
    await screen.findByText(/Whisper Large v3/);

    const resolve = screen.getByRole("region", {
      name: strings.models.resolveTitle,
    });

    await user.click(
      within(resolve).getByRole("button", {
        name: strings.models.resolveButton,
      }),
    );

    expect(
      await within(resolve).findByText(strings.models.resolveNotFound),
    ).toBeInTheDocument();
    expect(
      within(resolve).queryByText(strings.apiError.loadFailedTitle),
    ).not.toBeInTheDocument();
  });
});
