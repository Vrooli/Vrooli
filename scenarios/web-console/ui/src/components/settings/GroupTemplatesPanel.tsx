import { useCallback, useEffect, useState } from "react";
import { GripVertical, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MasterDetail } from "@vrooli/react-component-library/MasterDetail/1";
import { Sortable } from "@vrooli/react-component-library/Sortable/1";

import {
  deleteGroupTemplate,
  listGroupTemplates,
  upsertGroupTemplate,
  type GroupTemplateDTO,
  type StartMode,
  type TemplateRoleDTO,
} from "../../api/grouptemplates";
import { HEADER_COLORS } from "../../consts/config";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";

// [REQ:P0-014g] Group Templates

const emptyRole = (): TemplateRoleDTO => ({
  label: "",
  command: "",
  working_dir: "",
  incoming_prompt: "",
  backend: "",
  target_id: "",
  // A new role WAITS. Defaulting to eager would let adding a role to a
  // template quietly cost a process on the next use.
  start_mode: "waiting",
});

/**
 * The template manager.
 *
 * Every template here is an ordinary row, including the shipped example: the
 * delete control on it is the same control as on one the operator wrote, with
 * no guard and no warning that it is special, because it is not.
 */
export default function GroupTemplatesPanel() {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState<GroupTemplateDTO[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [draft, setDraft] = useState<GroupTemplateDTO | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "request-error">("loading");
  const [saving, setSaving] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const next = await listGroupTemplates();
      setTemplates(next);
      setStatus("ready");
      return next;
    } catch (error) {
      console.error("Failed to load group templates:", error);
      setStatus("request-error");
      return [];
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  useEffect(() => {
    setDraft(templates.find((tpl) => tpl.id === selectedId) ?? null);
  }, [selectedId, templates]);

  const save = useCallback(async (next: GroupTemplateDTO) => {
    if (!next.name.trim()) return;
    setSaving(true);
    try {
      // use_count is deliberately omitted: editing content must not reset how
      // often the template has been used.
      await upsertGroupTemplate({
        id: next.id || undefined,
        name: next.name,
        color: next.color,
        roles: next.roles,
      });
      const refreshed = await refresh();
      if (!next.id) setSelectedId(refreshed.find((tpl) => tpl.name === next.name)?.id ?? null);
    } catch (error) {
      console.error("Failed to save group template:", error);
    } finally {
      setSaving(false);
    }
  }, [refresh]);

  const remove = useCallback(async (id: string) => {
    try {
      await deleteGroupTemplate(id);
      setSelectedId((prev) => (prev === id ? null : prev));
      await refresh();
    } catch (error) {
      console.error("Failed to delete group template:", error);
    }
  }, [refresh]);

  const create = useCallback(() => {
    setSelectedId(null);
    setDraft({ id: "", name: "", color: HEADER_COLORS[0], roles: [emptyRole()], use_count: 0 });
  }, []);

  const patchRole = (index: number, patch: Partial<TemplateRoleDTO>) => {
    setDraft((prev) => prev && ({
      ...prev,
      roles: prev.roles.map((role, i) => (i === index ? { ...role, ...patch } : role)),
    }));
  };

  const items = templates.map((tpl) => ({
    id: tpl.id,
    title: tpl.name,
    summary: t(strings.groupTemplates.roles) + `: ${String(tpl.roles.length)}`,
    meta: t(strings.groupTemplates.usedCount, { count: tpl.use_count }),
    value: tpl,
  }));

  return (
    <div data-testid="group-templates-panel" className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.tabTemplates)}
        title={t(strings.groupTemplates.title)}
        description={t(strings.groupTemplates.reorderHint)}
      />

      <div className="flex justify-end">
        <Button data-testid="group-templates-create" variant="outline" size="sm" onClick={create}>
          <Plus className="me-1.5 h-4 w-4" aria-hidden />
          {t(strings.groupTemplates.create)}
        </Button>
      </div>

      {templates.length === 0 && status === "ready" && !draft && (
        <p data-testid="group-templates-empty" className="py-6 text-center text-sm text-wc-text-muted">
          {t(strings.groupTemplates.empty)}
        </p>
      )}

      <MasterDetail
        items={items}
        selectedId={selectedId}
        onSelect={(item) => { setSelectedId(item.id); }}
        status={status === "loading" ? "loading" : status === "request-error" ? "request-error" : "default"}
        label={t(strings.groupTemplates.title)}
        renderMaster={(item, state) => (
          <div data-testid={`group-template-${item.id}`} className="flex items-center gap-2">
            <button type="button" className="min-w-0 flex-1 text-start" onClick={state.select}>
              <span className="block truncate text-sm text-wc-text-primary">{item.title}</span>
              <span className="block truncate text-[11px] text-wc-text-faint">{item.summary}</span>
            </button>
            <button
              type="button"
              data-testid={`group-template-delete-${item.id}`}
              aria-label={t(strings.groupTemplates.delete)}
              title={t(strings.groupTemplates.delete)}
              className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
              onClick={() => { void remove(item.id); }}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        )}
        renderDetail={() => null}
      />

      {draft && (
        <SettingsCard className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="template-name" className="text-xs font-medium text-wc-text-secondary">
              {t(strings.groupTemplates.name)}
            </label>
            <input
              id="template-name"
              data-testid="group-template-name"
              value={draft.name}
              onChange={(event) => { setDraft({ ...draft, name: event.target.value }); }}
              className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none focus:border-wc-accent"
            />
          </div>

          <div className="space-y-1.5">
            <span className="text-xs font-medium text-wc-text-secondary">{t(strings.groupTemplates.color)}</span>
            <div className="flex flex-wrap gap-1.5">
              {HEADER_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  data-testid={`group-template-color-${color}`}
                  className={cn(
                    "h-5 w-5 rounded-full border border-wc-default transition hover:scale-110",
                    color === draft.color && "ring-2 ring-wc-accent",
                  )}
                  style={{ backgroundColor: color }}
                  onClick={() => { setDraft({ ...draft, color }); }}
                />
              ))}
            </div>
          </div>

          <div data-testid="group-templates-role-list" className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-wc-text-secondary">{t(strings.groupTemplates.roles)}</span>
              <button
                type="button"
                data-testid="group-templates-role-add"
                onClick={() => { setDraft({ ...draft, roles: [...draft.roles, emptyRole()] }); }}
                className="min-h-11 rounded-lg px-2 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
              >
                {t(strings.groupTemplates.addRole)}
              </button>
            </div>

            {/* Order is content: it is the order roles appear in a new group. */}
            <Sortable
              label={t(strings.groupTemplates.roles)}
              items={draft.roles.map((role, index) => ({ id: `role-${String(index)}`, value: role, label: role.label }))}
              onReorder={(reordered) => {
                setDraft({ ...draft, roles: reordered.map((item) => item.value) });
              }}
              renderItem={(item, state) => {
                const index = state.index;
                const role = item.value;
                return (
                  <div className="space-y-2 rounded-lg border border-wc-default bg-wc-surface-base/40 p-3">
                    <div className="flex items-center gap-2">
                      <GripVertical className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
                      <input
                        data-testid={`group-template-role-label-${String(index)}`}
                        value={role.label}
                        placeholder={t(strings.roles.roleLabelPlaceholder)}
                        onChange={(event) => { patchRole(index, { label: event.target.value }); }}
                        className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none focus:border-wc-accent"
                      />
                      <select
                        data-testid={`group-template-role-mode-${String(index)}`}
                        aria-label={t(strings.groupTemplates.startMode)}
                        value={role.start_mode}
                        onChange={(event) => { patchRole(index, { start_mode: event.target.value as StartMode }); }}
                        className="min-h-11 shrink-0 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary outline-none focus:border-wc-accent"
                      >
                        <option value="eager">{t(strings.groupTemplates.startModeEager)}</option>
                        <option value="waiting">{t(strings.groupTemplates.startModeWaiting)}</option>
                      </select>
                      <button
                        type="button"
                        data-testid={`group-template-role-remove-${String(index)}`}
                        aria-label={t(strings.groupTemplates.removeRole)}
                        onClick={() => {
                          setDraft({ ...draft, roles: draft.roles.filter((_, i) => i !== index) });
                        }}
                        className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>

                    <input
                      data-testid={`group-template-role-command-${String(index)}`}
                      value={role.command}
                      placeholder={t(strings.roles.roleCommandPlaceholder)}
                      onChange={(event) => { patchRole(index, { command: event.target.value }); }}
                      className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 font-mono text-xs text-wc-text-primary outline-none focus:border-wc-accent"
                    />

                    <div className="space-y-1">
                      <textarea
                        data-testid={`group-template-role-prompt-${String(index)}`}
                        value={role.incoming_prompt}
                        rows={2}
                        placeholder={t(strings.roles.roleIncomingPrompt)}
                        onChange={(event) => { patchRole(index, { incoming_prompt: event.target.value }); }}
                        className="w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-xs text-wc-text-primary outline-none focus:border-wc-accent"
                      />
                      <p className="text-[11px] text-wc-text-faint">
                        {t(strings.roles.roleIncomingPromptHint)}
                      </p>
                    </div>
                  </div>
                );
              }}
            />
          </div>

          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => { setDraft(null); setSelectedId(null); }}>
              {t(strings.roles.cancel)}
            </Button>
            <Button
              data-testid="group-template-save"
              size="sm"
              disabled={saving || !draft.name.trim()}
              onClick={() => { void save(draft); }}
            >
              {t(strings.roles.save)}
            </Button>
          </div>
        </SettingsCard>
      )}
    </div>
  );
}
