import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Check, ChevronDown, Clock, GripVertical, Loader2, Pencil, Play, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";

import { listGroupTemplates, type GroupTemplateDTO, type StartMode, type TemplateRoleDTO } from "../../api/grouptemplates";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { Button } from "../ui/button";

// [REQ:P0-014g] Group Templates

/** What the launcher hands back when the operator creates a group. */
export interface GroupCreationRequest {
  name: string;
  color: string;
  roles: TemplateRoleDTO[];
  /** The template this came from, so its use count can move. */
  templateId?: string;
}

interface GroupModePanelProps {
  open: boolean;
  onCreate: (request: GroupCreationRequest) => void;
  isCreating: boolean;
  disabled: boolean;
  /**
   * The machine trigger, rendered beside the template trigger.
   *
   * Template and machine are the two one-line decisions this mode opens with,
   * so they share a row. The launcher owns machine state, so it passes the
   * control in rather than this panel reaching for it.
   */
  machineSlot?: ReactNode;
  /** Cancel the whole dialog — the footer's left-hand escape. */
  onCancel?: () => void;
  /** Opens the template editor, which owns the saved recipe. */
  onEditTemplates?: () => void;
}

const blankRole = (): TemplateRoleDTO => ({
  label: "",
  command: "",
  working_dir: "",
  incoming_prompt: "",
  backend: "",
  target_id: "",
  start_mode: "waiting",
});

const ACCENT = "rgb(var(--wc-accent))";

/**
 * A field that reads as text until you touch it.
 *
 * The role list used to be three boxed inputs squeezed onto one line, which
 * made a role look like a form row and left each control too small to hit.
 * These are full-width, borderless at rest, and outlined on focus — the row
 * reads as "Planner · claude · receives …" and edits in place.
 */
function InlineField({
  testId,
  value,
  placeholder,
  ariaLabel,
  onChange,
  mono = false,
  className,
}: {
  testId: string;
  value: string;
  placeholder: string;
  ariaLabel: string;
  onChange: (next: string) => void;
  mono?: boolean;
  className?: string;
}) {
  return (
    <input
      data-testid={testId}
      value={value}
      aria-label={ariaLabel}
      placeholder={placeholder}
      onChange={(event) => { onChange(event.target.value); }}
      className={cn(
        "min-h-11 w-full min-w-0 rounded-lg border border-transparent bg-transparent px-2 text-wc-text-primary outline-none transition placeholder:text-wc-text-faint hover:border-wc-default focus:border-wc-accent focus:bg-wc-surface-input",
        mono ? "font-mono text-xs" : "text-sm",
        className,
      )}
    />
  );
}

