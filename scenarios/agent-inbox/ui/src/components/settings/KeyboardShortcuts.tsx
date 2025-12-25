import { Command, Navigation, MessageSquare } from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";

interface ShortcutItem {
  keys: string[];
  description: string;
}

interface ShortcutCategory {
  title: string;
  icon: React.ReactNode;
  shortcuts: ShortcutItem[];
}

const isMac = typeof navigator !== "undefined" && navigator.platform.includes("Mac");
const modKey = isMac ? "Cmd" : "Ctrl";

const shortcutCategories: ShortcutCategory[] = [
  {
    title: "Navigation",
    icon: <Navigation className="h-4 w-4" />,
    shortcuts: [
      { keys: [modKey, "K"], description: "Focus search" },
      { keys: [modKey, "1"], description: "Go to Inbox" },
      { keys: [modKey, "2"], description: "Go to Starred" },
      { keys: [modKey, "3"], description: "Go to Archived" },
      { keys: ["Esc"], description: "Close dialog / deselect chat" },
    ],
  },
  {
    title: "Chat",
    icon: <MessageSquare className="h-4 w-4" />,
    shortcuts: [
      { keys: [modKey, "N"], description: "New chat" },
      { keys: ["Enter"], description: "Send message" },
      { keys: ["Shift", "Enter"], description: "New line in message" },
      { keys: [modKey, "S"], description: "Toggle star on current chat" },
      { keys: [modKey, "E"], description: "Archive current chat" },
    ],
  },
  {
    title: "General",
    icon: <Command className="h-4 w-4" />,
    shortcuts: [
      { keys: [modKey, ","], description: "Open settings" },
      { keys: ["?"], description: "Show keyboard shortcuts" },
    ],
  },
];

interface KeyboardShortcutsProps {
  open: boolean;
  onClose: () => void;
}

export function KeyboardShortcuts({ open, onClose }: KeyboardShortcutsProps) {
  return (
    <Dialog open={open} onClose={onClose} className="max-w-lg">
      <DialogHeader onClose={onClose}>Keyboard Shortcuts</DialogHeader>
      <DialogBody className="space-y-6 max-h-[60vh] overflow-y-auto">
        {shortcutCategories.map((category) => (
          <section key={category.title}>
            <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
              {category.icon}
              {category.title}
            </h3>
            <div className="space-y-2">
              {category.shortcuts.map((shortcut, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between py-2 px-3 rounded-lg bg-white/5"
                >
                  <span className="text-sm text-slate-400">{shortcut.description}</span>
                  <div className="flex items-center gap-1">
                    {shortcut.keys.map((key, keyIndex) => (
                      <span key={keyIndex} className="flex items-center gap-1">
                        {keyIndex > 0 && <span className="text-slate-600">+</span>}
                        <kbd className="px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 font-mono min-w-[24px] text-center">
                          {key}
                        </kbd>
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </section>
        ))}
        <p className="text-xs text-slate-600 text-center pt-2">
          Shortcuts are disabled when typing in input fields
        </p>
      </DialogBody>
    </Dialog>
  );
}
