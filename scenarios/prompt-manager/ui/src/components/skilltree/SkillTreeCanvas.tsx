/**
 * SkillTreeCanvas - Main entry point for the 3D avatar visualization.
 *
 * Skill selection is now handled via the sidebar in skill selection mode.
 * When "Set Skills" is clicked on an avatar, it triggers the sidebar to
 * enter skill selection mode with checkboxes.
 */

import { Suspense, useCallback, useState } from 'react'
import { Canvas } from '@react-three/fiber'
import { Loader } from '@react-three/drei'
import type { Prompt } from '@/types'
import type { CombineFormat } from '@/types/skilltree'
import { useResolvedTheme } from '@/hooks/use-theme'
import { useSelectionStore } from '@/stores/selectionStore'
import { useSkillSelectionStore } from '@/stores/skillSelectionStore'
import { useCameraStore } from '@/stores/cameraStore'
import { useAvatarData } from '@/hooks/useAvatarData'
import { AvatarProvider } from './AvatarProvider'
import { SkillTreeScene } from './SkillTreeScene'
import { SkillTreeControls } from './SkillTreeControls'
import { CombinePanel } from './CombinePanel'
import { AvatarOverlay } from './AvatarOverlay'
import { AvatarCustomizeModal } from '../avatar/AvatarCustomizeModal'
import { ConfirmDialog } from '../shared/ConfirmDialog'

// Default camera values (moved outside component to satisfy exhaustive-deps)
const DEFAULT_CAMERA_POSITION: [number, number, number] = [0, 5, 10]
const DEFAULT_CAMERA_TARGET: [number, number, number] = [0, 0, 0]

interface SkillTreeCanvasProps {
  prompts: Prompt[]
  onSelectPrompt?: (promptId: string) => void
  onCombinePrompts?: (combined: string, format: CombineFormat) => void
  avatarType?: string
  /** ID of the avatar to display in the scene (uses first avatar if not specified) */
  activeAvatarId?: string
  className?: string
}

