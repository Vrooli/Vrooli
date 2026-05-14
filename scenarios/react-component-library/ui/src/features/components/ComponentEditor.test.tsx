import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeGetComponentContentResponse,
  makeUpdateComponentContentResponse,
} from "./mocks/factories";
import { makeComponentsMocks } from "./mocks/components";

// Monaco mounts a virtual DOM heavy enough to break jsdom (workers,
// canvas measurement). Swap it for a plain <textarea> stub so the
// editor's surrounding state-machine — load → edit → save — is the
// thing the test actually exercises.
vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: (props: {
    value?: string;
    onChange?: (v: string | undefined) => void;
  }) => {
    const { value, onChange } = props;
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
