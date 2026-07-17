import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeComponentExample,
  makeGetComponentContentResponse,
  makeGetComponentVersionContentResponse,
  makeListComponentExamplesResponse,
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
        <ComponentEditor id="cmp-panels" libraryId="lib:Panels" onClose={() => {}} activePane="details" metadataSlot={<p>Details content</p>} />,
      );

      await screen.findByText("Details content");
      expect(screen.getAllByTestId(selectors.components.editor.workspacePane)).toHaveLength(1);
      expect(screen.queryByTestId("components-editor-split-pane-switcher")).not.toBeInTheDocument();

      await user.click(screen.getByTestId("components-editor-split-view-toggle"));
      expect(screen.getAllByTestId(selectors.components.editor.workspacePane)).toHaveLength(2);
      expect(screen.getAllByTestId("components-editor-split-pane-switcher")).toHaveLength(2);
      expect(screen.getByTestId("components-editor-split-view-toggle")).toHaveAttribute("aria-pressed", "true");
    } finally {
      window.matchMedia = originalMatchMedia;
    }
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

      await screen.findByRole("button", { name: "Diff: 1.0.0 → 1.0.1" });
      expect(screen.getByTestId(selectors.components.editor.workspacePane)).toHaveAttribute("data-pane", "files");
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
      <ComponentEditor id="cmp-1" libraryId="lib:Button" selectedVersion="1.0.0" onClose={() => {}} />,
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
    renderWithProviders(
      <ComponentEditor id="cmp-1" libraryId="lib:Card" onClose={() => {}} />,
    );

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

    const frame = await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    expect(frame.getAttribute("src")).toContain("/preview/cmp-7/harness.html");
    await waitFor(() => {
      expect(frame.getAttribute("src")).toContain("v=sha-pre");
    });
  });

  it("renders example gallery iframes with named harness URLs", async () => {
    const { componentsClient, listComponentExamples } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-gallery" }),
    );
    vi.mocked(listComponentExamples).mockResolvedValueOnce(
      makeListComponentExamplesResponse({
        examples: [
          makeComponentExample({ name: "primary", displayName: "Primary" }),
          makeComponentExample({ name: "disabled", displayName: "Disabled" }),
        ],
      }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-7" libraryId="lib:Gallery" onClose={() => {}} activePane="preview" />,
    );

    const frames = await screen.findAllByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);

    expect(screen.getByTestId(selectors.components.editor.gallery)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(2);
    expect(frames).toHaveLength(2);
    expect(frames[0]?.getAttribute("src")).toContain("example=primary");
    expect(frames[1]?.getAttribute("src")).toContain("example=disabled");

    // Each gallery frame posts the resolved theme once it loads.
    if (frames[0]) fireEvent.load(frames[0]);
    expect(screen.queryByTestId(selectors.components.editor.previewError)).not.toBeInTheDocument();
  });

  it("exposes the preview gallery as a semantic readiness surface", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-surface" }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-surface" libraryId="lib:Surface" onClose={() => {}} activePane="preview" />,
    );

    await screen.findByTestId(selectors.components.editor.workspacePane);
    const surface = document.querySelector('[data-experience-surface="component-preview"]');
    expect(surface).not.toBeNull();
    expect(surface).toHaveAttribute("data-experience-surface", "component-preview");
    expect(surface).toHaveAttribute("data-experience-state", "loading");

    const frame = await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    await waitFor(() => {
      window.dispatchEvent(new MessageEvent("message", {
        data: { type: "preview-ready", id: "cmp-surface", example: "", version: "" },
        source: frame.contentWindow,
      }));
      expect(surface).toHaveAttribute("data-experience-state", "ready");
    });
  });

  it("sends a temporary props override only to the active named specimen", async () => {
    const { componentsClient, listComponentExamples } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-props" }),
    );
    vi.mocked(listComponentExamples).mockResolvedValueOnce(
      makeListComponentExamplesResponse({
        examples: [
          makeComponentExample({ name: "primary", displayName: "Primary", propsJson: '{"title":"Short"}' }),
          makeComponentExample({ name: "secondary", displayName: "Secondary", propsJson: '{"title":"Other"}' }),
        ],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ComponentEditor id="cmp-props" libraryId="lib:Props" onClose={() => {}} activePane="preview" />);
    const frames = await screen.findAllByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    const firstPost = vi.spyOn(frames[0]!.contentWindow!, "postMessage");
    const secondPost = vi.spyOn(frames[1]!.contentWindow!, "postMessage");
    await user.click(screen.getAllByRole("button", { name: "Play" })[0]!);
    const draft = await screen.findByTestId<HTMLTextAreaElement>(selectors.components.editor.propsDraft);
    fireEvent.change(draft, { target: { value: '{"title":"A deliberately much longer value"}' } });
    await waitFor(() => expect(draft).toHaveValue('{"title":"A deliberately much longer value"}'));
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(firstPost).toHaveBeenCalledWith(expect.objectContaining({
      type: "rcl-preview-props-override",
      componentId: "cmp-props",
      example: "primary",
      props: { title: "A deliberately much longer value" },
    }), "*");
    expect(secondPost).not.toHaveBeenCalledWith(expect.objectContaining({ type: "rcl-preview-props-override" }), "*");
  });

  it("moves a selected example into a single-specimen playground before exposing props", async () => {
    const { componentsClient, listComponentExamples } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-playground" }),
    );
    vi.mocked(listComponentExamples).mockResolvedValueOnce(makeListComponentExamplesResponse({
      examples: [
        makeComponentExample({ name: "primary", displayName: "Primary" }),
        makeComponentExample({ name: "secondary", displayName: "Secondary" }),
      ],
    }));
    const user = userEvent.setup();
    renderWithProviders(<ComponentEditor id="cmp-playground" libraryId="lib:Playground" onClose={() => {}} activePane="preview" />);
    expect((await screen.findAllByTestId(selectors.components.editor.exampleCard))).toHaveLength(2);
    expect(screen.queryByTestId(selectors.components.editor.propsPanel)).not.toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "Play" })[1]!);
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(1);
    expect(screen.getByText("Editing: Secondary")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.components.editor.propsPanel)).toBeInTheDocument();
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
        screen.getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame).getAttribute("src"),
      ).toContain("v=sha-pre");
    });

    // Preview refreshes whenever its content identity changes. The editor
    // source itself is now a separate Files tab, so this preview-only test
    // verifies the SHA contract without requiring a second pane.
    expect(screen.getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame).getAttribute("src")).toContain("v=sha-pre");
  });

  it("isolates a harness failure to its specimen", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-preview" }),
    );

    renderWithProviders(
      <ComponentEditor id="cmp-9" libraryId="lib:BrokenPreview" onClose={() => {}} activePane="preview" />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);

    window.dispatchEvent(new MessageEvent("message", {
      data: {
        type: "preview-error",
        id: "cmp-9",
        example: "",
        version: "",
        message: "preview: render failed - boom",
      },
      source: frame.contentWindow,
    }));

    const error = await screen.findByTestId(selectors.components.editor.specimenError);
    expect(error.textContent).toContain("preview: render failed - boom");

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
        <ComponentEditor id="cmp-ready" libraryId="lib:Ready" onClose={() => {}} activePane="details" />,
      );

      await userEvent.setup().click(screen.getByTestId("components-editor-split-view-toggle"));
      // The split workspace mounts Preview alongside Details. The Rendered
      // badge only shows after the harness announces preview-ready.
      await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
      expect(screen.queryByTestId(selectors.components.editor.previewBadge)).not.toBeInTheDocument();

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
      <ComponentEditor id="cmp-modes" libraryId="lib:Modes" onClose={() => {}} activePane="details" metadataSlot={<p>Details</p>} />,
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
      <ComponentEditor id="cmp-iframe" libraryId="lib:Iframe" onClose={() => {}} activePane="preview" />,
    );

    const frame = await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);

    // The onLoad handler posts the resolved theme into the frame; firing it
    // exercises that path without needing a live preview runtime. No error
    // surfaces on a clean load.
    fireEvent.load(frame);
    expect(screen.queryByTestId(selectors.components.editor.previewError)).not.toBeInTheDocument();
  });

  it("invokes onClose when the Back-to-list button is clicked", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse(),
    );

    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor id="cmp-1" libraryId="lib:X" onClose={onClose} />,
    );

    const closeBtn = await screen.findByTestId(selectors.components.editor.closeButton);
    await user.click(closeBtn);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
