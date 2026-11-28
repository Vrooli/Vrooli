import { useRef } from "react";
import Editor from "@monaco-editor/react";

import "./monaco";

export interface CodeEditorProps {
  /** Monaco model identity. Must be unique per asset + version + file. */
  modelPath: string;
  /** File path used to choose the language; only the extension matters. */
  filePath: string;
  value: string;
  readOnly?: boolean;
  height?: number | string;
  onChange?: (value: string) => void;
  /** Cmd/Ctrl+S. Wired through Monaco so the browser's save dialog never wins. */
  onSave?: () => void;
}

function languageFor(filePath: string): string {
  const extension = filePath.slice(filePath.lastIndexOf(".")).toLowerCase();
  if (extension === ".json") return "json";
  if (extension === ".css") return "css";
  if (extension === ".js" || extension === ".jsx" || extension === ".mjs") return "javascript";
  return "typescript";
}

export default function CodeEditor({
  modelPath,
  filePath,
  value,
  readOnly = false,
  height = 360,
  onChange,
  onSave,
}: CodeEditorProps) {
  const saveRef = useRef(onSave);
  saveRef.current = onSave;

  return (
    <Editor
      height={height}
      path={modelPath}
      language={languageFor(filePath)}
      theme="rcl-nocturne"
      value={value}
      onChange={(next) => onChange?.(next ?? "")}
      onMount={(instance, monaco) => {
        instance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
          saveRef.current?.();
        });
      }}
      loading={
        <span className="wb-skeleton" style={{ display: "block", height: 340 }} aria-hidden />
      }
      options={{
        readOnly,
        domReadOnly: readOnly,
        automaticLayout: true,
        minimap: { enabled: false },
        fontFamily:
          'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
        fontSize: 12,
        lineHeight: 19,
        tabSize: 2,
        insertSpaces: true,
        formatOnPaste: true,
        formatOnType: true,
        autoIndent: "full",
        wordWrap: "off",
        scrollBeyondLastLine: false,
        renderLineHighlight: "line",
        padding: { top: 10, bottom: 10 },
        scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
        overviewRulerLanes: 0,
        fixedOverflowWidgets: true,
        contextmenu: true,
        quickSuggestions: true,
        suggestOnTriggerCharacters: true,
      }}
    />
  );
}
