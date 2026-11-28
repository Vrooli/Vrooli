import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { ModelCapacityNote } from "./ModelCapacityNote";

describe("ModelCapacityNote", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it.each([
    ["a model with a processor path says it runs there when the graphics card is busy", true, "Runs on the processor if the graphics card is busy (slower)"],
    ["a graphics-card-only model says it may wait", false, "May wait for the graphics card if it's busy"],
  ])("%s", (_name, cpuCapable, text) => {
    renderWithProviders(<ModelCapacityNote model={{ gpuViable: true, cpuCapable }} />);
    expect(screen.getByTestId(selectors.workspace.capacityNote).textContent).toBe(text);
  });

  it.each([
    ["a model that is not running on the graphics card", { gpuViable: false, cpuCapable: true }],
    ["no model yet", null],
  ])("says nothing for %s", (_name, model) => {
    renderWithProviders(<ModelCapacityNote model={model} />);
    expect(screen.queryByTestId(selectors.workspace.capacityNote)).toBeNull();
  });
});
