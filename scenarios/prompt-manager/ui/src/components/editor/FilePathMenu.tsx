/**
 * FilePathMenu - Dropdown menu for file path details and actions.
 *
 * Shows:
 * - Breadcrumb display of the folder/mode path
 * - Editable filename input
 * - Copy relative path button (relative to skills folder)
 * - Copy project path button (relative to Vrooli root - useful for coding agents)
 * - Copy full path button (absolute path)
 * - Storage location toggle (Core/Local)
 */

import { useState, useRef, useEffect, useLayoutEffect, useCallback, type ReactNode } from 'react'
import { ChevronRight, Copy, Check, FolderOpen, HardDrive } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FolderType } from '@/types'

interface FilePathMenuProps {
  /** The filename (e.g., "my-skill.md") */
  file: string
  /** Array of mode segments (e.g., ['mode1', 'mode2']) */
  modes?: string[]
  /** The folder type (local, core, drafts) */
  folder?: FolderType
  /** Called when filename is changed */
  onFileChange: (file: string) => void
  /** Called when folder is changed */
  onFolderChange?: (folder: FolderType) => void
  /** Absolute path to skill directory (from API) */
  skillDir?: string
  /** Absolute path to SKILL.md file (from API) */
  contentPath?: string
  /** Override label for the root breadcrumb segment */
  rootLabel?: string
  /** Override breadcrumb segments (defaults to modes) */
  pathSegments?: string[]
  /** Override file extension (defaults to derived from file) */
  fileExtension?: string
  /** Override relative path */
  relativePath?: string
  /** Override project path */
  projectPath?: string
  /** Override full/absolute path */
  fullPath?: string
  /** Disable filename editing */
  isEditable?: boolean
  /** Override the trigger icon */
  triggerIcon?: ReactNode
  /** Additional class names */
  className?: string
}

/**
 * Dropdown menu for file path details and actions.
 */
