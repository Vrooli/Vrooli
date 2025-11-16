import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import 'react-flow-renderer/dist/style.css'
import OverviewDashboard from './components/OverviewDashboard'
import TechTreeCanvas from './components/TechTreeCanvas'
import useTechTreeData from './hooks/useTechTreeData'
import { sectorCategoryOptions } from './utils/constants'
import { formatTimestamp } from './utils/formatters'
import AppHeader from './components/layout/AppHeader'
import ViewToggle from './components/layout/ViewToggle'
import CreateSectorModal from './components/modals/CreateSectorModal'
import CreateStageModal from './components/modals/CreateStageModal'
import LinkScenarioModal from './components/modals/LinkScenarioModal'
import ConfirmDialog from './components/modals/ConfirmDialog'
import TreeManagementModal from './components/modals/TreeManagementModal'
import useFullscreenManager from './hooks/useFullscreenManager'
import { useDesignerModals } from './hooks/useDesignerModals'
import { useConfirmDialog } from './hooks/useConfirmDialog'
import { useTreeModal } from './hooks/useTreeModal'
import { useAIStageIdeas } from './hooks/useAIStageIdeas'
import { deleteStage, deleteSector, deleteScenarioMapping } from './services/techTree'
import AIStageIdeasModal from './components/modals/AIStageIdeasModal'
import type {
  ScenarioCatalogEntry,
  ProgressionStage,
  Sector,
  StageDependency,
  StrategicMilestone,
  StrategicValueBreakdown,
  StrategicValuePreset,
  StrategicValueSettings,
  TechTreeSummary,
  TreeViewMode,
  GraphViewMode,
  SectorSortOption
} from './types/techTree'
import {
  calculateStrategicValueBreakdown,
  DEFAULT_STRATEGIC_VALUE_SETTINGS,
  BUILT_IN_STRATEGIC_VALUE_PRESETS,
  DEFAULT_STRATEGIC_VALUE_PRESET_ID
} from './utils/strategicValue'

const VIEW_MODE_STORAGE_KEY = 'techTreeDesigner:viewMode'
const SECTOR_SORT_STORAGE_KEY = 'techTreeDesigner:sectorSort'
const STRATEGIC_VALUE_SETTINGS_KEY = 'techTreeDesigner:valueSettings'
const STRATEGIC_VALUE_PRESETS_KEY = 'techTreeDesigner:customValuePresets'
const STRATEGIC_VALUE_ACTIVE_PRESET_KEY = 'techTreeDesigner:activeValuePreset'

const isSectorSortOption = (value: string | null): value is SectorSortOption =>
  value === 'most-progress' ||
  value === 'least-progress' ||
  value === 'most-strategic' ||
  value === 'least-strategic' ||
  value === 'alpha-asc' ||
  value === 'alpha-desc'

const getStrategicValueScore = (sector: Sector) => {
  const stageProgressTotal = sector.stages?.reduce(
    (sum, stage) => sum + (stage.progress_percentage || 0),
    0
  )
  const scenarioLinks = sector.stages?.reduce(
    (sum, stage) => sum + (stage.scenario_mappings?.length || 0),
    0
  )
  return (sector.progress_percentage || 0) * 2 + stageProgressTotal + scenarioLinks * 15
}

const sortSectorsByPreference = (items: Sector[], sort: SectorSortOption) => {
  const sectors = [...items]
  switch (sort) {
    case 'most-progress':
      return sectors.sort((a, b) => (b.progress_percentage || 0) - (a.progress_percentage || 0))
    case 'least-progress':
      return sectors.sort((a, b) => (a.progress_percentage || 0) - (b.progress_percentage || 0))
    case 'most-strategic':
      return sectors.sort((a, b) => getStrategicValueScore(b) - getStrategicValueScore(a))
    case 'least-strategic':
      return sectors.sort((a, b) => getStrategicValueScore(a) - getStrategicValueScore(b))
    case 'alpha-asc':
      return sectors.sort((a, b) => a.name.localeCompare(b.name))
    case 'alpha-desc':
      return sectors.sort((a, b) => b.name.localeCompare(a.name))
    default:
      return sectors
  }
}

