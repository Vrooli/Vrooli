/**
 * WorldCanvas - Main entry point for the 3D agent visualization.
 *
 * Skill selection is handled via the sidebar for skill operations.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#component-hierarchy
// DOC: docs/internal/SEAMS.md#3d-world-testing-seams

import { Suspense, useCallback, useState, useMemo, useEffect, useRef } from 'react'
import { Canvas } from '@react-three/fiber'
import { Loader } from '@react-three/drei'
import type { Skill } from '@/types'
import type { DisplayFormat } from '@/types/world'
import type { FurnitureInstance } from '@/types/furniture'
import type { DecorationInstance } from '@/types/decoration'
import { useResolvedTheme } from '@/hooks/use-theme'
import { useSelectionStore } from '@/stores/selectionStore'
import { useCameraStore, type CameraMode } from '@/stores/cameraStore'
import { useFurnitureStore, useFurnitureList, useSeatedAgents } from '@/stores/furnitureStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useAgentData } from '@/hooks/useAgentData'
import { useWorldDefaults } from '@/hooks/useWorldDefaults'
import { AgentProvider } from './AgentProvider'
import { WorldScene, type AgentWithPosition } from './WorldScene'
import { WorldControls } from './WorldControls'
import { DisplayPanel } from './DisplayPanel'
import { AgentOverlay } from './AgentOverlay'
import { WorldEditorToolbar, ObjectPalette } from './editor'
import { FurnitureContextMenu } from './furniture'
import { SeatEditorOverlay } from './furniture/SeatEditorOverlay'
import { DecorationContextMenu } from './decorations'
import { AgentCustomizeModal } from '../agent/AgentCustomizeModal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { RenderPipeline } from './rendering/RenderPipeline'
import { EnvironmentSetup } from './rendering/EnvironmentSetup'
import { MaterialProvider } from './materials/MaterialProvider'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { WorldErrorProvider } from './WorldErrorProvider'
import { cursorRef } from './cursorRef'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { selectors } from '@/constants/selectors'

// Default camera values (moved outside component to satisfy exhaustive-deps)
const DEFAULT_CAMERA_POSITION: [number, number, number] = [0, 5, 10]
const DEFAULT_CAMERA_TARGET: [number, number, number] = [0, 0, 0]

/** Y position for agents standing on ground (inner offset group in SlimeAgent handles body radius) */
const AGENT_GROUND_Y = 0

/**
 * Generate a deterministic position for an agent based on its ID.
 * Uses a simple hash to ensure the same agent always gets the same position.
 */
function generateAgentPosition(agentId: string, index: number, total: number): [number, number, number] {
  // Simple hash from agent ID for deterministic randomness
  let hash = 0
  for (let i = 0; i < agentId.length; i++) {
    hash = ((hash << 5) - hash) + agentId.charCodeAt(i)
    hash = hash & hash // Convert to 32-bit integer
  }

  // If only one agent, place at origin
  if (total === 1) {
    return [0, AGENT_GROUND_Y, 0]
  }

  // Distribute agents in a circular pattern with some randomness
  const radius = 4 + (total > 5 ? 2 : 0) // Larger radius for more agents
  const angleOffset = (hash % 1000) / 1000 * Math.PI * 2
  const baseAngle = (index / total) * Math.PI * 2 + angleOffset

  // Add some radial variation based on hash
  const radialVariation = ((hash % 100) / 100 - 0.5) * 1.5

  const x = Math.cos(baseAngle) * (radius + radialVariation)
  const z = Math.sin(baseAngle) * (radius + radialVariation)

  return [x, AGENT_GROUND_Y, z]
}

interface WorldCanvasProps {
  skills: Skill[]
  onSelectSkill?: (skillId: string) => void
  onDisplaySkills?: (combined: string, format: DisplayFormat) => void
  agentType?: string
  className?: string
}