/** The template chooser: a trigger and a menu, matching the machine picker. */
function TemplatePicker({
  templates,
  templateId,
  onSelect,
}: {
  templates: GroupTemplateDTO[];
  templateId: string;
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const selected = templates.find((tpl) => tpl.id === templateId) ?? null;

  useEscapeKey(open, () => { setOpen(false); });

  return (
    <div className="relative min-w-0 flex-1 basis-[13rem]">
      <button
        type="button"
        id="launcher-template-picker"
        data-testid="launcher-template-picker"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`${t(strings.launcher.template)}: ${selected ? selected.name : t(strings.launcher.noTemplate)}`}
        onClick={() => { setOpen((prev) => !prev); }}
        className="flex min-h-11 w-full items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-start text-sm transition hover:border-wc-accent"
        style={selected?.color ? { borderColor: selected.color } : undefined}
      >
        <span
          className="h-3 w-3 shrink-0 rounded-full border border-wc-default"
          style={{ backgroundColor: selected?.color ?? "transparent" }}
          aria-hidden
        />
        <span className="min-w-0 flex-1 truncate text-wc-text-primary">
          {selected ? selected.name : t(strings.launcher.noTemplate)}
        </span>
        <ChevronDown className={cn("h-4 w-4 shrink-0 text-wc-text-faint transition", open && "rotate-180")} aria-hidden />
      </button>

      {open && (
        <div
          role="listbox"
          aria-label={t(strings.launcher.template)}
          data-testid="launcher-template-menu"
          className="absolute inset-x-0 top-full z-30 mt-1 max-h-64 overflow-y-auto rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl"
        >
          {[null, ...templates].map((tpl) => {
            const id = tpl?.id ?? "";
            const isSelected = id === templateId;
            return (
              <button
                key={id || "none"}
                type="button"
                role="option"
                aria-selected={isSelected}
                data-testid={`launcher-template-option-${id || "none"}`}
                onClick={() => { onSelect(id); setOpen(false); }}
                className={cn(
                  "flex min-h-11 w-full items-center gap-2.5 px-3 py-2 text-start transition",
                  isSelected ? "bg-wc-accent/10" : "hover:bg-wc-surface-input",
                )}
              >
                <span
                  className="h-3 w-3 shrink-0 rounded-full border border-wc-default"
                  style={{ backgroundColor: tpl?.color ?? "transparent" }}
                  aria-hidden
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm text-wc-text-primary">{tpl ? tpl.name : t(strings.launcher.noTemplate)}</span>
                  {tpl && (
                    <span className="block truncate text-[11px] text-wc-text-faint">
                      {t(strings.groupTemplates.roles)}: {tpl.roles.length}
                    </span>
                  )}
                </span>
                {isSelected && <Check className="h-4 w-4 shrink-0 text-wc-accent" aria-hidden />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

/**
 * Create a whole group in one action.
 *
 * The role list here is a working copy. Editing it changes THIS group and
 * never the stored template — a template is a starting point, and an operator
 * who drops a role for one piece of work has not asked to change the recipe.
 * The panel says so on screen rather than leaving it to be discovered.
 */
export default function GroupModePanel({
  open,
  onCreate,
  isCreating,
  disabled,
  machineSlot,
  onCancel,
  onEditTemplates,
}: GroupModePanelProps) {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState<GroupTemplateDTO[]>([]);
  const [templateId, setTemplateId] = useState<string>("");
  const [name, setName] = useState("");
  const [roles, setRoles] = useState<TemplateRoleDTO[]>([]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    listGroupTemplates()
      .then((next) => { if (!cancelled) setTemplates(next); })
      .catch((error: unknown) => { console.error("Failed to load group templates:", error); });
    return () => { cancelled = true; };
  }, [open]);

  const template = useMemo(
    () => templates.find((tpl) => tpl.id === templateId) ?? null,
    [templateId, templates],
  );

  useEffect(() => {
    // Selecting a template seeds the working copy. It does not bind to it.
    setRoles(template ? template.roles.map((role) => ({ ...role })) : []);
  }, [template]);

  const patchRole = (index: number, patch: Partial<TemplateRoleDTO>) => {
    setRoles((prev) => prev.map((role, i) => (i === index ? { ...role, ...patch } : role)));
  };

  const rail = template?.color ?? ACCENT;
  const canCreate = name.trim().length > 0 && !isCreating && !disabled;

  return (
    <div className="space-y-4">
      {/* Template and machine: the two one-line decisions, on one line. */}
      <div className="flex flex-wrap items-center gap-2">
        <TemplatePicker templates={templates} templateId={templateId} onSelect={setTemplateId} />
        {machineSlot}
      </div>

      {/* The name is the one thing an operator must type, so it gets the
          full width and a real label rather than sharing a two-up grid. */}
      <div className="space-y-1.5">
        <label htmlFor="launcher-group-name" className="block text-sm font-medium text-wc-text-primary">
          {t(strings.launcher.nameThisWork)}
        </label>
        <input
          id="launcher-group-name"
          data-testid="launcher-group-name"
          value={name}
          onChange={(event) => { setName(event.target.value); }}
          placeholder={t(strings.launcher.groupNamePlaceholder)}
          className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none placeholder:text-wc-text-faint focus:border-wc-accent"
        />
        <p className="text-[11px] text-wc-text-faint">
          {t(strings.launcher.groupName)} · {t(strings.launcher.colorAutomatic)}
        </p>
      </div>

      <div className="border-t border-wc-default" />

      <div data-testid="launcher-group-role-list" className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
            {t(strings.launcher.roleList)}
          </span>
          <button
            type="button"
            data-testid="launcher-group-role-add"
            onClick={() => { setRoles((prev) => [...prev, blankRole()]); }}
            className="inline-flex min-h-11 items-center gap-1 rounded-lg px-2 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
          >
            <Plus className="h-3.5 w-3.5" aria-hidden />
            {t(strings.launcher.addRole)}
          </button>
        </div>

        {roles.length === 0 && (
          <p data-testid="launcher-group-role-empty" className="rounded-lg border border-dashed border-wc-default px-3 py-6 text-center text-sm text-wc-text-muted">
            {t(strings.launcher.noRoles)}
          </p>
        )}

        {roles.map((role, index) => {
          const waiting = role.start_mode !== "eager";
          return (
            <div
              key={index}
              data-testid={`launcher-group-role-${String(index)}`}
              className={cn(
                "rounded-xl border bg-wc-surface-base/40 py-1.5 pe-2 ps-1.5",
                waiting ? "border-dashed border-wc-default" : "border-wc-default",
              )}
              style={{ borderInlineStartWidth: 3, borderInlineStartColor: rail, borderInlineStartStyle: "solid" }}
            >
              <div className="flex items-center gap-1">
                <GripVertical className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
                <InlineField
                  testId={`launcher-group-role-label-${String(index)}`}
                  value={role.label}
                  placeholder={t(strings.roles.roleLabelPlaceholder)}
                  ariaLabel={t(strings.roles.roleLabel)}
                  onChange={(next) => { patchRole(index, { label: next }); }}
                  className="font-medium"
                />
                {/* The chip is the cost disclosure: only "starts now" spends a
                    process when the group is created. */}
                <button
                  type="button"
                  data-testid={`launcher-group-role-mode-${String(index)}`}
                  aria-pressed={!waiting}
                  onClick={() => {
                    patchRole(index, { start_mode: (waiting ? "eager" : "waiting") as StartMode });
                  }}
                  className={cn(
                    "flex min-h-11 shrink-0 items-center gap-1.5 rounded-full border px-2.5 text-[11px] font-medium transition",
                    waiting
                      ? "border-dashed border-wc-default text-wc-text-faint hover:text-wc-text-secondary"
                      : "border-wc-accent bg-wc-accent/10 text-wc-accent",
                  )}
                >
                  {waiting
                    ? <Clock className="h-3.5 w-3.5" aria-hidden />
                    : <Play className="h-3.5 w-3.5" aria-hidden />}
                  <span className="truncate">{waiting ? t(strings.launcher.startsWaiting) : t(strings.launcher.startsNow)}</span>
                </button>
                <button
                  type="button"
                  data-testid={`launcher-group-role-remove-${String(index)}`}
                  aria-label={t(strings.launcher.removeRole, { label: role.label })}
                  onClick={() => { setRoles((prev) => prev.filter((_, i) => i !== index)); }}
                  className="flex h-11 w-9 shrink-0 items-center justify-center rounded-lg text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
                >
                  <Trash2 className="h-4 w-4" aria-hidden />
                </button>
              </div>

              <div className="flex items-center gap-1 ps-5">
                <span className="shrink-0 font-mono text-xs text-wc-text-faint" aria-hidden>$</span>
                <InlineField
                  testId={`launcher-group-role-command-${String(index)}`}
                  value={role.command}
                  placeholder={t(strings.roles.roleCommandPlaceholder)}
                  ariaLabel={t(strings.roles.roleCommand)}
                  onChange={(next) => { patchRole(index, { command: next }); }}
                  mono
                />
              </div>

              {/* A waiting role exists to be handed something, so what it
                  receives belongs on the row rather than in a template editor
                  the operator has to go find. */}
              {waiting && (
                <div className="flex items-center gap-1 ps-5">
                  <span className="shrink-0 text-[11px] text-wc-text-faint">↳</span>
                  <InlineField
                    testId={`launcher-group-role-prompt-${String(index)}`}
                    value={role.incoming_prompt}
                    placeholder={t(strings.launcher.promptPlaceholder)}
                    ariaLabel={t(strings.roles.roleIncomingPrompt)}
                    onChange={(next) => { patchRole(index, { incoming_prompt: next }); }}
                    className="text-xs"
                  />
                </div>
              )}
            </div>
          );
        })}

        {template && (
          <p className="px-1 text-[11px] text-wc-text-faint">{t(strings.launcher.editsNotSaved)}</p>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2 border-t border-wc-default pt-3">
        {onEditTemplates && (
          <button
            type="button"
            data-testid="launcher-edit-templates"
            onClick={onEditTemplates}
            className="flex min-h-11 items-center gap-1.5 rounded-lg px-2 text-xs text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
          >
            <Pencil className="h-3.5 w-3.5 shrink-0" aria-hidden />
            {t(strings.launcher.editTemplate)}
          </button>
        )}
        <span className="flex-1" />
        {onCancel && (
          <Button variant="outline" size="sm" onClick={onCancel}>{t(strings.roles.cancel)}</Button>
        )}
        <Button
          data-testid="launcher-create-group"
          size="sm"
          disabled={!canCreate}
          onClick={() => {
            onCreate({
              name: name.trim(),
              color: template?.color ?? "",
              roles,
              templateId: template?.id,
            });
          }}
        >
          {isCreating
            ? <Loader2 className="me-1.5 h-4 w-4 animate-spin" aria-hidden />
            : <Plus className="me-1.5 h-4 w-4" aria-hidden />}
          {isCreating ? t(strings.launcher.creatingGroup) : t(strings.launcher.createGroup)}
        </Button>
      </div>
    </div>
  );
}
