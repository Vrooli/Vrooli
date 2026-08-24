import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  ComponentVersionSchema,
  ListComponentVersionsResponseSchema,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { renderWithProviders } from "../../test-utils";
import {
  makeGetComponentContentResponse,
  makeGetComponentVersionContentResponse,
  makeUpdateComponentContentResponse,
} from "./mocks/factories";
import { makeComponentsMocks } from "./mocks/components";

const monacoMocks = vi.hoisted(() => ({
  setJavaScriptDiagnosticsOptions: vi.fn(),
  setTypeScriptDiagnosticsOptions: vi.fn(),
}));

// Monaco mounts a virtual DOM heavy enough to break jsdom (workers,
// canvas measurement). Swap it for a plain <textarea> stub so the
// editor's surrounding state-machine — load → edit → save — is the
// thing the test actually exercises.
vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: (props: {
    beforeMount?: (monaco: {
      languages: {
        typescript: {
          javascriptDefaults: {
            setDiagnosticsOptions: (options: unknown) => void;
          };
          typescriptDefaults: {
            setDiagnosticsOptions: (options: unknown) => void;
          };
        };
      };
    }) => void;
    value?: string;
    onChange?: (v: string | undefined) => void;
  }) => {
    const { beforeMount, value, onChange } = props;
    beforeMount?.({
      languages: {
        typescript: {
          javascriptDefaults: {
            setDiagnosticsOptions: monacoMocks.setJavaScriptDiagnosticsOptions,
          },
          typescriptDefaults: {
            setDiagnosticsOptions: monacoMocks.setTypeScriptDiagnosticsOptions,
          },
        },
      },
    });
    return (
      <textarea
        data-testid="monaco-stub"
        value={value ?? ""}
        onChange={(e) => onChange?.(e.target.value)}
      />
    );
  },
}));

vi.mock("../../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/components")>();
  return { ...actual, ...makeComponentsMocks() };
});

// The code panel's AdoptionFileTree resolves file placement over the adoptions
// transport. These editor tests exercise load/edit/save, not placement, so the
// resolver returns "no manifest" and the tree renders its flat fallback.
vi.mock("../../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/adoptions")>();
  return {
    ...actual,
    adoptionsClient: {
      ...actual.adoptionsClient,
      resolveAdoptionPath: vi.fn().mockResolvedValue({
        path: "",
        source: 0,
        slot: "",
        warnings: [],
        files: [],
        template: "",
        manifestResolved: false,
      }),
    },
  };
});

// ThemeSwitcher (TH-003) only mounts after previewReady; nothing to
// stub for these tests since they never reach that state.

