import { memo, FC, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { TerminalSquare } from 'lucide-react';
import Editor from '@monaco-editor/react';

const DEFAULT_EXPRESSION = 'return document.title;';
const DEFAULT_TIMEOUT_MS = 30000;
const MIN_TIMEOUT_MS = 100;
const MAX_TIMEOUT_MS = 120000;

const clampTimeout = (value: number): number => {
  if (!Number.isFinite(value)) {
    return DEFAULT_TIMEOUT_MS;
  }
  return Math.min(Math.max(Math.round(value), MIN_TIMEOUT_MS), MAX_TIMEOUT_MS);
};

const ScriptNode: FC<NodeProps> = ({ data, selected, id }) => {
  const { getNodes, setNodes } = useReactFlow();
  const nodeData = (data ?? {}) as Record<string, unknown>;

  const initialExpression = typeof nodeData.expression === 'string' && nodeData.expression.trim().length > 0
    ? (nodeData.expression as string)
    : DEFAULT_EXPRESSION;
  const initialTimeout = clampTimeout(Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT_MS));
  const initialStoreResult = typeof nodeData.storeResult === 'string' ? nodeData.storeResult : '';

  const [expression, setExpression] = useState<string>(initialExpression);
  const [timeoutMs, setTimeoutMs] = useState<number>(initialTimeout);
  const [storeResult, setStoreResult] = useState<string>(initialStoreResult);

  useEffect(() => {
    setExpression(initialExpression);
  }, [initialExpression]);

  useEffect(() => {
    setTimeoutMs(initialTimeout);
  }, [initialTimeout]);

  useEffect(() => {
    setStoreResult(initialStoreResult);
  }, [initialStoreResult]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      return {
        ...node,
        data: {
          ...(node.data ?? {}),
          ...updates,
        },
      };
    }));
  }, [getNodes, setNodes, id]);

  useEffect(() => {
    const handle = setTimeout(() => {
      updateNodeData({ expression });
    }, 250);
    return () => clearTimeout(handle);
  }, [expression, updateNodeData]);

  const editorOptions = useMemo(() => ({
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    automaticLayout: true,
    fontSize: 13,
    wordWrap: 'on' as const,
    lineNumbers: 'off' as const,
  }), []);

  const handleTimeoutBlur = useCallback(() => {
    updateNodeData({ timeoutMs: timeoutMs });
  }, [timeoutMs, updateNodeData]);

  const handleStoreResultBlur = useCallback(() => {
    updateNodeData({ storeResult: storeResult.trim() });
  }, [storeResult, updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-2">
        <TerminalSquare size={16} className="text-teal-300" />
        <span className="font-semibold text-sm">Script</span>
      </div>

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
            value={timeoutMs}
            onChange={(event) => setTimeoutMs(clampTimeout(Number(event.target.value)))}
            onBlur={handleTimeoutBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Store result as</label>
          <input
            type="text"
            placeholder="Optional variable name"
            value={storeResult}
            onChange={(event) => setStoreResult(event.target.value)}
            onBlur={handleStoreResultBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-gray-500 mt-1">Result is available to future nodes via the upcoming variable system.</p>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ScriptNode);