export function FilePathMenu({
  file,
  modes = [],
  folder,
  onFileChange,
  onFolderChange,
  skillDir,
  contentPath,
  rootLabel,
  pathSegments,
  fileExtension,
  relativePath,
  projectPath,
  fullPath,
  isEditable = true,
  triggerIcon,
  className,
}: FilePathMenuProps) {
  const [isOpen, setIsOpen] = useState(false)
  const extension = (() => {
    if (fileExtension) {
      return fileExtension.startsWith('.') ? fileExtension : `.${fileExtension}`
    }
    const base = file.split('/').pop() ?? file
    const lastDot = base.lastIndexOf('.')
    return lastDot > 0 ? base.slice(lastDot) : ''
  })()
  // Handle undefined/empty file prop defensively
  const safeFile = file || `untitled${extension}`
  // Extract filename without extension for editing
  const filenameWithoutExt = extension ? safeFile.slice(0, -extension.length) : safeFile
  const [editingName, setEditingName] = useState(filenameWithoutExt)
  const [copiedRelative, setCopiedRelative] = useState(false)
  const [copiedProject, setCopiedProject] = useState(false)
  const [copiedFull, setCopiedFull] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)

  // Position state for viewport-aware placement
  const [position, setPosition] = useState<{ top: number; left: number; right?: number }>({ top: 0, left: 0 })

  // Sync editing name with prop when menu opens
  useEffect(() => {
    if (isOpen) {
      setEditingName(extension ? safeFile.slice(0, -extension.length) : safeFile)
      // Focus input after a short delay
      if (isEditable) {
        setTimeout(() => inputRef.current?.focus(), 50)
      }
    }
  }, [isOpen, safeFile, extension, isEditable])

  // Calculate position when menu opens
  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return

    const trigger = triggerRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const menuWidth = Math.min(320, viewportWidth - 16) // Max width with padding
    const menuHeight = 320 // Approximate max height

    let top = trigger.bottom + 4
    let left = trigger.right - menuWidth
    let right: number | undefined = undefined

    // Ensure menu doesn't go off left edge
    if (left < 8) {
      left = 8
    }

    // Ensure menu doesn't go off right edge
    if (left + menuWidth > viewportWidth - 8) {
      left = viewportWidth - menuWidth - 8
    }

    // If menu would go below viewport, position above trigger
    if (top + menuHeight > viewportHeight - 8) {
      top = trigger.top - menuHeight - 4
      if (top < 8) {
        top = 8 // Keep some margin from top
      }
    }

    // On mobile, use right positioning for stability
    if (viewportWidth < 640) {
      right = 8
      left = 8 // Full width minus padding
    }

    setPosition({ top, left, right })
  }, [isOpen])

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        // Save filename if changed
        const newFilename = `${editingName.trim()}${extension}`
        if (isEditable && newFilename !== safeFile && editingName.trim()) {
          onFileChange(newFilename)
        }
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen, editingName, safeFile, extension, onFileChange, isEditable])

  // Close on Escape
  useEffect(() => {
    if (!isOpen) return

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setEditingName(extension ? safeFile.slice(0, -extension.length) : safeFile) // Reset to original
        setIsOpen(false)
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, safeFile, extension])

  // Build path segments for breadcrumb display only (modes are UI categories, not directory structure)
  const breadcrumbSegments = (pathSegments ?? modes).filter(Boolean)
  const rootSegment = rootLabel ?? folder ?? 'path'

  // Use server-provided paths when available (ground truth from API)
  // Fall back to constructed paths for backwards compatibility
  const resolvedFullPath = fullPath ?? (contentPath || (skillDir ? `${skillDir}/SKILL.md` : ''))

  // Compute project path: relative to project root (useful for coding agents)
  // Extract from absolute path by finding "scenarios/" prefix
  let resolvedProjectPath = projectPath ?? ''
  if (!resolvedProjectPath && contentPath) {
    const scenariosIndex = contentPath.indexOf('scenarios/')
    resolvedProjectPath = scenariosIndex !== -1 ? contentPath.slice(scenariosIndex) : contentPath
  } else if (!resolvedProjectPath && skillDir) {
    const scenariosIndex = skillDir.indexOf('scenarios/')
    resolvedProjectPath = scenariosIndex !== -1 ? `${skillDir.slice(scenariosIndex)}/SKILL.md` : `${skillDir}/SKILL.md`
  }

  // Relative path: just the store-relative path
  const resolvedRelativePath = relativePath ?? (skillDir
    ? (skillDir.split('/store/').pop() ?? '') + '/SKILL.md'
    : folder
      ? `skills/packs/${folder}/${safeFile.replace(extension, '')}/SKILL.md`
      : '')

  // Copy function - wrapped to avoid returning promise in onClick
  const handleCopyRelative = useCallback(() => {
    if (!resolvedRelativePath) return
    void navigator.clipboard.writeText(resolvedRelativePath).then(() => {
      setCopiedRelative(true)
      setTimeout(() => setCopiedRelative(false), 2000)
    }).catch((err: unknown) => {
      console.error('Failed to copy:', err)
    })
  }, [resolvedRelativePath])

  const handleCopyProject = useCallback(() => {
    if (!resolvedProjectPath) return
    void navigator.clipboard.writeText(resolvedProjectPath).then(() => {
      setCopiedProject(true)
      setTimeout(() => setCopiedProject(false), 2000)
    }).catch((err: unknown) => {
      console.error('Failed to copy:', err)
    })
  }, [resolvedProjectPath])

  const handleCopyFull = useCallback(() => {
    if (!resolvedFullPath) return
    void navigator.clipboard.writeText(resolvedFullPath).then(() => {
      setCopiedFull(true)
      setTimeout(() => setCopiedFull(false), 2000)
    }).catch((err: unknown) => {
      console.error('Failed to copy:', err)
    })
  }, [resolvedFullPath])

  const handleNameKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (isEditable && editingName.trim()) {
        onFileChange(`${editingName.trim()}${extension}`)
        setIsOpen(false)
      }
    }
  }

  // Get storage display info
  const getStorageInfo = (f: FolderType) => {
    switch (f) {
      case 'local':
        return { icon: HardDrive, label: 'Local', description: 'Personal skill, gitignored' }
      case 'core':
        return { icon: FolderOpen, label: 'Core', description: 'Shared skill, git-tracked' }
      case 'drafts':
        return { icon: HardDrive, label: 'Drafts', description: 'Draft skill' }
      default:
        return { icon: HardDrive, label: 'Local', description: 'Personal skill' }
    }
  }

  const storageInfo = folder ? getStorageInfo(folder) : { icon: FolderOpen, label: rootSegment, description: '' }
  const StorageIcon = storageInfo.icon

  return (
    <div ref={menuRef} className={cn('relative', className)}>
      {/* Trigger button - shows filename */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1 px-1.5 sm:px-2 py-0.5 rounded text-xs',
          'text-foreground hover:bg-muted transition-colors',
          'border border-transparent hover:border-border'
        )}
        title="Click to view path details"
        aria-label={`Open file path menu for ${safeFile}`}
      >
        {triggerIcon ?? <StorageIcon className="h-3 w-3 text-muted-foreground" />}
        <span className="hidden sm:inline font-medium">{safeFile}</span>
      </button>

      {/* Dropdown menu - fixed position for viewport awareness */}
      {isOpen && (
        <div
          ref={dropdownRef}
          style={{
            position: 'fixed',
            top: position.top,
            left: position.right !== undefined ? position.left : position.left,
            right: position.right,
            width: position.right !== undefined ? 'auto' : undefined,
            maxWidth: 'calc(100vw - 16px)',
          }}
          className={cn(
            'z-50 bg-card border border-border rounded-lg shadow-lg',
            'p-3 space-y-3'
          )}
        >
          {/* Storage location toggle */}
          {onFolderChange && folder && (
            <div className="space-y-1.5">
              <label className="text-xs text-muted-foreground font-medium block">
                Storage Location
              </label>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => onFolderChange('core')}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm transition-colors',
                    folder === 'core'
                      ? 'bg-emerald-600/20 text-emerald-300 border border-emerald-500/30'
                      : 'bg-muted text-muted-foreground hover:text-foreground border border-transparent hover:border-border'
                  )}
                >
                  <FolderOpen className="h-4 w-4" />
                  Core
                </button>
                <button
                  type="button"
                  onClick={() => onFolderChange('local')}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm transition-colors',
                    folder === 'local'
                      ? 'bg-primary/20 text-primary border border-primary/30'
                      : 'bg-muted text-muted-foreground hover:text-foreground border border-transparent hover:border-border'
                  )}
                >
                  <HardDrive className="h-4 w-4" />
                  Local
                </button>
              </div>
              <p className="text-[10px] text-muted-foreground">
                {storageInfo.description}
              </p>
            </div>
          )}

          {/* Path breadcrumb - without "Path" label */}
          <div className="flex items-center flex-wrap gap-1 text-sm">
            <span className="text-muted-foreground">{rootSegment}</span>
            {breadcrumbSegments.length > 0 && (
              <ChevronRight className="h-3 w-3 text-muted-foreground" />
            )}
            {breadcrumbSegments.map((segment, index) => (
              <span key={index} className="flex items-center">
                <span className="text-muted-foreground">{segment}</span>
                {index < breadcrumbSegments.length - 1 && (
                  <ChevronRight className="h-3 w-3 text-muted-foreground mx-0.5" />
                )}
              </span>
            ))}
          </div>

          {/* Filename input */}
          <div>
            <label className="text-xs text-muted-foreground font-medium mb-1 block">
              Filename
            </label>
            <div className="flex items-center gap-1">
              <input
                ref={inputRef}
                type="text"
                value={editingName}
                onChange={(e) => setEditingName(e.target.value)}
                onKeyDown={handleNameKeyDown}
                disabled={!isEditable}
                className={cn(
                  'flex-1 px-2 py-1 text-sm rounded',
                  'bg-muted border border-border',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary',
                  !isEditable && 'opacity-60 cursor-not-allowed'
                )}
                placeholder="File name"
              />
              {extension && <span className="text-sm text-muted-foreground">{extension}</span>}
            </div>
          </div>

          {/* File path display */}
          <div>
            <label className="text-xs text-muted-foreground font-medium mb-1 block">
              File Path
            </label>
            <p className="text-xs text-muted-foreground font-mono break-all select-all bg-muted/50 px-2 py-1.5 rounded">
              {resolvedProjectPath || resolvedFullPath || resolvedRelativePath}
            </p>
          </div>

          {/* Copy buttons - cleaner design */}
          <div className="flex flex-col gap-1.5">
            {resolvedRelativePath && (
              <button
                type="button"
                onClick={handleCopyRelative}
                className={cn(
                  'flex items-center justify-between gap-2 px-2 py-1.5 rounded text-sm',
                  'hover:bg-muted transition-colors w-full text-left'
                )}
              >
                <span className="text-foreground">Copy relative path</span>
                {copiedRelative ? (
                  <Check className="h-4 w-4 text-green-500 flex-shrink-0" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                )}
              </button>
            )}
            {resolvedProjectPath && (
              <button
                type="button"
                onClick={handleCopyProject}
                className={cn(
                  'flex items-center justify-between gap-2 px-2 py-1.5 rounded text-sm',
                  'hover:bg-muted transition-colors w-full text-left'
                )}
              >
                <span className="text-foreground">Copy project path</span>
                {copiedProject ? (
                  <Check className="h-4 w-4 text-green-500 flex-shrink-0" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                )}
              </button>
            )}
            {resolvedFullPath && (
              <button
                type="button"
                onClick={handleCopyFull}
                className={cn(
                  'flex items-center justify-between gap-2 px-2 py-1.5 rounded text-sm',
                  'hover:bg-muted transition-colors w-full text-left'
                )}
              >
                <span className="text-foreground">Copy full path</span>
                {copiedFull ? (
                  <Check className="h-4 w-4 text-green-500 flex-shrink-0" />
                ) : (
                  <Copy className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                )}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
