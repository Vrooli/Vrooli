import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import type { ComponentBundle } from "../data";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings.generated";
import { renderWithProviders } from "../../test-utils";

const api = vi.hoisted(() => ({
  getComponentContent: vi.fn(),
  getComponentVersionContent: vi.fn(),
  beginComponentDraft: vi.fn(),
  updateComponentContent: vi.fn(),
  checkComponentVersion: vi.fn(),
  publishComponentDraft: vi.fn(),
}));

vi.mock("../../api/components", () => ({
  getComponentContent: api.getComponentContent,
  componentsClient: {
    getComponentVersionContent: api.getComponentVersionContent,
  },
  beginComponentDraft: api.beginComponentDraft,
  updateComponentContent: api.updateComponentContent,
  checkComponentVersion: api.checkComponentVersion,
  publishComponentDraft: api.publishComponentDraft,
}));

vi.mock("./CodeEditor", () => ({
  default: ({
    value,
    onChange,
    onSave,
  }: {
    value: string;
    onChange?: (value: string) => void;
    onSave?: () => void;
  }) => (
    <textarea
      aria-label="editor"
      value={value}
      onChange={(event) => onChange?.(event.target.value)}
      onKeyDown={(event) => {
        if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") onSave?.();
      }}
    />
  ),
}));

import { SourceWorkbench } from "./SourceWorkbench";

function bundle(
  overrides: Partial<ComponentBundle["asset"]> = {},
  versions: ComponentBundle["versions"] = [],
): ComponentBundle {
  return {
    asset: {
      id: "cmp-1",
      libraryId: "react-component-library:Button/2",
      catalogId: "button",
      slug: "button",
      displayName: "Button",
      latestVersion: "1.1.0",
      version: "1.1.0",
      draftVersion: "",
      sourcePath: "components/button/versions/1.1.0/Button.tsx",
      manifestPath: "components/button/component.json",
      dependencies: [],
      tags: [],
      ...overrides,
    },
    versions,
    stories: [],
    testReports: [],
    partial: [],
  } as unknown as ComponentBundle;
}

function versionRow(
  version: string,
  status: ComponentVersionStatus,
  files: Array<{ path: string; contentSha256: string; isEntry?: boolean }>,
) {
  return {
    id: `${version}-row`,
    componentId: "cmp-1",
    libraryId: "react-component-library:Button/2",
    version,
    status,
    sourcePath: `components/button/versions/${version}/${files[0]?.path ?? "Button.tsx"}`,
    contentSha256: files[0]?.contentSha256 ?? "",
    files: files.map((file) => ({ ...file, isEntry: Boolean(file.isEntry), slot: "" })),
  } as unknown as ComponentBundle["versions"][number];
}

