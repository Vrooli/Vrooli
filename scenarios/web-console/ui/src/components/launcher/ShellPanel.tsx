import { Input } from "@vrooli/react-component-library/Input";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { slugify } from "../../lib/slugify";
import type { AgentCard } from "./agentGrid";
import { FOCUS_RING } from "./controls";

/**
 * A plain terminal. It says that agent settings do not apply: typing `claude`
 * here starts it without the drawer's limit or model, and the drawer never
 * implies otherwise.
 */
export function ShellPanel({ command, onCommand, onSubmit, shortcuts, onShortcut, disabled }: {
  command: string;
  onCommand: (value: string) => void;
  onSubmit: () => void;
  shortcuts: readonly AgentCard[];
  onShortcut: (command: string) => void;
  disabled: boolean;
}) {
  const { t } = useTranslation();
  return (
    <section aria-label={t(strings.terminalLauncher.customCommand)} className="grid gap-4 px-4 pt-4" data-testid="launcher-shell-panel">
      <InputGroup testId="launcher-custom-group" size="lg">
        <InputGroup.Adornment side="leading" className="font-mono text-sm">$</InputGroup.Adornment>
        <InputGroup.Field>
          <Input data-testid="launcher-custom-input" type="text" autoComplete="off" spellCheck={false} aria-label={t(strings.terminalLauncher.customCommand)}
            value={command} onChange={(event) => { onCommand(event.target.value); }} onKeyDown={(event) => { if (event.key === "Enter") onSubmit(); }}
            placeholder={t(strings.terminalLauncher.commandPlaceholder)} className="font-mono text-sm text-wc-text-primary placeholder:font-sans placeholder:text-wc-text-faint" />
        </InputGroup.Field>
      </InputGroup>
      {shortcuts.length > 0 && (
        <div>
          <h3 className="mb-1.5 text-xs text-wc-text-muted">{t(strings.terminalLauncher.shortcuts)}</h3>
          <div className="flex flex-wrap gap-[7px]">
            {shortcuts.map((card) => (
              <button key={card.key} type="button" data-testid={`launcher-shortcut-${slugify(card.label)}`} title={card.description || card.command} disabled={disabled}
                onClick={() => { onShortcut(card.command); }}
                className={cn("min-h-11 max-w-full truncate rounded-lg border border-wc-default bg-wc-surface-input px-2.5 font-mono text-[13px] text-wc-text-primary hover:border-wc-accent disabled:opacity-50", FOCUS_RING)}>
                {card.label}
              </button>
            ))}
          </div>
        </div>
      )}
      <p data-testid="launcher-shell-hint" className="text-xs leading-[18px] text-wc-text-muted">{t(strings.terminalLauncher.shellHint)}</p>
    </section>
  );
}
