import { memo, FC, useEffect, useMemo, useState } from 'react';
import type { NodeProps } from 'reactflow';
import { TerminalSquare } from 'lucide-react';
import Editor from '@monaco-editor/react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  textInputHandler,
  numberInputHandler,
} from '@hooks/useSyncedField';
import type { EvaluateParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';

const DEFAULT_EXPRESSION = 'return document.title;';
const DEFAULT_TIMEOUT_MS = 30000;
const MIN_TIMEOUT_MS = 100;
const MAX_TIMEOUT_MS = 120000;

const ScriptNode: FC<NodeProps> = ({ selected, id }) => {
  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<EvaluateParams>(id);

  // Expression needs special handling (debounced update)
  const [expression, setExpression] = useState<string>(() => {
    const value = params?.expression;
    return typeof value === 'string' && value.trim().length > 0 ? value : DEFAULT_EXPRESSION;
  });

  // Sync expression from params
  useEffect(() => {
    const value = params?.expression;
    setExpression(typeof value === 'string' && value.trim().length > 0 ? value : DEFAULT_EXPRESSION);
  }, [params?.expression]);

  // Debounced expression update
  useEffect(() => {
    const handle = setTimeout(() => {
      updateParams({ expression });
    }, 250);
    return () => clearTimeout(handle);
  }, [expression, updateParams]);

  // UI-specific fields
  const timeoutMs = useSyncedNumber(getValue<number>('timeoutMs') ?? DEFAULT_TIMEOUT_MS, {
    min: MIN_TIMEOUT_MS,
    max: MAX_TIMEOUT_MS,
    fallback: DEFAULT_TIMEOUT_MS,
    onCommit: (v) => updateData({ timeoutMs: v }),
  });
  const storeResult = useSyncedString(getValue<string>('storeResult') ?? '', {
    onCommit: (v) => updateData({ storeResult: v || undefined }),
  });

  const editorOptions = useMemo(
    () => ({
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      wordWrap: 'on' as const,
      lineNumbers: 'off' as const,
    }),
    [],
  );

  return (
    <BaseNode selected={selected} icon={TerminalSquare} iconClassName="text-teal-300" title="Script">
      <div className="rounded border border-gray-700 overflow-hidden mb-3">
        <Editor
          height="180px"
          defaultLanguage="javascript"
          theme="vs-dark"
          value={expression}
          onChange={(value) => setExpression(value ?? '')}
          options={editorOptions}
        />
      </div>

      <div className="space-y-2 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={MIN_TIMEOUT_MS}
            max={MAX_TIMEOUT_MS}
            value={timeoutMs.value}
            onChange={numberInputHandler(timeoutMs.setValue)}
            onBlur={timeoutMs.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Store result as</label>
          <input
            type="text"
            placeholder="Optional variable name"
            value={storeResult.value}
            onChange={textInputHandler(storeResult.setValue)}
            onBlur={storeResult.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-gray-500 mt-1">
            Result is available to future nodes via the upcoming variable system.
          </p>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(ScriptNode);
