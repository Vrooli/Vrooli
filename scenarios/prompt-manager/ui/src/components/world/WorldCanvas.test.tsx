import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { WorldCanvas } from './WorldCanvas'
import { selectors } from '@/constants/selectors'

const {
  mockCreateAgent,
  mockDeleteAgent,
  mockExitZoom,
  mockSetAgentPositions,
  mockSetSceneSnapshot,
  mockSetSelectedSkillIds,
  mockSetSelectedTeamId,
  mockUnseatAgent,
  mockUpdateAgent,
  mockZoomToAgent,
} = vi.hoisted(() => ({
  mockCreateAgent: vi.fn(),
  mockDeleteAgent: vi.fn(),
  mockExitZoom: vi.fn(),
  mockSetAgentPositions: vi.fn(),
  mockSetSceneSnapshot: vi.fn(),
  mockSetSelectedSkillIds: vi.fn(),
  mockSetSelectedTeamId: vi.fn(),
  mockUnseatAgent: vi.fn(),
  mockUpdateAgent: vi.fn(),
  mockZoomToAgent: vi.fn(),
}))

const environmentState = vi.hoisted(() => ({
  current: { type: 'outdoor-park' as const },
  realTimeMode: false,
  setDreiPreset: vi.fn(),
  setEnvironment: vi.fn(),
  setRealTimeMode: vi.fn(),
  setSyncWithTheme: vi.fn(),
  setTimeValue: vi.fn(),
  syncWithTheme: false,
  timeValue: 12,
}))

const performanceState = vi.hoisted(() => ({
  config: { maxFps: 'auto' as const },
}))

vi.mock('@react-three/fiber', () => ({
  Canvas: (_props: { children: ReactNode }) => (
    <div data-testid="mock-r3f-canvas" />
  ),
}))

vi.mock('@react-three/drei', () => ({
  Loader: () => null,
}))

vi.mock('@/hooks/use-theme', () => ({
  useResolvedTheme: () => 'light',
}))

vi.mock('@/hooks/useAgentData', () => ({
  useAgentData: () => ({
    agents: [
      { id: 'agent-1', displayName: 'Agent One', appearance: null, fileOrder: [] },
    ],
    createAgent: mockCreateAgent,
    deleteAgent: mockDeleteAgent,
    isDeleting: false,
    isUpdating: false,
    updateAgent: mockUpdateAgent,
  }),
}))

vi.mock('@/hooks/useWorldDefaults', () => ({
  useWorldDefaults: () => {},
}))

vi.mock('@/hooks/useTeamActivity', () => ({
  useTeamActivity: () => {},
}))

vi.mock('@/hooks/useTeamGathering', () => ({
  useTeamGathering: () => {},
}))

vi.mock('@/stores/selectionStore', () => ({
  useSelectionStore: (selector: (state: Record<string, unknown>) => unknown) => selector({
    selectedSkillIds: [],
    setSelectedSkillIds: mockSetSelectedSkillIds,
    setSelectedTeamId: mockSetSelectedTeamId,
  }),
}))

vi.mock('@/stores/cameraStore', () => ({
  useCameraStore: (selector: (state: Record<string, unknown>) => unknown) => selector({
    exitZoom: mockExitZoom,
    focusedAgentId: null,
    mode: 'freeform',
    zoomToAgent: mockZoomToAgent,
    zoomToAgentRequested: 0,
  }),
}))

vi.mock('@/stores/furnitureStore', () => ({
  useFurnitureList: () => [],
  useSeatedAgents: () => ({}),
  useFurnitureStore: (selector: (state: Record<string, unknown>) => unknown) => selector({
    getAgentSeatPosition: () => null,
    seatAgent: vi.fn(),
    unseatAgent: mockUnseatAgent,
  }),
}))

vi.mock('@/stores/decorationStore', () => ({
  useDecorationList: () => [],
}))

vi.mock('@/stores/environmentStore', () => ({
  useEnvironmentStore: (selector: (state: typeof environmentState) => unknown) => selector(environmentState),
}))

vi.mock('@/stores/graphicsStore', () => ({
  useGraphicsStore: (selector: (state: Record<string, unknown>) => unknown) => selector({
    config: {
      antialiasing: 'fxaa',
      dpr: [1, 2],
      materialQuality: 'standard',
      shadows: true,
    },
    tier: 'medium',
  }),
}))

vi.mock('@/stores/performanceStore', () => {
  const usePerformanceStore = Object.assign(
    (selector: (state: typeof performanceState) => unknown) => selector(performanceState),
    {
      getState: () => ({
        config: performanceState.config,
        setSceneSnapshot: mockSetSceneSnapshot,
      }),
    },
  )

  return { usePerformanceStore }
})

vi.mock('@/stores/agentPositionStore', () => ({
  useAgentPositionStore: {
    getState: () => ({
      setAll: mockSetAgentPositions,
    }),
  },
}))

vi.mock('./AgentProvider', () => ({
  AgentProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock('./WorldScene', () => ({
  WorldScene: () => <div data-testid="mock-world-scene" />,
}))

vi.mock('./DisplayPanel', () => ({
  DisplayPanel: () => null,
}))

vi.mock('./AgentOverlay', () => ({
  AgentOverlay: () => null,
}))

vi.mock('./editor', () => ({
  ObjectPalette: ({ className }: { className?: string }) => <div className={className} />,
  WorldEditorToolbar: ({ className }: { className?: string }) => <div className={className} />,
}))

vi.mock('./furniture', () => ({
  FurnitureContextMenu: () => null,
}))

vi.mock('./furniture/SeatEditorOverlay', () => ({
  SeatEditorOverlay: () => null,
}))

vi.mock('./decorations', () => ({
  DecorationContextMenu: () => null,
}))

vi.mock('../agent/AgentCustomizeModal', () => ({
  AgentCustomizeModal: () => null,
}))

vi.mock('../shared/ConfirmDialog', () => ({
  ConfirmDialog: () => null,
}))

vi.mock('./rendering/RenderPipeline', () => ({
  RenderPipeline: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock('./rendering/EnvironmentSetup', () => ({
  EnvironmentSetup: () => null,
}))

vi.mock('./materials/MaterialProvider', () => ({
  MaterialProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock('../PanelErrorBoundary', () => ({
  PanelErrorBoundary: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock('./WorldErrorBoundary', () => ({
  WorldErrorBoundary: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock('./WorldErrorProvider', () => ({
  WorldErrorProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

describe('WorldCanvas', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders environment controls directly in world view', () => {
    render(<WorldCanvas skills={[]} />)

    expect(screen.getByTestId(selectors.world.canvas)).toBeInTheDocument()
    expect(screen.getByTestId(selectors.environment.controls)).toBeVisible()
    expect(screen.getByTestId(selectors.environment.sceneSpace)).toBeInTheDocument()
    expect(screen.getByTestId(selectors.environment.scenePark)).toBeInTheDocument()
    expect(screen.getByTestId(selectors.environment.sceneOffice)).toBeInTheDocument()
  })
})
