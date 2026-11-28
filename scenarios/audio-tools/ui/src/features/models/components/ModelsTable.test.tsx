import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../test-utils";
import { strings } from "../../../consts/strings";
import { ModelsTable } from "./ModelsTable";
import { makeModelRecord } from "../mocks/factories";
import type { ModelRow } from "../../../services/models";

const model = (overrides: Parameters<typeof makeModelRecord>[0] = {}) =>
  makeModelRecord(overrides) as unknown as ModelRow;

afterEach(() => cleanup());

const WHISPER_MODEL_ID = "whisper-large-v3";
const KOKORO_MODEL_ID = "kokoro-82m";
const WHISPER_ENGINE_ID = "whisper-local";
const KOKORO_ENGINE_ID = "kokoro-engine";

describe("ModelsTable", () => {
  it("expands only the selected model and exposes its details disclosure state", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <ModelsTable models={[
        model(),
        model({ id: KOKORO_MODEL_ID, name: "Kokoro 82M", providerIds: ["kokoro-tts"], engineIds: [KOKORO_ENGINE_ID] }),
      ]} />,
    );

    const rows = screen.getAllByRole("row");
    const whisperRow = rows.find((row) => within(row).queryByText(WHISPER_MODEL_ID));
    const kokoroRow = rows.find((row) => within(row).queryByText(KOKORO_MODEL_ID));
    expect(whisperRow).toBeDefined();
    expect(kokoroRow).toBeDefined();

    const whisperToggle = within(whisperRow!).getByRole("button", { name: strings.models.detailsToggle });
    await user.click(whisperToggle);
    expect(whisperToggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(WHISPER_ENGINE_ID)).toBeInTheDocument();

    const kokoroToggle = within(kokoroRow!).getByRole("button", { name: strings.models.detailsToggle });
    await user.click(kokoroToggle);
    expect(kokoroToggle).toHaveAttribute("aria-expanded", "true");
    expect(whisperToggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText(WHISPER_ENGINE_ID)).not.toBeInTheDocument();
    expect(screen.getByText(KOKORO_ENGINE_ID)).toBeInTheDocument();
  });
});