export function WorldCanvas({
  skills,
  onSelectSkill: _onSelectSkill,
  onDisplaySkills,
  agentType = 'geometric',
  className,
}: WorldCanvasProps) {
  // Note: onSelectSkill is kept for API compatibility but not used since
  // skill selection is now handled via the sidebar
  void _onSelectSkill
  // Theme for 3D colors
  const resolvedTheme = useResolvedTheme()
  const isDarkMode = resolvedTheme === 'dark'

  // Perf: Read DPR from graphics tier — hardcoded [1,2] renders 4x fragments on retina at low tier
  const dpr = useGraphicsStore((state) => state.config.dpr)

  // Seed world with default furniture/decorations on first load
  useWorldDefaults()

  // Selection state from centralized Zustand store
  const selectedSkillIds = useSelectionStore((state) => state.selectedSkillIds)
  const setSelectedSkillIds = useSelectionStore((state) => state.setSelectedSkillIds)

  // Camera store for agent zoom
  const cameraMode = useCameraStore((state) => state.mode)
  const focusedAgentId = useCameraStore((state) => state.focusedAgentId)
  const exitZoom = useCameraStore((state) => state.exitZoom)
  const zoomToAgent = useCameraStore((state) => state.zoomToAgent)

  // Agent data
  const { agents, updateAgent, deleteAgent, createAgent, isUpdating, isDeleting } = useAgentData()
  const focusedAgent = agents.find((a) => a.id === focusedAgentId) ?? null

  // Furniture and seating state (scene-aware)
  const sceneType = useEnvironmentStore((s) => s.current.type)
  const furnitureList = useFurnitureList()
  const seatedAgents = useSeatedAgents()
  const seatAgent = useFurnitureStore((state) => state.seatAgent)
  const unseatAgent = useFurnitureStore((state) => state.unseatAgent)
  const getAgentSeatPosition = useFurnitureStore((state) => state.getAgentSeatPosition)

  // Furniture context menu state
  const [selectedFurniture, setSelectedFurniture] = useState<FurnitureInstance | null>(null)

  // Decoration context menu state
  const [selectedDecoration, setSelectedDecoration] = useState<DecorationInstance | null>(null)

  // Convert seatedAgents object to Map for FurnitureContextMenu
  const seatedAgentsMap = useMemo(() => {
    return new Map(Object.entries(seatedAgents))
  }, [seatedAgents])

  // Track agent positions overridden by drag (edit mode)
  const [agentPositionOverrides, setAgentPositionOverrides] = useState<
    Record<string, [number, number, number]>
  >({})

  // Clear agent drag overrides when scene changes
  const prevSceneType = useRef(sceneType)
  useEffect(() => {
    if (sceneType !== prevSceneType.current) {
      setAgentPositionOverrides({})
      prevSceneType.current = sceneType
    }
  }, [sceneType])

  const handleAgentPositionChange = useCallback(
    (agentId: string, newPosition: [number, number, number]) => {
      setAgentPositionOverrides((prev) => ({ ...prev, [agentId]: newPosition }))
    },
    []
  )

  // Generate positions for all agents (memoized for stability)
  // Override positions for seated agents or drag-repositioned agents
  const agentsWithPositions = useMemo<AgentWithPosition[]>(() => {
    return agents.map((agent, index) => {
      // Check if agent has been repositioned by drag
      const dragOverride = agentPositionOverrides[agent.id]
      if (dragOverride) {
        return {
          agent,
          position: dragOverride,
          isSeated: false,
          seatRotation: 0,
        }
      }
      // Check if agent is seated
      const seatPosition = getAgentSeatPosition(agent.id)
      if (seatPosition) {
        return {
          agent,
          position: seatPosition.position,
          isSeated: true,
          seatRotation: seatPosition.rotation,
        }
      }
      return {
        agent,
        position: generateAgentPosition(agent.id, index, agents.length),
        isSeated: false,
        seatRotation: 0,
      }
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- furnitureList and seatedAgents are signal deps that force recalculation when furniture/seating state changes (getAgentSeatPosition is referentially stable)
  }, [agents, getAgentSeatPosition, seatedAgents, furnitureList, agentPositionOverrides])

  // Get position of focused agent (for camera targeting)
  const focusedAgentPosition = useMemo(() => {
    const found = agentsWithPositions.find((a) => a.agent.id === focusedAgentId)
    return found?.position ?? null
  }, [agentsWithPositions, focusedAgentId])

  // Handle agent click in 3D scene - opens the agent menu
  const handleAgentClick = useCallback((agentId: string, position: [number, number, number]) => {
    setSelectedFurniture(null) // Close furniture menu when clicking an agent
    setSelectedDecoration(null) // Close decoration menu too
    zoomToAgent(agentId, position)
  }, [zoomToAgent])

  // Handle furniture click - opens furniture context menu
  const handleFurnitureClick = useCallback((furniture: FurnitureInstance) => {
    setSelectedDecoration(null) // Close decoration menu when clicking furniture
    setSelectedFurniture(furniture)
  }, [])

  // Handle closing furniture context menu
  const handleCloseFurnitureMenu = useCallback(() => {
    setSelectedFurniture(null)
  }, [])

  // Handle decoration click - opens decoration context menu
  const handleDecorationClick = useCallback((decoration: DecorationInstance) => {
    setSelectedFurniture(null) // Close furniture menu when clicking decoration
    setSelectedDecoration(decoration)
  }, [])

  // Handle closing decoration context menu
  const handleCloseDecorationMenu = useCallback(() => {
    setSelectedDecoration(null)
  }, [])

  // Handle seating an agent
  const handleSitAgent = useCallback((agentId: string, furnitureId: string, seatIndex: number) => {
    seatAgent(agentId, furnitureId, seatIndex)
  }, [seatAgent])

  // Handle unseating an agent
  const handleUnsitAgent = useCallback((agentId: string) => {
    unseatAgent(agentId)
  }, [unseatAgent])

  // Handle camera mode change from settings popup
  const handleCameraModeChange = useCallback(
    (mode: CameraMode, agentId?: string, position?: [number, number, number]) => {
      if (mode === 'zoomed-agent') {
        // Find an agent to zoom to
        const targetAgent = agentId
          ? agentsWithPositions.find((a) => a.agent.id === agentId)
          : focusedAgentId
            ? agentsWithPositions.find((a) => a.agent.id === focusedAgentId)
            : agentsWithPositions[0]

        if (targetAgent) {
          zoomToAgent(targetAgent.agent.id, position ?? targetAgent.position)
        }
      }
      // Note: freeform and top-down are handled directly by the settings popup
    },
    [zoomToAgent, focusedAgentId, agentsWithPositions]
  )

  // Modal state
  const [isCustomizeModalOpen, setIsCustomizeModalOpen] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)

  // Camera state for 3D scene - use focused agent position when zoomed
  const cameraState = useMemo(() => {
    if (cameraMode === 'zoomed-agent' && focusedAgentPosition) {
      return {
        position: [
          focusedAgentPosition[0],
          focusedAgentPosition[1] + 2,
          focusedAgentPosition[2] + 5,
        ] as [number, number, number],
        target: focusedAgentPosition,
        zoom: 2,
      }
    }
    if (cameraMode === 'top-down') {
      return {
        position: [0, 20, 0.1] as [number, number, number],
        target: [0, 0, 0] as [number, number, number],
        zoom: 1,
      }
    }
    // Default freeform view
    return {
      position: DEFAULT_CAMERA_POSITION,
      target: DEFAULT_CAMERA_TARGET,
      zoom: 1,
    }
  }, [cameraMode, focusedAgentPosition])

  // Handle mouse move for cursor tracking
  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * 2 - 1
    const y = -((e.clientY - rect.top) / rect.height) * 2 + 1
    cursorRef.current = { x: x * 5, y: y * 3 }
  }, [])

  const handleMouseLeave = useCallback(() => {
    cursorRef.current = null
  }, [])

  // Handle display
  const handleDisplay = useCallback(
    (combined: string, format: DisplayFormat) => {
      onDisplaySkills?.(combined, format)
    },
    [onDisplaySkills]
  )

  // Get full skill objects for selected IDs
  const selectedSkillObjects = skills.filter((p) => selectedSkillIds.includes(p.id))

  // Agent overlay handlers
  const handleCloseOverlay = useCallback(() => {
    exitZoom()
  }, [exitZoom])

  const handleCustomize = useCallback(() => {
    setIsCustomizeModalOpen(true)
  }, [])

  const handleDuplicate = useCallback(async () => {
    if (!focusedAgent) return
    await createAgent({
      displayName: `${focusedAgent.displayName} (Copy)`,
      appearance: focusedAgent.appearance ? { ...focusedAgent.appearance } : undefined,
      fileOrder: [...focusedAgent.fileOrder],
    })
    exitZoom()
  }, [focusedAgent, createAgent, exitZoom])

  const handleDeleteClick = useCallback(() => {
    setIsDeleteDialogOpen(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (focusedAgentId) {
      await deleteAgent(focusedAgentId)
      exitZoom()
    }
    setIsDeleteDialogOpen(false)
  }, [focusedAgentId, deleteAgent, exitZoom])

  return (
    <div
      className={`relative w-full h-full bg-background ${className || ''}`}
      data-testid={selectors.world.canvas}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
    >
      {/* 3D Canvas */}
      <WorldErrorProvider>
        <WorldErrorBoundary componentName="WorldCanvas" className="absolute inset-0">
          <Canvas
            shadows
            camera={{
              position: cameraState.position,
              fov: 50,
              near: 0.1,
              far: 100,
            }}
            gl={{ antialias: true, alpha: false }}
            dpr={dpr}
          >
            <color attach="background" args={[isDarkMode ? '#0f172a' : '#f8fafc']} />
            <RenderPipeline>
              <EnvironmentSetup />
              <MaterialProvider>
                <AgentProvider agent={agentType}>
                  <Suspense fallback={null}>
                    <WorldScene
                      cameraState={cameraState}
                      selectedNodeIds={selectedSkillIds}
                      agentsWithPositions={agentsWithPositions}
                      onAgentClick={handleAgentClick}
                      onFurnitureClick={handleFurnitureClick}
                      onDecorationClick={handleDecorationClick}
                      onAgentPositionChange={handleAgentPositionChange}
                      isDarkMode={isDarkMode}
                    />
                  </Suspense>
                </AgentProvider>
              </MaterialProvider>
            </RenderPipeline>
          </Canvas>
        </WorldErrorBoundary>
      </WorldErrorProvider>

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
      <WorldControls
        nodeCount={skills.length}
        selectionCount={selectedSkillIds.length}
        agentCount={agents.length}
        onCameraModeChange={handleCameraModeChange}
      />

      {/* World Editor */}
      <WorldEditorToolbar className="absolute top-4 left-1/2 -translate-x-1/2" />
      <ObjectPalette className="absolute top-16 left-4" />

      {/* Furniture context menu */}
      {selectedFurniture && (
        <div className="absolute top-16 right-4">
          <FurnitureContextMenu
            furniture={selectedFurniture}
            agents={agents}
            onClose={handleCloseFurnitureMenu}
            onSitAgent={handleSitAgent}
            onUnsitAgent={handleUnsitAgent}
            seatedAgents={seatedAgentsMap}
          />
        </div>
      )}

      {/* Seat editor overlay */}
      <div className="absolute top-16 left-4 z-10">
        <SeatEditorOverlay />
      </div>

      {/* Decoration context menu */}
      {selectedDecoration && (
        <div className="absolute top-16 right-4">
          <DecorationContextMenu
            decoration={selectedDecoration}
            onClose={handleCloseDecorationMenu}
          />
        </div>
      )}

      {/* Display panel */}
      <DisplayPanel
        selectedSkills={selectedSkillObjects}
        onClear={() => setSelectedSkillIds([])}
        onDisplay={handleDisplay}
      />

      {/* Empty state */}
      {skills.length === 0 && (
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
            <h3 className="text-lg font-medium text-muted-foreground mb-2">No Skills Yet</h3>
            <p className="text-sm text-muted-foreground/70 max-w-xs">
              Create your first skill to see it appear in the skill tree.
            </p>
          </div>
        </div>
      )}

      {/* Agent overlay - appears when zoomed to agent */}
      <AgentOverlay
        agent={focusedAgent}
        isVisible={cameraMode === 'zoomed-agent'}
        onClose={handleCloseOverlay}
        onCustomize={handleCustomize}
        onDuplicate={() => void handleDuplicate()}
        onDelete={handleDeleteClick}
      />

      {/* Agent customize modal */}
      <AgentCustomizeModal
        isOpen={isCustomizeModalOpen}
        onClose={() => setIsCustomizeModalOpen(false)}
        agent={focusedAgent}
        onSave={async (updates) => {
          if (focusedAgentId) {
            await updateAgent(focusedAgentId, updates)
          }
        }}
        isLoading={isUpdating}
      />

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Agent"
        message={`Are you sure you want to delete "${focusedAgent?.displayName ?? 'this agent'}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={isDeleting}
      />
    </div>
  )
}
