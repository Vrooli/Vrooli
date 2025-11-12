import { useMemo } from 'react';
import type { Node } from 'reactflow';
import { useWorkflowStore } from '../stores/workflowStore';

export interface WorkflowVariableInfo {
  name: string;
  sourceNodeId: string;
  sourceType: string;
  sourceLabel: string;
}

const NODE_TYPE_LABELS: Record<string, string> = {
  navigate: 'Navigate',
  click: 'Click',
  hover: 'Hover',
  dragDrop: 'Drag & Drop',
  focus: 'Focus',
  blur: 'Blur',
  scroll: 'Scroll',
  select: 'Select',
  uploadFile: 'Upload File',
  setVariable: 'Set Variable',
  useVariable: 'Use Variable',
  setCookie: 'Set Cookie',
  getCookie: 'Get Cookie',
  clearCookie: 'Clear Cookie',
  setStorage: 'Set Storage',
  getStorage: 'Get Storage',
  clearStorage: 'Clear Storage',
  networkMock: 'Network Mock',
  evaluate: 'Script',
  extract: 'Extract',
  assert: 'Assert',
  shortcut: 'Shortcut',
  keyboard: 'Keyboard',
  screenshot: 'Screenshot',
};

const toTitleCase = (value: string | undefined): string => {
  if (!value) {
    return 'Workflow node';
  }
  if (NODE_TYPE_LABELS[value]) {
    return NODE_TYPE_LABELS[value];
  }
  const spaced = value.replace(/([a-z])([A-Z])/g, '$1 $2');
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
};

const pushVariable = (
  result: WorkflowVariableInfo[],
  seen: Set<string>,
  candidate: unknown,
  node: Node,
  reason: string,
) => {
  if (typeof candidate !== 'string') {
    return;
  }
  const trimmed = candidate.trim();
  if (!trimmed || seen.has(trimmed)) {
    return;
  }
  seen.add(trimmed);
  result.push({
    name: trimmed,
    sourceNodeId: node.id,
    sourceType: typeof node.type === 'string' ? node.type : 'node',
    sourceLabel: `${reason} • ${toTitleCase(node.type as string | undefined)}`.trim(),
  });
};

export const collectWorkflowVariables = (
  nodesInput: Node[] | undefined | null,
  excludeNodeId?: string,
): WorkflowVariableInfo[] => {
  if (!nodesInput || nodesInput.length === 0) {
    return [];
  }
  const seen = new Set<string>();
  const variables: WorkflowVariableInfo[] = [];

  for (const rawNode of nodesInput) {
    const node = rawNode as Node | null;
    if (!node || typeof node.id !== 'string') {
      continue;
    }
    if (excludeNodeId && node.id === excludeNodeId) {
      continue;
    }
    const type = typeof node.type === 'string' ? node.type : '';
    const data = (node.data ?? {}) as Record<string, unknown>;

    if (type === 'setVariable') {
      pushVariable(variables, seen, data.name, node, 'Set Variable name');
    }

    if (type === 'setVariable' || type === 'useVariable') {
      pushVariable(variables, seen, data.storeAs, node, 'Alias');
    }

    if (typeof data.storeResult === 'string') {
      pushVariable(variables, seen, data.storeResult, node, 'Stored result');
    }

    if (typeof data.storeIn === 'string') {
      pushVariable(variables, seen, data.storeIn, node, 'Extract result');
    }
  }

  variables.sort((a, b) => a.name.localeCompare(b.name));
  return variables;
};

export function useWorkflowVariables(excludeNodeId?: string): WorkflowVariableInfo[] {
  const nodes = useWorkflowStore((state) => state.nodes as Node[] | undefined);
  return useMemo(() => collectWorkflowVariables(nodes, excludeNodeId), [nodes, excludeNodeId]);
}