export function SkillTreeCanvas({
  prompts,
  onSelectPrompt: _onSelectPrompt,
  onCombinePrompts,
  avatarType = 'geometric',
  activeAvatarId,
  className,
}: SkillTreeCanvasProps) {
  // Note: onSelectPrompt is kept for API compatibility but not used since
  // skill selection is now handled via the sidebar
  void _onSelectPrompt
  // Theme for 3D colors
  const resolvedTheme = useResolvedTheme()
  const isDarkMode = resolvedTheme === 'dark'

  // Cursor tracking for avatar
  const [cursorPosition, setCursorPosition] = useState<{ x: number; y: number } | null>(null)

  // Selection state from centralized Zustand store
  const selectedPromptIds = useSelectionStore((state) => state.selectedPromptIds)
  const setSelectedPromptIds = useSelectionStore((state) => state.setSelectedPromptIds)

  // Skill selection store
  const enterSkillSelectionMode = useSkillSelectionStore((state) => state.enterSkillSelectionMode)

  // Camera store for avatar zoom
  const cameraMode = useCameraStore((state) => state.mode)
  const focusedAvatarId = useCameraStore((state) => state.focusedAvatarId)
  const exitZoom = useCameraStore((state) => state.exitZoom)
  const zoomToAvatar = useCameraStore((state) => state.zoomToAvatar)

  // Avatar data
  const { avatars, updateAvatar, deleteAvatar, createAvatar, isUpdating, isDeleting } = useAvatarData()
  const focusedAvatar = avatars.find((a) => a.id === focusedAvatarId) ?? null

  // Get the active avatar for display (default to first if not specified)
  const activeAvatar = avatars.find((a) => a.id === activeAvatarId) ?? avatars[0] ?? null
  const activeAvatarColors = activeAvatar
    ? { body: activeAvatar.bodyColor, head: activeAvatar.headColor, accent: activeAvatar.accentColor }
    : undefined

  // Handle avatar click in 3D scene
  const handleAvatarClick = useCallback(() => {
    if (activeAvatar) {
      zoomToAvatar(activeAvatar.id, [0, 0, 0]) // Avatar is at origin
    }
  }, [activeAvatar, zoomToAvatar])

  // Modal state
  const [isCustomizeModalOpen, setIsCustomizeModalOpen] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)

  // Camera state for 3D scene
  const [cameraState, setCameraState] = useState({
    position: DEFAULT_CAMERA_POSITION,
    target: DEFAULT_CAMERA_TARGET,
    zoom: 1,
  })

  // Camera controls
  const resetCamera = useCallback(() => {
    setCameraState({
      position: DEFAULT_CAMERA_POSITION,
      target: DEFAULT_CAMERA_TARGET,
      zoom: 1,
    })
  }, [])

  const zoomIn = useCallback(() => {
    setCameraState((prev) => ({
      ...prev,
      zoom: Math.min(prev.zoom * 1.2, 3),
    }))
  }, [])

  const zoomOut = useCallback(() => {
    setCameraState((prev) => ({
      ...prev,
      zoom: Math.max(prev.zoom / 1.2, 0.5),
    }))
  }, [])

  // Handle mouse move for cursor tracking
  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * 2 - 1
    const y = -((e.clientY - rect.top) / rect.height) * 2 + 1
    setCursorPosition({ x: x * 5, y: y * 3 })
  }, [])

  const handleMouseLeave = useCallback(() => {
    setCursorPosition(null)
  }, [])

  // Handle combine
  const handleCombine = useCallback(
    (combined: string, format: CombineFormat) => {
      onCombinePrompts?.(combined, format)
    },
    [onCombinePrompts]
  )

  // Get full prompt objects for selected IDs
  const selectedPromptObjects = prompts.filter((p) => selectedPromptIds.includes(p.id))

  // Avatar overlay handlers
  const handleCloseOverlay = useCallback(() => {
    exitZoom()
  }, [exitZoom])

  const handleCustomize = useCallback(() => {
    setIsCustomizeModalOpen(true)
  }, [])

  const handleSetSkills = useCallback(() => {
    // Enter skill selection mode via the store
    // This will trigger the sidebar to show checkboxes
    if (focusedAvatar) {
      enterSkillSelectionMode(
        focusedAvatar,
        focusedAvatar.skills,
        async (skillIds) => {
          await updateAvatar(focusedAvatar.id, { skills: skillIds })
        }
      )
      // Close the avatar overlay so the user can see the sidebar
      exitZoom()
    }
  }, [focusedAvatar, enterSkillSelectionMode, updateAvatar, exitZoom])

  const handleDuplicate = useCallback(async () => {
    if (!focusedAvatar) return
    await createAvatar({
      name: `${focusedAvatar.name} (Copy)`,
      bodyColor: focusedAvatar.bodyColor,
      headColor: focusedAvatar.headColor,
      accentColor: focusedAvatar.accentColor,
      skills: [...focusedAvatar.skills],
    })
    exitZoom()
  }, [focusedAvatar, createAvatar, exitZoom])

  const handleDeleteClick = useCallback(() => {
    setIsDeleteDialogOpen(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (focusedAvatarId) {
      await deleteAvatar(focusedAvatarId)
      exitZoom()
    }
    setIsDeleteDialogOpen(false)
  }, [focusedAvatarId, deleteAvatar, exitZoom])

  return (
    <div
      className={`relative w-full h-full bg-background ${className || ''}`}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
    >
      {/* 3D Canvas */}
      <Canvas
        shadows
        camera={{
          position: cameraState.position,
          fov: 50,
          near: 0.1,
          far: 100,
        }}
        gl={{ antialias: true, alpha: false }}
        dpr={[1, 2]}
      >
        <color attach="background" args={[isDarkMode ? '#0f172a' : '#f8fafc']} />
        <AvatarProvider avatar={avatarType}>
          <Suspense fallback={null}>
            <SkillTreeScene
              cameraState={cameraState}
              selectedNodeIds={selectedPromptIds}
              cursorPosition={cursorPosition}
              onAvatarClick={handleAvatarClick}
              avatarColors={activeAvatarColors}
              isDarkMode={isDarkMode}
            />
          </Suspense>
        </AvatarProvider>
      </Canvas>

      {/* Loading indicator */}
      <Loader
        containerStyles={{
          background: isDarkMode ? 'rgba(15, 23, 42, 0.9)' : 'rgba(248, 250, 252, 0.9)',
        }}
        barStyles={{
          background: '#6366f1',
        }}
        dataStyles={{
          color: isDarkMode ? '#e2e8f0' : '#1e293b',
          fontSize: '14px',
        }}
      />

      {/* UI Overlays */}
      <SkillTreeControls
        onZoomIn={zoomIn}
        onZoomOut={zoomOut}
        onReset={resetCamera}
        nodeCount={prompts.length}
        selectionCount={selectedPromptIds.length}
      />

      {/* Combine panel */}
      <CombinePanel
        selectedPrompts={selectedPromptObjects}
        onClear={() => setSelectedPromptIds([])}
        onCombine={handleCombine}
      />

      {/* Empty state */}
      {prompts.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="text-center">
            <div className="w-16 h-16 mx-auto mb-4 bg-muted rounded-2xl flex items-center justify-center">
              <svg
                className="w-8 h-8 text-muted-foreground"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-muted-foreground mb-2">No Prompts Yet</h3>
            <p className="text-sm text-muted-foreground/70 max-w-xs">
              Create your first prompt to see it appear in the skill tree.
            </p>
          </div>
        </div>
      )}

      {/* Avatar overlay - appears when zoomed to avatar */}
      <AvatarOverlay
        avatar={focusedAvatar}
        isVisible={cameraMode === 'zoomed-avatar'}
        onClose={handleCloseOverlay}
        onCustomize={handleCustomize}
        onSetSkills={handleSetSkills}
        onDuplicate={() => void handleDuplicate()}
        onDelete={handleDeleteClick}
      />

      {/* Avatar customize modal */}
      <AvatarCustomizeModal
        isOpen={isCustomizeModalOpen}
        onClose={() => setIsCustomizeModalOpen(false)}
        avatar={focusedAvatar}
        onSave={async (updates) => {
          if (focusedAvatarId) {
            await updateAvatar(focusedAvatarId, updates)
          }
        }}
        isLoading={isUpdating}
      />

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Avatar"
        message={`Are you sure you want to delete "${focusedAvatar?.name ?? 'this avatar'}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={isDeleting}
      />
    </div>
  )
}