const App: React.FC = () => {
  const {
    techTrees,
    treeLoading,
    selectedTreeId,
    setSelectedTreeId,
    selectedTreeSummary,
    sectors,
    milestones,
    dependencies,
    loading,
    graphNotice,
    setGraphNotice,
    scenarioCatalog,
    isSyncingCatalog,
    refreshTreeData,
    handleScenarioCatalogRefresh,
    handleScenarioVisibility,
    buildTreeAwarePath
  } = useTechTreeData()

  const [selectedSector, setSelectedSector] = useState<Sector | null>(null)
  const [viewMode, setViewMode] = useState<TreeViewMode>(() => {
    if (typeof window === 'undefined') {
      return 'overview'
    }
    const stored = window.localStorage.getItem(VIEW_MODE_STORAGE_KEY) as TreeViewMode | null
    return stored === 'designer' ? 'designer' : 'overview'
  })
  const [sectorSort, setSectorSort] = useState<SectorSortOption>(() => {
    if (typeof window === 'undefined') {
      return 'most-progress'
    }
    const stored = window.localStorage.getItem(SECTOR_SORT_STORAGE_KEY)
    return isSectorSortOption(stored) ? stored : 'most-progress'
  })
  const [strategicValueSettings, setStrategicValueSettings] = useState<StrategicValueSettings>(() => {
    if (typeof window === 'undefined') {
      return DEFAULT_STRATEGIC_VALUE_SETTINGS
    }
    try {
      const stored = window.localStorage.getItem(STRATEGIC_VALUE_SETTINGS_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        return {
          ...DEFAULT_STRATEGIC_VALUE_SETTINGS,
          ...parsed
        }
      }
    } catch (error) {
      console.warn('Unable to parse strategic value settings from storage', error)
    }
    return DEFAULT_STRATEGIC_VALUE_SETTINGS
  })
  const [isStrategicPanelOpen, setIsStrategicPanelOpen] = useState(false)
  const [customValuePresets, setCustomValuePresets] = useState<StrategicValuePreset[]>(() => {
    if (typeof window === 'undefined') {
      return []
    }
    try {
      const stored = window.localStorage.getItem(STRATEGIC_VALUE_PRESETS_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        if (Array.isArray(parsed)) {
          return parsed
        }
      }
    } catch (error) {
      console.warn('Unable to parse saved value presets', error)
    }
    return []
  })
  const [activeValuePresetId, setActiveValuePresetId] = useState<string | null>(() => {
    if (typeof window === 'undefined') {
      return DEFAULT_STRATEGIC_VALUE_PRESET_ID
    }
    const stored = window.localStorage.getItem(STRATEGIC_VALUE_ACTIVE_PRESET_KEY)
    return stored || DEFAULT_STRATEGIC_VALUE_PRESET_ID
  })

  // New simplified graph view mode system
  const [graphViewMode, setGraphViewMode] = useState<GraphViewMode>('tech-tree')
  const [showHiddenScenarios, setShowHiddenScenarios] = useState(false)
  const [showIsolatedScenarios, setShowIsolatedScenarios] = useState(false)

  const techTreeCanvasRef = useRef<HTMLDivElement | null>(null)

  const { canFullscreen, isFullscreen, isLayoutFullscreen, toggleFullscreen } = useFullscreenManager(
    techTreeCanvasRef
  )
  const selectedTree = selectedTreeSummary?.tree ?? null

  const { sectorModal, stageModal, scenarioModal } = useDesignerModals({
    selectedTreeId,
    selectedSector,
    sectors,
    refreshTreeData
  })

  const { confirmState, isProcessing, showConfirm, handleConfirm, handleCancel } = useConfirmDialog()

  const treeModal = useTreeModal({
    onSuccess: () => {
      setGraphNotice('Tech tree operation completed successfully')
      refreshTreeData()
    },
    onError: (error) => {
      setGraphNotice(`Error: ${error}`)
    }
  })

  const aiIdeasModal = useAIStageIdeas({
    selectedTreeId,
    onSuccess: (message) => {
      setGraphNotice(message)
      refreshTreeData()
    },
    onError: (error) => {
      setGraphNotice(`AI Error: ${error}`)
    }
  })

  const handleCreateTree = useCallback(() => {
    treeModal.openForCreate()
  }, [treeModal])

  const handleCloneTree = useCallback(() => {
    if (selectedTreeSummary) {
      treeModal.openForClone(selectedTreeSummary)
    }
  }, [selectedTreeSummary, treeModal])

  const handleGenerateAIIdeas = useCallback((sectorId: string) => {
    const sector = sectors.find(s => s.id === sectorId)
    if (sector) {
      aiIdeasModal.open(sector)
    }
  }, [sectors, aiIdeasModal])

  const sortedSectors = useMemo(() => sortSectorsByPreference(sectors, sectorSort), [
    sectors,
    sectorSort
  ])

  useEffect(() => {
    if (!sortedSectors.length) {
      setSelectedSector(null)
      return
    }
    if (!loading && !treeLoading) {
      const hasSelection = sortedSectors.some((sector) => sector.id === selectedSector?.id)
      if (!hasSelection) {
        setSelectedSector(sortedSectors[0])
      }
    }
  }, [loading, treeLoading, sortedSectors, selectedSector])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(VIEW_MODE_STORAGE_KEY, viewMode)
    }
  }, [viewMode])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(SECTOR_SORT_STORAGE_KEY, sectorSort)
    }
  }, [sectorSort])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(
        STRATEGIC_VALUE_SETTINGS_KEY,
        JSON.stringify(strategicValueSettings)
      )
    }
  }, [strategicValueSettings])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(
        STRATEGIC_VALUE_PRESETS_KEY,
        JSON.stringify(customValuePresets)
      )
    }
  }, [customValuePresets])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (activeValuePresetId) {
        window.localStorage.setItem(STRATEGIC_VALUE_ACTIVE_PRESET_KEY, activeValuePresetId)
      } else {
        window.localStorage.removeItem(STRATEGIC_VALUE_ACTIVE_PRESET_KEY)
      }
    }
  }, [activeValuePresetId])

  const scenarioEntryMap = useMemo(() => {
    const map = new Map<string, ScenarioCatalogEntry>()
    if (scenarioCatalog.scenarios) {
      scenarioCatalog.scenarios.forEach((entry) => {
        map.set(entry.name, entry)
        if (entry.name) {
          map.set(entry.name.toLowerCase(), entry)
        }
      })
    }
    return map
  }, [scenarioCatalog.scenarios])

  const strategicValueBreakdown: StrategicValueBreakdown = useMemo(
    () =>
      calculateStrategicValueBreakdown({
        milestones: milestones as StrategicMilestone[],
        sectors,
        settings: strategicValueSettings
      }),
    [milestones, sectors, strategicValueSettings]
  )

  const insightMetrics = useMemo(() => {
    if (!sectors.length && !milestones.length) {
      return {
        averageSectorProgress: 0,
        activeMilestones: 0,
        totalPotentialValue: 0
      }
    }

    const averageSectorProgress = sectors.length
      ? Math.round(
          sectors.reduce((sum, sector) => sum + (sector.progress_percentage || 0), 0) /
            sectors.length
        )
      : 0

    const activeMilestones = milestones.filter(
      (milestone) => (milestone.completion_percentage || 0) >= 50
    ).length

    const totalPotentialValue = strategicValueBreakdown.fullPotentialValue

    return { averageSectorProgress, activeMilestones, totalPotentialValue }
  }, [milestones, sectors, strategicValueBreakdown.fullPotentialValue])

  const treeBadgeClass = useMemo(() => {
    if (!selectedTree) {
      return 'tree-badge--idle'
    }
    if (selectedTree.tree_type === 'official') {
      return 'tree-badge--official'
    }
    if (selectedTree.tree_type === 'draft') {
      return 'tree-badge--draft'
    }
    return 'tree-badge--experimental'
  }, [selectedTree])

  const treeBadgeLabel = selectedTree ? selectedTree.tree_type.replace('_', ' ') : 'Unavailable'

  const treeStatsSummary = useMemo(() => {
    if (!selectedTreeSummary) {
      return null
    }
    const sectorsCount = selectedTreeSummary.sector_count ?? 0
    const stagesCount = selectedTreeSummary.stage_count ?? 0
    const mappingCount = selectedTreeSummary.scenario_mapping_count ?? 0
    return `${sectorsCount} sectors • ${stagesCount} stages • ${mappingCount} scenario links`
  }, [selectedTreeSummary])

  const isTreeSelectorDisabled = treeLoading || !techTrees.length

  const handleTreeSelectionChange = useCallback(
    (value: string | null) => {
      setSelectedTreeId(value)
    },
    [setSelectedTreeId]
  )

  const insightLoader = loading || treeLoading

  const handleGraphPersist = useCallback(
    (payload: { notice?: string; message?: string }) => {
      if (payload?.notice) {
        setGraphNotice(payload.notice)
      } else if (payload?.message) {
        setGraphNotice(payload.message)
      }
      refreshTreeData()
    },
    [refreshTreeData, setGraphNotice]
  )

  const handleEditStage = useCallback((stage: ProgressionStage) => {
    stageModal.openForEdit(stage)
  }, [stageModal])

  const handleEditSector = useCallback((sector: Sector) => {
    sectorModal.openForEdit(sector)
  }, [sectorModal])

  const handleCreateStage = useCallback((sectorId?: string, stageType?: string, parentStageId?: string) => {
    stageModal.open({ sectorId, stageType, parentStageId })
  }, [stageModal])

  const handleDeleteStage = useCallback((stageId: string, stageName: string) => {
    showConfirm({
      title: 'Delete Stage',
      message: `Are you sure you want to delete "${stageName}"? This action cannot be undone.`,
      confirmLabel: 'Delete Stage',
      isDangerous: true,
      onConfirm: async () => {
        await deleteStage(stageId, selectedTreeId || undefined)
        setGraphNotice(`Stage "${stageName}" deleted successfully`)
        refreshTreeData()
      }
    })
  }, [showConfirm, selectedTreeId, refreshTreeData, setGraphNotice])

  const handleDeleteSector = useCallback((sectorId: string, sectorName: string) => {
    showConfirm({
      title: 'Delete Sector',
      message: `Are you sure you want to delete "${sectorName}"? This will also delete all stages within this sector. This action cannot be undone.`,
      confirmLabel: 'Delete Sector',
      isDangerous: true,
      onConfirm: async () => {
        await deleteSector(sectorId, selectedTreeId || undefined)
        setGraphNotice(`Sector "${sectorName}" deleted successfully`)
        refreshTreeData()
      }
    })
  }, [showConfirm, selectedTreeId, refreshTreeData, setGraphNotice])

  const handleUnlinkScenario = useCallback((scenarioName: string, mappingId?: string) => {
    showConfirm({
      title: 'Unlink Scenario',
      message: `Are you sure you want to unlink scenario "${scenarioName}"? This action cannot be undone.`,
      confirmLabel: 'Unlink Scenario',
      isDangerous: true,
      onConfirm: async () => {
        if (mappingId) {
          await deleteScenarioMapping(mappingId, selectedTreeId || undefined)
          setGraphNotice(`Scenario "${scenarioName}" unlinked successfully`)
          refreshTreeData()
        }
      }
    })
  }, [showConfirm, selectedTreeId, refreshTreeData, setGraphNotice])

  const handleStrategicValueSettingsChange = useCallback(
    (settings: StrategicValueSettings, options?: { preservePreset?: boolean }) => {
      setStrategicValueSettings(settings)
      if (!options?.preservePreset) {
        setActiveValuePresetId(null)
      }
    },
    []
  )

  const handleStrategicPanelToggle = useCallback((open?: boolean) => {
    setIsStrategicPanelOpen((current) => (typeof open === 'boolean' ? open : !current))
  }, [])

  const valuePresets = useMemo(
    () => [...BUILT_IN_STRATEGIC_VALUE_PRESETS, ...customValuePresets],
    [customValuePresets]
  )

  const handleApplyValuePreset = useCallback(
    (presetId: string) => {
      const preset = valuePresets.find((item) => item.id === presetId)
      if (!preset) {
        return
      }
      setActiveValuePresetId(presetId)
      handleStrategicValueSettingsChange({ ...preset.settings }, { preservePreset: true })
    },
    [valuePresets, handleStrategicValueSettingsChange]
  )

  const handleCreateValuePreset = useCallback(
    ({ name, description }: { name: string; description?: string }) => {
      const trimmedName = name.trim()
      if (!trimmedName) {
        return
      }
      const preset: StrategicValuePreset = {
        id: `custom-${Date.now()}`,
        name: trimmedName,
        description: description?.trim() || undefined,
        settings: { ...strategicValueSettings }
      }
      setCustomValuePresets((current) => [...current, preset])
      setActiveValuePresetId(preset.id)
    },
    [strategicValueSettings]
  )

  const handleDeleteValuePreset = useCallback(
    (presetId: string) => {
      setCustomValuePresets((current) => current.filter((preset) => preset.id !== presetId))
      if (activeValuePresetId === presetId) {
        setActiveValuePresetId(null)
      }
    },
    [activeValuePresetId]
  )

  if (insightLoader) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <div className="text-center">
          <div className="loading-spinner mb-4"></div>
          <h3>Loading Strategic Intelligence...</h3>
          <p className="text-slate-400">Initializing civilization roadmap</p>
        </div>
      </div>
    )
  }

  const lastSyncedLabel = formatTimestamp(scenarioCatalog.last_synced)

  return (
    <main className="app-shell">
      <ViewToggle viewMode={viewMode} onChange={setViewMode} />

      <AppHeader
        viewMode={viewMode}
        techTrees={techTrees as TechTreeSummary[]}
        selectedTreeId={selectedTreeId}
        treeBadgeClass={treeBadgeClass}
        treeBadgeLabel={treeBadgeLabel}
        treeStatsSummary={treeStatsSummary}
        isTreeSelectorDisabled={isTreeSelectorDisabled}
        onTreeSelectionChange={handleTreeSelectionChange}
        onCreateSector={sectorModal.open}
        onCreateTree={handleCreateTree}
        onCloneTree={handleCloneTree}
        graphViewMode={graphViewMode}
        onGraphViewModeChange={setGraphViewMode}
        showHiddenScenarios={showHiddenScenarios}
        showIsolatedScenarios={showIsolatedScenarios}
        onToggleHiddenScenarios={setShowHiddenScenarios}
        onToggleIsolatedScenarios={setShowIsolatedScenarios}
        onSyncScenarios={handleScenarioCatalogRefresh}
        isSyncingScenarios={isSyncingCatalog}
        lastSyncedLabel={lastSyncedLabel}
        hiddenScenarios={scenarioCatalog.hidden || []}
        scenarioEntryMap={scenarioEntryMap}
        onToggleScenarioVisibility={handleScenarioVisibility}
      />

      {viewMode === 'overview' ? (
        <OverviewDashboard
          sectors={sortedSectors}
          selectedSector={selectedSector}
          onSelectSector={setSelectedSector}
          milestones={milestones as StrategicMilestone[]}
          insightMetrics={insightMetrics}
          onRequestNewStage={stageModal.open}
          sectorSort={sectorSort}
          onSectorSortChange={setSectorSort}
          strategicValueBreakdown={strategicValueBreakdown}
          strategicValueSettings={strategicValueSettings}
          onStrategicValueSettingsChange={handleStrategicValueSettingsChange}
          isStrategicValuePanelOpen={isStrategicPanelOpen}
          onStrategicValuePanelToggle={handleStrategicPanelToggle}
          valuePresets={valuePresets}
          customValuePresets={customValuePresets}
          activeValuePresetId={activeValuePresetId}
          onApplyValuePreset={handleApplyValuePreset}
          onCreateValuePreset={handleCreateValuePreset}
          onDeleteValuePreset={handleDeleteValuePreset}
        />
      ) : (
        <TechTreeCanvas
          sectors={sectors}
          dependencies={dependencies as StageDependency[]}
          selectedTreeId={selectedTreeId}
          graphNotice={graphNotice}
          techTreeCanvasRef={techTreeCanvasRef}
          isFullscreen={isFullscreen}
          isLayoutFullscreen={isLayoutFullscreen}
          toggleFullscreen={toggleFullscreen}
          canFullscreen={canFullscreen}
          onGraphPersist={handleGraphPersist}
          scenarioCatalog={scenarioCatalog}
          scenarioEntryMap={scenarioEntryMap}
          showLiveScenarios={graphViewMode === 'hybrid'}
          scenarioOnlyMode={graphViewMode === 'scenarios'}
          showHiddenScenarios={showHiddenScenarios}
          showIsolatedScenarios={showIsolatedScenarios}
          handleScenarioVisibility={handleScenarioVisibility}
          setGraphNotice={setGraphNotice}
          buildTreeAwarePath={buildTreeAwarePath}
          onLinkScenario={scenarioModal.openForStage}
          onCreateSector={sectorModal.open}
          onCreateStage={handleCreateStage}
          onGenerateAIStageIdeas={handleGenerateAIIdeas}
          onEditStage={handleEditStage}
          onEditSector={handleEditSector}
          onDeleteStage={handleDeleteStage}
          onDeleteSector={handleDeleteSector}
          onUnlinkScenario={handleUnlinkScenario}
          refreshTreeData={refreshTreeData}
        />
      )}

      <CreateSectorModal
        isOpen={sectorModal.isOpen}
        formState={sectorModal.formState}
        categoryOptions={sectorCategoryOptions}
        onFieldChange={sectorModal.onFieldChange}
        onSubmit={sectorModal.onSubmit}
        onClose={sectorModal.close}
        isSaving={sectorModal.isSaving}
        errorMessage={sectorModal.errorMessage}
        editMode={sectorModal.editMode}
        sectorId={sectorModal.sectorId}
      />

      <CreateStageModal
        isOpen={stageModal.isOpen}
        sectors={sectors}
        formState={stageModal.formState}
        onFieldChange={stageModal.onFieldChange}
        onSubmit={stageModal.onSubmit}
        onClose={stageModal.close}
        onGenerateIdea={stageModal.onGenerateIdea}
        isGeneratingIdea={stageModal.isGeneratingIdea}
        isSaving={stageModal.isSaving}
        errorMessage={stageModal.errorMessage}
        editMode={stageModal.editMode}
        stageId={stageModal.stageId}
      />

      <LinkScenarioModal
        isOpen={scenarioModal.isOpen}
        formState={scenarioModal.formState}
        targetStage={scenarioModal.targetStage}
        onFieldChange={scenarioModal.onFieldChange}
        onSubmit={scenarioModal.onSubmit}
        onClose={scenarioModal.close}
        isSaving={scenarioModal.isSaving}
        errorMessage={scenarioModal.errorMessage}
      />

      <ConfirmDialog
        isOpen={confirmState.isOpen}
        title={confirmState.title}
        message={confirmState.message}
        confirmLabel={confirmState.confirmLabel}
        isDangerous={confirmState.isDangerous}
        isProcessing={isProcessing}
        onConfirm={handleConfirm}
        onCancel={handleCancel}
      />

      <TreeManagementModal
        isOpen={treeModal.isOpen}
        mode={treeModal.mode}
        sourceTree={treeModal.sourceTree}
        formState={treeModal.formState}
        onFieldChange={treeModal.onFieldChange}
        onSubmit={treeModal.onSubmit}
        onClose={treeModal.close}
        isSaving={treeModal.isSaving}
        errorMessage={treeModal.errorMessage}
      />

      <AIStageIdeasModal
        isOpen={aiIdeasModal.isOpen}
        sector={aiIdeasModal.targetSector}
        onClose={aiIdeasModal.close}
        onGenerate={aiIdeasModal.handleGenerate}
        onCreateStages={aiIdeasModal.handleCreateStages}
      />
    </main>
  )
}

export default App
