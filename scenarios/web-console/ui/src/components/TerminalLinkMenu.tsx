import { Copy, ExternalLink, FileText } from "lucide-react";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import type { TerminalLinkActivation } from "../lib/terminalFileLinks";
import { copyText } from "../lib/clipboard";

interface TerminalLinkMenuProps {
  /** The link the user activated, with the pointer position to anchor to. */
  activation: TerminalLinkActivation;
  /** Open a detected file path in the in-app preview. */
  onOpenFile: (path: string) => void;
  onClose: () => void;
}

/**
 * Action menu for a clicked terminal link. Rather than a single hard-to-hit
 * action on precise underlined cells, activating a link (now spanning its full
 * wrapped span) opens this menu so Open and Copy are both one unambiguous tap —
 * the only workable model on touch, where there is no hover underline.
 */
/** Middle-truncate a long link so both ends stay readable in the menu title —
 *  the drawer header has no room for a full path, and the tail (the filename)
 *  is the part the operator most needs to see. */
function ellipsizeMiddle(value: string, max = 44): string {
  if (value.length <= max) return value;
  const keep = max - 1;
  const head = Math.ceil(keep * 0.4);
  const tail = keep - head;
  return `${value.slice(0, head)}…${value.slice(value.length - tail)}`;
}

export default function TerminalLinkMenu({ activation, onOpenFile, onClose }: TerminalLinkMenuProps) {
  const { kind, value, text, position } = activation;

  const items: ContextMenuItem[] = [
    kind === "file"
      ? {
          id: "open-file",
          label: "Open file",
          icon: <FileText className="h-4 w-4 shrink-0" />,
          testId: "link-open-file",
          onSelect: () => { onOpenFile(value); },
        }
      : {
          id: "open-url",
          label: "Open link",
          icon: <ExternalLink className="h-4 w-4 shrink-0" />,
          testId: "link-open-url",
          onSelect: () => { window.open(value, "_blank", "noopener,noreferrer"); },
        },
    {
      id: "copy",
      label: kind === "file" ? "Copy path" : "Copy link",
      icon: <Copy className="h-4 w-4 shrink-0" />,
      testId: "link-copy",
      onSelect: () => { void copyText(value); },
    },
  ];

  return (
    <ContextMenu
      open
      position={position}
      title={ellipsizeMiddle(text)}
      closeLabel="Close"
      testId="terminal-link-menu"
      onOpenChange={(next) => { if (!next) onClose(); }}
      items={items}
    />
  );
}
