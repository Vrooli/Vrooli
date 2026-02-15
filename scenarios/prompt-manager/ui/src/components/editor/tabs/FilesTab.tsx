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
  type DragEvent as ReactDragEvent,
  type ReactNode,
  type MouseEvent as ReactMouseEvent,
} from 'react'
import {
  File,
  FileText,
  Folder,
  FolderOpen,
  GripVertical,
  ListChecks,
  PanelLeftClose,
  PanelLeftOpen,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Sparkles,
  Trash2,
  X,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { toast } from '@/hooks/use-toast'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import type { AgentFileEntry } from '@/types/agent'
import type { HighlightRequest } from '@/lib/highlight'
import type { ContentSearchMatch } from '@/lib/schemas'
import { createHighlightMatch } from '@/lib/highlight'
import * as agentService from '@/services/agentService'
import { useResizableSplitPanel } from '@/hooks/useResizableSplitPanel'
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
  renameFileOrderPath: (fromPath: string, toPath: string, isDir: boolean) => void
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
  /** Cross-reference highlight request */
  highlightRequest?: HighlightRequest | null
  /** Called after highlight is applied (clears URL params) */
  onHighlightHandled?: () => void
}

interface FileNode {
  name: string
  path: string
  isDir: boolean
  children: FileNode[]
}

interface TemplateOption {
  id: string
  label: string
  content: string
}

const RESERVED_AGENT_FILE = 'agent.json'
const RECOMMENDED_AGENT_FILES = ['AGENTS.md', 'SOUL.md', 'TOOLS.md'] as const

