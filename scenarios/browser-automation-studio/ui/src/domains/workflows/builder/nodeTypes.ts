import type { NodeTypes } from 'reactflow';
import { ActionDefinitionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import ActionNode from '../nodes/ActionNode';

const legacyNodeTypes = [
  'browserAction', 'navigate', 'click', 'hover', 'dragDrop', 'focus', 'blur',
  'scroll', 'select', 'uploadFile', 'rotate', 'gesture', 'tabSwitch', 'frameSwitch',
  'conditional', 'setVariable', 'setCookie', 'getCookie', 'clearCookie', 'setStorage',
  'getStorage', 'clearStorage', 'networkMock', 'type', 'shortcut', 'keyboard',
  'evaluate', 'screenshot', 'wait', 'extract', 'assert', 'useVariable', 'subflow', 'loop',
];

const v2ActionTypes = ActionDefinitionSchema.fields.flatMap((field) => {
  if (field.oneof?.name !== 'params' || field.fieldKind !== 'message') return [];
  return [field.localName];
});

export const nodeTypes: NodeTypes = Object.fromEntries(
  [...new Set([...legacyNodeTypes, ...v2ActionTypes])].map((nodeType) => [nodeType, ActionNode]),
);

// Edge marker colors by theme - darker for light mode, lighter for dark mode
export const EDGE_MARKER_COLORS = {
  dark: '#6b7280',
  light: '#4b5563',
} as const;
