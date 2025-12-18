import Editor from '@monaco-editor/react';
import { selectors } from '@constants/selectors';
import type { WorkflowValidationResult } from '@/types/workflow';

interface CodeEditorPanelProps {
  codeValue: string;
  codeError: string | null;
  codeDirty: boolean;
  validationResult: WorkflowValidationResult | null;
  isValidatingCode: boolean;
  onCodeChange: (value: string | undefined) => void;
  onResetCode: () => void;
  onApplyChanges: () => void;
}

export function CodeEditorPanel({
  codeValue,
  codeError,
  codeDirty,
  validationResult,
  isValidatingCode,
  onCodeChange,
  onResetCode,
  onApplyChanges,
}: CodeEditorPanelProps) {
  return (
    <div
      className="absolute inset-0 flex flex-col bg-[#1e1e1e] border border-gray-800 rounded-lg overflow-hidden"
      data-testid={selectors.workflowBuilder.code.view}
    >
      <div
        className="flex-1 overflow-hidden"
        data-testid={selectors.workflowBuilder.code.editorContainer}
      >
        <Editor
          height="100%"
          defaultLanguage="json"
          value={codeValue}
          onChange={onCodeChange}
          theme="vs-dark"
          data-testid={selectors.workflowBuilder.code.editor}
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
            insertSpaces: true,
            wordWrap: 'on',
            formatOnPaste: true,
            formatOnType: true,
            renderWhitespace: 'selection',
            scrollbar: {
              verticalScrollbarSize: 10,
              horizontalScrollbarSize: 10,
            },
          }}
        />
      </div>
      <div
        className="flex items-center justify-between border-t border-gray-800 bg-[#252526] px-4 py-3"
        data-testid={selectors.workflowBuilder.code.toolbar}
      >
        <div className="flex items-center gap-3">
          <div
            className="text-xs text-gray-400"
            data-testid={selectors.workflowBuilder.code.lineCount}
          >
            {codeValue.split('\n').length} lines
          </div>
          {codeError && (
            <>
              <div className="w-px h-4 bg-gray-700" />
              <div
                className="text-xs text-red-400"
                data-testid={selectors.workflowBuilder.code.error}
              >
                {codeError}
              </div>
            </>
          )}
          {validationResult && !codeError && (
            <>
              <div className="w-px h-4 bg-gray-700" />
              <div
                className={`text-xs ${validationResult.valid ? 'text-emerald-400' : 'text-red-400'}`}
                data-testid={selectors.workflowBuilder.code.validation}
              >
                {validationResult.valid
                  ? `Validated · ${validationResult.stats.node_count} nodes`
                  : 'Validation failed'}
              </div>
            </>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onResetCode}
            className="px-3 py-1.5 rounded-md text-xs bg-gray-700 text-gray-200 hover:bg-gray-600 transition-all disabled:opacity-50"
            disabled={!codeDirty}
            data-testid={selectors.workflowBuilder.code.resetButton}
          >
            Reset
          </button>
          <button
            onClick={onApplyChanges}
            className="px-3 py-1.5 rounded-md text-xs bg-purple-600 text-white hover:bg-purple-500 transition-all disabled:opacity-50"
            disabled={!codeDirty || isValidatingCode}
            data-testid={selectors.workflowBuilder.code.applyButton}
          >
            {isValidatingCode ? 'Validating…' : 'Apply Changes'}
          </button>
        </div>
      </div>
    </div>
  );
}
