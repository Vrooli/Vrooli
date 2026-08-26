/**
 * @libraryId react-component-library:EditableResource
 * @displayName EditableResource
 * @description
 * @version 1.0.5
 * @tags ["patterns","forms","async","recovery","conflict","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource patterns.editable-resource */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useEffect, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import {
  ConflictResolutionFlow,
  type ConflictField,
} from "@vrooli/react-component-library/ConflictResolutionFlow/1.0.0";
import { Form } from "@vrooli/react-component-library/Form/1.0.0";
import {
  ResourceDetail,
  type ResourceDetailEntry,
  type ResourceDetailStatus,
} from "@vrooli/react-component-library/ResourceDetail/1.0.0";
import { UnsavedChangesFlow } from "@vrooli/react-component-library/UnsavedChangesFlow/1.0.0";

export interface EditableResourceEditorState<T> {
  draft: T;
  setDraft: (next: T | ((current: T) => T)) => void;
  dirty: boolean;
}
export interface EditableResourceProps<
  T extends Record<string, unknown> = Record<string, unknown>,
> {
  record: T;
  title: string;
  description?: string;
  entries?: ResourceDetailEntry[];
  renderEditor: (state: EditableResourceEditorState<T>) => ReactNode;
  onSave: (draft: T, signal: AbortSignal) => void | Promise<void>;
  onCancel?: () => void;
  onNavigateAway?: () => void;
  conflictFields?: ConflictField[];
  onResolveConflict?: (values: Record<string, unknown>) => void | Promise<void>;
  status?: ResourceDetailStatus | "submitting";
  defaultEditing?: boolean;
  permissionMessage?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-editable-resource] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; }
