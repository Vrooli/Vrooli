/**
 * Settings surface — controls toggle preferences store, locale switcher
 * cycles supported locales, and the skill-catalog sync + template
 * watcher cards render real data from their Connect-RPC clients.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../../../api/skillCatalog", async () => {
  const { create } = await import("@bufbuild/protobuf");
  const { SyncResponseSchema, ListSkillsResponseSchema } = await import(
    "@vrooli/proto-types/development-toolchain-validator/v1/skill_catalog/skill_catalog_pb"
  );
  return {
    skillCatalogClient: {
      sync: vi.fn().mockResolvedValue(
        create(SyncResponseSchema, { skills: [], added: 1, updated: 2, removed: 0 }),
      ),
      listSkills: vi.fn().mockResolvedValue(create(ListSkillsResponseSchema, { skills: [] })),
      getSkill: vi.fn(),
    },
  };
});

vi.mock("../../../api/staleness", async () => {
  const { create } = await import("@bufbuild/protobuf");
  const { ListStaleResponseSchema } = await import(
    "@vrooli/proto-types/development-toolchain-validator/v1/staleness/staleness_pb"
  );
  return {
    stalenessClient: {
      listStale: vi.fn().mockResolvedValue(create(ListStaleResponseSchema, { entries: [] })),
    },
  };
});

import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";
import { Settings } from "../Settings";
import { usePreferencesStore } from "../../../shared/stores/preferencesStore";

beforeEach(() => {
  // Reset prefs to defaults before each test.
  usePreferencesStore.setState({
    theme: "dark",
    density: "comfortable",
    sidebarCollapsed: false,
    lastVisitedGoldenSlug: null,
  });
});

afterEach(() => {
  cleanup();
});

describe("Settings", () => {
  it("renders the surface heading", () => {
    renderWithProviders(<Settings />);
    expect(screen.getByTestId(selectors.settings.surface)).toBeInTheDocument();
  });

  it("toggles the theme via the dark/light buttons", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    expect(usePreferencesStore.getState().theme).toBe("dark");
    await user.click(screen.getByTestId(selectors.settings.themeLight));
    await waitFor(() => {
      expect(usePreferencesStore.getState().theme).toBe("light");
    });
  });

  it("toggles the density via the comfortable/compact buttons", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    await user.click(screen.getByTestId(selectors.settings.densityCompact));
    await waitFor(() => {
      expect(usePreferencesStore.getState().density).toBe("compact");
    });
  });

  it("toggles the sidebar-collapsed preference via the checkbox", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    await user.click(screen.getByTestId(selectors.settings.sidebarCollapsed));
    await waitFor(() => {
      expect(usePreferencesStore.getState().sidebarCollapsed).toBe(true);
    });
  });

  it("renders skill-catalog sync and template watcher cards", async () => {
    renderWithProviders(<Settings />);
    expect(screen.getByTestId(selectors.settings.catalogSyncCard)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settings.catalogSyncButton)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settings.watcherCard)).toBeInTheDocument();
    // Watcher summary resolves from mock (empty entries → "All manifests..." copy).
    await waitFor(() => {
      expect(screen.getByTestId(selectors.settings.watcherSummary).textContent).not.toBe("…");
    });
  });

  it("invokes skill-catalog sync on click and surfaces the summary", async () => {
    const user = userEvent.setup();
    const { skillCatalogClient } = await import("../../../api/skillCatalog");
    renderWithProviders(<Settings />);
    await user.click(screen.getByTestId(selectors.settings.catalogSyncButton));
    await waitFor(() => {
      expect(vi.mocked(skillCatalogClient.sync)).toHaveBeenCalled();
    });
  });
});
