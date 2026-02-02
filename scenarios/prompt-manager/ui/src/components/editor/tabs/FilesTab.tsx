/**
 * FilesTab - Agent files browser and editor.
 *
 * Features:
 * - File tree of the agent folder with add/rename/delete
 * - Clicking agent.json shows appearance controls
 * - Clicking any other file opens markdown editor
 */

import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { File, FileText, Folder, FolderOpen, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { toast } from '@/hooks/use-toast'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import type { AgentFileEntry } from '@/types/agent'
import * as agentService from '@/services/agentService'
import { AppearanceTab } from './AppearanceTab'
import { EditorActionButtons, SkillContentEditor } from '../SkillContentEditor'

interface FilesTabProps {
  agentId: string
  formState: NormalizedAgentFormState
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
  updateFields: (updates: Partial<NormalizedAgentFormState>) => void
  isDirty: boolean
  dirtyCount: number
  onUndo: () => void
  onRedo: () => void
  canUndo: boolean
  canRedo: boolean
  onSave: () => void
  onDiscard: () => void
  isSaving: boolean
  isValid: boolean
}

interface FileNode {
  name: string
  path: string
  isDir: boolean
  children: FileNode[]
}

const RESERVED_AGENT_FILE = 'agent.json'

function buildFileTree(entries: AgentFileEntry[]): FileNode {
  const root: FileNode = { name: '', path: '', isDir: true, children: [] }
  const nodeMap = new Map<string, FileNode>()
  nodeMap.set('', root)

  const ensureNode = (path: string, name: string, isDir: boolean) => {
    const existing = nodeMap.get(path)
    if (existing) {
      if (isDir && !existing.isDir) {
        existing.isDir = true
      }
      return existing
    }
    const node: FileNode = { name, path, isDir, children: [] }
    nodeMap.set(path, node)
    return node
  }

  for (const entry of entries) {
    const parts = entry.path.split('/').filter(Boolean)
    let currentPath = ''
    let parent = root

    parts.forEach((part, index) => {
      currentPath = currentPath ? `${currentPath}/${part}` : part
      const isLeaf = index === parts.length - 1
      const nodeIsDir = isLeaf ? entry.isDir : true
      const node = ensureNode(currentPath, part, nodeIsDir)
      if (!parent.children.includes(node)) {
        parent.children.push(node)
      }
      parent = node
    })
  }

  const sortTree = (node: FileNode) => {
    node.children.sort((a, b) => {
      if (a.isDir !== b.isDir) {
        return a.isDir ? -1 : 1
      }
      return a.name.localeCompare(b.name)
    })
    node.children.forEach(sortTree)
  }
  sortTree(root)

  return root
}

function isMarkdownFile(path: string): boolean {
  return path.toLowerCase().endsWith('.md')
}

export function FilesTab({
  agentId,
  formState,
  updateField,
  updateFields: _updateFields,
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  onDiscard,
  isSaving,
  isValid,
}: FilesTabProps) {
  void _updateFields

  const [files, setFiles] = useState<AgentFileEntry[]>([])
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())
  const [fileContent, setFileContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [isFileLoading, setIsFileLoading] = useState(false)
  const [isFileSaving, setIsFileSaving] = useState(false)
  const [mode, setMode] = useState<'none' | 'add' | 'rename'>('none')
  const [pendingPath, setPendingPath] = useState('')

  const tree = useMemo(() => buildFileTree(files), [files])

  const selectedEntry = useMemo(
    () => (selectedPath ? files.find((file) => file.path === selectedPath) : undefined),
    [files, selectedPath]
  )

  const isReservedSelected = selectedPath === RESERVED_AGENT_FILE
  const isDirectorySelected = selectedEntry?.isDir ?? false
  const isFileEditorActive = Boolean(selectedPath && !isDirectorySelected && !isReservedSelected)
  const isFileDirty = isFileEditorActive && fileContent !== originalContent

  const refreshFiles = useCallback(async () => {
    try {
      const entries = await agentService.listAgentFiles(agentId)
      setFiles(entries)
    } catch (error) {
      console.warn('[FilesTab] Failed to load agent files:', error)
      toast({
        title: 'Unable to load files',
        description: 'Check the API server and try again.',
      })
    }
  }, [agentId])

  useEffect(() => {
    void refreshFiles()
  }, [refreshFiles])

  useEffect(() => {
    setExpandedPaths(new Set())
  }, [agentId])

  useEffect(() => {
    if (files.length === 0) {
      setSelectedPath(null)
      return
    }

    if (selectedPath && files.some((file) => file.path === selectedPath)) {
      return
    }

    const preferred =
      files.find((file) => file.path === 'SOUL.md')?.path ??
      files.find((file) => file.path === 'AGENTS.md')?.path ??
      files[0]?.path ??
      null
    setSelectedPath(preferred)
  }, [files, selectedPath])

  useEffect(() => {
    if (!selectedPath || isDirectorySelected || isReservedSelected) {
      setFileContent('')
      setOriginalContent('')
      setIsFileLoading(false)
      return
    }

    let cancelled = false
    setIsFileLoading(true)
    agentService.getAgentFileContent(agentId, selectedPath)
      .then((content) => {
        if (cancelled) return
        setFileContent(content)
        setOriginalContent(content)
      })
      .catch((error: unknown) => {
        console.warn('[FilesTab] Failed to load file content:', error)
        toast({
          title: 'Unable to load file',
          description: 'Check the API server and try again.',
        })
      })
      .finally(() => {
        if (!cancelled) setIsFileLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [agentId, selectedPath, isDirectorySelected, isReservedSelected])

  const ensureExpandedForPath = useCallback((path: string) => {
    const parts = path.split('/').filter(Boolean)
    if (parts.length <= 1) return
    const newExpanded = new Set(expandedPaths)
    let current = ''
    parts.slice(0, -1).forEach((part) => {
      current = current ? `${current}/${part}` : part
      newExpanded.add(current)
    })
    setExpandedPaths(newExpanded)
  }, [expandedPaths])

  const handleSelectPath = useCallback((path: string, isDir: boolean) => {
    if (path === selectedPath) {
      if (isDir) {
        setExpandedPaths((prev) => {
          const next = new Set(prev)
          if (next.has(path)) {
            next.delete(path)
          } else {
            next.add(path)
          }
          return next
        })
      }
      return
    }

    if (isFileDirty) {
      const confirmLeave = window.confirm('You have unsaved changes. Discard them?')
      if (!confirmLeave) return
    }

    if (isDir) {
      setExpandedPaths((prev) => {
        const next = new Set(prev)
        if (next.has(path)) {
          next.delete(path)
        } else {
          next.add(path)
        }
        return next
      })
      return
    }

    ensureExpandedForPath(path)
    setSelectedPath(path)
  }, [ensureExpandedForPath, isFileDirty, selectedPath])

  const handleSaveFile = useCallback(async () => {
    if (!selectedPath || !isFileEditorActive || !isFileDirty) return
    setIsFileSaving(true)
    try {
      await agentService.setAgentFileContent(agentId, selectedPath, fileContent)
      setOriginalContent(fileContent)
      toast({
        title: 'File saved',
        description: `${selectedPath} updated.`,
      })
    } catch (error) {
      console.warn('[FilesTab] Failed to save file:', error)
      toast({
        title: 'Unable to save file',
        description: 'Check the API server and try again.',
      })
    } finally {
      setIsFileSaving(false)
    }
  }, [agentId, selectedPath, isFileEditorActive, isFileDirty, fileContent])

  const handleDiscardFile = useCallback(() => {
    setFileContent(originalContent)
  }, [originalContent])

  const handleStartAdd = useCallback(() => {
    setMode('add')
    setPendingPath('NEW.md')
  }, [])

  const handleStartRename = useCallback(() => {
    if (!selectedPath || isDirectorySelected || isReservedSelected) return
    setMode('rename')
    setPendingPath(selectedPath)
  }, [selectedPath, isDirectorySelected, isReservedSelected])

  const handleCancelMode = useCallback(() => {
    setMode('none')
    setPendingPath('')
  }, [])

  const handleConfirmMode = useCallback(async () => {
    const trimmed = pendingPath.trim()
    if (!trimmed) return

    try {
      if (mode === 'add') {
        await agentService.createAgentFile(agentId, { path: trimmed, content: '' })
        setSelectedPath(trimmed)
        ensureExpandedForPath(trimmed)
      } else if (mode === 'rename' && selectedPath) {
        if (trimmed === selectedPath) {
          handleCancelMode()
          return
        }
        await agentService.renameAgentFile(agentId, { from: selectedPath, to: trimmed })
        setSelectedPath(trimmed)
        ensureExpandedForPath(trimmed)
      }
      await refreshFiles()
    } catch (error) {
      console.warn('[FilesTab] File operation failed:', error)
      toast({
        title: 'File operation failed',
        description: 'Check the file name and try again.',
      })
    } finally {
      setMode('none')
      setPendingPath('')
    }
  }, [agentId, mode, pendingPath, refreshFiles, selectedPath, ensureExpandedForPath, handleCancelMode])

  const handleDelete = useCallback(async () => {
    if (!selectedPath || isDirectorySelected || isReservedSelected) return
    const confirmed = window.confirm(`Delete ${selectedPath}? This cannot be undone.`)
    if (!confirmed) return

    try {
      await agentService.deleteAgentFile(agentId, selectedPath)
      setSelectedPath(null)
      await refreshFiles()
    } catch (error) {
      console.warn('[FilesTab] Failed to delete file:', error)
      toast({
        title: 'Unable to delete file',
        description: 'Check the API server and try again.',
      })
    }
  }, [agentId, selectedPath, isDirectorySelected, isReservedSelected, refreshFiles])

  const renderNode = (node: FileNode, depth = 0): ReactNode => {
    if (!node.path) {
      return <>{node.children.map((child) => renderNode(child, 0))}</>
    }

    const isSelected = selectedPath === node.path
    const isExpanded = expandedPaths.has(node.path)
    const icon = node.isDir
      ? (isExpanded ? <FolderOpen className="h-4 w-4" /> : <Folder className="h-4 w-4" />)
      : isMarkdownFile(node.path)
        ? <FileText className="h-4 w-4" />
        : <File className="h-4 w-4" />

    return (
      <div key={node.path}>
        <button
          type="button"
          onClick={() => handleSelectPath(node.path, node.isDir)}
          className={cn(
            'w-full flex items-center gap-2 rounded-md px-2 py-1 text-sm text-left',
            isSelected ? 'bg-primary/15 text-primary' : 'hover:bg-muted'
          )}
          style={{ paddingLeft: 8 + depth * 14 }}
        >
          {icon}
          <span className="truncate">{node.name}</span>
        </button>
        {node.isDir && isExpanded && node.children.map((child) => renderNode(child, depth + 1))}
      </div>
    )
  }

  return (
    <div className="h-full flex min-h-0">
      <div className="w-64 border-r border-border flex flex-col min-h-0">
        <div className="flex items-center justify-between px-3 py-2 border-b border-border">
          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Agent Files</span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => void refreshFiles()}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
              title="Refresh"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={handleStartAdd}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
              title="Add file"
            >
              <Plus className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={handleStartRename}
              className={cn(
                'p-1 rounded text-muted-foreground hover:text-foreground',
                !selectedPath || isDirectorySelected || isReservedSelected ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
              )}
              disabled={!selectedPath || isDirectorySelected || isReservedSelected}
              title="Rename file"
            >
              <Pencil className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={() => void handleDelete()}
              className={cn(
                'p-1 rounded text-muted-foreground hover:text-foreground',
                !selectedPath || isDirectorySelected || isReservedSelected ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
              )}
              disabled={!selectedPath || isDirectorySelected || isReservedSelected}
              title="Delete file"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>

        {mode !== 'none' && (
          <div className="px-3 py-2 border-b border-border space-y-2">
            <Input
              value={pendingPath}
              onChange={(event) => setPendingPath(event.target.value)}
              placeholder="path/to/file.md"
            />
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => void handleConfirmMode()}
                className="px-2 py-1 rounded bg-primary text-primary-foreground text-xs"
              >
                {mode === 'add' ? 'Create' : 'Rename'}
              </button>
              <button
                type="button"
                onClick={handleCancelMode}
                className="px-2 py-1 rounded text-xs text-muted-foreground hover:text-foreground"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        <div className="flex-1 overflow-y-auto px-2 py-2">
          {files.length === 0 ? (
            <div className="text-xs text-muted-foreground px-2 py-4">
              No files yet. Create a file to get started.
            </div>
          ) : (
            renderNode(tree)
          )}
        </div>
      </div>

      <div className="flex-1 min-w-0 flex flex-col">
        {!selectedPath && (
          <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
            Select a file to view or edit.
          </div>
        )}

        {selectedPath && isReservedSelected && (
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-semibold">agent.json</div>
                <div className="text-xs text-muted-foreground">Appearance settings</div>
              </div>
              <EditorActionButtons
                isDirty={isDirty}
                dirtyCount={dirtyCount}
                onUndo={onUndo}
                onRedo={onRedo}
                canUndo={canUndo}
                canRedo={canRedo}
                onSave={onSave}
                onDiscard={onDiscard}
                isSaving={isSaving}
                isValid={isValid}
              />
            </div>
            <AppearanceTab formState={formState} updateField={updateField} />
          </div>
        )}

        {selectedPath && isDirectorySelected && (
          <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
            Select a file to view or edit.
          </div>
        )}

        {selectedPath && isFileEditorActive && (
          <div className="flex-1 min-h-0">
            {isFileLoading ? (
              <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
                Loading file...
              </div>
            ) : (
              <SkillContentEditor
                value={fileContent}
                originalValue={originalContent}
                onChange={setFileContent}
                isDirty={isFileDirty}
                dirtyCount={isFileDirty ? 1 : 0}
                onSave={() => void handleSaveFile()}
                onDiscard={handleDiscardFile}
                isSaving={isFileSaving}
                isValid
                className="h-full"
              />
            )}
          </div>
        )}
      </div>
    </div>
  )
}