function buildDefaultFileOrder(paths: string[]): string[] {
  const normalized = [...paths]
  normalized.sort((a, b) => {
    const aLower = a.toLowerCase()
    const bLower = b.toLowerCase()
    if (aLower === 'soul.md') return bLower === 'soul.md' ? 0 : -1
    if (bLower === 'soul.md') return 1
    return aLower.localeCompare(bLower)
  })
  return normalized
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
  renameFileOrderPath,
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
  highlightRequest,
  onHighlightHandled,
}: FilesTabProps) {
  void _updateFields

  const {
    width: filesSidebarWidth,
    isResizing: isFilesSidebarResizing,
    isCollapsed: isFilesSidebarCollapsed,
    containerRef: filesContainerRef,
    handleResizeStart: handleFilesSidebarResizeStart,
    expand: expandFilesSidebar,
    collapse: collapseFilesSidebar,
  } = useResizableSplitPanel({
    defaultWidth: 256,
    minWidth: 180,
    maxWidthRatio: 0.5,
    snapCloseThreshold: 120,
    storageKey: 'pm.agentFilesSidebarWidth',
  })

  const [files, setFiles] = useState<AgentFileEntry[]>([])
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())
  const [fileContent, setFileContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [isFileLoading, setIsFileLoading] = useState(false)
  const [isFileSaving, setIsFileSaving] = useState(false)
  const [templateOptionsByFile, setTemplateOptionsByFile] = useState<Record<string, TemplateOption[]>>({})
  const [fileDialogOpen, setFileDialogOpen] = useState(false)
  const [dialogMode, setDialogMode] = useState<'add' | 'rename'>('add')
  const [pendingPath, setPendingPath] = useState('')
  const [renameSourcePath, setRenameSourcePath] = useState<string | null>(null)
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null)
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    path: string
    isDir: boolean
    isReserved: boolean
  } | null>(null)
  const skipFileLoadRef = useRef<string | null>(null)

  // Cross-reference highlight state
  const [highlightMatches, setHighlightMatches] = useState<ContentSearchMatch[]>([])
  const [highlightScrollToLine, setHighlightScrollToLine] = useState<number | null>(null)

  const tree = useMemo(() => buildFileTree(files), [files])
  const recommendedMissing = useMemo(
    () =>
      RECOMMENDED_AGENT_FILES.filter(
        (name) => !files.some((file) => file.path.toLowerCase() === name.toLowerCase())
      ),
    [files]
  )

  const markdownFiles = useMemo(
    () => files.filter((file) => !file.isDir && isMarkdownFile(file.path)),
    [files]
  )
  const markdownPaths = useMemo(
    () => markdownFiles.map((file) => file.path),
    [markdownFiles]
  )
  const defaultFileOrder = useMemo(
    () => buildDefaultFileOrder(markdownPaths),
    [markdownPaths]
  )
  const hasExplicitOrder = formState.fileOrder.length > 0
  const fileOrder = hasExplicitOrder ? formState.fileOrder : defaultFileOrder
  const markdownPathMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const path of markdownPaths) {
      map.set(path.toLowerCase(), path)
    }
    return map
  }, [markdownPaths])
  const orderEntries = useMemo(
    () =>
      fileOrder.map((path) => ({
        path,
        exists: markdownPathMap.has(path.toLowerCase()),
      })),
    [fileOrder, markdownPathMap]
  )
  const missingOrderEntries = useMemo(() => {
    if (!hasExplicitOrder) return []
    return formState.fileOrder.filter((path) => !markdownPathMap.has(path.toLowerCase()))
  }, [formState.fileOrder, hasExplicitOrder, markdownPathMap])
  const unlistedFiles = useMemo(() => {
    if (!hasExplicitOrder) return []
    const orderSet = new Set(formState.fileOrder.map((path) => path.toLowerCase()))
    return defaultFileOrder.filter((path) => !orderSet.has(path.toLowerCase()))
  }, [defaultFileOrder, formState.fileOrder, hasExplicitOrder])

  const selectedEntry = useMemo(
    () => (selectedPath ? files.find((file) => file.path === selectedPath) : undefined),
    [files, selectedPath]
  )

  const isReservedSelected = isReservedPath(selectedPath)
  const isDirectorySelected = selectedEntry?.isDir ?? false
  const isFileEditorActive = Boolean(selectedPath && !isDirectorySelected && !isReservedSelected)
  const isFileDirty = isFileEditorActive && fileContent !== originalContent
  const selectedTemplateKey = selectedPath ? selectedPath.split('/').pop()?.toLowerCase() : null
  const templateOptions = selectedTemplateKey ? templateOptionsByFile[selectedTemplateKey] : undefined

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
    let cancelled = false
    agentService.listAgentFileTemplates()
      .then((templates) => {
        if (cancelled) return
        const next: Record<string, TemplateOption[]> = {}
        for (const template of templates) {
          const key = template.fileName.toLowerCase()
          const option: TemplateOption = {
            id: template.id,
            label: template.name,
            content: template.content,
          }
          if (!next[key]) {
            next[key] = [option]
          } else {
            next[key].push(option)
          }
        }
        for (const key of Object.keys(next)) {
          next[key]?.sort((a, b) => a.label.localeCompare(b.label))
        }
        setTemplateOptionsByFile(next)
      })
      .catch((error: unknown) => {
        console.warn('[FilesTab] Failed to load templates:', error)
        toast({
          title: 'Unable to load templates',
          description: 'Check the API server and try again.',
        })
      })

    return () => {
      cancelled = true
    }
  }, [])

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

  // Handle highlight request: auto-select file and create match decorations
  useEffect(() => {
    if (!highlightRequest) {
      setHighlightMatches([])
      setHighlightScrollToLine(null)
      return
    }

    // Auto-select the file if specified and different from current
    if (highlightRequest.file && highlightRequest.file !== selectedPath) {
      // Check that the requested file exists
      if (files.some((f) => f.path === highlightRequest.file)) {
        setSelectedPath(highlightRequest.file)
      }
      // The match will be created once file content loads (via the dependency below)
      return
    }

    // File is already selected (or no file specified for skill highlights)
    // Create the highlight match from current file content
    if (fileContent && !isFileLoading) {
      const match = createHighlightMatch(fileContent, highlightRequest)
      if (match) {
        setHighlightMatches([match])
        setHighlightScrollToLine(highlightRequest.line)
      } else {
        setHighlightMatches([])
        setHighlightScrollToLine(highlightRequest.line)
      }
      onHighlightHandled?.()
    }
  }, [highlightRequest, selectedPath, fileContent, isFileLoading, files, onHighlightHandled])

  // Clear highlights when user selects a different file manually
  const prevSelectedPathRef = useRef(selectedPath)
  useEffect(() => {
    if (prevSelectedPathRef.current !== selectedPath) {
      // Only clear if this wasn't triggered by highlightRequest
      if (!highlightRequest || highlightRequest.file !== selectedPath) {
        setHighlightMatches([])
        setHighlightScrollToLine(null)
      }
      prevSelectedPathRef.current = selectedPath
    }
  }, [selectedPath, highlightRequest])

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

  const handleSetDefaultFileOrder = useCallback(() => {
    if (defaultFileOrder.length === 0) return
    updateField('fileOrder', defaultFileOrder)
  }, [defaultFileOrder, updateField])

  const handleAddUnlistedFiles = useCallback(() => {
    if (unlistedFiles.length === 0) return
    const next = [...formState.fileOrder, ...unlistedFiles]
    updateField('fileOrder', next)
  }, [formState.fileOrder, unlistedFiles, updateField])

  const handleRemoveMissingEntries = useCallback(() => {
    if (missingOrderEntries.length === 0) return
    const missingSet = new Set(missingOrderEntries.map((path) => path.toLowerCase()))
    const next = formState.fileOrder.filter((path) => !missingSet.has(path.toLowerCase()))
    updateField('fileOrder', next)
  }, [formState.fileOrder, missingOrderEntries, updateField])

  const reorderFileOrder = useCallback(
    (fromIndex: number, toIndex: number) => {
      if (fromIndex === toIndex) return
      const next = [...fileOrder]
      const [moved] = next.splice(fromIndex, 1)
      if (!moved) return
      next.splice(toIndex, 0, moved)
      updateField('fileOrder', next)
    },
    [fileOrder, updateField]
  )

  const handleDragStart = useCallback(
    (index: number) => (event: ReactDragEvent<HTMLDivElement>) => {
      setDraggedIndex(index)
      event.dataTransfer.effectAllowed = 'move'
      event.dataTransfer.setData('text/plain', String(index))
    },
    []
  )

  const handleDragOver = useCallback(
    (event: ReactDragEvent<HTMLDivElement>) => {
      event.preventDefault()
      event.dataTransfer.dropEffect = 'move'
    },
    []
  )

  const handleDrop = useCallback(
    (index: number) => (event: ReactDragEvent<HTMLDivElement>) => {
      event.preventDefault()
      const fallbackIndex = Number(event.dataTransfer.getData('text/plain'))
      const fromIndex = draggedIndex ?? (Number.isNaN(fallbackIndex) ? null : fallbackIndex)
      if (fromIndex === null) {
        setDraggedIndex(null)
        return
      }
      reorderFileOrder(fromIndex, index)
      setDraggedIndex(null)
    },
    [draggedIndex, reorderFileOrder]
  )

  const handleDragEnd = useCallback(() => {
    setDraggedIndex(null)
  }, [])

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
        const sourceEntry = files.find((entry) => entry.path.toLowerCase() === from.toLowerCase())
        renameFileOrderPath(from, to, sourceEntry?.isDir ?? false)
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
    [agentId, ensureExpandedForPath, files, refreshFiles, renameFileOrderPath, selectedPath]
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
    [agentId, refreshFiles, selectedPath]
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
            key={template.id}
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
      <div ref={filesContainerRef} className={cn('h-full flex min-h-0', isFilesSidebarResizing && 'select-none')}>
        {isFilesSidebarCollapsed ? (
          <div className="flex-shrink-0 w-10 border-r border-border flex flex-col items-center py-2">
            <button
              type="button"
              onClick={expandFilesSidebar}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
              title="Expand file list"
            >
              <PanelLeftOpen className="h-4 w-4" />
            </button>
          </div>
        ) : (
          <>
            <div className="flex-shrink-0 border-r border-border flex flex-col min-h-0" style={{ width: filesSidebarWidth }}>
              <div className="flex items-center justify-between px-3 py-2 border-b border-border">
                <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Agent Files</span>
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={collapseFilesSidebar}
                    className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
                    title="Collapse"
                  >
                    <PanelLeftClose className="h-4 w-4" />
                  </button>
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
            <div
              role="separator"
              aria-orientation="vertical"
              onMouseDown={handleFilesSidebarResizeStart}
              className="relative flex-shrink-0 w-3 cursor-col-resize group"
            >
              <div className="absolute left-1 top-0 h-full w-0.5 bg-border group-hover:bg-primary/50 transition-colors" />
            </div>
          </>
        )}

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
                  <div className="text-xs text-muted-foreground">Appearance + file ordering</div>
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
              <div className="rounded-lg border border-border/60 bg-muted/30 p-4 space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold">Prompt File Order</div>
                    <div className="text-xs text-muted-foreground">
                      Drag to reorder how markdown files are assembled for this agent.
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={handleSetDefaultFileOrder}
                    className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                    disabled={defaultFileOrder.length === 0}
                    title="Reset to default order"
                  >
                    <RotateCcw className="h-3.5 w-3.5" />
                    Reset default
                  </button>
                </div>

                {!hasExplicitOrder && defaultFileOrder.length > 0 && (
                  <div className="rounded-md border border-dashed border-border/80 bg-background/60 px-3 py-2 text-xs text-muted-foreground">
                    Using the default order (SOUL.md first, then A–Z). Set an explicit order to
                    customize.
                  </div>
                )}

                {hasExplicitOrder && (missingOrderEntries.length > 0 || unlistedFiles.length > 0) && (
                  <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200/90 space-y-1">
                    {missingOrderEntries.length > 0 && (
                      <div>
                        Missing files in order: {missingOrderEntries.join(', ')}
                      </div>
                    )}
                    {unlistedFiles.length > 0 && (
                      <div>
                        Unlisted markdown files: {unlistedFiles.join(', ')}
                      </div>
                    )}
                  </div>
                )}

                {orderEntries.length === 0 ? (
                  <div className="text-xs text-muted-foreground">
                    No markdown files found yet. Create one to start ordering.
                  </div>
                ) : (
                  <div className="space-y-2">
                    {orderEntries.map((entry, index) => (
                      <div
                        key={`${entry.path}-${index}`}
                        role="button"
                        tabIndex={0}
                        draggable={orderEntries.length > 1}
                        onDragStart={handleDragStart(index)}
                        onDragOver={handleDragOver}
                        onDrop={handleDrop(index)}
                        onDragEnd={handleDragEnd}
                        className={cn(
                          'flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs',
                          entry.exists ? 'border-border/70 bg-background/70' : 'border-amber-500/40 bg-amber-500/10',
                          orderEntries.length > 1 && 'cursor-grab active:cursor-grabbing',
                          draggedIndex === index && 'opacity-60'
                        )}
                      >
                        <GripVertical className="h-3.5 w-3.5 text-muted-foreground/70" />
                        <span className="flex-1 truncate">{entry.path}</span>
                        <span
                          className={cn(
                            'rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide',
                            entry.exists
                              ? 'bg-emerald-500/15 text-emerald-200'
                              : 'bg-amber-500/20 text-amber-200'
                          )}
                        >
                          {entry.exists ? 'Ready' : 'Missing'}
                        </span>
                      </div>
                    ))}
                  </div>
                )}

                <div className="flex flex-wrap gap-2 pt-1">
                  {!hasExplicitOrder && defaultFileOrder.length > 0 && (
                    <button
                      type="button"
                      onClick={handleSetDefaultFileOrder}
                      className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2.5 py-1 text-xs text-foreground hover:bg-muted"
                    >
                      <ListChecks className="h-3.5 w-3.5" />
                      Use default order
                    </button>
                  )}
                  {unlistedFiles.length > 0 && (
                    <button
                      type="button"
                      onClick={handleAddUnlistedFiles}
                      className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2.5 py-1 text-xs text-foreground hover:bg-muted"
                    >
                      <Plus className="h-3.5 w-3.5" />
                      Add unlisted files
                    </button>
                  )}
                  {missingOrderEntries.length > 0 && (
                    <button
                      type="button"
                      onClick={handleRemoveMissingEntries}
                      className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2.5 py-1 text-xs text-foreground hover:bg-muted"
                    >
                      <ListChecks className="h-3.5 w-3.5" />
                      Remove missing entries
                    </button>
                  )}
                </div>
              </div>
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
                  searchMatches={highlightMatches.length > 0 ? highlightMatches : undefined}
                  scrollToLine={highlightScrollToLine}
                  onScrollToLineHandled={() => setHighlightScrollToLine(null)}
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
