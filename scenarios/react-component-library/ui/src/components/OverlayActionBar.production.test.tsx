import { screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import type { OverlayActionList } from "../../../library/components/OverlayActionBar/versions/1.0.2-draft.1/OverlayActionBar";
import { fiveGermanOverlayActions } from "../../../library/usage-contracts/adversarial-fixtures";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
});

it("does not emit development usage warnings in a production Vite build", async () => {
  vi.stubEnv("DEV", false);
  const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
  const { OverlayActionBar } = await import(
    "../../../library/components/OverlayActionBar/versions/1.0.2-draft.1/OverlayActionBar"
  );

  renderWithProviders(
    <OverlayActionBar actions={fiveGermanOverlayActions as unknown as OverlayActionList} />,
  );

  expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
  expect(warning).not.toHaveBeenCalled();
});
