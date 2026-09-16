/**
 * TeamFilesTab - Shared files browser and editor for teams.
 *
 * Features:
 * - File tree scoped to the team's shared folder
 * - Add/rename/delete files and folders
 * - Markdown editor for all files
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
  Eye,
  Code2,
  AlertCircle,
  PanelLeftClose,
  PanelLeftOpen,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  X,
  MoreHorizontal,
  Search,
  Download,
  ExternalLink,
  WrapText,
  Type,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { toast } from '@/hooks/use-toast'
import { useResizableSplitPanel } from '@/hooks/useResizableSplitPanel'
import { useGlobalKeydown } from '@/hooks/useGlobalKeydown'
import { useIsMobile } from '@/hooks/useMediaQuery'
import type { TeamSharedFileEntry } from '@/types/team'
import type { HighlightRequest } from '@/lib/highlight'
import type { ContentSearchMatch, EffortWorkspace, EffortWorkspaceFile } from '@/lib/schemas'
import { createHighlightMatch } from '@/lib/highlight'
import * as teamService from '@/services/teamService'
import { SkillContentEditor } from '../SkillContentEditor'
import { FilePathMenu } from '../FilePathMenu'
import { DropdownItem, ToolbarDropdown } from '../ToolbarDropdown'
import { MarkdownRenderer } from '@/components/markdown'
import { CopyIconButton } from '@vrooli/react-component-library/CopyIconButton/1.0.1'
import { CollectionPage } from '@vrooli/react-component-library/CollectionPage/1.7.1'

interface TeamFilesTabProps {
  teamId: string
  /** Only finite delivery teams have a linked effort workspace projection. */
  showEffortWorkspaces?: boolean
  /** Cross-reference highlight request */
  highlightRequest?: HighlightRequest | null
  /** Called after highlight is applied (clears URL params) */
  onHighlightHandled?: () => void
  className?: string
}

interface FileNode {
  name: string
  path: string
  isDir: boolean
  children: FileNode[]
}

const RECOMMENDED_TEAM_FILES = ['TEAM.md'] as const

