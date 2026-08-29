import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import { distinctVariables } from "../../lib/snippetVars";

const VARIABLE_NAME = /^[a-z][a-z0-9_]*$/;

interface SnippetBodyEditorProps {
  body: string;
  onChange: (body: string) => void;
  bodyTestId?: string;
  countTestId?: string;
  rows?: number;
}

/** Shared snippet-body editor, including the select-to-variable interaction. */
export function SnippetBodyEditor({
  body,
  onChange,
  bodyTestId = "snippet-save-body",
  countTestId = "snippet-variable-count",
  rows,
}: SnippetBodyEditorProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const [variableError, setVariableError] = useState("");
  const variableCount = useMemo(() => distinctVariables(body).length, [body]);

  const trackSelection = () => {
    const textarea = textareaRef.current;
    if (!textarea || textarea.selectionStart === textarea.selectionEnd) {
      setSelection(null);
      return;
    }
    setSelection({ start: textarea.selectionStart, end: textarea.selectionEnd });
  };

  const makeVariable = () => {
    if (!selection) return;
    const variable = window.prompt(t(strings.snippets.variables.namePrompt)) ?? "";
    if (!VARIABLE_NAME.test(variable)) {
      setVariableError(t(strings.snippets.variables.invalidName));
      return;
    }
    onChange(`${body.slice(0, selection.start)}{{${variable}}}${body.slice(selection.end)}`);
    setVariableError("");
    setSelection(null);
  };

  return (
    <>
      <textarea
        ref={textareaRef}
        data-testid={bodyTestId}
        value={body}
        rows={rows}
        onChange={(event) => { onChange(event.target.value); }}
        onSelect={trackSelection}
        className="min-h-40 w-full rounded-lg border border-wc-default bg-wc-surface-input p-3 text-sm text-wc-text-primary"
      />
      <div className="flex min-h-11 items-center justify-between gap-2">
        <span data-testid={countTestId} className="text-xs text-wc-text-muted">
          {t(strings.snippets.variableCount, { count: variableCount })}
        </span>
        {selection && (
          <button
            type="button"
            data-testid="snippet-make-variable"
            onClick={makeVariable}
            className="min-h-11 rounded-lg px-3 text-sm text-wc-accent hover:bg-wc-surface-input"
          >
            {t(strings.snippets.makeVariable)}
          </button>
        )}
      </div>
      {variableError && (
        <p data-testid="snippet-variable-error" className="text-xs text-red-400">
          {variableError}
        </p>
      )}
    </>
  );
}
