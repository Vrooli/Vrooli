import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import {
  deleteHandoffRule,
  listHandoffRules,
  upsertHandoffRule,
  type HandoffRuleDTO,
  type RuleSource,
} from "../../api/handoffrules";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";
import { IconButton } from "@vrooli/react-component-library/IconButton";

// [REQ:P0-014h] Handoff Capture Rules

/**
 * The rules editor.
 *
 * The footer states the safety property in plain words, because an operator
 * writing a pattern needs to know the worst case: a wrong rule costs a
 * dismissed chip, never a message delivered to an agent. Without that line
 * the natural fear — "will this send something on its own?" — has no answer
 * on screen, and a cautious operator writes no rules at all.
 */
export default function HandoffRulesPanel() {
  const { t } = useTranslation();
  const [rules, setRules] = useState<HandoffRuleDTO[]>([]);
  const [saving, setSaving] = useState(false);

  const refresh = useCallback(async () => {
    try {
      setRules(await listHandoffRules());
    } catch (error) {
      console.error("Failed to load handoff rules:", error);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  const patch = useCallback(async (rule: HandoffRuleDTO, update: Partial<HandoffRuleDTO>) => {
    const next = { ...rule, ...update };
    // Optimistic: a toggle that waited for a round trip would feel broken.
    setRules((prev) => prev.map((r) => (r.id === rule.id ? next : r)));
    if (!next.name.trim() || !next.pattern.trim()) return;
    try {
      await upsertHandoffRule(next);
    } catch (error) {
      console.error("Failed to save handoff rule:", error);
      await refresh();
    }
  }, [refresh]);

  const create = useCallback(async () => {
    setSaving(true);
    try {
      await upsertHandoffRule({
        name: "New rule",
        // A new rule starts OFF. An enabled rule with a pattern the operator
        // has not finished writing would put a suggestion under every message.
        enabled: false,
        source: "file_path",
        pattern: "**/*.md",
        surfaces: ["messages"],
        sort_order: rules.length,
      });
      await refresh();
    } catch (error) {
      console.error("Failed to create handoff rule:", error);
    } finally {
      setSaving(false);
    }
  }, [refresh, rules.length]);

  const remove = useCallback(async (id: string) => {
    setRules((prev) => prev.filter((r) => r.id !== id));
    try {
      await deleteHandoffRule(id);
    } catch (error) {
      console.error("Failed to delete handoff rule:", error);
      await refresh();
    }
  }, [refresh]);

  return (
    <div data-testid="handoff-rules-panel" className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.tabHandoffRules)}
        title={t(strings.handoffRules.title)}
        description={t(strings.handoffRules.footer)}
      />

      <div className="flex justify-end">
        <Button
          data-testid="handoff-rules-create"
          variant="outline"
          size="sm"
          disabled={saving}
          onClick={() => { void create(); }}
        >
          <Plus className="me-1.5 h-4 w-4" aria-hidden />
          {t(strings.handoffRules.create)}
        </Button>
      </div>

      {rules.length === 0 ? (
        <p data-testid="handoff-rules-empty" className="py-6 text-center text-sm text-wc-text-muted">
          {t(strings.handoffRules.empty)}
        </p>
      ) : (
        <div className="space-y-2">
          {rules.map((rule) => (
            <SettingsCard key={rule.id} className="space-y-3">
              <div data-testid={`handoff-rule-${rule.id}`} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  data-testid={`handoff-rule-toggle-${rule.id}`}
                  aria-label={t(strings.handoffRules.enabled)}
                  checked={rule.enabled}
                  onChange={(event) => { void patch(rule, { enabled: event.target.checked }); }}
                  className="h-4 w-4 shrink-0 accent-[rgb(var(--wc-accent))]"
                />
                <input
                  data-testid={`handoff-rule-name-${rule.id}`}
                  aria-label={t(strings.handoffRules.name)}
                  value={rule.name}
                  onChange={(event) => { void patch(rule, { name: event.target.value }); }}
                  className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none focus:border-wc-accent"
                />
                <IconButton
                  data-testid={`handoff-rule-delete-${rule.id}`}
                  aria-label={t(strings.handoffRules.delete)}
                  surface="danger"
                  className="shrink-0"
                  onClick={() => { void remove(rule.id); }}
                >
                  <Trash2 />
                </IconButton>
              </div>

              <div className="flex flex-col gap-2 sm:flex-row">
                <select
                  data-testid={`handoff-rule-source-${rule.id}`}
                  aria-label={t(strings.handoffRules.source)}
                  value={rule.source}
                  onChange={(event) => { void patch(rule, { source: event.target.value as RuleSource }); }}
                  className="min-h-11 shrink-0 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary outline-none focus:border-wc-accent"
                >
                  <option value="file_path">{t(strings.handoffRules.sourceFilePath)}</option>
                  <option value="message_text">{t(strings.handoffRules.sourceMessageText)}</option>
                </select>
                <input
                  data-testid={`handoff-rule-pattern-${rule.id}`}
                  aria-label={t(strings.handoffRules.pattern)}
                  value={rule.pattern}
                  onChange={(event) => { void patch(rule, { pattern: event.target.value }); }}
                  className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 font-mono text-xs text-wc-text-primary outline-none focus:border-wc-accent"
                />
              </div>

              <p className="text-[11px] text-wc-text-faint">
                {rule.source === "file_path"
                  ? t(strings.handoffRules.patternGlobHint)
                  : t(strings.handoffRules.patternRegexHint)}
              </p>
            </SettingsCard>
          ))}
        </div>
      )}

      <p data-testid="handoff-rules-footer" className="rounded-lg border border-wc-default bg-wc-surface-base/40 px-3 py-2 text-xs text-wc-text-secondary">
        {t(strings.handoffRules.footer)}
      </p>
    </div>
  );
}
