/**
 * WorldCanvas - Main entry point for the 3D member visualization.
 *
 * Skill selection is now handled via the sidebar in skill selection mode.
 * When "Set Skills" is clicked on a member, it triggers the sidebar to
 * enter skill selection mode with checkboxes.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#component-hierarchy
// DOC: docs/SEAMS.md#3d-world-testing-seams

import { Suspense, useCallback, useState, useMemo } from 'react'
import { Canvas } from '@react-three/fiber'
import { Loader } from '@react-three/drei'
import type { Skill } from '@/types'
import type { CombineFormat } from '@/types/world'
import { useResolvedTheme } from '@/hooks/use-theme'
import { useSelectionStore } from '@/stores/selectionStore'
import { useSkillSelectionStore } from '@/stores/skillSelectionStore'
import { useCameraStore } from '@/stores/cameraStore'
import { useMemberData } from '@/hooks/useMemberData'
import { useWorldDefaults } from '@/hooks/useWorldDefaults'
import { MemberProvider } from './MemberProvider'
import { WorldScene, type MemberWithPosition } from './WorldScene'
import { WorldControls } from './WorldControls'
import { CombinePanel } from './CombinePanel'
import { MemberOverlay } from './MemberOverlay'
import { WorldEditorToolbar, ObjectPalette } from './editor'
import { MemberCustomizeModal } from '../member/MemberCustomizeModal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { RenderPipeline } from './rendering/RenderPipeline'
import { EnvironmentSetup } from './rendering/EnvironmentSetup'
import { MaterialProvider } from './materials/MaterialProvider'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { WorldErrorProvider } from './WorldErrorContext'

// Default camera values (moved outside component to satisfy exhaustive-deps)
const DEFAULT_CAMERA_POSITION: [number, number, number] = [0, 5, 10]
const DEFAULT_CAMERA_TARGET: [number, number, number] = [0, 0, 0]

/**
 * Generate a deterministic position for a member based on its ID.
 * Uses a simple hash to ensure the same member always gets the same position.
 */
function generateMemberPosition(memberId: string, index: number, total: number): [number, number, number] {
  // Simple hash from member ID for deterministic randomness
  let hash = 0
  for (let i = 0; i < memberId.length; i++) {
    hash = ((hash << 5) - hash) + memberId.charCodeAt(i)
    hash = hash & hash // Convert to 32-bit integer
  }

  // If only one member, place at origin
  if (total === 1) {
    return [0, 0, 0]
  }

  // Distribute members in a circular pattern with some randomness
  const radius = 4 + (total > 5 ? 2 : 0) // Larger radius for more members
  const angleOffset = (hash % 1000) / 1000 * Math.PI * 2
  const baseAngle = (index / total) * Math.PI * 2 + angleOffset

  // Add some radial variation based on hash
  const radialVariation = ((hash % 100) / 100 - 0.5) * 1.5

  const x = Math.cos(baseAngle) * (radius + radialVariation)
  const z = Math.sin(baseAngle) * (radius + radialVariation)

  return [x, 0, z]
}

interface WorldCanvasProps {
  skills: Skill[]
  onSelectSkill?: (skillId: string) => void
  onCombineSkills?: (combined: string, format: CombineFormat) => void
  memberType?: string
  className?: string
}

