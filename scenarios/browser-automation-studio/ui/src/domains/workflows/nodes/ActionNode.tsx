import { memo, type FC, type ReactNode } from 'react';
import type { NodeProps } from 'reactflow';
import { Handle, Position, useReactFlow } from 'reactflow';
import { ScalarType, type DescField } from '@bufbuild/protobuf';
import {
  Activity, AppWindow, ArrowLeftRight, Camera, CheckCircle2, Circle, Clock, Cookie,
  Database, Eye, Globe, Hand, HardDriveDownload, HardDriveUpload, Keyboard,
  MousePointer, MousePointer2, Network, Play, Recycle, RefreshCcw, ScrollText,
  ShieldCheck, Smartphone, TerminalSquare, Type, UploadCloud, Variable,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedJson, type UseSyncedFieldResult } from '@hooks/useSyncedField';
import type { ActionDefinition, ActionParamsField } from '@/domains/workflows/utils/normalizers';
import { actionTypeToNodeType, getActionParamsFieldName } from '@/domains/workflows/utils/normalizers';
import { ActionDefinitionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { NodeCheckbox, NodeNumberField, NodeSelectField, NodeSelectorField, NodeTextArea, NodeTextField, NodeUrlField } from './fields';
import { useElementPicker } from '@hooks/useElementPicker';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useResiliencePanelProps } from '@hooks/useResiliencePanel';
import ResiliencePanel from './ResiliencePanel';

type Params = Record<string, unknown>;
type FieldKind = 'text' | 'textarea' | 'number' | 'checkbox' | 'select' | 'selector' | 'url' | 'json';
interface FieldSpec {
  key: string;
  label: string;
  kind?: FieldKind;
  placeholder?: string;
  options?: Array<{ value: string; label: string }>;
  defaultValue?: string | number | boolean;
  min?: number;
  max?: number;
  rows?: number;
  description?: string;
  when?: (params: Params) => boolean;
}
interface NodeSpec {
  title: string;
  icon: LucideIcon;
  color: string;
  fields: FieldSpec[];
  className?: string;
  handles?: 'standard' | 'conditional' | 'loop';
}

const options = (values: string, labels: string[] = []) => values.split('|').map((value, index) => ({ value, label: labels[index] ?? value.replace(/([A-Z])/g, ' $1').replace(/^./, (c) => c.toUpperCase()) }));
const text = (key: string, label = key, placeholder?: string): FieldSpec => ({ key, label, kind: 'text', placeholder });
const area = (key: string, label = key, placeholder?: string, rows = 3): FieldSpec => ({ key, label, kind: 'textarea', placeholder, rows });
const number = (key: string, label = key, defaultValue = 0, min: number | null = 0, max?: number): FieldSpec => ({ key, label, kind: 'number', defaultValue, min: min ?? undefined, max });
const toggle = (key: string, label = key, defaultValue = false): FieldSpec => ({ key, label, kind: 'checkbox', defaultValue });
const select = (key: string, label: string, values: string, labels?: string[], defaultValue?: string): FieldSpec => {
  const choices = options(values, labels);
  return { key, label, kind: 'select', options: choices, defaultValue: defaultValue ?? choices[0]?.value };
};
const selector = (key = 'selector', label = 'Selector'): FieldSpec => ({ key, label, kind: 'selector', placeholder: 'CSS selector...' });
const url: FieldSpec = { key: 'url', label: 'URL', kind: 'url' };

