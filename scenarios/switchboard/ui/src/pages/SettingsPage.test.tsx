import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { SettingsPage } from "./SettingsPage";

describe("SettingsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("lists the loaded descriptors and the install facts", async () => {
    stubConsoleFetch(defaultRoutes());
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-descriptor-region")).toHaveAttribute("data-experience-state", "ready"));
    expect(screen.getByTestId("settings-descriptor-status")).toBeInTheDocument();
    expect(screen.getByTestId("settings-theme")).toBeInTheDocument();
    expect(screen.getByTestId("settings-metering-region")).toBeInTheDocument();
  });
});
