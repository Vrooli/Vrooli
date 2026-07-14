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

function makeBuiltinList() {
  return create(ListBuiltinThemesResponseSchema, {
    themes: [
      create(ThemeSchema, {
        id: "light",
        name: "Light",
        tokens: { "--color-primary": "#3b82f6" },
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

function Harness({ postToFrames }: { postToFrames: (message: unknown) => void }) {
  return <ThemeSwitcher postToFrames={postToFrames} previewReady={true} appResolvedTheme="light" />;
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

  it("posts rcl-theme-apply with tokens when a built-in is selected", async () => {
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

  it("applies scenario theme when Apply is clicked", async () => {
    const postSpy = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness postToFrames={postSpy} />);

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
    const payload = postSpy.mock.calls.at(-1)![0] as { themeId: string; tokens: Record<string, string> };
    expect(payload.themeId).toBe("flow-verifier");
    expect(payload.tokens["--color-primary"]).toBe("#ff00ff");
  });
});
