/**
 * Modal for creating and editing templates.
 * Provides form for all template fields including variables.
 *
 * Updated to support file-based templates:
 * - All templates are now editable (including defaults)
 * - Editing a default template creates a user override
 */

import type { ComponentType, SVGProps } from "react";
import { X, Pencil, Info, Sparkles, Loader2 } from "lucide-react";
import * as LucideIcons from "lucide-react";
import { CategoryPathEditor } from "@/components/shared/CategoryPathEditor";
import { ItemTreeSidebar } from "@/components/shared/ItemTreeSidebar";
import type { TemplateEditorModalProps } from "./types";
import { useTemplateEditorForm } from "./useTemplateEditorForm";
import { MetadataFields } from "./MetadataFields";
import { VariableEditor } from "./VariableEditor";
import { ContentEditor } from "./ContentEditor";
import { UnsavedChangesDialog } from "./UnsavedChangesDialog";

// Type for Lucide icon components
type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

// Get icon component from name
function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || Sparkles;
}

export function TemplateEditorModal({
  open,
  onClose,
  template,
  templateSource,
  defaultModes,
  onSave,
  readOnly = false,
  onEdit,
  allTemplates,
  onSelectTemplate,
  onSaveAll,
}: TemplateEditorModalProps) {
  const showSidebar = !!allTemplates && allTemplates.length > 0;

  const form = useTemplateEditorForm({
    template,
    templateSource,
    defaultModes,
    open,
    readOnly,
    allTemplates,
    onSave,
    onClose,
    onSelectTemplate,
    onSaveAll,
  });

  if (!open) return null;

  const modalTitle = readOnly
    ? "Template Preview"
    : form.isEditing
      ? "Edit Template"
      : "Create Template";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={form.handleClose}
      />

      {/* Modal */}
      <div className={`relative bg-slate-900 border border-white/10 rounded-xl w-full max-h-[90vh] shadow-xl mx-4 flex flex-col ${
        showSidebar ? "max-w-6xl" : "max-w-2xl md:max-w-5xl"
      }`}>
          {/* Header */}
          <div className="flex-shrink-0 flex items-center justify-between p-4 border-b border-white/10">
            <h2 className="text-lg font-semibold text-white">
              {modalTitle}
            </h2>
            <div className="flex items-center gap-2">
              {readOnly && onEdit && (
                <button
                  onClick={onEdit}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors"
                  title="Edit template"
                >
                  <Pencil className="h-4 w-4" />
                  Edit
                </button>
              )}
              <button
                onClick={form.handleClose}
                className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>

          {/* Info banner for editing defaults */}
          {form.isEditingDefault && (
            <div className={`flex-shrink-0 mx-4 mt-4 p-3 rounded-lg flex items-start gap-3 ${
              form.applyToDefault
                ? "bg-indigo-900/20 border border-indigo-500/30"
                : "bg-amber-900/20 border border-amber-500/30"
            }`}>
              <Info className={`h-5 w-5 flex-shrink-0 mt-0.5 ${
                form.applyToDefault ? "text-indigo-400" : "text-amber-400"
              }`} />
              <div className="flex-1">
                <p className={`text-sm font-medium ${
                  form.applyToDefault ? "text-indigo-200" : "text-amber-200"
                }`}>
                  {form.applyToDefault
                    ? "Updating default template"
                    : "Editing a default template"}
                </p>
                <p className={`text-xs mt-1 ${
                  form.applyToDefault ? "text-indigo-300/70" : "text-amber-300/70"
                }`}>
                  {form.applyToDefault
                    ? "Your changes will modify the default template directly. This affects all users."
                    : "Your changes will be saved as a custom version. The original default will remain available."}
                </p>
              </div>
              <button
                type="button"
                onClick={() => form.setApplyToDefault(!form.applyToDefault)}
                className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                  form.applyToDefault
                    ? "bg-amber-600/20 text-amber-300 hover:bg-amber-600/30"
                    : "bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30"
                }`}
              >
                {form.applyToDefault ? "Save as custom" : "Apply to default"}
              </button>
            </div>
          )}

          {/* Content with optional sidebar */}
          <div className="flex-1 min-h-0 overflow-hidden flex">
            {/* Tree Sidebar */}
            {allTemplates && allTemplates.length > 0 && (
              <ItemTreeSidebar
                items={form.itemsForTree}
                selectedItemId={template?.id ?? null}
                onSelectItem={(id) => {
                  const selected = allTemplates.find((t) => t.id === id);
                  if (selected) form.handleSelectTemplate(selected);
                }}
                dirtyItemIds={form.dirtyItemIds}
                expandedNodes={form.expandedTreeNodes}
                onToggleNode={form.toggleTreeNode}
                renderItemIcon={(item) => {
                  const IconComp = getIconComponent(item.icon || "Sparkles");
                  return <IconComp className="h-3.5 w-3.5 flex-shrink-0 text-slate-400" />;
                }}
                title="Templates"
                className={form.isSidebarCollapsed ? "" : "w-60 flex-shrink-0"}
                isCollapsed={form.isSidebarCollapsed}
                onToggleCollapse={() => form.setIsSidebarCollapsed((prev) => !prev)}
              />
            )}

            {/* Main Content */}
            <div className="flex-1 min-h-0 p-4 overflow-hidden">
              <div className="h-full md:grid md:grid-cols-[1fr_2fr] md:gap-6 space-y-4 md:space-y-0 overflow-y-auto md:overflow-hidden">
              {/* Left Column - Metadata */}
              <div className="space-y-4 md:h-full md:min-h-0 md:overflow-y-auto md:pr-2">
                <MetadataFields
                  name={form.name}
                  onNameChange={form.setName}
                  description={form.description}
                  onDescriptionChange={form.setDescription}
                  icon={form.icon}
                  onIconChange={form.setIcon}
                  draft={form.draft}
                  onDraftChange={form.setDraft}
                  readOnly={readOnly}
                  errors={form.errors}
                />

                {/* Category Path */}
                <CategoryPathEditor
                  value={form.modes}
                  onChange={form.setModes}
                  getSuggestionsAtLevel={form.getSuggestionsAtLevel}
                  label="Category Path"
                  placeholder="Select or type category..."
                  disabled={readOnly}
                  required={!readOnly}
                  error={form.errors.modes}
                />

                {/* Variables */}
                <VariableEditor
                  variables={form.variables}
                  errors={form.errors}
                  readOnly={readOnly}
                  onAdd={form.addVariable}
                  onUpdate={form.updateVariable}
                  onRemove={form.removeVariable}
                />

              </div>

              {/* Right Column - Content */}
              <ContentEditor
                content={form.content}
                onChange={form.setContent}
                readOnly={readOnly}
                errors={form.errors}
                undefinedVariables={form.undefinedVariables}
                showPreview={form.showPreview}
                onTogglePreview={() => form.setShowPreview(!form.showPreview)}
                previewContent={form.previewContent}
              />
            </div>
          </div>
          </div>

          {/* Footer */}
          <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 border-t border-white/10">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
            >
              {readOnly ? "Close" : "Cancel"}
            </button>
            {!readOnly && (
              <>
                {/* Show Save All when multiple items are dirty and onSaveAll is provided */}
                {showSidebar && form.dirtyCount > 1 && onSaveAll ? (
                  <button
                    onClick={() => { void form.handleSaveAll(); }}
                    disabled={form.isSavingAll}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {form.isSavingAll && <Loader2 className="h-4 w-4 animate-spin" />}
                    Save All Changes ({form.dirtyCount})
                  </button>
                ) : (
                  <button
                    onClick={form.handleSave}
                    className="px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors"
                  >
                    {form.isEditing ? "Save Changes" : "Create Template"}
                  </button>
                )}
              </>
            )}
          </div>
      </div>

      {/* Unsaved changes confirmation dialog */}
      {form.showCloseConfirm && (
        <UnsavedChangesDialog
          dirtyCount={form.dirtyCount}
          onKeepEditing={() => form.setShowCloseConfirm(false)}
          onDiscard={() => {
            form.setShowCloseConfirm(false);
            form.setPendingChanges(new Map());
            onClose();
          }}
        />
      )}
    </div>
  );
}
