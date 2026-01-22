/**
 * SkillTreeCanvas - Main entry point for the 3D skill tree visualization.
 * Wraps the R3F Canvas with providers and UI overlays.
 */

import { Suspense, useCallback, useState } from 'react'
import { Canvas } from '@react-three/fiber'
import { Loader } from '@react-three/drei'
import type { Prompt } from '@/types'
import type { CombineFormat } from '@/types/skilltree'
import { useSkillTree3D } from '@/hooks/useSkillTree3D'
import { useSelectionStore } from '@/stores/selectionStore'
import { useCameraStore } from '@/stores/cameraStore'
import { useAvatarData } from '@/hooks/useAvatarData'
import { AvatarProvider } from './AvatarProvider'
import { SkillTreeScene } from './SkillTreeScene'
import { SkillTreeControls } from './SkillTreeControls'
import { CombinePanel } from './CombinePanel'
import { AvatarOverlay } from './AvatarOverlay'
import { AvatarCustomizeModal } from '../avatar/AvatarCustomizeModal'
import { ConfirmDialog } from '../shared/ConfirmDialog'

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
  onSelectPrompt,
  onCombinePrompts,
  avatarType = 'geometric',
  activeAvatarId,
  className,
}: SkillTreeCanvasProps) {
  // Cursor tracking for avatar
  const [cursorPosition, setCursorPosition] = useState<{ x: number; y: number } | null>(null)

  // Selection state from centralized Zustand store
  const selectedPromptIds = useSelectionStore((state) => state.selectedPromptIds)
  const setSelectedPromptIds = useSelectionStore((state) => state.setSelectedPromptIds)
  const setSelectedPromptId = useSelectionStore((state) => state.setSelectedPromptId)

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

  // Handle node click - updates both multi-selection and single selection
  const handleNodeSelection = useCallback(
    (promptId: string) => {
      setSelectedPromptId(promptId)
      onSelectPrompt?.(promptId)
    },
    [setSelectedPromptId, onSelectPrompt]
  )

  // Skill tree 3D state
  const {
    treeData,
    cameraState,
    hoveredNodeId,
    handleNodeClick,
    handleNodeHover,
    resetCamera,
    zoomIn,
    zoomOut,
    nodeCount,
  } = useSkillTree3D({
    prompts,
    selectedPromptIds,
    onSelectionChange: setSelectedPromptIds,
    onNodeClick: handleNodeSelection,
  })

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
    // TODO: Open skill assignment panel
    // For now, just assign currently selected prompts
    if (focusedAvatarId && selectedPromptIds.length > 0) {
      void updateAvatar(focusedAvatarId, {
        skills: [...new Set([...(focusedAvatar?.skills ?? []), ...selectedPromptIds])],
      })
    }
  }, [focusedAvatarId, focusedAvatar, selectedPromptIds, updateAvatar])

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
      className={`relative w-full h-full bg-slate-900 ${className || ''}`}
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
        <color attach="background" args={['#0f172a']} />
        <AvatarProvider avatar={avatarType}>
          <Suspense fallback={null}>
            <SkillTreeScene
              treeData={treeData}
              cameraState={cameraState}
              selectedNodeIds={selectedPromptIds}
              hoveredNodeId={hoveredNodeId}
              cursorPosition={cursorPosition}
              onNodeClick={handleNodeClick}
              onNodeHover={handleNodeHover}
              onAvatarClick={handleAvatarClick}
              avatarColors={activeAvatarColors}
            />
          </Suspense>
        </AvatarProvider>
      </Canvas>

      {/* Loading indicator */}
      <Loader
        containerStyles={{
          background: 'rgba(15, 23, 42, 0.9)',
        }}
        barStyles={{
          background: '#6366f1',
        }}
        dataStyles={{
          color: '#e2e8f0',
          fontSize: '14px',
        }}
      />

      {/* UI Overlays */}
      <SkillTreeControls
        onZoomIn={zoomIn}
        onZoomOut={zoomOut}
        onReset={resetCamera}
        nodeCount={nodeCount}
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
            <div className="w-16 h-16 mx-auto mb-4 bg-slate-800 rounded-2xl flex items-center justify-center">
              <svg
                className="w-8 h-8 text-slate-600"
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
            <h3 className="text-lg font-medium text-slate-400 mb-2">No Prompts Yet</h3>
            <p className="text-sm text-slate-500 max-w-xs">
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