import { ComponentEditor } from "./ComponentEditor";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ComponentEditor", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("fetches and renders source content with the library id in the title", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({
        content: "// real content\n",
        sha256: "abc123def456789",
      }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-1" libraryId="react-component-library:Button" onClose={() => {}} />,
    );

    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.editor.title).textContent).toContain(
        "react-component-library:Button",
      );
    });
    await waitFor(() => {
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toBe(
        "// real content\n",
      );
    });
  });

  it("keeps the default workspace to one pane and enables a resizable desktop split view", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// split", sha256: "sha-panels" }),
    );

    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    try {
      window.localStorage.removeItem("rcl.component-editor.split-view.v1");
      const user = userEvent.setup();
      renderWithProviders(
        <ComponentEditor
          id="cmp-panels"
          libraryId="lib:Panels"
          onClose={() => {}}
          activePane="details"
          metadataSlot={<p>Details content</p>}
        />,
      );

      await screen.findByText("Details content");
      expect(screen.getAllByTestId(selectors.components.editor.workspacePane)).toHaveLength(1);
      expect(screen.queryByTestId("components-editor-split-pane-switcher")).not.toBeInTheDocument();

      await user.click(screen.getByTestId("components-editor-split-view-toggle"));
      expect(screen.getAllByTestId(selectors.components.editor.workspacePane)).toHaveLength(2);
      expect(screen.getAllByTestId("components-editor-split-pane-switcher")).toHaveLength(2);
      expect(screen.getByTestId("components-editor-split-view-toggle")).toHaveAttribute(
        "aria-pressed",
        "true",
      );
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });

  it("provides a viewport-sized preview fallback when native fullscreen is unavailable", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// fullscreen", sha256: "sha-fullscreen" }),
    );

    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor
        id="cmp-fullscreen"
        libraryId="lib:Fullscreen"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    const stage = await screen.findByTestId(selectors.components.editor.previewStage);
    const toggle = screen.getByTestId(selectors.components.editor.previewStageFullscreen);
    expect(stage).toHaveAttribute("data-preview-fullscreen", "false");

    await user.click(toggle);
    expect(stage).toHaveAttribute("data-preview-fullscreen", "true");
    expect(toggle).toHaveAttribute("aria-pressed", "true");

    await user.click(toggle);
    expect(stage).toHaveAttribute("data-preview-fullscreen", "false");
    expect(toggle).toHaveAttribute("aria-pressed", "false");
  });

  it("opens the Files diff tab when a comparison arrives from Details", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// comparison source", sha256: "sha-diff" }),
    );
    try {
      const onCloseComparison = vi.fn();
      const user = userEvent.setup();
      renderWithProviders(
        <ComponentEditor
          id="cmp-diff"
          libraryId="lib:Diff"
          onClose={() => {}}
          comparison={{ fromLabel: "1.0.0", toLabel: "1.0.1", rows: [] }}
          onCloseComparison={onCloseComparison}
        />,
      );

      await screen.findByRole("tab", { name: "Diff: 1.0.0 → 1.0.1" });
      expect(screen.getByTestId(selectors.components.editor.workspacePane)).toHaveAttribute(
        "data-pane",
        "files",
      );
      await user.click(screen.getByTestId(selectors.components.editor.filesDiffClose));
      expect(onCloseComparison).toHaveBeenCalledTimes(1);
    } finally {
    }
  });

  it("keeps Monaco syntax diagnostics while disabling misleading semantic diagnostics", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "const x = 1;", sha256: "sha-diag" }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-diag" libraryId="lib:Diagnostics" onClose={() => {}} />,
    );

    await screen.findByTestId<HTMLTextAreaElement>("monaco-stub");

    const expected = {
      noSemanticValidation: true,
      noSyntaxValidation: false,
    };
    expect(monacoMocks.setTypeScriptDiagnosticsOptions).toHaveBeenCalledWith(expected);
    expect(monacoMocks.setJavaScriptDiagnosticsOptions).toHaveBeenCalledWith(expected);
  });

  it("loads a selected historical version read-only instead of current source", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentVersionContent).mockResolvedValueOnce(
      makeGetComponentVersionContentResponse({ content: "export const Historical = () => null;" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-1"
        libraryId="lib:Button"
        selectedVersion="1.0.0"
        onClose={() => {}}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toBe(
        "export const Historical = () => null;",
      );
    });
    expect(componentsClient.getComponentVersionContent).toHaveBeenCalledWith({
      componentId: "cmp-1",
      version: "1.0.0",
    });
  });

  it("disables Save until the buffer is dirty and forwards expectedSha256", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-original" }),
    );
    vi.mocked(componentsClient.updateComponentContent).mockResolvedValueOnce(
      makeUpdateComponentContentResponse({ sha256: "sha-new" }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ComponentEditor id="cmp-1" libraryId="lib:Card" onClose={() => {}} />);

    const saveBtn = await screen.findByTestId(selectors.components.editor.saveButton);
    await waitFor(() => {
      expect(saveBtn).toBeDisabled();
    });

    const stub = screen.getByTestId<HTMLTextAreaElement>("monaco-stub");
    fireEvent.change(stub, { target: { value: "v2" } });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.editor.dirtyBadge)).toBeInTheDocument();
    });
    expect(saveBtn).not.toBeDisabled();

    await user.click(saveBtn);

    await waitFor(() => {
      expect(componentsClient.updateComponentContent).toHaveBeenCalledTimes(1);
    });
    const call = vi.mocked(componentsClient.updateComponentContent).mock.calls[0]![0];
    expect(call).toMatchObject({
      id: "cmp-1",
      content: "v2",
      expectedSha256: "sha-original",
    });

    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.editor.savedToast)).toBeInTheDocument();
    });
  });

  it("renders the preview iframe pointing at the harness URL with a sha-based cache-buster", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-pre" }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-7" libraryId="lib:Iframe" onClose={() => {}} activePane="preview" />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    expect(frame.getAttribute("src")).toContain("/preview/cmp-7/harness.html");
    // The preview runtime is guarded by the same Cloudflare Access session as
    // the app. An opaque sandbox origin omits that session for module imports.
    expect(frame.getAttribute("sandbox")).toBe("allow-scripts allow-same-origin");
    await waitFor(() => {
      expect(frame.getAttribute("src")).toContain("v=sha-pre");
    });
  });

  it("renders the selected named state in the workbench canvas", async () => {
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-gallery" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-7",
          libraryId: "lib:Gallery",
          version: "1.0.0",
          schemaVersion: 2,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson:
            '[{"id":"primary","name":"Primary","description":"Primary action in its normal state.","args":{}},{"id":"disabled","name":"Disabled","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });

    renderWithProviders(
      <ComponentEditor
        id="cmp-7"
        libraryId="lib:Gallery"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    await screen.findAllByTestId(selectors.components.editor.storyPickerItem);
    const frames = await screen.findAllByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );

    expect(screen.getByTestId(selectors.components.editor.gallery)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(1);
    expect(screen.getByTestId(selectors.components.editor.storyDescription)).toHaveTextContent(
      "Primary action in its normal state.",
    );
    expect(screen.getAllByTestId(selectors.components.editor.storyPickerItem)[0]).toHaveAttribute(
      "title",
      "Primary action in its normal state.",
    );
    expect(frames).toHaveLength(1);
    expect(frames[0]?.getAttribute("src")).toContain("story=primary");
    expect(vi.mocked(listComponentStories)).toHaveBeenCalledWith({
      componentId: "cmp-7",
      version: "1.0.0",
      limit: 1,
    });

    // Each gallery frame posts the resolved theme once it loads.
    if (frames[0]) fireEvent.load(frames[0]);
    expect(screen.queryByTestId(selectors.components.editor.previewError)).not.toBeInTheDocument();
  });

  it("uses the manifest latest version when history lists a draft first", async () => {
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "latest-source", sha256: "sha-latest" }),
    );
    vi.mocked(componentsClient.listComponentVersions).mockResolvedValueOnce(
      create(ListComponentVersionsResponseSchema, {
        versions: [
          create(ComponentVersionSchema, { version: "0.3.2-draft.1" }),
          create(ComponentVersionSchema, { version: "0.3.2" }),
        ],
      }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "latest-story",
          componentId: "cmp-markdown",
          libraryId: "react-component-library:markdown-renderer",
          version: "0.3.2",
          schemaVersion: 2,
          kind: "component",
          title: "",
          argsJson: "{}",
          environmentJson: "{}",
          storiesJson: '[{"id":"default","name":"Default","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });

    renderWithProviders(
      <ComponentEditor
        id="cmp-markdown"
        libraryId="react-component-library:markdown-renderer"
        latestVersion="0.3.2"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    await waitFor(() => {
      expect(vi.mocked(listComponentStories)).toHaveBeenCalledWith({
        componentId: "cmp-markdown",
        version: "0.3.2",
        limit: 1,
      });
    });
    expect(vi.mocked(listComponentStories)).not.toHaveBeenCalledWith({
      componentId: "cmp-markdown",
      version: "0.3.2-draft.1",
      limit: 1,
    });
  });

  it("loads the selected version-local companion file instead of a shared runtime alias", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent)
      .mockResolvedValueOnce(
        makeGetComponentContentResponse({
          content: 'export { FilterBar } from "./FilterBar";\n',
          sha256: "sha-entry",
        }),
      )
      .mockResolvedValueOnce(
        makeGetComponentContentResponse({
          content: "export function InlineCode() { return null; }\n",
          sha256: "sha-companion",
        }),
      );
    vi.mocked(componentsClient.getComponentVersionContent)
      .mockResolvedValueOnce(
        makeGetComponentVersionContentResponse({
          content: '{"stories":[{"id":"default","name":"Default","args":{}}]}',
          version: { contentSha256: "sha-story" },
        }),
      )
      .mockResolvedValueOnce(
        makeGetComponentVersionContentResponse({
          content: "export function InlineCodeHarness() { return <InlineCode />; }\n",
          version: { contentSha256: "sha-harness" },
        }),
      )
      .mockResolvedValueOnce(
        makeGetComponentVersionContentResponse({
          content: '{"contract":{"kind":"rcl-component-experience-contract"}}',
          version: { contentSha256: "sha-contract" },
        }),
      );
    vi.mocked(componentsClient.listComponentVersions).mockResolvedValueOnce(
      create(ListComponentVersionsResponseSchema, {
        versions: [
          create(ComponentVersionSchema, {
            version: "1.0.0",
            files: [
              { path: "FilterBar.tsx", isEntry: true },
              { path: "InlineCode.tsx", isEntry: false },
              { path: "experience-contract.json", isEntry: false },
              { path: "story.json", isEntry: false },
              { path: "story.tsx", isEntry: false },
            ],
          }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor
        id="cmp-files"
        libraryId="react-component-library:FilterBar"
        latestVersion="1.0.0"
        onClose={() => {}}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain("FilterBar");
    });
    await user.click(screen.getByRole("tab", { name: "InlineCode.tsx" }));

    await waitFor(() => {
      expect(componentsClient.getComponentContent).toHaveBeenLastCalledWith({
        id: "cmp-files",
        path: "InlineCode.tsx",
      });
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain("InlineCode");
    });

    await user.click(screen.getByRole("tab", { name: "story.json" }));

    await waitFor(() => {
      expect(componentsClient.getComponentVersionContent).toHaveBeenLastCalledWith({
        componentId: "cmp-files",
        version: "1.0.0",
        path: "story.json",
      });
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain("default");
      expect(screen.getByTestId(selectors.components.editor.saveButton)).toBeDisabled();
    });

    await user.click(screen.getByRole("tab", { name: "story.tsx" }));

    await waitFor(() => {
      expect(componentsClient.getComponentVersionContent).toHaveBeenLastCalledWith({
        componentId: "cmp-files",
        version: "1.0.0",
        path: "story.tsx",
      });
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain("InlineCode");
      expect(screen.getByTestId(selectors.components.editor.saveButton)).toBeDisabled();
    });

    await user.click(screen.getByRole("tab", { name: "experience-contract.json" }));

    await waitFor(() => {
      expect(componentsClient.getComponentVersionContent).toHaveBeenLastCalledWith({
        componentId: "cmp-files",
        version: "1.0.0",
        path: "experience-contract.json",
      });
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain(
        "rcl-component-experience-contract",
      );
      expect(screen.getByTestId(selectors.components.editor.saveButton)).toBeDisabled();
      expect(screen.getByTestId("components-editor-pretty-json")).toHaveAttribute(
        "aria-pressed",
        "true",
      );
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toContain("\n");
    });

    await user.click(screen.getByTestId("components-editor-pretty-json"));
    expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toBe(
      '{"contract":{"kind":"rcl-component-experience-contract"}}',
    );
  });

  it("shows a structural source skeleton while the file request is pending", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(
      <ComponentEditor
        id="cmp-loading"
        libraryId="react-component-library:Button"
        onClose={() => {}}
      />,
    );

    expect(await screen.findByTestId(selectors.components.editor.loading)).toHaveAttribute(
      "role",
      "status",
    );
    expect(
      screen.getByTestId(selectors.components.editor.loading).querySelectorAll(".animate-pulse")
        .length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText("Loading source...")).not.toBeInTheDocument();
  });

  it("shows preview events newest-first and clears the active story log", async () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-events" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-events",
          libraryId: "lib:Events",
          version: "1.0.0",
          schemaVersion: 2,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson: '[{"id":"primary","name":"Primary","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });
    try {
      renderWithProviders(
        <ComponentEditor
          id="cmp-events"
          libraryId="lib:Events"
          onClose={() => {}}
          activePane="preview"
        />,
      );
      await screen.findByRole("button", { name: "Primary" });
      await screen.findByTestId(selectors.components.editor.previewToolsPanel);
      const diagnostics = await screen.findByTestId(
        selectors.components.editor.previewDiagnostics,
      );
      expect(diagnostics).toHaveTextContent('"componentId": "cmp-events"');
      expect(diagnostics).toHaveTextContent('"storyId": "primary"');
      const writeText = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: { writeText },
      });
      fireEvent.click(screen.getByTestId(selectors.components.editor.previewDiagnosticsCopy));
      expect(writeText).toHaveBeenCalledWith(expect.stringContaining('"kit": "vrooli-default"'));
      const frame = await screen.findByTestId<HTMLIFrameElement>(
        selectors.components.editor.previewFrame,
      );
      window.dispatchEvent(
        new MessageEvent("message", {
          data: {
            type: "rcl-preview-event",
            id: "cmp-events",
            story: "primary",
            version: "1.0.0",
            name: "change",
            args: ["#51cf66"],
            ts: 1,
          },
          source: frame.contentWindow,
        }),
      );
      expect(
        (await screen.findAllByTestId(selectors.components.editor.previewEventItem))[0],
      ).toHaveTextContent('change("#51cf66")');
      await userEvent
        .setup()
        .click(screen.getByTestId(selectors.components.editor.previewEventsClear));
      expect(
        screen.getByTestId(selectors.components.editor.previewEventsEmpty),
      ).toBeInTheDocument();
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });

  it("caps the active preview event stream at 200 entries", async () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-event-cap" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-event-cap",
          libraryId: "lib:Events",
          version: "1.0.0",
          schemaVersion: 2,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson: '[{"id":"primary","name":"Primary","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });
    try {
      renderWithProviders(
        <ComponentEditor
          id="cmp-event-cap"
          libraryId="lib:Events"
          onClose={() => {}}
          activePane="preview"
        />,
      );
      await screen.findByRole("button", { name: "Primary" });
      await screen.findByTestId(selectors.components.editor.previewToolsPanel);
      const frame = await screen.findByTestId<HTMLIFrameElement>(
        selectors.components.editor.previewFrame,
      );
      for (let index = 0; index <= 200; index++) {
        window.dispatchEvent(
          new MessageEvent("message", {
            data: {
              type: "rcl-preview-event",
              id: "cmp-event-cap",
              story: "primary",
              version: "1.0.0",
              name: `event-${index}`,
              args: [],
              ts: index,
            },
            source: frame.contentWindow,
          }),
        );
      }
      const eventItems = await screen.findAllByTestId(selectors.components.editor.previewEventItem);
      expect(eventItems).toHaveLength(200);
      expect(eventItems[0]).toHaveTextContent("event-200()");
      expect(eventItems.at(-1)).toHaveTextContent("event-1()");
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });

  it("exposes the preview gallery as a semantic readiness surface", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-surface" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-surface"
        libraryId="lib:Surface"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    await screen.findByTestId(selectors.components.editor.workspacePane);
    const surface = document.querySelector('[data-experience-surface="component-preview"]');
    expect(surface).not.toBeNull();
    expect(surface).toHaveAttribute("data-experience-surface", "component-preview");
    expect(surface).toHaveAttribute("data-experience-state", "loading");

    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    await waitFor(() => {
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { type: "preview-ready", id: "cmp-surface", example: "", version: "" },
          source: frame.contentWindow,
        }),
      );
      expect(surface).toHaveAttribute("data-experience-state", "ready");
    });
  });

  it("keeps frame experiments temporary until the author explicitly saves them", async () => {
    const { componentsClient, listComponentStories, listPreviewFrames, persistPreviewFrame } =
      await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-frame-save" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-frame-save",
          libraryId: "lib:FrameSave",
          version: "1.0.0",
          schemaVersion: 3,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson: '[{"id":"primary","name":"Primary","args":{}}]',
          contractJson:
            '{"schemaVersion":3,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"primary","name":"Primary","args":{}}]}',
          sourcePath: "story.json",
        },
      ],
    });
    vi.mocked(listPreviewFrames).mockResolvedValueOnce({
      candidates: [
        {
          asset: "navigation.page",
          version: "1.0.0",
          region: "content",
          capability: "",
          fixture: "",
          label: "navigation.page",
          compatible: true,
          diagnosticCode: "",
          diagnostic: "",
        },
      ],
    });
    vi.mocked(persistPreviewFrame).mockResolvedValueOnce({
      componentId: "cmp-frame-save",
      version: "1.0.1-draft.1",
      storyId: "primary",
      storyJson: "{}",
      sourcePath: "story.json",
    });

    renderWithProviders(
      <ComponentEditor
        id="cmp-frame-save"
        libraryId="lib:FrameSave"
        latestVersion="1.0.0"
        selectedStory="primary"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    const picker = await screen.findByTestId<HTMLSelectElement>("components-editor-frame-picker");
    expect(persistPreviewFrame).not.toHaveBeenCalled();
    await userEvent.selectOptions(picker, "navigation.page");
    const save = await screen.findByTestId("components-editor-frame-save");
    await userEvent.click(save);
    await waitFor(() => expect(persistPreviewFrame).toHaveBeenCalledTimes(1));
    expect(persistPreviewFrame).toHaveBeenCalledWith(
      expect.objectContaining({
        componentId: "cmp-frame-save",
        storyId: "primary",
        asset: "navigation.page",
        frameVersion: "1.0.0",
      }),
    );
  });

  it("sends a temporary props override to the visible selected state", async () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-props" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-props",
          libraryId: "lib:Props",
          version: "1.0.0",
          schemaVersion: 1,
          kind: "component",
          title: "",
          argsJson: '{"fields":[{"path":"title","kind":"text"}]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson:
            '[{"id":"primary","name":"Primary","args":{"title":"Short"}},{"id":"secondary","name":"Secondary","args":{"title":"Other"}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor
        id="cmp-props"
        libraryId="lib:Props"
        onClose={() => {}}
        activePane="preview"
      />,
    );
    await screen.findByRole("button", { name: "Primary" });
    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    const firstPost = vi.spyOn(frame.contentWindow!, "postMessage");
    const title = await screen.findByLabelText("title");
    fireEvent.change(title, { target: { value: "A deliberately much longer value" } });
    await waitFor(() => expect(title).toHaveValue("A deliberately much longer value"));
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(firstPost).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "rcl-preview-props-override",
        componentId: "cmp-props",
        story: "primary",
        props: { title: "A deliberately much longer value" },
      }),
      "*",
    );
    window.matchMedia = originalMatchMedia;
  });

  it("keeps state, canvas, and controls together without a playground mode", async () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-playground" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-playground",
          libraryId: "lib:Playground",
          version: "1.0.0",
          schemaVersion: 1,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson:
            '[{"id":"primary","name":"Primary","args":{}},{"id":"secondary","name":"Secondary","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor
        id="cmp-playground"
        libraryId="lib:Playground"
        onClose={() => {}}
        activePane="preview"
      />,
    );
    await screen.findByRole("button", { name: "Primary" });
    expect(await screen.findAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(1);
    const toolsToggle = screen.getByTestId(selectors.components.editor.previewToolsToggle);
    expect(toolsToggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toolsToggle);
    expect(toolsToggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId(selectors.components.editor.propsPanel)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Secondary" }));
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(1);
    expect(screen.getByRole("button", { name: "Secondary" })).toHaveAttribute(
      "aria-current",
      "true",
    );
    expect(screen.getByTestId(selectors.components.editor.propsPanel)).toBeInTheDocument();
    window.matchMedia = originalMatchMedia;
  });

  it("does not mount the resizable preview tools panel on a narrow viewport", async () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-mobile-tools" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-mobile-tools",
          libraryId: "lib:Mobile",
          version: "1.0.0",
          schemaVersion: 1,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson: '[{"id":"primary","name":"Primary","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });

    try {
      renderWithProviders(
        <ComponentEditor
          id="cmp-mobile-tools"
          libraryId="lib:Mobile"
          onClose={() => {}}
          activePane="preview"
        />,
      );
      await screen.findByRole("button", { name: "Primary" });
      expect(
        screen.queryByTestId(selectors.components.editor.previewToolsPanel),
      ).not.toBeInTheDocument();
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });

  it("reloads the iframe when save returns a new sha256", async () => {
    const { componentsClient } = await import("../../api/components");
    // First GET returns the pre-save sha; post-save invalidation triggers
    // a refetch that returns the new sha — both flows feed baselineSha.
    vi.mocked(componentsClient.getComponentContent)
      .mockResolvedValueOnce(makeGetComponentContentResponse({ content: "v1", sha256: "sha-pre" }))
      .mockResolvedValue(makeGetComponentContentResponse({ content: "v2", sha256: "sha-post" }));
    vi.mocked(componentsClient.updateComponentContent).mockResolvedValueOnce(
      makeUpdateComponentContentResponse({ sha256: "sha-post" }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-8" libraryId="lib:Reload" onClose={() => {}} activePane="preview" />,
    );

    await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    await waitFor(() => {
      expect(
        screen
          .getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame)
          .getAttribute("src"),
      ).toContain("v=sha-pre");
    });

    // Preview refreshes whenever its content identity changes. The editor
    // source itself is now a separate Files tab, so this preview-only test
    // verifies the SHA contract without requiring a second pane.
    expect(
      screen
        .getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame)
        .getAttribute("src"),
    ).toContain("v=sha-pre");
  });

  it("isolates a harness failure to its specimen", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-preview" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-9"
        libraryId="lib:BrokenPreview"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );

    window.dispatchEvent(
      new MessageEvent("message", {
        data: {
          type: "preview-error",
          id: "cmp-9",
          example: "",
          version: "",
          message: "preview: render failed - boom",
        },
        source: frame.contentWindow,
      }),
    );

    const error = await screen.findByTestId(selectors.components.editor.specimenError);
    expect(error.textContent).toContain("preview: render failed - boom");
  });

  it("reports loading, ready, and partial preview states to the asset-page readiness contract", async () => {
    const states: string[] = [];
    renderWithProviders(
      <ComponentEditor
        id="cmp-state"
        libraryId="lib:State"
        onClose={() => {}}
        activePane="preview"
        onPreviewExperienceStateChange={(state) => states.push(state)}
      />,
    );
    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    expect(states).toContain("loading");
    window.dispatchEvent(
      new MessageEvent("message", {
        data: {
          type: "preview-ready",
          id: "cmp-state",
          story: "__default__",
          version: "__current__",
        },
        source: frame.contentWindow,
      }),
    );
    await waitFor(() => expect(states).toContain("ready"));
    window.dispatchEvent(
      new MessageEvent("message", {
        data: {
          type: "preview-error",
          id: "cmp-state",
          story: "__default__",
          version: "__current__",
          message: "failed",
        },
        source: frame.contentWindow,
      }),
    );
    await waitFor(() => expect(states).toContain("partial"));
  });

  it("compares two canvas cases without switching to focus mode", async () => {
    const boundsSpy = vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
      x: 0,
      y: 0,
      width: 640,
      height: 420,
      top: 0,
      right: 640,
      bottom: 420,
      left: 0,
      toJSON: () => ({}),
    });
    const { componentsClient, listComponentStories } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-compare" }),
    );
    vi.mocked(listComponentStories).mockResolvedValueOnce({
      stories: [
        {
          id: "contract",
          componentId: "cmp-compare",
          libraryId: "lib:Compare",
          version: "1.0.0",
          schemaVersion: 1,
          kind: "component",
          title: "",
          argsJson: '{"fields":[]}',
          environmentJson: '{"fixtures":[]}',
          storiesJson:
            '[{"id":"primary","name":"Primary","args":{}},{"id":"disabled","name":"Disabled","args":{}},{"id":"loading","name":"Loading","args":{}},{"id":"error","name":"Error","args":{}},{"id":"success","name":"Success","args":{}}]',
          contractJson: "{}",
          sourcePath: "story.json",
        },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor
        id="cmp-compare"
        libraryId="lib:Compare"
        onClose={() => {}}
        activePane="preview"
      />,
    );
    await screen.findByRole("button", { name: "Primary" });
    await user.click(screen.getByTestId("components-editor-stage-mode"));
    const comparisonButtons = await screen.findAllByTestId(
      selectors.components.editor.exampleCompare,
    );
    expect(screen.queryAllByTestId(selectors.components.editor.storyPickerItem)).toHaveLength(0);
    expect(comparisonButtons[0]).toHaveAccessibleName("Compare Primary");
    await user.click(comparisonButtons[0]!);
    expect(screen.getByTestId(selectors.components.editor.gallery)).toHaveAttribute(
      "data-preview-stage-mode",
      "false",
    );
    const updatedComparisonButtons = await screen.findAllByTestId(
      selectors.components.editor.exampleCompare,
    );
    await user.click(updatedComparisonButtons[1]!);
    expect(await screen.findAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(2);
    await user.click(screen.getByTestId(selectors.components.editor.storySheetAll));
    expect(await screen.findAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(4);
    await user.click(screen.getByTestId(selectors.components.editor.comparisonClear));
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(5);
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)[0]).toHaveClass(
      "resize",
      "overflow-auto",
      "min-h-[18rem]",
      "max-h-[80vh]",
    );
    expect(
      screen.getAllByTestId(selectors.components.editor.exampleDimensions)[0],
    ).toHaveTextContent("640 × 420");

    const caseTitles = () =>
      screen
        .getAllByTestId(selectors.components.editor.exampleTitle)
        .map((title) => title.textContent);
    const primaryMoveHandle = screen.getByRole("button", {
      name: "Move Primary. Use arrow keys to reorder.",
    });
    fireEvent.keyDown(primaryMoveHandle, { key: "ArrowRight" });
    expect(caseTitles()).toEqual(["Disabled", "Primary", "Loading", "Error", "Success"]);

    let draggedIdentity = "";
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn((_type: string, value: string) => {
        draggedIdentity = value;
      }),
      getData: vi.fn(() => draggedIdentity),
    } as unknown as DataTransfer;
    const disabledMoveHandle = screen.getByRole("button", {
      name: "Move Disabled. Use arrow keys to reorder.",
    });
    const primaryCard = screen
      .getAllByTestId(selectors.components.editor.exampleTitle)[1]
      ?.closest("section");
    expect(primaryCard).not.toBeNull();
    vi.spyOn(primaryCard!, "getBoundingClientRect").mockReturnValue({
      x: 0,
      y: 0,
      width: 100,
      height: 100,
      top: 0,
      right: 100,
      bottom: 100,
      left: 0,
      toJSON: () => ({}),
    });
    fireEvent.dragStart(disabledMoveHandle, { dataTransfer });
    expect(dataTransfer.setData).toHaveBeenCalled();
    expect(draggedIdentity).toBe(disabledMoveHandle.closest("section")?.dataset.specimen);
    const canvasSurface = screen.getByLabelText("Preview canvas. Drag to pan, scroll to zoom.");
    fireEvent.dragOver(canvasSurface, { dataTransfer, clientX: 900, clientY: 900 });
    expect(primaryCard).toHaveAttribute("data-drop-placement", "after");
    expect(disabledMoveHandle.closest("section")).toHaveClass(
      "opacity-40",
      "scale-[0.98]",
      "border-dashed",
    );
    fireEvent.drop(canvasSurface, { dataTransfer, clientX: 900, clientY: 900 });
    expect(dataTransfer.getData).toHaveBeenCalled();
    expect(caseTitles()).toEqual(["Primary", "Disabled", "Loading", "Error", "Success"]);

    await user.click(screen.getByRole("button", { name: "Focus Disabled" }));
    expect(screen.getByTestId(selectors.components.editor.gallery)).toHaveAttribute(
      "data-preview-stage-mode",
      "true",
    );
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(1);
    boundsSpy.mockRestore();
  });

  it("marks the preview ready and posts the resolved theme on the desktop side-by-side layout", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// ready", sha256: "sha-ready" }),
    );

    const originalMatchMedia = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    try {
      renderWithProviders(
        <ComponentEditor
          id="cmp-ready"
          libraryId="lib:Ready"
          onClose={() => {}}
          activePane="details"
        />,
      );

      await userEvent.setup().click(screen.getByTestId("components-editor-split-view-toggle"));
      // The split workspace mounts Preview alongside Details. The Rendered
      // badge only shows after the harness announces preview-ready.
      await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
      expect(
        screen.queryByTestId(selectors.components.editor.previewBadge),
      ).not.toBeInTheDocument();

      // The first-fetch baselineSha effect resets the ready set right after
      // mount, so a single announce can race. Re-announce until the badge
      // sticks — exactly what a live harness does on each (re)load.
      const frame = screen.getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
      await waitFor(() => {
        window.dispatchEvent(
          new MessageEvent("message", {
            data: { type: "preview-ready", id: "cmp-ready", example: "", version: "" },
            source: frame.contentWindow,
          }),
        );
        expect(screen.getByTestId(selectors.components.editor.previewBadge)).toBeInTheDocument();
      });
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });

  it("uses a single unheaded pane until split view is explicitly enabled", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// modes", sha256: "sha-modes" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-modes"
        libraryId="lib:Modes"
        onClose={() => {}}
        activePane="details"
        metadataSlot={<p>Details</p>}
      />,
    );

    expect(await screen.findByText("Details")).toBeInTheDocument();
    expect(screen.queryByTestId("components-editor-split-pane-switcher")).not.toBeInTheDocument();
  });

  it("posts the resolved theme to the preview iframe once it loads", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// iframe", sha256: "sha-iframe" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-iframe"
        libraryId="lib:Iframe"
        onClose={() => {}}
        activePane="preview"
      />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );

    // The onLoad handler posts the resolved theme into the frame; firing it
    // exercises that path without needing a live preview runtime. No error
    // surfaces on a clean load.
    fireEvent.load(frame);
    expect(screen.queryByTestId(selectors.components.editor.previewError)).not.toBeInTheDocument();
  });

  it("renders structural previews as one declared-viewport specimen stage", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// stage", sha256: "sha-stage" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-stage"
        libraryId="lib:PageFrame"
        onClose={() => {}}
        activePane="preview"
        stageMode
      />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    fireEvent.load(frame);
    const card = await screen.findByTestId(selectors.components.editor.exampleCard);
    expect(screen.getByTestId("components-editor-stage-mode")).toHaveTextContent("Focus");
    expect(screen.getByTestId(selectors.components.editor.gallery)).toHaveAttribute(
      "data-preview-stage-mode",
      "true",
    );
    expect(card).toContainElement(frame);
    expect(frame).not.toHaveClass("h-[20rem]");
    expect(frame.style.width).toBe("1280px");
    expect(frame.style.height).toBe("720px");
  });

  it("keeps gallery previews fluid instead of sizing the gallery as a device", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// gallery", sha256: "sha-gallery-fluid" }),
    );

    renderWithProviders(
      <ComponentEditor
        id="cmp-gallery-fluid"
        libraryId="lib:Gallery"
        onClose={() => {}}
        activePane="preview"
        stageMode={false}
      />,
    );

    const gallery = await screen.findByTestId(selectors.components.editor.gallery);
    const frame = await screen.findByTestId<HTMLIFrameElement>(
      selectors.components.editor.previewFrame,
    );
    expect(gallery.style.width).toBe("");
    expect(gallery.style.height).toBe("");
    expect(frame.style.width).toBe("100%");
    expect(frame.style.height).toBe("100%");
  });

  it("invokes onClose when the Back-to-list button is clicked", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse(),
    );

    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<ComponentEditor id="cmp-1" libraryId="lib:X" onClose={onClose} />);

    const closeBtn = await screen.findByTestId(selectors.components.editor.closeButton);
    await user.click(closeBtn);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
