import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { renderWithProviders } from "../../test-utils";
import {
  GetBuiltinThemeResponseSchema,
  GetThemeFromScenarioResponseSchema,
  ListBuiltinThemesResponseSchema,
  ThemeSchema,
} from "@vrooli/proto-types/react-component-library/v1/themes/themes_pb";

vi.mock("../../api/themes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/themes")>();
  return {
    ...actual,
    themesClient: {
      listBuiltinThemes: vi.fn(),
      getBuiltinTheme: vi.fn(),
      getThemeFromScenario: vi.fn(),
    },
  };
});

import { ThemeSwitcher } from "./ThemeSwitcher";
import { themesClient } from "../../api/themes";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import type { ColorScheme } from "../../hooks/useDeviceFilters";

function makeBuiltinList() {
  return create(ListBuiltinThemesResponseSchema, {
    themes: [
      // Server still names these "Light"/"Dark"; the UI must re-label them
      // so a pack can never be confused for a color-scheme mode.
      create(ThemeSchema, {
        id: "light",
        name: "Light",
        tokens: { "--color-primary": "#3b82f6" },
        source: "builtin",
      }),
      create(ThemeSchema, {
        id: "dark",
        name: "Dark",
        tokens: { "--color-primary": "#1e3a8a" },
        source: "builtin",
      }),
    ],
  });
}

function makeBuiltinTheme() {
  return create(GetBuiltinThemeResponseSchema, {
    theme: create(ThemeSchema, {
      id: "light",
      name: "Light",
      tokens: { "--color-primary": "#3b82f6", "--rounded-md": "8px" },
      source: "builtin",
    }),
  });
}

function makeScenarioTheme() {
  return create(GetThemeFromScenarioResponseSchema, {
    theme: create(ThemeSchema, {
      id: "flow-verifier",
      name: "Flow verifier",
      tokens: { "--color-primary": "#ff00ff" },
      source: "scenario:flow-verifier",
    }),
  });
}

function Harness({
  postToFrames,
  setColorScheme = () => {},
  colorScheme = "system",
}: {
  postToFrames: (message: unknown) => void;
  setColorScheme?: (scheme: ColorScheme) => void;
  colorScheme?: ColorScheme;
}) {
  return (
    <ThemeSwitcher
      postToFrames={postToFrames}
      previewReady={true}
      colorScheme={colorScheme}
      setColorScheme={setColorScheme}
    />
  );
}

describe("ThemeSwitcher", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(themesClient.listBuiltinThemes).mockResolvedValue(makeBuiltinList());
    vi.mocked(themesClient.getBuiltinTheme).mockResolvedValue(makeBuiltinTheme());
    vi.mocked(themesClient.getThemeFromScenario).mockResolvedValue(makeScenarioTheme());
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("posts rcl-theme-apply with tokens when a built-in pack is selected", async () => {
    const postSpy = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness postToFrames={postSpy} />);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.components.themeSwitcher.select).querySelector(
          'option[value="builtin:light"]',
        ),
      ).not.toBeNull();
    });

    await user.selectOptions(
      screen.getByTestId(selectors.components.themeSwitcher.select),
      "builtin:light",
    );

    await waitFor(() => {
      expect(postSpy).toHaveBeenCalled();
    });
    const payload = postSpy.mock.calls.at(-1)![0] as { type: string; themeId: string; tokens: Record<string, string> };
    expect(payload.type).toBe("rcl-theme-apply");
    expect(payload.themeId).toBe("light");
    expect(payload.tokens["--color-primary"]).toBe("#3b82f6");
    expect(payload.tokens["--rounded-md"]).toBe("8px");
  });

  it("re-labels the built-in Light/Dark packs so they never read as modes", async () => {
    const postSpy = vi.fn();
    renderWithProviders(<Harness postToFrames={postSpy} />);

    const select = screen.getByTestId(selectors.components.themeSwitcher.select);
    await waitFor(() => {
      expect(select.querySelector('option[value="builtin:light"]')).not.toBeNull();
    });
    expect(select.querySelector('option[value="builtin:light"]')?.textContent).toBe("Slate");
    expect(select.querySelector('option[value="builtin:dark"]')?.textContent).toBe("Midnight");
  });

  it("single-owner: picking any token pack never changes the color-scheme mode", async () => {
    const postSpy = vi.fn();
    const setColorScheme = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <Harness postToFrames={postSpy} setColorScheme={setColorScheme} colorScheme="dark" />,
    );

    const select = screen.getByTestId(selectors.components.themeSwitcher.select);
    await waitFor(() => {
      expect(select.querySelector('option[value="builtin:light"]')).not.toBeNull();
    });

    await user.selectOptions(select, "builtin:light");
    await waitFor(() => {
      expect(postSpy).toHaveBeenCalled();
    });

    // The mode setter is never touched by a pack change...
    expect(setColorScheme).not.toHaveBeenCalled();
    // ...and no rcl-resolved-theme (the mode axis) is ever posted from here.
    for (const call of postSpy.mock.calls) {
      expect((call[0] as { type?: string }).type).not.toBe("rcl-resolved-theme");
    }
  });

  it("the segmented mode toggle is the single writer of light/dark", async () => {
    const postSpy = vi.fn();
    const setColorScheme = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <Harness postToFrames={postSpy} setColorScheme={setColorScheme} colorScheme="system" />,
    );

    await user.click(screen.getByTestId(selectors.components.themeSwitcher.modeDark));
    expect(setColorScheme).toHaveBeenCalledWith("dark");

    await user.click(screen.getByTestId(selectors.components.themeSwitcher.modeLight));
    expect(setColorScheme).toHaveBeenCalledWith("light");
  });

  it("applies a scenario DESIGN.md pack from the import disclosure", async () => {
    const postSpy = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness postToFrames={postSpy} />);

    await user.click(screen.getByTestId(selectors.components.themeSwitcher.importToggle));
    await user.type(
      screen.getByTestId(selectors.components.themeSwitcher.scenarioInput),
      "flow-verifier",
    );
    await user.click(
      screen.getByTestId(selectors.components.themeSwitcher.scenarioApply),
    );

    await waitFor(() => {
      expect(postSpy).toHaveBeenCalled();
    });
    const payload = postSpy.mock.calls.at(-1)![0] as { type: string; themeId: string; tokens: Record<string, string> };
    expect(payload.type).toBe("rcl-theme-apply");
    expect(payload.themeId).toBe("flow-verifier");
    expect(payload.tokens["--color-primary"]).toBe("#ff00ff");
  });
});
