import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import type { ComponentBundle } from "../data";
import { setLocale } from "../../i18n";
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
  componentsClient: {
    getComponentContent: api.getComponentContent,
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
      />,
    );

    await screen.findByText(/export const draft = 1/);
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
    expect(screen.getByRole("button", { name: /Reload source/ })).toBeInTheDocument();
  });
});