const specs: Record<string, NodeSpec> = {
  browserAction: { title: 'Browser Action', icon: Activity, color: 'text-slate-300', fields: [text('action'), text('selector'), text('value')] },
  navigate: { title: 'Navigate', icon: Globe, color: 'text-blue-400', className: 'w-80', fields: [select('destinationType', 'Destination', 'NAVIGATE_DESTINATION_TYPE_URL|NAVIGATE_DESTINATION_TYPE_SCENARIO', ['URL', 'App'], 'NAVIGATE_DESTINATION_TYPE_URL'), { ...url, when: (p) => !String(p.destinationType ?? '').toUpperCase().endsWith('SCENARIO') }, { ...text('scenario', 'Scenario', 'Scenario name'), when: (p) => String(p.destinationType ?? '').toUpperCase().endsWith('SCENARIO') }, { ...text('scenarioPath', 'Path'), when: (p) => String(p.destinationType ?? '').toUpperCase().endsWith('SCENARIO') }, text('waitForSelector', 'Wait for selector'), select('waitUntil', 'Wait until', 'NAVIGATE_WAIT_EVENT_LOAD|NAVIGATE_WAIT_EVENT_DOM_CONTENT_LOADED|NAVIGATE_WAIT_EVENT_NETWORK_IDLE'), number('timeoutMs', 'Timeout (ms)', 30000), number('waitForMs', 'Wait after navigation (ms)')] },
  click: { title: 'Click', icon: MousePointer, color: 'text-green-400', fields: [selector(), select('button', 'Mouse button', 'MOUSE_BUTTON_LEFT|MOUSE_BUTTON_RIGHT|MOUSE_BUTTON_MIDDLE'), number('clickCount', 'Click count', 1, 1, 3), number('delayMs', 'Press duration (ms)'), area('modifiers', 'Keyboard modifiers', 'One per line'), toggle('force', 'Force click'), toggle('scrollIntoView', 'Scroll into view')] },
  hover: { title: 'Hover', icon: MousePointer2, color: 'text-amber-300', fields: [selector(), number('timeoutMs', 'Timeout (ms)', 30000)] },
  dragDrop: { title: 'Drag & Drop', icon: Hand, color: 'text-pink-300', fields: [text('sourceSelector', 'Drag from selector'), text('targetSelector', 'Drop on selector'), number('offsetX', 'Horizontal offset', 0, null), number('offsetY', 'Vertical offset', 0, null), number('targetOffsetX', 'Target horizontal offset', 0, null), number('targetOffsetY', 'Target vertical offset', 0, null), number('steps', 'Steps', 18, 1), number('delayMs', 'Delay per step (ms)'), number('timeoutMs', 'Timeout (ms)', 30000)] },
  focus: { title: 'Focus', icon: Eye, color: 'text-blue-300', fields: [selector('selector', 'Target selector'), toggle('scroll', 'Scroll into view'), number('timeoutMs', 'Timeout (ms)', 5000), number('waitForMs', 'Wait after action (ms)')] },
  blur: { title: 'Blur', icon: Eye, color: 'text-blue-300', fields: [selector('selector', 'Target selector'), number('timeoutMs', 'Timeout (ms)', 5000), number('waitForMs', 'Wait after action (ms)')] },
  scroll: { title: 'Scroll', icon: ScrollText, color: 'text-purple-300', fields: [selector(), number('x', 'Horizontal position'), number('y', 'Vertical position'), number('deltaX', 'Horizontal distance', 0, null), number('deltaY', 'Vertical distance', 500, null), select('behavior', 'Behavior', 'SCROLL_BEHAVIOR_AUTO|SCROLL_BEHAVIOR_SMOOTH')] },
  select: { title: 'Select Option', icon: CheckCircle2, color: 'text-cyan-300', fields: [selector(), { ...text('value', 'Value'), when: (p) => (p.selectBy ?? 'value') === 'value' }, { ...text('label', 'Label'), when: (p) => p.selectBy === 'label' }, { ...number('index', 'Index'), when: (p) => p.selectBy === 'index' }, number('timeoutMs', 'Timeout (ms)', 5000), number('waitForMs', 'Wait after action (ms)')] },
  uploadFile: { title: 'Upload File', icon: UploadCloud, color: 'text-pink-300', fields: [selector('selector', 'Target selector'), area('filePaths', 'File paths', 'One path per line'), number('timeoutMs', 'Timeout (ms)', 30000)] },
  rotate: { title: 'Rotate', icon: Smartphone, color: 'text-sky-300', fields: [select('orientation', 'Orientation', 'DEVICE_ORIENTATION_PORTRAIT|DEVICE_ORIENTATION_LANDSCAPE'), number('angle', 'Angle', 0, 0, 360)] },
  gesture: { title: 'Gesture', icon: Hand, color: 'text-rose-300', fields: [select('gestureType', 'Gesture', 'GESTURE_TYPE_TAP|GESTURE_TYPE_DOUBLE_TAP|GESTURE_TYPE_LONG_PRESS|GESTURE_TYPE_SWIPE|GESTURE_TYPE_PINCH'), selector(), select('direction', 'Swipe direction', 'SWIPE_DIRECTION_UP|SWIPE_DIRECTION_DOWN|SWIPE_DIRECTION_LEFT|SWIPE_DIRECTION_RIGHT'), number('distance', 'Distance'), number('scale', 'Scale', 1), number('durationMs', 'Duration (ms)', 500), number('steps', 'Steps', 10, 1), number('stepDelayMs', 'Step delay (ms)'), text('traceLabel', 'Trace label'), number('idleAfterMs', 'Idle after (ms)'), number('wheelDeltaY', 'Wheel delta', 0, null), toggle('ctrlKey', 'Control key')] },
  tabSwitch: { title: 'Tab Switch', icon: AppWindow, color: 'text-violet-300', fields: [select('action', 'Action', 'TAB_SWITCH_ACTION_SWITCH|TAB_SWITCH_ACTION_OPEN|TAB_SWITCH_ACTION_CLOSE'), text('url', 'URL'), number('index', 'Tab index'), text('title', 'Tab title'), text('urlPattern', 'URL pattern')] },
  frameSwitch: { title: 'Frame Switch', icon: ArrowLeftRight, color: 'text-lime-300', fields: [select('action', 'Action', 'FRAME_SWITCH_ACTION_ENTER|FRAME_SWITCH_ACTION_PARENT|FRAME_SWITCH_ACTION_TOP'), text('selector', 'IFrame selector'), text('frameId', 'Frame ID'), text('frameUrl', 'Frame URL'), number('timeoutMs', 'Timeout (ms)', 30000)] },
  conditional: { title: 'Conditional', icon: Activity, color: 'text-indigo-300', handles: 'conditional', fields: [select('conditionType', 'Condition type', 'expression|element|variable', ['Expression (JS)', 'Element Presence', 'Workflow Variable'], 'expression'), { ...area('expression', 'Expression', 'return true;', 3), when: (p) => String(p.conditionType ?? 'EXPRESSION').toUpperCase().endsWith('EXPRESSION') }, { ...selector(), when: (p) => String(p.conditionType ?? '').toUpperCase().endsWith('ELEMENT') || p.conditionType === 'element' }, { ...text('variable', 'Variable name'), when: (p) => String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') || p.conditionType === 'variable' }, { ...select('operator', 'Operator', 'equals|not_equals|contains|starts_with|ends_with|gt|gte|lt|lte'), when: (p) => String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') || p.conditionType === 'variable' }, { ...text('value', 'Comparison value'), when: (p) => String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') || p.conditionType === 'variable' }, toggle('negate', 'Negate condition'), number('timeoutMs', 'Timeout (ms)', 10000, 100), number('pollIntervalMs', 'Poll interval (ms)', 250, 50, 2000)] },
  setVariable: { title: 'Set Variable', icon: Variable, color: 'text-sky-300', fields: [text('name', 'Variable name'), select('sourceType', 'Source', 'static|expression|element'), { ...select('valueType', 'Value type', 'text|number|boolean|json'), when: (p) => String(p.sourceType ?? 'STATIC').toUpperCase().endsWith('STATIC') }, { ...area('value', 'Value'), when: (p) => String(p.sourceType ?? 'STATIC').toUpperCase().endsWith('STATIC') }, { ...area('expression', 'Expression'), when: (p) => String(p.sourceType ?? '').toUpperCase().endsWith('EXPRESSION') || p.sourceType === 'expression' }, { ...selector(), when: (p) => String(p.sourceType ?? '').toUpperCase().endsWith('EXTRACT') || p.sourceType === 'element' }, { ...select('extractType', 'Extract as', 'text|html|attribute'), when: (p) => String(p.sourceType ?? '').toUpperCase().endsWith('EXTRACT') || p.sourceType === 'element' }, { ...text('attribute', 'Attribute'), when: (p) => String(p.sourceType ?? '').toUpperCase().endsWith('EXTRACT') || p.sourceType === 'element' }, text('storeAs', 'Store as'), number('timeoutMs', 'Timeout (ms)'), toggle('allMatches', 'Collect all matches')] },
  setCookie: { title: 'Set Cookie', icon: Cookie, color: 'text-amber-300', fields: [text('name', 'Cookie name'), area('value', 'Value'), url, text('domain', 'Domain'), text('path', 'Path'), text('expiresAt', 'Expires at'), select('sameSite', 'SameSite', 'strict|lax|none'), toggle('secure', 'Secure'), toggle('httpOnly', 'HTTP only'), number('ttlSeconds', 'Lifetime (seconds)'), number('timeoutMs', 'Timeout (ms)', 30000), number('waitForMs', 'Wait after action (ms)')] },
  getCookie: { title: 'Get Cookie', icon: Cookie, color: 'text-teal-300', fields: [text('name', 'Cookie name'), text('domain', 'Domain'), text('path', 'Path'), text('storeAs', 'Store result as'), number('timeoutMs', 'Timeout (ms)', 30000)] },
  clearCookie: { title: 'Clear Cookie', icon: Cookie, color: 'text-rose-300', fields: [toggle('clearAll', 'Remove every cookie in the current context'), { ...text('name', 'Cookie name', 'authToken'), when: (p) => !p.clearAll }, { ...url, when: (p) => !p.clearAll }, { ...text('domain', 'Domain'), when: (p) => !p.clearAll }, { ...text('path', 'Path'), when: (p) => !p.clearAll }, number('timeoutMs', 'Timeout (ms)', 30000), number('waitForMs', 'Wait after action (ms)')] },
  setStorage: { title: 'Set Storage', icon: HardDriveUpload, color: 'text-violet-300', fields: [select('storageType', 'Storage', 'localStorage|sessionStorage'), text('key', 'Key'), select('valueType', 'Value type', 'text|json'), area('value', 'Value'), number('timeoutMs', 'Timeout (ms)', 15000), number('waitForMs', 'Wait after action (ms)')] },
  getStorage: { title: 'Get Storage', icon: HardDriveDownload, color: 'text-teal-300', fields: [select('storageType', 'Storage', 'localStorage|sessionStorage'), text('key', 'Key'), text('storeAs', 'Store result as'), number('timeoutMs', 'Timeout (ms)', 15000)] },
  clearStorage: { title: 'Clear Storage', icon: HardDriveUpload, color: 'text-red-300', fields: [select('storageType', 'Storage', 'localStorage|sessionStorage'), toggle('clearAll', 'Remove every entry'), { ...text('key', 'Key'), when: (p) => !p.clearAll }, number('timeoutMs', 'Timeout (ms)', 15000), number('waitForMs', 'Wait after action (ms)')] },
  networkMock: { title: 'Network Mock', icon: Network, color: 'text-fuchsia-300', fields: [select('operation', 'Operation', 'NETWORK_MOCK_OPERATION_ADD|NETWORK_MOCK_OPERATION_REMOVE|NETWORK_MOCK_OPERATION_CLEAR'), text('urlPattern', 'URL pattern'), text('method', 'Method'), number('statusCode', 'Status code', 200, 100, 599), area('body', 'Response body'), number('delayMs', 'Delay (ms)')] },
  type: { title: 'Type Text', icon: Type, color: 'text-cyan-300', fields: [url, selector(), area('value', 'Text'), toggle('isSensitive', 'Sensitive value'), toggle('submit', 'Press Enter after typing'), toggle('clearFirst', 'Clear field first'), number('delayMs', 'Key delay (ms)')] },
  shortcut: { title: 'Shortcut', icon: Keyboard, color: 'text-slate-300', fields: [area('shortcut', 'Shortcuts', 'Ctrl+A, Backspace'), selector('selector', 'Focus selector')] },
  keyboard: { title: 'Keyboard', icon: Keyboard, color: 'text-lime-300', fields: [select('action', 'Action', 'KEY_ACTION_PRESS|KEY_ACTION_DOWN|KEY_ACTION_UP|KEY_ACTION_TYPE'), text('key', 'Key'), area('keys', 'Keys', 'One key per line'), area('modifiers', 'Modifiers', 'One per line')] },
  evaluate: { title: 'Script', icon: TerminalSquare, color: 'text-teal-300', fields: [area('expression', 'JavaScript', 'return document.title;', 5), text('storeResult', 'Store result as')] },
  screenshot: { title: 'Screenshot', icon: Camera, color: 'text-blue-300', fields: [toggle('fullPage', 'Full page'), selector(), number('quality', 'JPEG quality', 90, 1, 100)] },
  wait: { title: 'Wait', icon: Clock, color: 'text-gray-400', fields: [{ ...number('durationMs', 'Milliseconds', 1000, 0, undefined), when: (p) => p.waitType === 'time' }, { ...selector(), when: (p) => p.waitType !== 'time' }, number('timeoutMs', 'Timeout (ms)', 30000)] },
  extract: { title: 'Extract Data', icon: Database, color: 'text-pink-400', fields: [selector(), select('extractType', 'Extract', 'EXTRACT_TYPE_TEXT|EXTRACT_TYPE_ATTRIBUTE|EXTRACT_TYPE_VALUE|EXTRACT_TYPE_HTML'), text('attributeName', 'Attribute name'), text('propertyName', 'Property name'), text('storeAs', 'Store result as'), number('timeoutMs', 'Timeout (ms)', 5000)] },
  assert: { title: 'Assert', icon: ShieldCheck, color: 'text-emerald-300', fields: [text('label', 'Label'), selector(), select('mode', 'Assertion', 'ASSERTION_MODE_EXISTS|ASSERTION_MODE_NOT_EXISTS|ASSERTION_MODE_EQUALS|ASSERTION_MODE_CONTAINS|ASSERTION_MODE_VISIBLE'), { ...area('expected', 'Expected value'), kind: 'json' }, toggle('negated', 'Negate assertion'), toggle('caseSensitive', 'Case sensitive'), text('attributeName', 'Attribute name'), text('failureMessage', 'Failure message'), number('timeoutMs', 'Timeout (ms)', 5000)] },
  useVariable: { title: 'Use Variable', icon: Recycle, color: 'text-sky-300', fields: [text('name', 'Variable name'), text('storeAs', 'Store as'), area('transform', 'Transform'), toggle('required', 'Fail if variable is missing')] },
  subflow: { title: 'Subflow', icon: Play, color: 'text-violet-400', className: 'w-80', fields: [text('workflowId', 'Workflow ID'), text('workflowPath', 'Workflow path'), number('workflowVersion', 'Workflow version', 1, 1)] },
  loop: { title: 'Loop', icon: RefreshCcw, color: 'text-slate-200', handles: 'loop', fields: [select('loopType', 'Loop type', 'LOOP_TYPE_FOREACH|LOOP_TYPE_REPEAT|LOOP_TYPE_WHILE', ['For Each (array variable)', 'Repeat (fixed count)', 'While (boolean condition)'], 'LOOP_TYPE_FOREACH'), { ...text('arraySource', 'Array variable'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('FOREACH') }, { ...number('count', 'Repeat count', 1, 1), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('REPEAT') }, number('maxIterations', 'Maximum iterations', 100, 1, 1000), { ...text('itemVariable', 'Item variable', 'loop.item'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('FOREACH') }, { ...text('indexVariable', 'Index variable', 'loop.index'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('FOREACH') }, { ...select('condition.type', 'Condition type', 'LOOP_CONDITION_TYPE_VARIABLE|LOOP_CONDITION_TYPE_EXPRESSION'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('WHILE') }, { ...text('condition.variable', 'Condition variable'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('WHILE') && String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') }, { ...select('condition.operator', 'Condition operator', 'LOOP_CONDITION_OPERATOR_TRUTHY|LOOP_CONDITION_OPERATOR_EQUALS|LOOP_CONDITION_OPERATOR_NOT_EQUALS|LOOP_CONDITION_OPERATOR_CONTAINS'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('WHILE') && String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') }, { ...text('condition.value', 'Condition value'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('WHILE') && String(p.conditionType ?? '').toUpperCase().endsWith('VARIABLE') }, { ...area('condition.expression', 'Condition expression'), when: (p) => String(p.loopType ?? '').toUpperCase().endsWith('WHILE') && String(p.conditionType ?? '').toUpperCase().endsWith('EXPRESSION') }, number('iterationTimeoutMs', 'Iteration timeout (ms)', 45000), number('totalTimeoutMs', 'Total timeout (ms)', 300000)] },
  cookieStorage: { title: 'Cookie / Storage', icon: Cookie, color: 'text-amber-300', fields: [select('operation', 'Operation', 'COOKIE_OPERATION_GET|COOKIE_OPERATION_SET|COOKIE_OPERATION_DELETE|COOKIE_OPERATION_CLEAR', ['Get', 'Set', 'Delete', 'Clear'], 'COOKIE_OPERATION_GET'), select('storageType', 'Storage', 'STORAGE_TYPE_COOKIE|STORAGE_TYPE_LOCAL_STORAGE|STORAGE_TYPE_SESSION_STORAGE', ['Cookie', 'localStorage', 'sessionStorage'], 'STORAGE_TYPE_COOKIE'), { ...text('key', 'Name / key'), when: (p) => !String(p.operation ?? '').toUpperCase().endsWith('CLEAR') }, { ...area('value', 'Value'), when: (p) => String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...text('cookieOptions.domain', 'Cookie domain'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...text('cookieOptions.path', 'Cookie path'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...number('cookieOptions.expires', 'Expires (Unix seconds)'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...toggle('cookieOptions.httpOnly', 'HTTP only'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...toggle('cookieOptions.secure', 'Secure'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }, { ...select('cookieOptions.sameSite', 'SameSite', 'COOKIE_SAME_SITE_STRICT|COOKIE_SAME_SITE_LAX|COOKIE_SAME_SITE_NONE'), when: (p) => String(p.storageType ?? '').toUpperCase().endsWith('COOKIE') && String(p.operation ?? '').toUpperCase().endsWith('SET') }] },
};

const dataFields: Record<string, FieldSpec[]> = {
  select: [select('selectBy', 'Select by', 'value|label|index', undefined, 'value'), toggle('multiple', 'Allow multiple'), { ...area('values', 'Values', 'One value per line'), when: (p) => Boolean(p.multiple) }],
  wait: [select('waitType', 'Wait for', 'time|element|navigation', ['Wait for time', 'Wait for element', 'Wait for navigation'], 'time')],
};

const displayType = (type: string | undefined): string => {
  if (!type) return 'navigate';
  if (type === 'input') return 'type';
  if (type === 'evaluate') return 'evaluate';
  if (type === 'selectOption') return 'select';
  if (type === 'useVariable') return 'setVariable';
  if (['setCookie', 'getCookie', 'clearCookie', 'setStorage', 'getStorage', 'clearStorage'].includes(type)) return 'cookieStorage';
  return type;
};

const resolveValue = (current: unknown, fallback: FieldSpec['defaultValue'], kind: FieldKind): string | number | boolean => {
  if (kind === 'checkbox') return typeof current === 'boolean' ? current : Boolean(fallback ?? false);
  if (kind === 'number') return typeof current === 'number' ? current : Number(fallback ?? 0);
  if (kind === 'textarea' && Array.isArray(current)) return current.join('\n');
  return typeof current === 'string' ? current : String(fallback ?? '');
};

const schemaFieldSpec = (field: DescField): FieldSpec => {
  const label = field.localName.replace(/([A-Z])/g, ' $1').replace(/^./, (character) => character.toUpperCase());
  if (field.fieldKind === 'enum') return { key: field.localName, label, kind: 'select' };
  if (field.fieldKind === 'list') return { key: field.localName, label, kind: 'textarea', rows: 2 };
  if (field.fieldKind === 'map' || field.fieldKind === 'message') return { key: field.localName, label, kind: 'json', rows: 3 };
  if (field.scalar === ScalarType.BOOL) return { key: field.localName, label, kind: 'checkbox' };
  if ([ScalarType.INT32, ScalarType.UINT32, ScalarType.SINT32, ScalarType.FIXED32, ScalarType.SFIXED32, ScalarType.INT64, ScalarType.UINT64, ScalarType.SINT64, ScalarType.FIXED64, ScalarType.SFIXED64, ScalarType.FLOAT, ScalarType.DOUBLE].includes(field.scalar)) {
    return { key: field.localName, label, kind: 'number' };
  }
  return { key: field.localName, label, kind: 'text' };
};

const ActionNodeJsonField: FC<{
  value: Record<string, unknown>;
  label: string;
  onCommit: (value: Record<string, unknown>) => void;
}> = ({ value, label, onCommit }) => {
  const field = useSyncedJson(value, { onCommit });
  return <NodeTextArea field={field} label={label} rows={3} />;
};

const ActionNode: FC<NodeProps> = ({ selected, id }) => {
  const { getNodes } = useReactFlow();
  const node = getNodes().find((candidate) => candidate.id === id);
  const action = (node as (typeof node & { action?: ActionDefinition }) | undefined)?.action;
  const { params: typedParams, updateParams } = useActionParams<NonNullable<ActionDefinition[ActionParamsField]>>(id);
  const params = typedParams as Params | undefined;
  const { getValue, updateData } = useNodeData(id);
  const actionNodeType = action?.type ? actionTypeToNodeType(action.type) : undefined;
  const type = displayType(actionNodeType && actionNodeType !== 'unknown' ? actionNodeType : node?.type);
  const spec = specs[type] ?? { title: type, icon: Circle, color: 'text-gray-400', fields: [] };
  const paramsFieldName = action?.type ? getActionParamsFieldName(action.type) : null;
  const paramsMessageField = paramsFieldName
    ? ActionDefinitionSchema.fields.find((item) => item.localName === paramsFieldName && item.fieldKind === 'message')
    : undefined;
  const paramsSchemaFields = paramsMessageField?.fieldKind === 'message' ? paramsMessageField.message.fields : [];
  const schemaField = (key: string): DescField | undefined => {
    const [parentKey = key, nestedKey] = key.split('.');
    const parent = paramsSchemaFields.find((item) => item.localName === parentKey);
    return nestedKey && parent?.fieldKind === 'message'
      ? parent.message.fields.find((item) => item.localName === nestedKey)
      : parent;
  };
  const picker = useElementPicker(id);
  const { effectiveUrl } = useUrlInheritance(id);
  const resilience = useResiliencePanelProps(id);
  const field = <T extends string | number | boolean>(key: string, fallback?: T): UseSyncedFieldResult<T> => {
    const isNodeDataField = (type === 'select' && ['selectBy', 'multiple', 'waitForMs', 'values'].includes(key)) || (type === 'wait' && key === 'waitType') || !schemaField(key);
    const [parentKey = key, nestedKey] = key.split('.');
    const rawValue = isNodeDataField ? getValue<unknown>(key) : nestedKey
      ? (params?.[parentKey] as Record<string, unknown> | undefined)?.[nestedKey]
      : params?.[key];
    const schema = schemaField(key);
    const enumDescriptor = schema?.fieldKind === 'enum' || schema?.fieldKind === 'list' ? schema.enum : undefined;
    const enumValues = enumDescriptor?.values.filter((item) => item.number !== 0) ?? [];
    const enumName = (candidate: unknown): unknown => Array.isArray(candidate) ? candidate.map(enumName) : typeof candidate === 'string' && enumValues.length
      ? enumValues.find((item) => item.name.toLowerCase().endsWith(`_${candidate.toLowerCase()}`))?.name ?? candidate
      : candidate;
    const normalizedValue = enumName(rawValue ?? fallback);
    const displayValue = (schema?.fieldKind === 'list' || (isNodeDataField && key === 'values')) && Array.isArray(normalizedValue) ? normalizedValue.join('\n') : normalizedValue;
    const value = (displayValue ?? '') as T;
    const updateActionField = (resolved: unknown) => {
      if (nestedKey) {
        const parentValue = (params?.[parentKey] as Record<string, unknown> | undefined) ?? {};
        updateParams({ [parentKey]: { ...parentValue, [nestedKey]: resolved } });
        return;
      }
      const oneof = schema?.oneof;
      if (!oneof) {
        updateParams({ [key]: resolved });
        return;
      }
      const siblings = paramsSchemaFields.filter((item) => item.oneof === oneof && item.localName !== key);
      updateParams({ ...Object.fromEntries(siblings.map((item) => [item.localName, undefined])), [key]: resolved });
    };
    return {
      value,
      setValue: (next) => {
        let resolved: unknown = typeof next === 'function' ? (next as (previous: T) => T)(value) : next;
        if ((schema?.fieldKind === 'list' || (isNodeDataField && key === 'values')) && typeof resolved === 'string') resolved = resolved.split('\n').map((item) => item.trim()).filter(Boolean);
        resolved = enumName(resolved);
        if (isNodeDataField) {
          updateData({ [key]: resolved });
          if (type === 'wait' && key === 'waitType') updateParams({ state: resolved === 'navigation' ? 'WAIT_STATE_NAVIGATION' : undefined });
        } else updateActionField(resolved);
      },
      commit: () => undefined,
      isDirty: false,
    };
  };
  const nestedCondition = params?.condition as Record<string, unknown> | undefined;
  const paramsForVisibility = {
    ...(params ?? {}),
    ...(nestedCondition ? { conditionType: nestedCondition.type, conditionVariable: nestedCondition.variable } : {}),
    ...getNodes().find((candidate) => candidate.id === id)?.data,
  };
  const configuredKeys = new Set([...spec.fields, ...(dataFields[type] ?? [])].map((item) => item.key.split('.')[0]));
  const generatedFields = paramsSchemaFields.filter((item) => !configuredKeys.has(item.localName)).map(schemaFieldSpec);
  const visibleFields = [...spec.fields, ...generatedFields, ...(dataFields[type] ?? [])].filter((item) => !item.when || item.when(paramsForVisibility));
  const Icon = spec.icon;
  const renderField = (item: FieldSpec): ReactNode => {
    const kind = item.kind ?? 'text';
    const value = params?.[item.key] ?? getValue<unknown>(item.key);
    const fieldSchema = schemaField(item.key);
    const fieldEnum = fieldSchema?.fieldKind === 'enum' || fieldSchema?.fieldKind === 'list' ? fieldSchema.enum : undefined;
    const schemaOptions = fieldEnum ? fieldEnum.values
      .filter((option) => option.number !== 0)
      .map((option) => ({ value: option.name, label: option.localName.replace(/([A-Z])/g, ' $1').replace(/^./, (character) => character.toUpperCase()) })) : [];
    const defaultValue = item.defaultValue ?? (kind === 'select' ? schemaOptions[0]?.value : undefined);
    const initial = resolveValue(value, defaultValue, kind);
    if (kind === 'json') {
      return <ActionNodeJsonField key={item.key} label={item.label} value={(value && typeof value === 'object' ? value : {}) as Record<string, unknown>} onCommit={(next) => updateParams({ [item.key]: next })} />;
    }
    const state = field(item.key, initial);
    switch (kind) {
      case 'checkbox': return <NodeCheckbox key={item.key} field={state as UseSyncedFieldResult<boolean>} label={item.label} />;
      case 'number': return <NodeNumberField key={item.key} field={state as UseSyncedFieldResult<number>} label={item.label} min={item.min} max={item.max} />;
      case 'select': {
        return <NodeSelectField key={item.key} field={state as UseSyncedFieldResult<string>} label={item.label} options={schemaOptions.length ? schemaOptions : item.options ?? []} />;
      }
      case 'textarea': return <NodeTextArea key={item.key} field={state as UseSyncedFieldResult<string>} label={item.label} rows={item.rows} placeholder={item.placeholder} />;
      case 'selector': return <NodeSelectorField key={item.key} field={state as UseSyncedFieldResult<string>} effectiveUrl={effectiveUrl} label={item.label} placeholder={item.placeholder} onElementSelect={picker.onSelect} />;
      case 'url': return schemaField(item.key)
        ? <NodeTextField key={item.key} field={state as UseSyncedFieldResult<string>} label={item.label} placeholder={item.placeholder} />
        : <NodeUrlField key={item.key} nodeId={id} />;
      default: return <NodeTextField key={item.key} field={state as UseSyncedFieldResult<string>} label={item.label} placeholder={item.placeholder} description={item.description} />;
    }
  };
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''} ${spec.className ?? ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" data-testid="workflow-node-target-handle" />
      {spec.handles === 'loop' ? <>
        <Handle type="source" position={Position.Bottom} id="loopBody" className="node-handle" style={{ left: '30%', background: '#38bdf8' }} />
        <Handle type="source" position={Position.Bottom} id="loopAfter" className="node-handle" style={{ left: '70%', background: '#7c3aed' }} />
        <Handle type="target" position={Position.Left} id="loopContinue" className="node-handle" style={{ background: '#22c55e' }} />
        <Handle type="target" position={Position.Right} id="loopBreak" className="node-handle" style={{ background: '#f87171' }} />
      </> : spec.handles !== 'conditional' && <Handle type="source" position={Position.Bottom} className="node-handle" data-testid="workflow-node-source-handle" />}
      {spec.handles === 'conditional' && <>
        <Handle type="source" position={Position.Bottom} id="ifTrue" className="node-handle" style={{ left: '35%', background: '#22c55e' }} />
        <Handle type="source" position={Position.Bottom} id="ifFalse" className="node-handle" style={{ left: '65%', background: '#ef4444' }} />
      </>}
      <div className="flex items-center gap-2 mb-2"><Icon size={16} className={spec.color} /><span className="font-semibold text-sm">{spec.title}</span></div>
      <div className="space-y-2 text-xs">
        {visibleFields.map(renderField)}
      </div>
      {['click', 'dragDrop', 'type'].includes(type) && <ResiliencePanel {...resilience} />}
    </div>
  );
};

export default memo(ActionNode);
