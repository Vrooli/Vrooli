import { Copy, Play } from "lucide-react";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import type { TerminalCommandActivation } from "../lib/terminalCommandBlocks";
import { copyText } from "../lib/clipboard";

interface TerminalCommandMenuProps {
  activation: TerminalCommandActivation;
  onRun?: (command: string) => void;
  onClose: () => void;
}

function ellipsizeMiddle(value: string, max = 52): string {
  if (value.length <= max) return value;
  const keep = max - 1;
  const head = Math.ceil(keep * 0.4);
  const tail = keep - head;
  return `${value.slice(0, head)}…${value.slice(value.length - tail)}`;
}

export default function TerminalCommandMenu({ activation, onRun, onClose }: TerminalCommandMenuProps) {
  const { value, text, position, lineCount } = activation;

  const items: ContextMenuItem[] = [
    {
      id: "copy-command",
      label: lineCount > 1 ? `Copy ${String(lineCount)} lines` : "Copy command",
      icon: <Copy className="h-4 w-4 shrink-0" />,
      testId: "command-copy",
      onSelect: () => { void copyText(value); },
    },
    ...(onRun ? [{
      id: "run-command",
      label: "Run command",
      icon: <Play className="h-4 w-4 shrink-0" />,
      testId: "command-run",
      onSelect: () => { onRun(value); },
    }] : []),
  ];

  return (
    <ContextMenu
      open
      position={position}
      title={ellipsizeMiddle(text)}
      closeLabel="Close"
      testId="terminal-command-menu"
      onOpenChange={(next) => { if (!next) onClose(); }}
      items={items}
    />
  );
}