function buildFileTree(entries: TeamSharedFileEntry[]): FileNode {
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

type TeamFileKind = 'markdown' | 'code' | 'data' | 'html' | 'image' | 'audio' | 'video' | 'pdf' | 'unsupported'

function teamFileKind(path: string): TeamFileKind {
  const lower = path.toLowerCase()
  if (lower.endsWith('.md') || lower.endsWith('.mdx')) return 'markdown'
  if (lower.endsWith('.json') || lower.endsWith('.yaml') || lower.endsWith('.yml') || lower.endsWith('.csv')) return 'data'
  if (lower.endsWith('.html') || lower.endsWith('.htm')) return 'html'
  if (/\.(ts|tsx|js|jsx|go|py|rs|sh|css|scss|sql|toml|xml|diff|patch)$/.test(lower)) return 'code'
  if (/\.(png|jpe?g|gif|webp|svg)$/.test(lower)) return 'image'
  if (/\.(mp3|wav|ogg|m4a)$/.test(lower)) return 'audio'
  if (/\.(mp4|webm|mov)$/.test(lower)) return 'video'
  if (lower.endsWith('.pdf')) return 'pdf'
  return 'unsupported'
}

function formatFileSize(size?: number): string {
  if (!Number.isFinite(size)) return 'Size unavailable'
  if ((size ?? 0) < 1024) return `${size ?? 0} B`
  if ((size ?? 0) < 1024 * 1024) return `${((size ?? 0) / 1024).toFixed(1)} KB`
  return `${((size ?? 0) / (1024 * 1024)).toFixed(1)} MB`
}

function kindLabel(kind: TeamFileKind): string {
  const labels: Record<TeamFileKind, string> = { markdown: 'Markdown', code: 'Source', data: 'Structured data', html: 'HTML', image: 'Image', audio: 'Audio', video: 'Video', pdf: 'PDF', unsupported: 'Unsupported' }
  return labels[kind]
}

function fileMimeType(kind: TeamFileKind, path: string): string {
  if (kind === 'html') return 'text/html'
  if (kind === 'markdown') return 'text/markdown'
  if (kind === 'data' && path.toLowerCase().endsWith('.json')) return 'application/json'
  const lower = path.toLowerCase()
  if (lower.endsWith('.svg')) return 'image/svg+xml'
  if (kind === 'image') return 'image/*'
  if (kind === 'audio') return 'audio/*'
  if (kind === 'video') return 'video/*'
  if (kind === 'pdf') return 'application/pdf'
  return 'text/plain'
}

function isDataUrl(content: string): boolean {
  return /^data:[^,]+,/.test(content.trim())
}

export function TeamFilesTab({ teamId, showEffortWorkspaces = false, highlightRequest, onHighlightHandled, className }: TeamFilesTabProps) {
  const isMobile = useIsMobile()
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
    storageKey: 'pm.teamFilesSidebarWidth',
  })

  const [files, setFiles] = useState<TeamSharedFileEntry[]>([])
  const [effortWorkspaces, setEffortWorkspaces] = useState<EffortWorkspace[]>([])
  const [effortWorkspacesUnavailable, setEffortWorkspacesUnavailable] = useState<Array<{ effortRef: string; reason: string }>>([])
  const [effortWorkspacesLoading, setEffortWorkspacesLoading] = useState(false)
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [selectedEffortFile, setSelectedEffortFile] = useState<{ workspace: EffortWorkspace; file: EffortWorkspaceFile } | null>(null)
  const [fileFilter, setFileFilter] = useState('')
  const [mobilePreview, setMobilePreview] = useState(false)
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
  } | null>(null)
  const skipFileLoadRef = useRef<string | null>(null)
  const loadGenerationRef = useRef(0)
  const loadedTeamRef = useRef<string | null>(null)

  // Cross-reference highlight state
  const [highlightMatches, setHighlightMatches] = useState<ContentSearchMatch[]>([])
  const [highlightScrollToLine, setHighlightScrollToLine] = useState<number | null>(null)

  const filteredFiles = useMemo(() => {
    const query = fileFilter.trim().toLowerCase()
    if (!query) return files
    const matchingPaths = new Set<string>()
    files.forEach((file) => {
      if (file.path.toLowerCase().includes(query)) {
        const parts = file.path.split('/').filter(Boolean)
        parts.forEach((_, index) => matchingPaths.add(parts.slice(0, index + 1).join('/')))
      }
    })
    return files.filter((file) => matchingPaths.has(file.path))
  }, [fileFilter, files])
  const tree = useMemo(() => buildFileTree(filteredFiles), [filteredFiles])
  const recommendedMissing = useMemo(
    () =>
      RECOMMENDED_TEAM_FILES.filter(
        (name) => !files.some((file) => file.path.toLowerCase() === name.toLowerCase())
      ),
    [files]
  )

  const selectedEntry = useMemo(
    () => (selectedPath ? files.find((file) => file.path === selectedPath) : undefined),
    [files, selectedPath]
  )

  const isDirectorySelected = selectedEntry?.isDir ?? false
  const isFileEditorActive = Boolean(selectedPath && !isDirectorySelected)
  const selectedKind = selectedPath ? teamFileKind(selectedPath) : 'unsupported'
  const selectedNeedsArtifactPreview = ['image', 'audio', 'video', 'pdf', 'unsupported'].includes(selectedKind)
  const isEffortFileSelected = selectedEffortFile !== null
  const isFileDirty = isFileEditorActive && !isEffortFileSelected && fileContent !== originalContent

  const refreshFiles = useCallback(async () => {
    const generation = ++loadGenerationRef.current

    // Team navigation must never show a previous team's shared files. Clear the
    // listing synchronously when the target team changes; the generation guard
    // discards any in-flight response for the previous team.
    if (loadedTeamRef.current !== teamId) {
      loadedTeamRef.current = teamId
      setFiles([])
      setSelectedPath(null)
      setFileContent('')
      setOriginalContent('')
    }

    try {
      const entries = await teamService.listTeamSharedFiles(teamId)
      if (generation !== loadGenerationRef.current) return
      setFiles(entries)
    } catch (error) {
      if (generation !== loadGenerationRef.current) return
      console.warn('[TeamFilesTab] Failed to load shared files:', error)
      toast({
        title: 'Unable to load shared files',
        description: 'Check the API server and try again.',
      })
    }
  }, [teamId])

  useEffect(() => {
    void refreshFiles()
  }, [refreshFiles])

  const refreshEffortWorkspaces = useCallback(async () => {
    if (!showEffortWorkspaces) {
      setEffortWorkspaces([])
      setEffortWorkspacesUnavailable([])
      return
    }
    setEffortWorkspacesLoading(true)
    try {
      const result = await teamService.listEffortWorkspaces(teamId)
      setEffortWorkspaces(result.workspaces)
      setEffortWorkspacesUnavailable(result.unavailable)
    } catch (error) {
      console.warn('[TeamFilesTab] Failed to load effort workspaces:', error)
      setEffortWorkspaces([])
      setEffortWorkspacesUnavailable([{ effortRef: 'linked effort', reason: error instanceof Error ? error.message : 'Unable to load effort workspaces' }])
    } finally {
      setEffortWorkspacesLoading(false)
    }
  }, [showEffortWorkspaces, teamId])

  useEffect(() => {
    void refreshEffortWorkspaces()
  }, [refreshEffortWorkspaces])

  useEffect(() => {
    setExpandedPaths(new Set())
    setSelectedEffortFile(null)
  }, [teamId])

  useEffect(() => {
    if (selectedEffortFile) return
    if (files.length === 0) {
      setSelectedPath(null)
      return
    }

    if (selectedPath && files.some((file) => file.path === selectedPath)) {
      return
    }

    const preferred =
      files.find((file) => file.path === 'TEAM.md')?.path ??
      files[0]?.path ??
      null
    setSelectedPath(preferred)
  }, [files, selectedPath, selectedEffortFile])

  useEffect(() => {
    if (selectedEffortFile) return
    if (!selectedPath || isDirectorySelected) {
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
    teamService.getTeamSharedFileContent(teamId, selectedPath)
      .then((content) => {
        if (cancelled) return
        setFileContent(content)
        setOriginalContent(content)
      })
      .catch((error: unknown) => {
        console.warn('[TeamFilesTab] Failed to load file content:', error)
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
  }, [teamId, selectedPath, isDirectorySelected, selectedEffortFile])

  useEffect(() => {
    if (!selectedEffortFile) return
    let cancelled = false
    setIsFileLoading(true)
    setFileContent('')
    setOriginalContent('')
    teamService.getEffortWorkspaceContent(selectedEffortFile.workspace.effortRef, selectedEffortFile.file.path)
      .then((result) => {
        if (!cancelled) setFileContent(result.content)
      })
      .catch((error: unknown) => {
        if (cancelled) return
        console.warn('[TeamFilesTab] Failed to load effort workspace file:', error)
        toast({
          title: 'Unable to load effort file',
          description: 'The linked effort workspace may no longer be available.',
        })
      })
      .finally(() => {
        if (!cancelled) setIsFileLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [selectedEffortFile])

  // Handle highlight request: auto-select file and create match decorations
  useEffect(() => {
    if (!highlightRequest) {
      setHighlightMatches([])
      setHighlightScrollToLine(null)
      return
    }

    // Auto-select the file if specified and different from current
    if (highlightRequest.file && highlightRequest.file !== selectedPath) {
      if (files.some((f) => f.path === highlightRequest.file)) {
        setSelectedPath(highlightRequest.file)
      }
      return
    }

    // File is already selected — create the highlight match
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
    setSelectedEffortFile(null)
    setSelectedPath(path)
  }, [ensureExpandedForPath, isFileDirty, selectedPath])

  const handleSelectEffortFile = useCallback((workspace: EffortWorkspace, file: EffortWorkspaceFile) => {
    if (file.isDir) return
    if (isFileDirty && !window.confirm('You have unsaved changes. Discard them?')) return
    setContextMenu(null)
    setSelectedPath(null)
    setSelectedEffortFile({ workspace, file })
  }, [isFileDirty])

  const handleSaveFile = useCallback(async () => {
    if (!selectedPath || !isFileEditorActive || !isFileDirty) return
    setIsFileSaving(true)
    try {
      await teamService.setTeamSharedFileContent(teamId, selectedPath, fileContent)
      setOriginalContent(fileContent)
      toast({
        title: 'File saved',
        description: `${selectedPath} updated.`,
      })
    } catch (error) {
      console.warn('[TeamFilesTab] Failed to save file:', error)
      toast({
        title: 'Unable to save file',
        description: 'Check the API server and try again.',
      })
    } finally {
      setIsFileSaving(false)
    }
  }, [teamId, selectedPath, isFileEditorActive, isFileDirty, fileContent])

  const handleDiscardFile = useCallback(() => {
    setFileContent(originalContent)
  }, [originalContent])

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
      if (!target) return
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
        await teamService.renameTeamSharedFile(teamId, { from, to })
        if (selectedPath === from) {
          skipFileLoadRef.current = to
          setSelectedPath(to)
        }
        ensureExpandedForPath(to)
        await refreshFiles()
        return true
      } catch (error) {
        console.warn('[TeamFilesTab] File operation failed:', error)
        toast({
          title: 'File operation failed',
          description: 'Check the file name and try again.',
        })
        return false
      }
    },
    [teamId, ensureExpandedForPath, refreshFiles, selectedPath]
  )

  const handleConfirmDialog = useCallback(async () => {
    const trimmed = pendingPath.trim()
    if (!trimmed) return

    let didSucceed = false

    try {
      if (dialogMode === 'add') {
        await teamService.createTeamSharedFile(teamId, { path: trimmed, content: '' })
        setSelectedPath(trimmed)
        ensureExpandedForPath(trimmed)
        await refreshFiles()
        didSucceed = true
      } else if (renameSourcePath) {
        didSucceed = await handleRenameFile(renameSourcePath, trimmed)
      }
    } catch (error) {
      console.warn('[TeamFilesTab] File operation failed:', error)
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
    teamId,
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
      if (!target) return
      const confirmed = window.confirm(`Delete ${target}? This cannot be undone.`)
      if (!confirmed) return

      try {
        await teamService.deleteTeamSharedFile(teamId, target)
        if (selectedPath === target) {
          setSelectedPath(null)
        }
        await refreshFiles()
      } catch (error) {
        console.warn('[TeamFilesTab] Failed to delete file:', error)
        toast({
          title: 'Unable to delete file',
          description: 'Check the API server and try again.',
        })
      } finally {
        setContextMenu(null)
      }
    },
    [teamId, refreshFiles, selectedPath]
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
      })
    },
    []
  )

  const selectSharedPath = useCallback((path: string, isDir: boolean) => {
    handleSelectPath(path, isDir)
    if (!isDir) setMobilePreview(true)
  }, [handleSelectPath])

  const selectEffortPath = useCallback((workspace: EffortWorkspace, file: EffortWorkspaceFile) => {
    handleSelectEffortFile(workspace, file)
    if (!file.isDir) setMobilePreview(true)
  }, [handleSelectEffortFile])

  const handleRenameSelectedFile = useCallback(
    async (nextFile: string) => {
      if (!selectedPath || isDirectorySelected) return
      const parentDir = selectedPath.split('/').slice(0, -1).join('/')
      const nextPath = parentDir ? `${parentDir}/${nextFile}` : nextFile
      if (!nextPath || nextPath === selectedPath) return
      await handleRenameFile(selectedPath, nextPath)
    },
    [handleRenameFile, isDirectorySelected, selectedPath]
  )

  const filePathMenu = useMemo<ReactNode | null>(() => {
    if (!selectedPath || isDirectorySelected) return null

    const segments = selectedPath.split('/').filter(Boolean)
    const baseName = segments.pop() ?? selectedPath
    const dirSegments = segments

    const relativePath = `teams/${teamId}/shared/${selectedPath}`
    const projectPath = `scenarios/prompt-manager/store/teams/${teamId}/shared/${selectedPath}`

    return (
      <FilePathMenu
        file={baseName}
        rootLabel="teams"
        pathSegments={[teamId, 'shared', ...dirSegments]}
        onFileChange={(nextFile) => void handleRenameSelectedFile(nextFile)}
        relativePath={relativePath}
        projectPath={projectPath}
        isEditable
        className="flex-shrink-0"
      />
    )
  }, [handleRenameSelectedFile, isDirectorySelected, selectedPath, teamId])

  const fileActionsMenu = useMemo<ReactNode | null>(() => {
    if (!selectedPath || isDirectorySelected) return null

    return (
      <ToolbarDropdown
        icon={<MoreHorizontal className="h-4 w-4" />}
        label="File actions"
        showChevron={false}
        align="right"
        className="h-8 w-8 p-0"
      >
        <DropdownItem
          onClick={() => handleStartRename()}
          icon={<Pencil className="h-4 w-4" />}
          label="Rename file"
        />
        <DropdownItem
          onClick={() => void handleDelete()}
          icon={<Trash2 className="h-4 w-4 text-destructive" />}
          label="Delete file"
        />
      </ToolbarDropdown>
    )
  }, [handleDelete, handleStartRename, isDirectorySelected, selectedPath])

  const headerRight = useMemo<ReactNode | null>(() => {
    if (filePathMenu && fileActionsMenu) {
      return (
        <>
          {filePathMenu}
          {fileActionsMenu}
        </>
      )
    }
    return filePathMenu ?? fileActionsMenu
  }, [fileActionsMenu, filePathMenu])

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
          onClick={() => selectSharedPath(node.path, node.isDir)}
          onContextMenu={(event) => handleContextMenu(event, node)}
          className={cn(
            'w-full flex items-center gap-2 rounded-md px-2 py-1 text-sm text-left',
            isSelected ? 'bg-primary/15 text-primary' : 'hover:bg-muted'
          )}
          style={{ paddingLeft: 8 + depth * 14 }}
        >
          {icon}
          <span className="min-w-0 flex-1 whitespace-normal break-all">{node.name}</span>
        </button>
        {node.isDir && isExpanded && node.children.map((child) => renderNode(child, depth + 1))}
      </div>
    )
  }

  const workspaceHeader = (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-primary/20 bg-primary/5 px-3 py-2">
      <div className="min-w-0">
        <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-primary">Team workspace</p>
        <p className="break-words text-sm text-muted-foreground">Shared files and linked effort evidence</p>
      </div>
      <span className="shrink-0 rounded-full border border-border px-2 py-1 text-[11px] text-muted-foreground">{files.length} shared</span>
    </div>
  )
  const workspaceFilters = (
    <label className="relative block">
      <Search aria-hidden="true" className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <Input value={fileFilter} onChange={(event) => setFileFilter(event.target.value)} placeholder="Filter files and linked evidence" aria-label="Filter team files" className="h-11 pl-9" />
    </label>
  )
  const fileCollection = (
    <div ref={filesContainerRef} className={cn('min-h-[24rem] min-w-0 overflow-x-hidden', isFilesSidebarResizing && 'select-none')}>
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
            <div
              className={cn('flex-shrink-0 border-r border-border flex min-w-0 flex-col min-h-0 w-full md:w-auto', mobilePreview && 'hidden md:flex')}
              style={isMobile ? undefined : { width: filesSidebarWidth }}
            >
              <div className="flex items-center justify-between px-3 py-2 border-b border-border">
                <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Files</span>
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
                    onClick={() => { void refreshFiles(); void refreshEffortWorkspaces() }}
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
                </div>
              </div>

              <div className="flex-1 overflow-y-auto px-2 py-2">
                <div className="mb-3">
                  <div className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Team shared</div>
                {files.length === 0 ? (
                  <div className="text-xs text-muted-foreground px-2 py-4">
                    No shared files yet. Create a file to get started.
                  </div>
                ) : filteredFiles.length === 0 ? (
                  <div className="px-2 py-4 text-xs text-muted-foreground">No files match “{fileFilter}”.</div>
                ) : (
                  renderNode(tree)
                )}
                </div>
                {showEffortWorkspaces && (
                  <div className="border-t border-border pt-3">
                    <div className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Effort workspaces</div>
                    {effortWorkspacesLoading ? <div className="px-2 py-2 text-xs text-muted-foreground">Loading linked workspaces…</div> : null}
                    {!effortWorkspacesLoading && effortWorkspaces.length === 0 && effortWorkspacesUnavailable.length === 0 ? (
                      <div className="px-2 py-2 text-xs text-muted-foreground">No linked workspace files.</div>
                    ) : null}
                    {effortWorkspacesUnavailable.length > 0 ? (
                      <div className="px-2 py-2 text-xs text-muted-foreground" title={effortWorkspacesUnavailable.map((item) => `${item.effortRef}: ${item.reason}`).join('\n')}>
                        Linked workspace unavailable
                      </div>
                    ) : null}
                    {effortWorkspaces.map((workspace) => (
                      <div key={workspace.effortRef} className="mt-2">
                        <div className="px-2 text-xs font-medium text-foreground" title={workspace.effortRef}>{workspace.slug}</div>
                        {workspace.files.map((file) => (
                          <button
                            key={`${workspace.effortRef}:${file.path}`}
                            type="button"
                            disabled={file.isDir}
                            onClick={() => selectEffortPath(workspace, file)}
                            className={cn(
                              'w-full flex items-center gap-2 rounded-md px-2 py-1 text-left text-sm text-muted-foreground',
                              selectedEffortFile?.workspace.effortRef === workspace.effortRef && selectedEffortFile.file.path === file.path ? 'bg-primary/15 text-primary' : 'hover:bg-muted',
                              file.isDir && 'cursor-default'
                            )}
                          >
                            {file.isDir ? <Folder className="h-4 w-4" /> : <FileText className="h-4 w-4" />}
                            <span className="min-w-0 flex-1 whitespace-normal break-all">{file.path}</span>
                          </button>
                        ))}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
            <div
              role="separator"
              aria-orientation="vertical"
              onMouseDown={handleFilesSidebarResizeStart}
              className="relative hidden w-3 flex-shrink-0 cursor-col-resize group md:block"
            >
              <div className="absolute left-1 top-0 h-full w-0.5 bg-border group-hover:bg-primary/50 transition-colors" />
            </div>
          </>
        )}

  </div>
  )
  const fileInspector = (
    <div className={cn('min-h-[24rem] min-w-0 flex min-h-0 flex-col', !mobilePreview && 'md:flex', mobilePreview ? 'flex' : 'hidden md:flex')}>
          {(mobilePreview || selectedEffortFile || isFileEditorActive) && <div className="flex items-start gap-2 border-b border-border px-3 py-2 md:hidden"><span className="min-w-0 break-words text-xs text-muted-foreground">{selectedEffortFile?.file.path ?? selectedPath ?? 'Preview'}</span></div>}
          {!selectedPath && !selectedEffortFile && (
            <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
              Select a file to view or edit.
            </div>
          )}

          {selectedEffortFile && (
            <div className="flex h-full min-h-0 flex-col overflow-hidden">
              {isFileLoading ? <p className="p-4 text-sm text-muted-foreground">Loading file…</p> : (
                <TeamFileArtifactPreview
                  path={selectedEffortFile.file.path}
                  content={fileContent}
                  size={selectedEffortFile.file.size}
                  sourceLabel={`${selectedEffortFile.workspace.slug} · linked effort workspace`}
                  readOnly
                />
              )}
            </div>
          )}

          {selectedPath && !selectedEffortFile && isDirectorySelected && (
            <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
              Select a file to view or edit.
            </div>
          )}

          {selectedPath && !selectedEffortFile && isFileEditorActive && (
            <div className="flex-1 min-h-0">
              {isFileLoading ? (
                <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
                  Loading file...
                </div>
              ) : selectedNeedsArtifactPreview ? (
                <TeamFileArtifactPreview
                  path={selectedPath}
                  content={fileContent}
                  size={selectedEntry?.size}
                  sourceLabel="team shared files"
                />
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
  )

  return (
    <>
      <div className={cn('h-full min-h-0 min-w-0 max-w-full overflow-x-hidden', className)}>
        <CollectionPage
          state="ready"
          gutter="none"
          mobilePane={mobilePreview ? 'inspector' : 'collection'}
          onMobilePaneChange={(pane) => setMobilePreview(pane === 'inspector')}
          backLabel="Back to files"
          detailLabel="Selected file"
          detailTitle={selectedEffortFile?.file.path ?? selectedPath ?? 'No file selected'}
          regions={{
            header: workspaceHeader,
            filters: workspaceFilters,
            collection: fileCollection,
            inspector: fileInspector,
          }}
        />
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
              <label className="text-sm font-medium text-foreground" htmlFor="team-file-path">
                File path
              </label>
              <Input
                id="team-file-path"
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
        <TeamFileContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          path={contextMenu.path}
          isDir={contextMenu.isDir}
          onClose={() => setContextMenu(null)}
          onRename={() => handleStartRename(contextMenu.path)}
          onDelete={() => void handleDelete(contextMenu.path)}
        />
      )}
    </>
  )
}

function TeamFileArtifactPreview({
  path,
  content,
  size,
  sourceLabel,
  readOnly = false,
}: {
  path: string
  content: string
  size?: number
  sourceLabel: string
  readOnly?: boolean
}) {
  const kind = teamFileKind(path)
  const [mode, setMode] = useState<'rendered' | 'source'>(kind === 'markdown' || kind === 'html' ? 'rendered' : 'source')
  const [copied, setCopied] = useState(false)
  const [fontSize, setFontSize] = useState<'sm' | 'md' | 'lg'>('md')
  const [wrap, setWrap] = useState(true)
  const previewIdentity = `${kind}:${path}`
  const previousPreviewIdentity = useRef(previewIdentity)
  const canDownload = Boolean(content)
  const canRenderDataMedia = isDataUrl(content)
  const isTruncated = content.endsWith('#truncated')
  const previewContent = content.replace(/#truncated$/, '')
  const textContent = content.replace(/^data:[^,]+,/, '')
  const sourceClass = fontSize === 'sm' ? 'text-[11px]' : fontSize === 'lg' ? 'text-sm' : 'text-xs'
  useEffect(() => {
    if (previousPreviewIdentity.current === previewIdentity) return
    previousPreviewIdentity.current = previewIdentity
    setMode(kind === 'markdown' || kind === 'html' ? 'rendered' : 'source')
    setFontSize('md')
    setWrap(true)
  }, [kind, previewIdentity])
  const createPreviewBlob = async () => {
    if (isDataUrl(previewContent)) {
      try {
        return await (await fetch(previewContent)).blob()
      } catch {
        // Keep the escape hatch usable even when the browser refuses a data URL.
      }
    }
    return new Blob([content], { type: fileMimeType(kind, path) })
  }
  const download = async () => {
    if (!canDownload) return
    const blob = await createPreviewBlob()
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = path.split('/').pop() || 'download'
    anchor.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 0)
  }
  const open = async () => {
    if (!canDownload) return
    const blob = await createPreviewBlob()
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank', 'noopener,noreferrer')
    window.setTimeout(() => URL.revokeObjectURL(url), 30_000)
  }
  const lines = content.split('\n')

  return (
    <section className="flex min-h-0 flex-1 flex-col bg-card" data-testid="team-file-artifact-preview" data-preview-mode={mode} aria-label={`${path} preview`}>
      <header className="shrink-0 border-b border-border bg-gradient-to-r from-primary/10 via-card to-card px-4 py-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <span className="grid h-8 w-8 shrink-0 place-items-center rounded-xl border border-primary/20 bg-primary/10 text-primary">
                {kind === 'markdown' ? <FileText className="h-4 w-4" aria-hidden="true" /> : <File className="h-4 w-4" aria-hidden="true" />}
              </span>
              <div className="min-w-0">
                <h4 className="break-all text-sm font-semibold text-foreground">{path}</h4>
                <p className="mt-0.5 text-xs text-muted-foreground">{kindLabel(kind)} · {formatFileSize(size)} · {sourceLabel}</p>
                {isTruncated && <p className="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">Preview truncated at the transport limit. Open the owning workspace for the complete artifact.</p>}
              </div>
            </div>
          </div>
          <div className="flex shrink-0 flex-wrap items-center gap-1.5">
            {(kind === 'markdown' || kind === 'html') && (
              <div className="flex rounded-lg border border-border bg-muted/30 p-0.5" role="group" aria-label="Preview mode">
                <button type="button" onClick={() => setMode('rendered')} aria-pressed={mode === 'rendered'} className={cn('inline-flex min-h-8 items-center gap-1 rounded-md px-2 text-xs font-medium', mode === 'rendered' && 'bg-background text-primary shadow-sm')}><Eye className="h-3.5 w-3.5" aria-hidden="true" />Rendered</button>
                <button type="button" onClick={() => setMode('source')} aria-pressed={mode === 'source'} className={cn('inline-flex min-h-8 items-center gap-1 rounded-md px-2 text-xs font-medium', mode === 'source' && 'bg-background text-primary shadow-sm')}><Code2 className="h-3.5 w-3.5" aria-hidden="true" />Source</button>
              </div>
            )}
            {['markdown', 'code', 'data', 'html'].includes(kind) && (
              <>
                <button type="button" onClick={() => setWrap((value) => !value)} aria-pressed={wrap} className={cn('inline-flex min-h-8 items-center gap-1 rounded-lg border border-border px-2 text-xs font-medium hover:bg-muted', wrap && 'bg-muted')}><WrapText className="h-3.5 w-3.5" aria-hidden="true" />Wrap</button>
                <div className="flex items-center rounded-lg border border-border" role="group" aria-label="Preview text size">
                  <Type className="mx-1.5 h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
                  {(['sm', 'md', 'lg'] as const).map((size) => <button key={size} type="button" onClick={() => setFontSize(size)} aria-pressed={fontSize === size} className={cn('min-h-8 px-1.5 text-xs font-medium hover:bg-muted', fontSize === size && 'bg-muted text-primary')}>{size === 'sm' ? 'A−' : size === 'lg' ? 'A+' : 'A'}</button>)}
                </div>
              </>
            )}
            {canDownload && <>
              <button type="button" onClick={download} className="inline-flex min-h-8 items-center gap-1 rounded-lg border border-border px-2 text-xs font-medium hover:bg-muted"><Download className="h-3.5 w-3.5" aria-hidden="true" />Download</button>
              <button type="button" onClick={open} className="inline-flex min-h-8 items-center gap-1 rounded-lg border border-border px-2 text-xs font-medium hover:bg-muted"><ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />Open</button>
            </>}
            <span className="inline-flex min-h-8 items-center gap-1 rounded-lg border border-border px-1 text-xs font-medium hover:bg-muted">
              <CopyIconButton value={path} aria-label="Copy file path" title={copied ? 'Path copied' : 'Copy file path'} copiedLabel="Path copied" failedLabel="Copy failed" onCopied={() => { setCopied(true); window.setTimeout(() => setCopied(false), 1600) }} className="h-7 w-7" />
              <span aria-hidden="true" className="pr-1">{copied ? 'Copied' : 'Copy path'}</span>
            </span>
            <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-[11px] font-medium text-amber-700 dark:text-amber-300">{readOnly ? 'Read only' : 'Editable'}</span>
          </div>
        </div>
      </header>

      {kind === 'markdown' && mode === 'rendered' ? (
        <div className="min-h-0 flex-1 overflow-auto px-5 py-5">
          {content.trim() ? <MarkdownRenderer content={content} /> : <EmptyTeamFileState label="This Markdown file is empty." />}
        </div>
      ) : kind === 'html' && mode === 'rendered' ? (
        <div className="min-h-0 flex-1 overflow-auto p-5">
          <div className="mb-3 flex items-start gap-2 rounded-xl border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-800 dark:text-amber-200">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span>HTML is isolated in a sandboxed preview. Scripts, forms, and same-origin access are disabled.</span>
          </div>
          {content.trim() ? <iframe title={`${path} rendered preview`} sandbox="" srcDoc={content} className="min-h-[28rem] w-full rounded-2xl border border-border bg-white" /> : <EmptyTeamFileState label="This HTML file is empty." />}
        </div>
      ) : kind === 'html' ? (
        <div className="min-h-0 flex-1 overflow-auto p-5">
          <TeamFileSource content={content} className={sourceClass} wrap={wrap} />
        </div>
      ) : ['image', 'audio', 'video', 'pdf', 'unsupported'].includes(kind) ? (
        <div className="min-h-0 flex-1 overflow-auto p-5">
          {canRenderDataMedia && kind === 'image' ? <div className="flex min-h-48 items-center justify-center rounded-2xl border border-border bg-[linear-gradient(45deg,#eee_25%,transparent_25%),linear-gradient(-45deg,#eee_25%,transparent_25%),linear-gradient(45deg,transparent_75%,#eee_75%),linear-gradient(-45deg,transparent_75%,#eee_75%)] bg-[length:24px_24px] bg-[position:0_0,0_12px,12px_-12px,-12px_0] p-6"><img src={previewContent} alt={path} className="max-h-[min(70vh,48rem)] max-w-full rounded-xl object-contain" /></div> : canRenderDataMedia && kind === 'audio' ? <audio controls src={previewContent} className="w-full" /> : canRenderDataMedia && kind === 'video' ? <video controls src={previewContent} className="max-h-[70vh] w-full rounded-xl" /> : canRenderDataMedia && kind === 'pdf' ? <iframe title={`${path} PDF preview`} src={previewContent} className="min-h-[36rem] w-full rounded-xl border border-border" /> : <div className="flex min-h-48 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-muted/20 p-6 text-center"><AlertCircle className="mb-3 h-6 w-6 text-muted-foreground" aria-hidden="true" /><h5 className="text-sm font-semibold text-foreground">Preview unavailable from this transport</h5><p className="mt-1 max-w-md text-sm text-muted-foreground">{kindLabel(kind)} files are identified honestly, but this bounded Prompt Manager read did not provide safe bytes for an in-shell preview.</p><p className="mt-3 text-xs text-muted-foreground">Use Download or Open when content is available, or the owning workspace's permitted action.</p></div>}
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto p-4">
          {content ? <TeamFileSource content={textContent} lines={lines} className={sourceClass} wrap={wrap} /> : <EmptyTeamFileState label="This file is empty." />}
        </div>
      )}
    </section>
  )
}

function TeamFileSource({ content, lines = content.split('\n'), className = 'text-xs', wrap = false }: { content: string; lines?: string[]; className?: string; wrap?: boolean }) {
  return (
    <pre className={cn('overflow-auto rounded-2xl border border-border bg-slate-950 p-4 leading-6 text-slate-100 shadow-inner', className, wrap && 'whitespace-pre-wrap break-words')} data-testid="team-file-source">
      {lines.map((line, index) => <span key={index} className="block"><span className="mr-4 inline-block w-8 select-none text-right text-slate-500">{index + 1}</span>{line || ' '}</span>)}
    </pre>
  )
}

function EmptyTeamFileState({ label }: { label: string }) {
  return <div className="flex min-h-48 items-center justify-center rounded-2xl border border-dashed border-border bg-muted/20 p-6 text-center text-sm text-muted-foreground">{label}</div>
}

interface TeamFileContextMenuProps {
  x: number
  y: number
  path: string
  isDir: boolean
  onClose: () => void
  onRename: () => void
  onDelete: () => void
}

function TeamFileContextMenu({
  x,
  y,
  path,
  isDir,
  onClose,
  onRename,
  onDelete,
}: TeamFileContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    const timer = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
    }, 0)

    return () => {
      clearTimeout(timer)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [onClose])

  useGlobalKeydown((event) => {
    if (event.key === 'Escape') {
      onClose()
    }
  }, { target: 'document' })

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

  const itemClass = (variant: 'default' | 'danger' = 'default') =>
    cn(
      'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-sm transition-colors',
      variant === 'danger'
        ? 'text-rose-600 hover:text-rose-700 hover:bg-rose-50'
        : 'text-foreground hover:bg-muted'
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
      <div className="break-words px-2 pt-2 pb-1 text-xs text-muted-foreground">
        {displayName}
      </div>
      <div className="p-1">
        <button
          type="button"
          onClick={() => {
            onRename()
            onClose()
          }}
          className={itemClass()}
        >
          <Pencil className="h-4 w-4" />
          <span>Rename {label}</span>
        </button>
        <button
          type="button"
          onClick={() => {
            onDelete()
            onClose()
          }}
          className={itemClass('danger')}
        >
          <Trash2 className="h-4 w-4" />
          <span>Delete {label}</span>
        </button>
      </div>
    </div>
  )
}