[data-rcl-editable-resource-editor] { display: grid; gap: var(--space-md, 1rem); padding: var(--space-md, 1rem); border: var(--border-hairline, 1px) solid var(--color-primary, #2563eb); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); box-shadow: var(--elev-raised, 0 3px 12px rgb(15 23 42 / .06)); }
[data-rcl-editable-resource-editor-header] { display: grid; gap: var(--space-2xs, .35rem); }
[data-rcl-editable-resource-editor-title] { font: var(--text-subtitle, 700 1rem/1.35 system-ui, sans-serif); }
[data-rcl-editable-resource-editor-header] > span { color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .875rem/1.4 system-ui, sans-serif); }
[data-rcl-editable-resource-editor] label { display: grid; gap: var(--space-2xs, .35rem); color: var(--color-foreground, #0f172a); font: var(--text-label, 650 .8125rem/1.35 system-ui, sans-serif); }
[data-rcl-editable-resource-editor] input, [data-rcl-editable-resource-editor] textarea, [data-rcl-editable-resource-editor] select { box-sizing: border-box; min-block-size: var(--tap-target-min, 44px); inline-size: 100%; padding: var(--space-xs, .625rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid var(--color-border-strong, #94a3b8); border-radius: var(--radius-control, .625rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); font: var(--text-body, 400 .875rem/1.4 system-ui, sans-serif); }
[data-rcl-editable-resource-editor] input::placeholder, [data-rcl-editable-resource-editor] textarea::placeholder { color: var(--color-muted-foreground, #64748b); opacity: 1; }
[data-rcl-editable-resource-editor] input:focus-visible, [data-rcl-editable-resource-editor] textarea:focus-visible, [data-rcl-editable-resource-editor] select:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-primary, #2563eb) 28%, transparent); outline-offset: 1px; border-color: var(--color-primary, #2563eb); }
[data-rcl-editable-resource-actions] { display: flex; flex-wrap: wrap; gap: var(--space-xs, .625rem); }
[data-rcl-editable-resource-actions] button { min-block-size: var(--tap-target-min, 44px); padding: var(--space-2xs, .35rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid currentColor; border-radius: var(--radius-control, .625rem); background: transparent; color: var(--color-primary, #2563eb); cursor: pointer; font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); }
[data-rcl-editable-resource-actions] button[type="submit"] { background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #fff); }
[data-rcl-editable-resource-actions] button:disabled { cursor: not-allowed; opacity: .5; }
@media (forced-colors: active) { [data-rcl-editable-resource-editor], [data-rcl-editable-resource-editor] input, [data-rcl-editable-resource-editor] textarea, [data-rcl-editable-resource-editor] select { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } }
`;

function same<T>(left: T, right: T) {
  try {
    return JSON.stringify(left) === JSON.stringify(right);
  } catch {
    return Object.is(left, right);
  }
}

export const EditableResource = withClassName(function EditableResource<
  T extends Record<string, unknown>,
>({
  record,
  title,
  description,
  entries,
  renderEditor,
  onSave,
  onCancel,
  onNavigateAway,
  conflictFields,
  onResolveConflict,
  status = "default",
  defaultEditing = false,
  permissionMessage,
  className,
  style,
}: EditableResourceProps<T>) {
  const [draft, setDraft] = useState(record);
  const [editing, setEditing] = useState(defaultEditing);
  const [phase, setPhase] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [error, setError] = useState<string>();
  const controller = useMemo(() => new AbortController(), []);
  const dirty = editing && !same(draft, record);
  useEffect(() => {
    if (!editing) setDraft(record);
  }, [editing, record]);
  useEffect(() => () => controller.abort(), [controller]);

  const save = async () => {
    setPhase("submitting");
    setError(undefined);
    try {
      await onSave(draft, controller.signal);
      setPhase("success");
      setEditing(false);
    } catch (caught) {
      setPhase("error");
      setError(
        caught instanceof Error
          ? caught.message
          : "We could not save these changes. Your draft is preserved.",
      );
      throw caught;
    }
  };
  const cancel = () => {
    setDraft(record);
    setEditing(false);
    setPhase("idle");
    setError(undefined);
    onCancel?.();
  };
  const resourceStatus: ResourceDetailStatus =
    status === "submitting" ? "refreshing" : status === "success" ? "default" : status;
  if (conflictFields?.length)
    return (
      <div data-rcl-editable-resource className={className} style={style}>
        <style data-rcl-editable-resource-styles dangerouslySetInnerHTML={{ __html: styles }} />
        <ConflictResolutionFlow fields={conflictFields} onResolve={onResolveConflict} />
      </div>
    );
  return (
    <div data-rcl-editable-resource className={className} style={style}>
      <style data-rcl-editable-resource-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <UnsavedChangesFlow
        isDirty={dirty}
        onSave={save}
        onDiscard={cancel}
        onLeave={onNavigateAway}
        saveState={
          phase === "submitting"
            ? "saving"
            : phase === "error"
              ? "error"
              : phase === "success"
                ? "saved"
                : "idle"
        }
      >
        <ResourceDetail
          title={title}
          description={description}
          entries={entries}
          status={resourceStatus}
          permissionMessage={permissionMessage}
          actions={
            <div data-rcl-editable-resource-actions>
              <button
                data-testid="patterns.editable-resource"
                type="button"
                onClick={() => setEditing(true)}
                disabled={editing || status === "offline" || status === "permission-denied"}
              >
                {useStrings("patterns.editable-resource.edit", "Edit")}
              </button>
            </div>
          }
        >
          <>
            {editing ? (
              <div data-rcl-editable-resource-editor>
                <div data-rcl-editable-resource-editor-header>
                  <strong data-rcl-editable-resource-editor-title>Edit {title}</strong>
                  <span>
                    {useStrings(
                      "patterns.editable-resource.draft-changes-stay-local-until-you-save-span-div",
                      "Draft changes stay local until you save.",
                    )}
                  </span>
                </div>
                <Form
                  aria-label={`Edit ${title}`}
                  onSubmit={() => void save()}
                  footer={
                    <div data-rcl-editable-resource-actions>
                      <button
                        data-testid="patterns.editable-resource"
                        type="button"
                        onClick={cancel}
                        disabled={phase === "submitting"}
                      >
                        {useStrings("patterns.editable-resource.cancel", "Cancel")}
                      </button>
                      <button
                        data-testid="patterns.editable-resource"
                        type="submit"
                        disabled={phase === "submitting"}
                      >
                        {phase === "submitting" ? "Saving…" : "Save changes"}
                      </button>
                    </div>
                  }
                >
                  {renderEditor({ draft, setDraft, dirty })}
                </Form>
                {error ? <div role="alert">{error}</div> : null}
              </div>
            ) : null}
          </>
        </ResourceDetail>
      </UnsavedChangesFlow>
    </div>
  );
});
