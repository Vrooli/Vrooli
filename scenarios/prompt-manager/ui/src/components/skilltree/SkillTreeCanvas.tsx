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
import { AvatarProvider } from './AvatarProvider'
import { SkillTreeScene } from './SkillTreeScene'
import { SkillTreeControls } from './SkillTreeControls'
import { CombinePanel } from './CombinePanel'

interface SkillTreeCanvasProps {
  prompts: Prompt[]
  onSelectPrompt?: (promptId: string) => void
  onCombinePrompts?: (combined: string, format: CombineFormat) => void
  avatarType?: string
  className?: string
}

export function SkillTreeCanvas({
  prompts,
  onSelectPrompt,
  onCombinePrompts,
  avatarType = 'geometric',
  className,
}: SkillTreeCanvasProps) {
  // Cursor tracking for avatar
  const [cursorPosition, setCursorPosition] = useState<{ x: number; y: number } | null>(null)

  // Selection state from centralized Zustand store
  const selectedPromptIds = useSelectionStore((state) => state.selectedPromptIds)
  const setSelectedPromptIds = useSelectionStore((state) => state.setSelectedPromptIds)
  const setSelectedPromptId = useSelectionStore((state) => state.setSelectedPromptId)

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
    </div>
  )
}
