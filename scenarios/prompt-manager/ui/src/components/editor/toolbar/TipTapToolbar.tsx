/**
 * TipTapToolbar - Unified toolbar for the TipTap editor.
 *
 * Features:
 * - Responsive design (dropdown menus on mobile, inline buttons on desktop)
 * - All formatting controls in one component
 * - Consistent styling
 * - Editor type toggle integration
 */

import type { Editor } from '@tiptap/react'
import {
  Bold,
  Italic,
  Strikethrough,
  Code,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  Quote,
  Minus,
  FileCode,
  Highlighter,
  Link as LinkIcon,
  Unlink,
} from 'lucide-react'
import { ToolbarButton, ToolbarDivider } from './ToolbarButton'
import { ToolbarDropdown, DropdownItem } from '../ToolbarDropdown'
import { useToolbarActions } from './useToolbarActions'
import { useIsMobile } from '@/hooks/useMediaQuery'
import {
  EditorActionButtons,
  EditorToggle,
  ViewToggle,
  type EditorActionState,
  type EditorType,
  type ViewMode,
} from '../SkillContentEditor'

export interface TipTapToolbarProps extends EditorActionState {
  /** The TipTap editor instance */
  editor: Editor
  /** Callback to open the link dialog */
  onOpenLinkDialog: () => void
  /** Callback to remove a link */
  onRemoveLink: () => void
  /** Current editor type (code/wysiwyg) */
  editorType?: EditorType
  /** Callback when editor type changes */
  onEditorTypeChange?: (type: EditorType) => void
  /** Current view mode (edit/preview) */
  viewMode?: ViewMode
  /** Callback when view mode changes */
  onViewModeChange?: (mode: ViewMode) => void
}

/**
 * Unified toolbar component for the TipTap editor.
 */
export function TipTapToolbar({
  editor,
  onOpenLinkDialog,
  onRemoveLink,
  editorType,
  onEditorTypeChange,
  viewMode,
  onViewModeChange,
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  onSaveAll,
  isSaving,
  isValid,
}: TipTapToolbarProps) {
  const actions = useToolbarActions(editor)
  const isMobile = useIsMobile()
  const editorActions: EditorActionState = {
    isDirty,
    dirtyCount,
    onUndo,
    onRedo,
    canUndo,
    canRedo,
    onSave,
    onSaveAll,
    isSaving,
    isValid,
  }

  if (isMobile) {
    return (
      <MobileToolbar
        actions={actions}
        onOpenLinkDialog={onOpenLinkDialog}
        onRemoveLink={onRemoveLink}
        editorType={editorType}
        onEditorTypeChange={onEditorTypeChange}
        viewMode={viewMode}
        onViewModeChange={onViewModeChange}
        editorActions={editorActions}
      />
    )
  }

  return (
    <DesktopToolbar
      actions={actions}
      onOpenLinkDialog={onOpenLinkDialog}
      onRemoveLink={onRemoveLink}
      editorType={editorType}
      onEditorTypeChange={onEditorTypeChange}
      viewMode={viewMode}
      onViewModeChange={onViewModeChange}
      editorActions={editorActions}
    />
  )
}

interface ToolbarContentProps {
  actions: ReturnType<typeof useToolbarActions>
  onOpenLinkDialog: () => void
  onRemoveLink: () => void
  editorType?: EditorType
  onEditorTypeChange?: (type: EditorType) => void
  viewMode?: ViewMode
  onViewModeChange?: (mode: ViewMode) => void
  editorActions?: EditorActionState
}

/**
 * Mobile toolbar with dropdown menus for grouped buttons.
 */
