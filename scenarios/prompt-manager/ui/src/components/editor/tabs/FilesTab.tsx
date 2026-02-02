/**
 * FilesTab - Agent files browser and editor.
 *
 * Features:
 * - File tree of the agent folder with add/rename/delete
 * - Clicking agent.json shows appearance controls
 * - Clicking any other file opens markdown editor
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type MouseEvent as ReactMouseEvent,
} from 'react'
import {
  File,
  FileText,
  Folder,
  FolderOpen,
  Pencil,
  Plus,
  RefreshCw,
  Sparkles,
  Trash2,
  X,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { toast } from '@/hooks/use-toast'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import type { AgentFileEntry } from '@/types/agent'
import * as agentService from '@/services/agentService'
import { AppearanceTab } from './AppearanceTab'
import { EditorActionButtons, SkillContentEditor } from '../SkillContentEditor'
import { DropdownItem, ToolbarDropdown } from '../ToolbarDropdown'
import { FilePathMenu } from '../FilePathMenu'

interface FilesTabProps {
  agentId: string
  agentDir?: string
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
const RECOMMENDED_AGENT_FILES = ['AGENTS.md', 'SOUL.md', 'TOOLS.md'] as const
const AGENT_FILE_TEMPLATES: Record<string, { label: string; content: string }[]> = {
  'AGENTS.MD': [
    {
      label: 'OpenClaw-inspired starter',
      content: `# AGENTS

## Start of Session
- Read \`SOUL.md\` to align tone and boundaries.
- Read \`TOOLS.md\` before using any tools.
- Scan this file for current procedures and priorities.

## Operating Principles
- Be accurate and candid. If unsure, ask.
- Prefer small, reversible changes.
- Keep outputs concise and actionable.

## External Actions
- Never send messages, make purchases, or trigger external side effects without explicit approval.

## Memory Hygiene
- Write durable decisions back into the relevant files.
- If you create a new process, document it here.

## Output Style
- Use short headings and bullet points.
- Provide checklists for multi-step tasks.
`,
    },
  ],
  'SOUL.MD': [
    {
      label: 'OpenClaw-inspired starter',
      content: `# SOUL

## Core Truths
- Be genuinely helpful, not performatively helpful.
- Protect privacy and sensitive data.
- Optimize for clarity and correctness.

## Boundaries
- No external actions without explicit user approval.
- If context is missing or ambiguous, ask before assuming.

## Communication Style
- Direct, calm, and specific.
- Avoid fluff and avoid sycophancy.
- Admit uncertainty when it exists.

## Domain Identity (Optional)
- This agent specializes in: [describe focus].
`,
    },
  ],
  'TOOLS.MD': [
    {
      label: 'Vrooli skill primer',
      content: `# TOOLS

## Tool Access
You have access to tools via Vrooli skills. Before using a tool, read its skill.

Use:
\`prompt-manager skill read <skill-id>\`

## Common Skills
- browser-automation-studio — Web navigation and UI validation.
- e2e-testing — End-to-end browser testing.
- api-steer — API design and integration guidance.
- cli-steer — CLI ergonomics and conventions.

## Usage Rules
- Follow skill instructions verbatim.
- If a tool is missing, request the skill or ask for guidance.
`,
    },
  ],
}

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

function isReservedPath(path?: string | null): boolean {
  if (!path) return false
  const baseName = path.split('/').pop()
  return baseName?.toLowerCase() === RESERVED_AGENT_FILE
}

export function FilesTab({
  agentId,
  agentDir,
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
  const [fileDialogOpen, setFileDialogOpen] = useState(false)
  const [dialogMode, setDialogMode] = useState<'add' | 'rename'>('add')
  const [pendingPath, setPendingPath] = useState('')
  const [renameSourcePath, setRenameSourcePath] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    path: string
    isDir: boolean
    isReserved: boolean
  } | null>(null)
  const skipFileLoadRef = useRef<string | null>(null)

  const tree = useMemo(() => buildFileTree(files), [files])
  const recommendedMissing = useMemo(
    () =>
      RECOMMENDED_AGENT_FILES.filter(
        (name) => !files.some((file) => file.path.toLowerCase() === name.toLowerCase())
      ),
    [files]
  )

  const selectedEntry = useMemo(
    () => (selectedPath ? files.find((file) => file.path === selectedPath) : undefined),
    [files, selectedPath]
  )

  const isReservedSelected = isReservedPath(selectedPath)
  const isDirectorySelected = selectedEntry?.isDir ?? false
  const isFileEditorActive = Boolean(selectedPath && !isDirectorySelected && !isReservedSelected)
  const isFileDirty = isFileEditorActive && fileContent !== originalContent
  const selectedTemplateKey = selectedPath ? selectedPath.split('/').pop()?.toUpperCase() : null
  const templateOptions =
    selectedTemplateKey && AGENT_FILE_TEMPLATES[selectedTemplateKey]
      ? AGENT_FILE_TEMPLATES[selectedTemplateKey]
      : undefined

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
      if (skipFileLoadRef.current === selectedPath) {
        skipFileLoadRef.current = null
      }
      setFileContent('')
      setOriginalContent('')
      setIsFileLoading(false)
      return
    }

    if (skipFileLoadRef.current === selectedPath) {
      skipFileLoadRef.current = null
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
    setContextMenu(null)
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

  const handleApplyTemplate = useCallback(
    (content: string) => {
      if (isFileDirty) {
        const confirmed = window.confirm('Replace the current draft with this template?')
        if (!confirmed) return
      }
      setFileContent(content)
    },
    [isFileDirty]
  )

  const handleStartAdd = useCallback(() => {
    setDialogMode('add')
    setRenameSourcePath(null)
    setPendingPath(recommendedMissing[0] ?? 'NEW.md')
    setFileDialogOpen(true)
    setContextMenu(null)
  }, [recommendedMissing])

  const handleStartRename = useCallback(
    (path?: string) => {
      const target = path ?? selectedPath
      if (!target || isReservedPath(target)) return
      setDialogMode('rename')
      setRenameSourcePath(target)
      setPendingPath(target)
      setFileDialogOpen(true)
      setContextMenu(null)
    },
    [selectedPath]
  )

  const handleCloseDialog = useCallback(() => {
    setFileDialogOpen(false)
    setPendingPath('')
    setRenameSourcePath(null)
  }, [])

  const handleRenameFile = useCallback(
    async (from: string, to: string): Promise<boolean> => {
      if (from === to) return true
      try {
        await agentService.renameAgentFile(agentId, { from, to })
        if (selectedPath === from) {
          skipFileLoadRef.current = to
          setSelectedPath(to)
        }
        ensureExpandedForPath(to)
        await refreshFiles()
        return true
      } catch (error) {
        console.warn('[FilesTab] File operation failed:', error)
        toast({
          title: 'File operation failed',
          description: 'Check the file name and try again.',
        })
        return false
      }
    },
    [agentId, ensureExpandedForPath, refreshFiles, selectedPath, toast]
  )

  const handleConfirmDialog = useCallback(async () => {
    const trimmed = pendingPath.trim()
    if (!trimmed) return

    let didSucceed = false

    try {
      if (dialogMode === 'add') {
        await agentService.createAgentFile(agentId, { path: trimmed, content: '' })
        setSelectedPath(trimmed)
        ensureExpandedForPath(trimmed)
        await refreshFiles()
        didSucceed = true
      } else if (renameSourcePath) {
        didSucceed = await handleRenameFile(renameSourcePath, trimmed)
      }
    } catch (error) {
      console.warn('[FilesTab] File operation failed:', error)
      toast({
        title: 'File operation failed',
        description: 'Check the file name and try again.',
      })
    } finally {
      if (didSucceed) {
        handleCloseDialog()
      }
    }
  }, [
    agentId,
    dialogMode,
    pendingPath,
    renameSourcePath,
    ensureExpandedForPath,
    handleCloseDialog,
    handleRenameFile,
    refreshFiles,
    toast,
  ])

  const handleDelete = useCallback(
    async (path?: string) => {
      const target = path ?? selectedPath
      if (!target || isReservedPath(target)) return
      const confirmed = window.confirm(`Delete ${target}? This cannot be undone.`)
      if (!confirmed) return

      try {
        await agentService.deleteAgentFile(agentId, target)
        if (selectedPath === target) {
          setSelectedPath(null)
        }
        await refreshFiles()
      } catch (error) {
        console.warn('[FilesTab] Failed to delete file:', error)
        toast({
          title: 'Unable to delete file',
          description: 'Check the API server and try again.',
        })
      } finally {
        setContextMenu(null)
      }
    },
    [agentId, refreshFiles, selectedPath, toast]
  )

  const handleContextMenu = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>, node: FileNode) => {
      event.preventDefault()
      event.stopPropagation()
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        path: node.path,
        isDir: node.isDir,
        isReserved: isReservedPath(node.path),
      })
    },
    []
  )

  const handleRenameSelectedFile = useCallback(
    async (nextFile: string) => {
      if (!selectedPath || isDirectorySelected || isReservedSelected) return
      const parentDir = selectedPath.split('/').slice(0, -1).join('/')
      const nextPath = parentDir ? `${parentDir}/${nextFile}` : nextFile
      if (!nextPath || nextPath === selectedPath) return
      await handleRenameFile(selectedPath, nextPath)
    },
    [handleRenameFile, isDirectorySelected, isReservedSelected, selectedPath]
  )

  const filePathMenu = useMemo<ReactNode | null>(() => {
    if (!selectedPath || isDirectorySelected) return null

    const segments = selectedPath.split('/').filter(Boolean)
    const baseName = segments.pop() ?? selectedPath
    const dirSegments = segments

    const resolvedAgentDir = agentDir ?? ''
    const fullPath = resolvedAgentDir ? `${resolvedAgentDir}/${selectedPath}` : ''

    let projectPath = ''
    if (resolvedAgentDir) {
      const scenariosIndex = resolvedAgentDir.indexOf('scenarios/')
      const basePath = scenariosIndex !== -1 ? resolvedAgentDir.slice(scenariosIndex) : resolvedAgentDir
      projectPath = `${basePath}/${selectedPath}`
    }

    const relativeBase = resolvedAgentDir
      ? (resolvedAgentDir.split('/store/').pop() ?? '')
      : `agents/${agentId}`
    const relativePath = relativeBase ? `${relativeBase}/${selectedPath}` : ''

    return (
      <FilePathMenu
        file={baseName}
        rootLabel="agents"
        pathSegments={[agentId, ...dirSegments]}
        onFileChange={(nextFile) => void handleRenameSelectedFile(nextFile)}
        relativePath={relativePath}
        projectPath={projectPath}
        fullPath={fullPath}
        isEditable={!isReservedSelected}
        className="flex-shrink-0"
      />
    )
  }, [agentDir, agentId, handleRenameSelectedFile, isDirectorySelected, isReservedSelected, selectedPath])

  const templateHeader = useMemo<ReactNode | null>(() => {
    if (!isFileEditorActive || !templateOptions?.length) {
      return null
    }

    return (
      <ToolbarDropdown
        icon={<Sparkles className="h-4 w-4" />}
        label="Templates"
        align="right"
        className="h-8 px-2"
      >
        {templateOptions.map((template) => (
          <DropdownItem
            key={template.label}
            onClick={() => handleApplyTemplate(template.content)}
            icon={<FileText className="h-4 w-4" />}
            label={template.label}
          />
        ))}
      </ToolbarDropdown>
    )
  }, [handleApplyTemplate, isFileEditorActive, templateOptions])

  const headerRight = useMemo<ReactNode | null>(() => {
    if (filePathMenu && templateHeader) {
      return (
        <>
          {filePathMenu}
          {templateHeader}
        </>
      )
    }
    return filePathMenu ?? templateHeader
  }, [filePathMenu, templateHeader])

  const isDialogValid = pendingPath.trim().length > 0

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
          onContextMenu={(event) => handleContextMenu(event, node)}
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
    <>
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
                onClick={() => handleStartRename()}
                className={cn(
                  'p-1 rounded text-muted-foreground hover:text-foreground',
                  !selectedPath || isReservedSelected ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
                )}
                disabled={!selectedPath || isReservedSelected}
                title="Rename file"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => void handleDelete()}
                className={cn(
                  'p-1 rounded text-muted-foreground hover:text-foreground',
                  !selectedPath || isReservedSelected ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
                )}
                disabled={!selectedPath || isReservedSelected}
                title="Delete file"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </div>

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
                <div className="flex items-center gap-2">
                  {filePathMenu}
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
                  headerRight={headerRight ?? undefined}
                  className="h-full"
                />
              )}
            </div>
          )}
        </div>
      </div>

      {fileDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={handleCloseDialog}
          />
          <div
            className={cn(
              'relative w-full max-w-md mx-4 p-4',
              'bg-card border border-border rounded-lg shadow-xl',
              'animate-in fade-in-0 zoom-in-95 duration-200'
            )}
          >
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-foreground">
                {dialogMode === 'add' ? 'Create New File' : 'Rename File'}
              </h2>
              <button
                type="button"
                onClick={handleCloseDialog}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {dialogMode === 'add' && recommendedMissing.length > 0 && (
              <div className="mb-4">
                <div className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Recommended Files
                </div>
                <div className="flex flex-wrap gap-2 mt-2">
                  {recommendedMissing.map((name) => {
                    const isSelected = pendingPath.trim().toLowerCase() === name.toLowerCase()
                    return (
                      <button
                        key={name}
                        type="button"
                        onClick={() => setPendingPath(name)}
                        className={cn(
                          'px-2 py-1 rounded-md text-xs border transition-colors',
                          isSelected
                            ? 'bg-primary/20 border-primary text-primary'
                            : 'border-border text-foreground hover:bg-muted'
                        )}
                      >
                        {name}
                      </button>
                    )
                  })}
                </div>
              </div>
            )}

            <div className="mb-4 space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="agent-file-path">
                File path
              </label>
              <Input
                id="agent-file-path"
                value={pendingPath}
                onChange={(event) => setPendingPath(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.preventDefault()
                    void handleConfirmDialog()
                  }
                }}
                placeholder="path/to/file.md"
              />
              <p className="text-xs text-muted-foreground">
                You can create folders by including slashes in the path.
              </p>
            </div>

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={handleCloseDialog}
                className={cn(
                  'px-4 py-2 text-sm rounded-lg transition-colors',
                  'bg-muted hover:bg-muted/80 text-foreground'
                )}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleConfirmDialog()}
                disabled={!isDialogValid}
                className={cn(
                  'px-4 py-2 text-sm rounded-lg transition-colors',
                  'bg-primary hover:bg-primary/90 text-primary-foreground',
                  !isDialogValid && 'opacity-50 cursor-not-allowed'
                )}
              >
                {dialogMode === 'add' ? 'Create' : 'Rename'}
              </button>
            </div>
          </div>
        </div>
      )}

      {contextMenu && (
        <AgentFileContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          path={contextMenu.path}
          isDir={contextMenu.isDir}
          isReserved={contextMenu.isReserved}
          onClose={() => setContextMenu(null)}
          onRename={() => handleStartRename(contextMenu.path)}
          onDelete={() => void handleDelete(contextMenu.path)}
        />
      )}
    </>
  )
}

interface AgentFileContextMenuProps {
  x: number
  y: number
  path: string
  isDir: boolean
  isReserved: boolean
  onClose: () => void
  onRename: () => void
  onDelete: () => void
}

function AgentFileContextMenu({
  x,
  y,
  path,
  isDir,
  isReserved,
  onClose,
  onRename,
  onDelete,
}: AgentFileContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    const timer = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleEscape)
    }, 0)

    return () => {
      clearTimeout(timer)
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [onClose])

  useEffect(() => {
    if (!menuRef.current) return
    const rect = menuRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let adjustedX = x
    let adjustedY = y

    if (x + rect.width > viewportWidth) {
      adjustedX = viewportWidth - rect.width - 8
    }
    if (y + rect.height > viewportHeight) {
      adjustedY = viewportHeight - rect.height - 8
    }

    menuRef.current.style.left = `${adjustedX}px`
    menuRef.current.style.top = `${adjustedY}px`
  }, [x, y])

  const handleRename = useCallback(() => {
    if (isReserved) return
    onRename()
    onClose()
  }, [isReserved, onRename, onClose])

  const handleDelete = useCallback(() => {
    if (isReserved) return
    onDelete()
    onClose()
  }, [isReserved, onDelete, onClose])

  const itemClass = (enabled: boolean, variant: 'default' | 'danger' = 'default') =>
    cn(
      'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm transition-colors',
      enabled
        ? variant === 'danger'
          ? 'text-rose-600 hover:text-rose-700 hover:bg-rose-50'
          : 'text-foreground hover:bg-muted'
        : 'text-muted-foreground/50 cursor-not-allowed'
    )

  const label = isDir ? 'folder' : 'file'
  const displayName = path.split('/').pop() ?? path

  return (
    <div
      ref={menuRef}
      className={cn(
        'fixed z-50 min-w-[180px] overflow-visible rounded-md',
        'bg-popover border border-border shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100'
      )}
      style={{ left: x, top: y }}
    >
      <div className="px-2 pt-2 pb-1 text-xs text-muted-foreground truncate">
        {displayName}
      </div>
      <div className="p-1">
        <button
          type="button"
          onClick={handleRename}
          disabled={isReserved}
          className={itemClass(!isReserved)}
        >
          <Pencil className="h-4 w-4" />
          <span>Rename {label}</span>
        </button>
        <button
          type="button"
          onClick={handleDelete}
          disabled={isReserved}
          className={itemClass(!isReserved, 'danger')}
        >
          <Trash2 className="h-4 w-4" />
          <span>Delete {label}</span>
        </button>
      </div>
    </div>
  )
}
