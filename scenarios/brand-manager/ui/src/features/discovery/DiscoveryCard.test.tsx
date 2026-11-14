/**
 * DiscoveryCard tests — focused on the discovery-card surface only. Renders
 * <DiscoveryCard /> directly so failures point at discovery-feature behaviour,
 * not shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeDiscoveryResult, makeDiscoverySource, makeDraftBrand, makeDraftColors, makeDraftIdentity } from "./mocks/factories";
import { makeDiscoveryMocks } from "./mocks/discovery";

vi.mock("../../api/discovery", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/discovery")>();
  return { ...actual, ...makeDiscoveryMocks() };
});

import { DiscoveryCard } from "./DiscoveryCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("DiscoveryCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("keeps the scan button disabled until a scenario is entered", async () => {
    renderWithProviders(<DiscoveryCard />);
    const button = screen.getByTestId(selectors.discovery.scanButton);
    expect(button).toBeDisabled();

    const user = userEvent.setup();
    await user.type(screen.getByTestId(selectors.discovery.scenarioInput), "web-console");
    expect(button).toBeEnabled();
  });

  it("scans and renders sources, the draft brand, and suggestions", async () => {
    const { discoverScenario } = await import("../../api/discovery");
    vi.mocked(discoverScenario).mockResolvedValueOnce(
      makeDiscoveryResult({
        scenario: "web-console",
        confidence: 0.6,
        sources: [makeDiscoverySource({ file: ".vrooli/branding.json", type: "branding_json" })],
        draftBrand: makeDraftBrand({
          identity: makeDraftIdentity({ displayName: "Acme" }),
          colors: makeDraftColors({ primary: "#112233" }),
        }),
        suggestions: ["No logo found. Consider uploading a brand logo."],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<DiscoveryCard />);
    await user.type(screen.getByTestId(selectors.discovery.scenarioInput), "web-console");
    await user.click(screen.getByTestId(selectors.discovery.scanButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.discovery.results)).toBeInTheDocument();
    });
    const sources = screen.getByTestId(selectors.discovery.sourcesList);
    expect(sources.textContent).toContain(".vrooli/branding.json");
    expect(screen.getByTestId(selectors.discovery.draft).textContent).toContain("Acme");
    expect(screen.getByTestId(selectors.discovery.suggestionsList).textContent).toContain("logo");
  });

  it("shows the empty state when a scan finds no sources", async () => {
    const { discoverScenario } = await import("../../api/discovery");
    vi.mocked(discoverScenario).mockResolvedValueOnce(
      makeDiscoveryResult({ scenario: "blank", suggestions: ["No color system found. Consider defining brand colors."] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<DiscoveryCard />);
    await user.type(screen.getByTestId(selectors.discovery.scenarioInput), "blank");
    await user.click(screen.getByTestId(selectors.discovery.scanButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.discovery.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.discovery.sourcesList)).not.toBeInTheDocument();
  });
});