function MobileToolbar({
  actions,
  onOpenLinkDialog,
  onRemoveLink,
  editorType,
  onEditorTypeChange,
  viewMode,
  onViewModeChange,
  editorActions,
}: ToolbarContentProps) {
  const hasEditorActions = Boolean(
    editorActions?.onUndo || editorActions?.onRedo || editorActions?.onSave || editorActions?.onSaveAll
  )

  return (
    <div className="flex-shrink-0 flex flex-wrap items-center gap-0.5 px-2 py-1.5 border-b border-border">
      {hasEditorActions && (
        <>
          <EditorActionButtons {...editorActions} />
          <ToolbarDivider />
        </>
      )}
      {/* Headings dropdown */}
      <ToolbarDropdown
        icon={<Heading1 className="h-4 w-4" />}
        label="Headings"
        hasActiveItem={actions.hasActiveHeading()}
      >
        <DropdownItem
          onClick={actions.toggleHeading1}
          isActive={actions.isHeading1Active()}
          icon={<Heading1 className="h-4 w-4" />}
          label="Heading 1"
        />
        <DropdownItem
          onClick={actions.toggleHeading2}
          isActive={actions.isHeading2Active()}
          icon={<Heading2 className="h-4 w-4" />}
          label="Heading 2"
        />
        <DropdownItem
          onClick={actions.toggleHeading3}
          isActive={actions.isHeading3Active()}
          icon={<Heading3 className="h-4 w-4" />}
          label="Heading 3"
        />
      </ToolbarDropdown>

      {/* Text formatting dropdown */}
      <ToolbarDropdown
        icon={<Bold className="h-4 w-4" />}
        label="Formatting"
        hasActiveItem={actions.hasActiveTextFormat()}
      >
        <DropdownItem
          onClick={actions.toggleBold}
          isActive={actions.isBoldActive()}
          icon={<Bold className="h-4 w-4" />}
          label="Bold"
        />
        <DropdownItem
          onClick={actions.toggleItalic}
          isActive={actions.isItalicActive()}
          icon={<Italic className="h-4 w-4" />}
          label="Italic"
        />
        <DropdownItem
          onClick={actions.toggleStrike}
          isActive={actions.isStrikeActive()}
          icon={<Strikethrough className="h-4 w-4" />}
          label="Strikethrough"
        />
        <DropdownItem
          onClick={actions.toggleHighlight}
          isActive={actions.isHighlightActive()}
          icon={<Highlighter className="h-4 w-4" />}
          label="Highlight"
        />
      </ToolbarDropdown>

      <ToolbarDivider />

      {/* Code buttons */}
      <ToolbarButton onClick={actions.toggleCode} isActive={actions.isCodeActive()} title="Inline Code">
        <Code className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleCodeBlock} isActive={actions.isCodeBlockActive()} title="Code Block">
        <FileCode className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* Link buttons */}
      <ToolbarButton onClick={onOpenLinkDialog} isActive={actions.isLinkActive()} title="Add Link">
        <LinkIcon className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={onRemoveLink} disabled={!actions.isLinkActive()} title="Remove Link">
        <Unlink className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* List buttons */}
      <ToolbarButton onClick={actions.toggleBulletList} isActive={actions.isBulletListActive()} title="Bullet List">
        <List className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleOrderedList} isActive={actions.isOrderedListActive()} title="Numbered List">
        <ListOrdered className="h-4 w-4" />
      </ToolbarButton>

      {/* Editor type toggle */}
      {editorType && onEditorTypeChange && (
        <>
          <div className="flex-1" />
          <div className="flex items-center gap-2">
            <EditorToggle editorType={editorType} onEditorTypeChange={onEditorTypeChange} />
            {viewMode && onViewModeChange && (
              <ViewToggle viewMode={viewMode} onViewModeChange={onViewModeChange} />
            )}
          </div>
        </>
      )}
    </div>
  )
}

/**
 * Desktop toolbar with all buttons inline.
 */
function DesktopToolbar({
  actions,
  onOpenLinkDialog,
  onRemoveLink,
  editorType,
  onEditorTypeChange,
  viewMode,
  onViewModeChange,
  editorActions,
}: ToolbarContentProps) {
  const hasEditorActions = Boolean(
    editorActions?.onUndo || editorActions?.onRedo || editorActions?.onSave || editorActions?.onSaveAll
  )

  return (
    <div className="flex-shrink-0 flex flex-wrap items-center gap-0.5 px-2 py-1.5 border-b border-border">
      {hasEditorActions && (
        <>
          <EditorActionButtons {...editorActions} />
          <ToolbarDivider />
        </>
      )}
      {/* Headings */}
      <ToolbarButton onClick={actions.toggleHeading1} isActive={actions.isHeading1Active()} title="Heading 1">
        <Heading1 className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleHeading2} isActive={actions.isHeading2Active()} title="Heading 2">
        <Heading2 className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleHeading3} isActive={actions.isHeading3Active()} title="Heading 3">
        <Heading3 className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* Text formatting */}
      <ToolbarButton onClick={actions.toggleBold} isActive={actions.isBoldActive()} title="Bold">
        <Bold className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleItalic} isActive={actions.isItalicActive()} title="Italic">
        <Italic className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleStrike} isActive={actions.isStrikeActive()} title="Strikethrough">
        <Strikethrough className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleHighlight} isActive={actions.isHighlightActive()} title="Highlight">
        <Highlighter className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* Code */}
      <ToolbarButton onClick={actions.toggleCode} isActive={actions.isCodeActive()} title="Inline Code">
        <Code className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleCodeBlock} isActive={actions.isCodeBlockActive()} title="Code Block">
        <FileCode className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* Links */}
      <ToolbarButton onClick={onOpenLinkDialog} isActive={actions.isLinkActive()} title="Add Link">
        <LinkIcon className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={onRemoveLink} disabled={!actions.isLinkActive()} title="Remove Link">
        <Unlink className="h-4 w-4" />
      </ToolbarButton>

      <ToolbarDivider />

      {/* Lists and block elements */}
      <ToolbarButton onClick={actions.toggleBulletList} isActive={actions.isBulletListActive()} title="Bullet List">
        <List className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleOrderedList} isActive={actions.isOrderedListActive()} title="Numbered List">
        <ListOrdered className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.toggleBlockquote} isActive={actions.isBlockquoteActive()} title="Blockquote">
        <Quote className="h-4 w-4" />
      </ToolbarButton>
      <ToolbarButton onClick={actions.insertHorizontalRule} title="Horizontal Rule">
        <Minus className="h-4 w-4" />
      </ToolbarButton>

      {/* Editor type toggle */}
      {editorType && onEditorTypeChange && (
        <>
          <div className="flex-1" />
          <div className="flex items-center gap-2">
            <EditorToggle editorType={editorType} onEditorTypeChange={onEditorTypeChange} />
            {viewMode && onViewModeChange && (
              <ViewToggle viewMode={viewMode} onViewModeChange={onViewModeChange} />
            )}
          </div>
        </>
      )}
    </div>
  )
}
