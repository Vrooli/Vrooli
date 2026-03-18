import { Moon, Sun, Keyboard, MessageCircle, AlignLeft } from "lucide-react";
import { Button } from "../ui/button";
import { SettingsSection } from "./SettingsControls";
import type { Theme, ViewMode } from "./settingsTypes";

interface ChoiceButtonProps {
  label: string;
  icon: React.ReactNode;
  selected: boolean;
  onClick: () => void;
  testId?: string;
}

function ChoiceButton({ label, icon, selected, onClick, testId }: ChoiceButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
        selected
          ? "bg-indigo-500/20 border-indigo-500 text-white"
          : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
      }`}
      data-testid={testId}
    >
      {icon}
      <span className="text-sm">{label}</span>
    </button>
  );
}

interface GeneralSettingsTabProps {
  theme: Theme;
  onThemeChange: (theme: Theme) => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onShowKeyboardShortcuts: () => void;
  onClose: () => void;
}

export function GeneralSettingsTab({
  theme,
  onThemeChange,
  viewMode,
  onViewModeChange,
  onShowKeyboardShortcuts,
  onClose,
}: GeneralSettingsTabProps) {
  return (
    <div className="space-y-6">
      <SettingsSection title="Appearance">
        <div className="flex gap-2">
          <ChoiceButton
            label="Dark"
            icon={<Moon className="h-4 w-4" />}
            selected={theme === "dark"}
            onClick={() => onThemeChange("dark")}
            testId="theme-dark-button"
          />
          <ChoiceButton
            label="Light"
            icon={<Sun className="h-4 w-4" />}
            selected={theme === "light"}
            onClick={() => onThemeChange("light")}
            testId="theme-light-button"
          />
        </div>
      </SettingsSection>

      <SettingsSection
        title="Chat View"
        description="Choose how messages are displayed in conversations"
      >
        <div className="flex gap-2">
          <ChoiceButton
            label="Bubble"
            icon={<MessageCircle className="h-4 w-4" />}
            selected={viewMode === "bubble"}
            onClick={() => onViewModeChange("bubble")}
            testId="view-mode-bubble-button"
          />
          <ChoiceButton
            label="Compact"
            icon={<AlignLeft className="h-4 w-4" />}
            selected={viewMode === "compact"}
            onClick={() => onViewModeChange("compact")}
            testId="view-mode-compact-button"
          />
        </div>
        <p className="text-xs text-slate-600 mt-2">
          Compact mode uses full width, ideal for code-heavy conversations
        </p>
      </SettingsSection>

      <SettingsSection title="Keyboard">
        <Button
          variant="secondary"
          onClick={() => {
            onShowKeyboardShortcuts();
            onClose();
          }}
          className="w-full justify-start gap-2"
          data-testid="keyboard-shortcuts-button"
        >
          <Keyboard className="h-4 w-4" />
          View Keyboard Shortcuts
        </Button>
      </SettingsSection>
    </div>
  );
}
