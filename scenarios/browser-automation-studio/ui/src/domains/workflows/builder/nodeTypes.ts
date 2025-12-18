import type { NodeTypes } from 'reactflow';
import AssertNode from '../nodes/AssertNode';
import BlurNode from '../nodes/BlurNode';
import BrowserActionNode from '../nodes/BrowserActionNode';
import ClearCookieNode from '../nodes/ClearCookieNode';
import ClearStorageNode from '../nodes/ClearStorageNode';
import ClickNode from '../nodes/ClickNode';
import ConditionalNode from '../nodes/ConditionalNode';
import DragDropNode from '../nodes/DragDropNode';
import ExtractNode from '../nodes/ExtractNode';
import FocusNode from '../nodes/FocusNode';
import FrameSwitchNode from '../nodes/FrameSwitchNode';
import GestureNode from '../nodes/GestureNode';
import GetCookieNode from '../nodes/GetCookieNode';
import GetStorageNode from '../nodes/GetStorageNode';
import HoverNode from '../nodes/HoverNode';
import KeyboardNode from '../nodes/KeyboardNode';
import LoopNode from '../nodes/LoopNode';
import NavigateNode from '../nodes/NavigateNode';
import NetworkMockNode from '../nodes/NetworkMockNode';
import RotateNode from '../nodes/RotateNode';
import ScreenshotNode from '../nodes/ScreenshotNode';
import ScriptNode from '../nodes/ScriptNode';
import ScrollNode from '../nodes/ScrollNode';
import SelectNode from '../nodes/SelectNode';
import SetCookieNode from '../nodes/SetCookieNode';
import SetStorageNode from '../nodes/SetStorageNode';
import SetVariableNode from '../nodes/SetVariableNode';
import ShortcutNode from '../nodes/ShortcutNode';
import SubflowNode from '../nodes/SubflowNode';
import TabSwitchNode from '../nodes/TabSwitchNode';
import TypeNode from '../nodes/TypeNode';
import UploadFileNode from '../nodes/UploadFileNode';
import UseVariableNode from '../nodes/UseVariableNode';
import WaitNode from '../nodes/WaitNode';

export const nodeTypes: NodeTypes = {
  browserAction: BrowserActionNode,
  navigate: NavigateNode,
  click: ClickNode,
  hover: HoverNode,
  dragDrop: DragDropNode,
  focus: FocusNode,
  blur: BlurNode,
  scroll: ScrollNode,
  select: SelectNode,
  uploadFile: UploadFileNode,
  rotate: RotateNode,
  gesture: GestureNode,
  tabSwitch: TabSwitchNode,
  frameSwitch: FrameSwitchNode,
  conditional: ConditionalNode,
  setVariable: SetVariableNode,
  setCookie: SetCookieNode,
  getCookie: GetCookieNode,
  clearCookie: ClearCookieNode,
  setStorage: SetStorageNode,
  getStorage: GetStorageNode,
  clearStorage: ClearStorageNode,
  networkMock: NetworkMockNode,
  type: TypeNode,
  shortcut: ShortcutNode,
  keyboard: KeyboardNode,
  evaluate: ScriptNode,
  screenshot: ScreenshotNode,
  wait: WaitNode,
  extract: ExtractNode,
  assert: AssertNode,
  useVariable: UseVariableNode,
  subflow: SubflowNode,
  loop: LoopNode,
};

// Edge marker colors by theme - darker for light mode, lighter for dark mode
export const EDGE_MARKER_COLORS = {
  dark: '#6b7280', // gray-500 - visible on dark backgrounds
  light: '#4b5563', // gray-600 - visible on light backgrounds
} as const;