describe("SourceWorkbench", () => {
  beforeEach(async () => {
    await setLocale("en");
    window.sessionStorage.clear();
    vi.clearAllMocks();
    api.getComponentVersionContent.mockResolvedValue({
      content: "export const released = true;\n",
      version: versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
        { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
      ]),
    });
    api.getComponentContent.mockResolvedValue({
      content: "export const draft = 1;\n",
      sha256: "sha-draft",
      sourcePath: "components/button/versions/1.2.0-draft.1/Button.tsx",
    });
  });

  afterEach(() => {
    cleanup();
    window.sessionStorage.clear();
    vi.clearAllMocks();
  });

  it("reads a released version and begins a draft through the authoring RPC", async () => {
    const onVersion = vi.fn();
    api.beginComponentDraft.mockResolvedValue({
      component: {
        id: "cmp-1",
        libraryId: "react-component-library:Button/2",
        draftVersion: "1.2.0-draft.1",
      },
      version: "1.2.0-draft.1",
      sourcePath: "components/button/versions/1.2.0-draft.1/Button.tsx",
      artifactPaths: [],
      previewPath: "",
    });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({}, [
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.1.0"
        onVersion={onVersion}
      />,
    );

    await screen.findByText(/export const released = true/);
    const begin = screen.getByRole("button", { name: /Begin draft/ });
    fireEvent.click(begin);

    await waitFor(() =>
      expect(api.beginComponentDraft).toHaveBeenCalledWith({
        component: "react-component-library:Button/2",
        bump: "patch",
      }),
    );
    expect(onVersion).toHaveBeenCalledWith("1.2.0-draft.1");
  });

  it("saves with the read digest, then checks and publishes the draft", async () => {
    const onVersion = vi.fn();
    const onSaved = vi.fn();
    api.updateComponentContent.mockResolvedValue({
      sha256: "sha-saved",
      sourcePath: "components/button/versions/1.2.0-draft.1/Button.tsx",
    });
    api.checkComponentVersion.mockResolvedValue({
      component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
      version: "1.2.0-draft.1",
      passed: true,
      checks: [{ stage: "typescript", verdict: "passed", message: "compiled", remediation: "" }],
      previewPath: "",
    });
    api.publishComponentDraft.mockResolvedValue({
      component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
      version: "1.2.0",
      sourcePath: "components/button/versions/1.2.0/Button.tsx",
      artifactPaths: [],
      previewPath: "",
    });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={onVersion}
        onSaved={onSaved}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    // Source is readable without loading the Monaco editor. The editor module
    // is reached only after the author explicitly enters draft editing.
    expect(screen.queryByLabelText(/editor/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));

    const editor = await screen.findByLabelText(/editor/);
    fireEvent.change(editor, { target: { value: "export const draft = 2;\n" } });
    fireEvent.click(screen.getByRole("button", { name: /Save draft/ }));

    await waitFor(() =>
      expect(api.updateComponentContent).toHaveBeenCalledWith({
        id: "cmp-1",
        content: "export const draft = 2;\n",
        expectedSha256: "sha-draft",
        path: "",
      }),
    );
    await screen.findByText(/Saved to the draft/);
    expect(onSaved).toHaveBeenCalledWith("1.2.0-draft.1", "", "sha-saved");

    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    await waitFor(() =>
      expect(api.checkComponentVersion).toHaveBeenCalledWith({
        component: "react-component-library:Button/2",
        version: "1.2.0-draft.1",
      }),
    );
    await screen.findByText(/typescript/);

    fireEvent.click(screen.getByRole("button", { name: /Publish…/ }));
    fireEvent.change(screen.getByLabelText(/Changelog/), {
      target: { value: "Tighten the label." },
    });
    fireEvent.click(screen.getByRole("button", { name: /Publish release/ }));

    await waitFor(() =>
      expect(api.publishComponentDraft).toHaveBeenCalledWith({
        component: "react-component-library:Button/2",
        draftVersion: "1.2.0-draft.1",
        changelogMd: "Tighten the label.",
      }),
    );
    expect(onVersion).toHaveBeenCalledWith("1.2.0");
  });

  it("turns a stale digest into a named conflict with a reload action", async () => {
    api.updateComponentContent.mockRejectedValue(
      new ConnectError(
        "content sha256 mismatch (got new, expected sha-draft)",
        Code.FailedPrecondition,
      ),
    );

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));
    fireEvent.change(await screen.findByLabelText(/editor/), {
      target: { value: "export const draft = 3;\n" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Save draft/ }));

    await screen.findByText(/The file changed on disk/);
    fireEvent.click(screen.getByRole("button", { name: /Reload source/ }));

    expect(await screen.findByText(/export const draft = 1/)).toBeInTheDocument();
    expect(screen.queryByLabelText(/editor/)).not.toBeInTheDocument();
    expect(screen.queryByText(/The file changed on disk/)).not.toBeInTheDocument();
    expect(api.getComponentContent).toHaveBeenLastCalledWith({ id: "cmp-1", path: "" });
  });

  it("does not request a preview reload when the saved digest is unchanged", async () => {
    const onSaved = vi.fn();
    api.updateComponentContent.mockResolvedValue({
      sha256: "sha-draft",
      sourcePath: "components/button/versions/1.2.0-draft.1/Button.tsx",
    });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
        onSaved={onSaved}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));
    fireEvent.change(await screen.findByLabelText(/editor/), {
      target: { value: "export const draft = 1; // formatting only\n" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Save draft/ }));

    await screen.findByText(/Saved to the draft/);
    expect(onSaved).not.toHaveBeenCalled();
  });

  it("keeps the saved-source callback untouched on failure and recovers after a retry", async () => {
    const onSaved = vi.fn();
    api.updateComponentContent.mockRejectedValueOnce(new Error("write failed"));

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
        onSaved={onSaved}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));
    const editor = await screen.findByLabelText(/editor/);
    fireEvent.change(editor, { target: { value: "export const draft = 2;\n" } });
    fireEvent.click(screen.getByRole("button", { name: /Save draft/ }));

    await screen.findByText(/write failed/);
    expect(editor).toHaveValue("export const draft = 2;\n");
    expect(onSaved).not.toHaveBeenCalled();

    api.updateComponentContent.mockResolvedValueOnce({
      sha256: "sha-recovered",
      sourcePath: "components/button/versions/1.2.0-draft.1/Button.tsx",
    });
    fireEvent.click(screen.getByRole("button", { name: /Save draft/ }));

    await screen.findByText(/Saved to the draft/);
    expect(onSaved).toHaveBeenCalledTimes(1);
    expect(onSaved).toHaveBeenCalledWith("1.2.0-draft.1", "", "sha-recovered");
  });

  it("discards an unsaved file edit before letting the reviewer inspect another source file", async () => {
    api.getComponentContent.mockImplementation(({ path }: { path: string }) =>
      Promise.resolve({
        content:
          path === "utils.ts" ? "export const helper = true;\n" : "export const draft = 1;\n",
        sha256: path === "utils.ts" ? "sha-utils" : "sha-draft",
        sourcePath: `components/button/versions/1.2.0-draft.1/${path || "Button.tsx"}`,
      }),
    );

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
            { path: "utils.ts", contentSha256: "sha-utils" },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));
    fireEvent.change(await screen.findByLabelText(/editor/), {
      target: { value: "export const draft = 2;\n" },
    });
    expect(screen.getByText(/Unsaved/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Discard/ }));
    expect(await screen.findByText(/export const draft = 1/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.component.sourceFile)), {
      target: { value: "utils.ts" },
    });

    expect(await screen.findByText(/export const helper = true/)).toBeInTheDocument();
    expect(api.getComponentContent).toHaveBeenLastCalledWith({ id: "cmp-1", path: "utils.ts" });
    expect(screen.queryByText(/Unsaved/)).not.toBeInTheDocument();
  });

  it("keeps publishing gated after failed checks and lets the author retry before canceling preparation", async () => {
    api.checkComponentVersion
      .mockResolvedValueOnce({
        component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
        version: "1.2.0-draft.1",
        passed: false,
        checks: [
          {
            stage: "accessibility",
            verdict: "failed",
            message: "The button needs an accessible name.",
            remediation: "Add an aria-label to the icon-only action.",
          },
        ],
        previewPath: "",
      })
      .mockResolvedValueOnce({
        component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
        version: "1.2.0-draft.1",
        passed: true,
        checks: [
          {
            stage: "accessibility",
            verdict: "passed",
            message: "The button has an accessible name.",
            remediation: "",
          },
        ],
        previewPath: "",
      });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    expect(await screen.findByText(/The button needs an accessible name/)).toBeInTheDocument();
    expect(screen.getByText(/Add an aria-label to the icon-only action/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Publish…/ })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    expect(await screen.findByText(/The button has an accessible name/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Publish…/ }));
    expect(screen.getByLabelText(/Changelog/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Cancel/ }));
    expect(screen.queryByLabelText(/Changelog/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Publish…/ })).toBeInTheDocument();
    expect(api.checkComponentVersion).toHaveBeenCalledTimes(2);
    expect(api.publishComponentDraft).not.toHaveBeenCalled();
  });

  it("opens an available draft from a released version", async () => {
    const onVersion = vi.fn();
    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.1.0"
        onVersion={onVersion}
      />,
    );

    expect(await screen.findByText(/Draft 1.2.0-draft.1 is active/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Open draft/ }));
    expect(onVersion).toHaveBeenCalledWith("1.2.0-draft.1");
  });

  it("uses the selected version bump when beginning a draft", async () => {
    api.beginComponentDraft.mockResolvedValue({ version: "1.2.0-draft.1" });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({}, [
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.1.0"
        onVersion={vi.fn()}
      />,
    );

    await screen.findByText(/export const released = true/);
    fireEvent.change(screen.getByLabelText(/Version bump/), { target: { value: "minor" } });
    fireEvent.click(screen.getByRole("button", { name: /Begin draft/ }));

    await waitFor(() =>
      expect(api.beginComponentDraft).toHaveBeenCalledWith({
        component: "react-component-library:Button/2",
        bump: "minor",
      }),
    );
  });

  it("keeps the release available and lets the author retry a failed draft start", async () => {
    api.beginComponentDraft
      .mockRejectedValueOnce(new Error("draft service unavailable"))
      .mockResolvedValueOnce({ version: "1.2.0-draft.1" });
    const onVersion = vi.fn();

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({}, [
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.1.0"
        onVersion={onVersion}
      />,
    );

    await screen.findByText(/export const released = true/);
    fireEvent.click(screen.getByRole("button", { name: /Begin draft/ }));
    expect(await screen.findByText(/draft service unavailable/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Begin draft/ })).toBeEnabled();

    fireEvent.click(screen.getByRole("button", { name: /Begin draft/ }));
    await waitFor(() => expect(onVersion).toHaveBeenCalledWith("1.2.0-draft.1"));
    expect(api.beginComponentDraft).toHaveBeenCalledTimes(2);
  });

  it("keeps publish preparation open after a failed publish and allows retry", async () => {
    api.checkComponentVersion.mockResolvedValue({
      component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
      version: "1.2.0-draft.1",
      passed: true,
      checks: [{ stage: "typescript", verdict: "passed", message: "compiled", remediation: "" }],
      previewPath: "",
    });
    api.publishComponentDraft
      .mockRejectedValueOnce(new Error("publish service unavailable"))
      .mockResolvedValueOnce({ version: "1.2.0" });
    const onVersion = vi.fn();

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={onVersion}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    await screen.findByText(/compiled/);
    fireEvent.click(screen.getByRole("button", { name: /Publish…/ }));
    fireEvent.change(screen.getByLabelText(/Changelog/), {
      target: { value: "Fix the button label." },
    });
    fireEvent.click(screen.getByRole("button", { name: /Publish release/ }));

    expect(await screen.findByText(/publish service unavailable/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Changelog/)).toHaveValue("Fix the button label.");
    expect(screen.getByRole("button", { name: /Publish release/ })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: /Publish release/ }));

    await waitFor(() => expect(onVersion).toHaveBeenCalledWith("1.2.0"));
    expect(api.publishComponentDraft).toHaveBeenCalledTimes(2);
  });

  it("restores a draft edit that was stashed before the workbench was reopened", async () => {
    const props = {
      bundle: bundle({ draftVersion: "1.2.0-draft.1" }, [
        versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
          { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
        ]),
      ]),
      version: "1.2.0-draft.1",
      onVersion: vi.fn(),
    };
    const firstRender = renderWithProviders(<SourceWorkbench {...props} />);

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Edit source/ }));
    fireEvent.change(await screen.findByRole("textbox", { name: /editor/ }), {
      target: { value: "export const draft = 9;\n" },
    });
    expect(
      window.sessionStorage.getItem("rcl.workbench.source-draft.cmp-1.1.2.0-draft.1.entry"),
    ).toBe("export const draft = 9;\n");

    firstRender.unmount();
    renderWithProviders(<SourceWorkbench {...props} />);

    expect(await screen.findByRole("textbox", { name: /editor/ })).toHaveValue(
      "export const draft = 9;\n",
    );
    expect(
      screen.getByText(i18n.t(strings.workbench.component.sourceRestored)),
    ).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.workbench.component.unsaved))).toBeInTheDocument();
  });

  it("surfaces a failed check request and can retry successfully", async () => {
    api.checkComponentVersion
      .mockRejectedValueOnce(new Error("check service unavailable"))
      .mockResolvedValueOnce({
        component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
        version: "1.2.0-draft.1",
        passed: true,
        checks: [{ stage: "typescript", verdict: "passed", message: "compiled", remediation: "" }],
        previewPath: "",
      });

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
      />,
    );

    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    expect(await screen.findByText(/check service unavailable/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    expect(await screen.findByText(/compiled/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Publish…/ })).toBeInTheDocument();
  });

  it("renders empty draft check results", async () => {
    api.checkComponentVersion.mockResolvedValue({
      component: { id: "cmp-1", libraryId: "react-component-library:Button/2" },
      version: "1.2.0-draft.1",
      passed: false,
      checks: [],
      previewPath: "",
    });
    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({ draftVersion: "1.2.0-draft.1" }, [
          versionRow("1.2.0-draft.1", ComponentVersionStatus.DRAFT, [
            { path: "Button.tsx", contentSha256: "sha-draft", isEntry: true },
          ]),
        ])}
        version="1.2.0-draft.1"
        onVersion={vi.fn()}
      />,
    );
    await screen.findByText(/export const draft = 1/);
    fireEvent.click(screen.getByRole("button", { name: /Run checks/ }));
    expect(
      await screen.findByText(i18n.t(strings.workbench.component.noChecksTitle)),
    ).toBeInTheDocument();
    expect(screen.getByText(/returned no stages/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Publish…/ })).not.toBeInTheDocument();
  });

  it("shows the source owner error when a file cannot be loaded", async () => {
    api.getComponentVersionContent.mockRejectedValue(new Error("source unavailable"));

    renderWithProviders(
      <SourceWorkbench
        bundle={bundle({}, [
          versionRow("1.1.0", ComponentVersionStatus.RELEASED, [
            { path: "Button.tsx", contentSha256: "sha-release", isEntry: true },
          ]),
        ])}
        version="1.1.0"
        onVersion={vi.fn()}
      />,
    );

    expect(await screen.findByText(/source owner did not return this file/)).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: /Source code/ })).not.toBeInTheDocument();
  });
});
