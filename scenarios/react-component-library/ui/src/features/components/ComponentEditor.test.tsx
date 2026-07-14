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
    expect(screen.getByTestId(selectors.components.editor.shaHash).textContent).toContain(
      "abc123def456",
    );
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
      <ComponentEditor id="cmp-7" libraryId="lib:Iframe" onClose={() => {}} />,
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

    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor id="cmp-7" libraryId="lib:Gallery" onClose={() => {}} />,
    );

    await user.click(await screen.findByTestId(selectors.components.editor.previewModeButton));
    const frames = await screen.findAllByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);

    expect(screen.getByTestId(selectors.components.editor.gallery)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.components.editor.exampleCard)).toHaveLength(2);
    expect(frames).toHaveLength(2);
    expect(frames[0]?.getAttribute("src")).toContain("example=primary");
    expect(frames[1]?.getAttribute("src")).toContain("example=disabled");
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

    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor id="cmp-8" libraryId="lib:Reload" onClose={() => {}} />,
    );

    await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    await waitFor(() => {
      expect(
        screen.getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame).getAttribute("src"),
      ).toContain("v=sha-pre");
    });

    const stub = screen.getByTestId<HTMLTextAreaElement>("monaco-stub");
    fireEvent.change(stub, { target: { value: "v2" } });
    await user.click(screen.getByTestId(selectors.components.editor.saveButton));

    await waitFor(() => {
      const post = screen
        .getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame)
        .getAttribute("src");
      expect(post).toContain("v=sha-post");
    });
  });

  it("shows a retryable preview failure when the harness posts preview-error", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "v1", sha256: "sha-preview" }),
    );

    const user = userEvent.setup();
    renderWithProviders(
      <ComponentEditor id="cmp-9" libraryId="lib:BrokenPreview" onClose={() => {}} />,
    );

    await screen.findByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame);
    await user.click(screen.getByRole("button", { name: "Preview" }));

    window.dispatchEvent(new MessageEvent("message", {
      data: {
        type: "preview-error",
        id: "cmp-9",
        message: "preview: render failed - boom",
      },
    }));

    const error = await screen.findByTestId(selectors.components.editor.previewError);
    expect(error.textContent).toContain("preview: render failed - boom");

    const before = screen
      .getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame)
      .getAttribute("src");
    await user.click(screen.getByTestId(selectors.components.editor.previewRetryButton));

    await waitFor(() => {
      const after = screen
        .getByTestId<HTMLIFrameElement>(selectors.components.editor.previewFrame)
        .getAttribute("src");
      expect(after).not.toBe(before);
      expect(after).toContain("r=1");
    });
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