export function WorldCanvas({
  skills,
  onSelectSkill: _onSelectSkill,
  onCombineSkills,
  memberType = 'geometric',
  className,
}: WorldCanvasProps) {
  // Note: onSelectSkill is kept for API compatibility but not used since
  // skill selection is now handled via the sidebar
  void _onSelectSkill
  // Theme for 3D colors
  const resolvedTheme = useResolvedTheme()
  const isDarkMode = resolvedTheme === 'dark'

  // Seed world with default furniture/decorations on first load
  useWorldDefaults()

  // Cursor tracking for member
  const [cursorPosition, setCursorPosition] = useState<{ x: number; y: number } | null>(null)

  // Selection state from centralized Zustand store
  const selectedSkillIds = useSelectionStore((state) => state.selectedSkillIds)
  const setSelectedSkillIds = useSelectionStore((state) => state.setSelectedSkillIds)

  // Skill selection store
  const enterSkillSelectionMode = useSkillSelectionStore((state) => state.enterSkillSelectionMode)

  // Camera store for member zoom
  const cameraMode = useCameraStore((state) => state.mode)
  const focusedMemberId = useCameraStore((state) => state.focusedMemberId)
  const exitZoom = useCameraStore((state) => state.exitZoom)
  const zoomToMember = useCameraStore((state) => state.zoomToMember)
  const cycleCameraMode = useCameraStore((state) => state.cycleCameraMode)

  // Member data
  const { members, updateMember, deleteMember, createMember, isUpdating, isDeleting } = useMemberData()
  const focusedMember = members.find((a) => a.id === focusedMemberId) ?? null

  // Generate positions for all members (memoized for stability)
  const membersWithPositions = useMemo<MemberWithPosition[]>(() => {
    return members.map((member, index) => ({
      member,
      position: generateMemberPosition(member.id, index, members.length),
    }))
  }, [members])

  // Get position of focused member (for camera targeting)
  const focusedMemberPosition = useMemo(() => {
    const found = membersWithPositions.find((a) => a.member.id === focusedMemberId)
    return found?.position ?? null
  }, [membersWithPositions, focusedMemberId])

  // Handle member click in 3D scene - opens the member menu
  const handleMemberClick = useCallback((memberId: string, position: [number, number, number]) => {
    zoomToMember(memberId, position)
  }, [zoomToMember])

  // Handle camera mode cycling
  const handleCycleCameraMode = useCallback(() => {
    // Get the first member for zoom target (or focused if available)
    const targetMember = focusedMemberId
      ? membersWithPositions.find((a) => a.member.id === focusedMemberId)
      : membersWithPositions[0]

    cycleCameraMode(targetMember?.member.id, targetMember?.position)
  }, [cycleCameraMode, focusedMemberId, membersWithPositions])

  // Modal state
  const [isCustomizeModalOpen, setIsCustomizeModalOpen] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)

  // Camera state for 3D scene - use focused member position when zoomed
  const cameraState = useMemo(() => {
    if (cameraMode === 'zoomed-member' && focusedMemberPosition) {
      return {
        position: [
          focusedMemberPosition[0],
          focusedMemberPosition[1] + 2,
          focusedMemberPosition[2] + 5,
        ] as [number, number, number],
        target: focusedMemberPosition,
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
  }, [cameraMode, focusedMemberPosition])

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
      onCombineSkills?.(combined, format)
    },
    [onCombineSkills]
  )

  // Get full skill objects for selected IDs
  const selectedSkillObjects = skills.filter((p) => selectedSkillIds.includes(p.id))

  // Member overlay handlers
  const handleCloseOverlay = useCallback(() => {
    exitZoom()
  }, [exitZoom])

  const handleCustomize = useCallback(() => {
    setIsCustomizeModalOpen(true)
  }, [])

  const handleSetSkills = useCallback(() => {
    // Enter skill selection mode via the store
    // This will trigger the sidebar to show checkboxes
    if (focusedMember) {
      enterSkillSelectionMode(
        focusedMember,
        focusedMember.skills,
        async (skillIds) => {
          await updateMember(focusedMember.id, { skills: skillIds })
        }
      )
      // Close the member overlay so the user can see the sidebar
      exitZoom()
    }
  }, [focusedMember, enterSkillSelectionMode, updateMember, exitZoom])

  const handleDuplicate = useCallback(async () => {
    if (!focusedMember) return
    await createMember({
      name: `${focusedMember.name} (Copy)`,
      bodyColor: focusedMember.bodyColor,
      headColor: focusedMember.headColor,
      accentColor: focusedMember.accentColor,
      skills: [...focusedMember.skills],
    })
    exitZoom()
  }, [focusedMember, createMember, exitZoom])

  const handleDeleteClick = useCallback(() => {
    setIsDeleteDialogOpen(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (focusedMemberId) {
      await deleteMember(focusedMemberId)
      exitZoom()
    }
    setIsDeleteDialogOpen(false)
  }, [focusedMemberId, deleteMember, exitZoom])

  return (
    <div
      className={`relative w-full h-full bg-background ${className || ''}`}
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
            dpr={[1, 2]}
          >
            <color attach="background" args={[isDarkMode ? '#0f172a' : '#f8fafc']} />
            <RenderPipeline>
              <EnvironmentSetup />
              <MaterialProvider>
                <MemberProvider member={memberType}>
                  <Suspense fallback={null}>
                    <WorldScene
                      cameraState={cameraState}
                      selectedNodeIds={selectedSkillIds}
                      cursorPosition={cursorPosition}
                      membersWithPositions={membersWithPositions}
                      onMemberClick={handleMemberClick}
                      isDarkMode={isDarkMode}
                    />
                  </Suspense>
                </MemberProvider>
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
        cameraMode={cameraMode}
        onCycleCameraMode={handleCycleCameraMode}
        nodeCount={skills.length}
        selectionCount={selectedSkillIds.length}
        memberCount={members.length}
      />

      {/* World Editor */}
      <WorldEditorToolbar className="absolute top-4 left-1/2 -translate-x-1/2" />
      <ObjectPalette className="absolute top-16 left-4" />

      {/* Combine panel */}
      <CombinePanel
        selectedSkills={selectedSkillObjects}
        onClear={() => setSelectedSkillIds([])}
        onCombine={handleCombine}
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

      {/* Member overlay - appears when zoomed to member */}
      <MemberOverlay
        member={focusedMember}
        isVisible={cameraMode === 'zoomed-member'}
        onClose={handleCloseOverlay}
        onCustomize={handleCustomize}
        onSetSkills={handleSetSkills}
        onDuplicate={() => void handleDuplicate()}
        onDelete={handleDeleteClick}
      />

      {/* Member customize modal */}
      <MemberCustomizeModal
        isOpen={isCustomizeModalOpen}
        onClose={() => setIsCustomizeModalOpen(false)}
        member={focusedMember}
        onSave={async (updates) => {
          if (focusedMemberId) {
            await updateMember(focusedMemberId, updates)
          }
        }}
        isLoading={isUpdating}
      />

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Member"
        message={`Are you sure you want to delete "${focusedMember?.name ?? 'this member'}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={isDeleting}
      />
    </div>
  )
}
