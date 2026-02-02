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
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  X,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { toast } from '@/hooks/use-toast'
import type { TeamSharedFileEntry } from '@/types/team'
import * as teamService from '@/services/teamService'
import { SkillContentEditor } from '../SkillContentEditor'
import { FilePathMenu } from '../FilePathMenu'

interface TeamFilesTabProps {
  teamId: string
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

export function TeamFilesTab({ teamId, className }: TeamFilesTabProps) {
  const [files, setFiles] = useState<TeamSharedFileEntry[]>([])
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
  } | null>(null)
  const skipFileLoadRef = useRef<string | null>(null)

  const tree = useMemo(() => buildFileTree(files), [files])
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
  const isFileDirty = isFileEditorActive && fileContent !== originalContent

  const refreshFiles = useCallback(async () => {
    try {
      const entries = await teamService.listTeamSharedFiles(teamId)
      setFiles(entries)
    } catch (error) {
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

  useEffect(() => {
    setExpandedPaths(new Set())
  }, [teamId])

  useEffect(() => {
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
  }, [files, selectedPath])

  useEffect(() => {
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
  }, [teamId, selectedPath, isDirectorySelected])

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
      <div className={cn('h-full flex min-h-0', className)}>
        <div className="w-64 border-r border-border flex flex-col min-h-0">
          <div className="flex items-center justify-between px-3 py-2 border-b border-border">
            <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Shared Files</span>
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
                  !selectedPath ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
                )}
                disabled={!selectedPath}
                title="Rename file"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => void handleDelete()}
                className={cn(
                  'p-1 rounded text-muted-foreground hover:text-foreground',
                  !selectedPath ? 'opacity-50 cursor-not-allowed' : 'hover:bg-muted'
                )}
                disabled={!selectedPath}
                title="Delete file"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto px-2 py-2">
            {files.length === 0 ? (
              <div className="text-xs text-muted-foreground px-2 py-4">
                No shared files yet. Create a file to get started.
              </div>
            ) : (
              renderNode(tree)
            )}
          </div>
        </div>

        <div className="flex-1 min-w-0 min-h-0 flex flex-col">
          {!selectedPath && (
            <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
              Select a file to view or edit.
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
                  headerRight={filePathMenu ?? undefined}
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
      <div className="px-2 pt-2 pb-1 text-xs text-muted-foreground truncate">
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
