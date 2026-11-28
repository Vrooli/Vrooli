import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";

const editorMock = vi.hoisted(() => vi.fn(() => null));

vi.mock("@monaco-editor/react", () => ({ default: editorMock }));
vi.mock("./monaco", () => ({}));

import Editor, { type EditorProps, type Monaco } from "@monaco-editor/react";
import CodeEditor from "./CodeEditor";

function latestProps(): EditorProps {
  const call = vi.mocked(Editor).mock.calls.at(-1);
  if (!call) throw new Error("Expected the editor component to render.");
  return call[0];
}

describe("CodeEditor", () => {
  afterEach(() => vi.clearAllMocks());

  it.each([
    ["manifest.JSON", "json"],
    ["styles.css", "css"],
    ["script.jsx", "javascript"],
    ["worker.mjs", "javascript"],
    ["component.tsx", "typescript"],
  ])("selects the language for %s", (filePath, language) => {
    renderWithProviders(
      <CodeEditor modelPath="asset@1/file" filePath={filePath} value="source" />,
      { withoutRouter: true },
    );

    expect(latestProps().language).toBe(language);
  });

  it("passes editor configuration and normalizes an empty change", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <CodeEditor
        modelPath="asset@2/component.tsx"
        filePath="component.tsx"
        value="export const Button = () => null"
        onChange={onChange}
      />,
      { withoutRouter: true },
    );

    const props = latestProps();
    expect(props).toMatchObject({
      height: 360,
      path: "asset@2/component.tsx",
      language: "typescript",
      theme: "rcl-nocturne",
      value: "export const Button = () => null",
      options: {
        readOnly: false,
        domReadOnly: false,
        automaticLayout: true,
        minimap: { enabled: false },
      },
      loading: expect.anything(),
    });
    props.onChange?.(undefined, {} as never);
    expect(onChange).toHaveBeenCalledWith("");
  });

  it("keeps the registered save command wired to the latest callback", () => {
    const firstSave = vi.fn();
    const nextSave = vi.fn();
    const { rerender } = renderWithProviders(
      <CodeEditor
        modelPath="asset@2/component.tsx"
        filePath="component.tsx"
        value="source"
        readOnly
        height="50vh"
        onSave={firstSave}
      />,
      { withoutRouter: true },
    );

    const addCommand = vi.fn();
    const instance = { addCommand };
    const monaco = { KeyMod: { CtrlCmd: 8 }, KeyCode: { KeyS: 2 } } as unknown as Monaco;
    latestProps().onMount?.(instance as never, monaco);
    const [shortcut, callback] = addCommand.mock.calls[0] ?? [];
    expect(shortcut).toBe(10);
    expect(callback).toEqual(expect.any(Function));

    rerender(
      <CodeEditor
        modelPath="asset@2/component.tsx"
        filePath="component.tsx"
        value="source"
        readOnly
        height="50vh"
        onSave={nextSave}
      />,
    );
    callback?.();

    expect(firstSave).not.toHaveBeenCalled();
    expect(nextSave).toHaveBeenCalledOnce();
    expect(latestProps().options).toMatchObject({ readOnly: true, domReadOnly: true });
    expect(latestProps().height).toBe("50vh");
  });
});
