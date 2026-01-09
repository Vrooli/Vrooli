import { useSectorModal } from './useSectorModal'
import { useStageModal } from './useStageModal'
import { useScenarioModal } from './useScenarioModal'
import type { Sector } from '../types/techTree'

interface UseDesignerModalsArgs {
  selectedTreeId: string | null
  selectedSector: Sector | null
  sectors: Sector[]
  refreshTreeData: () => void
}

/**
 * Composed hook that manages all three designer modals (sector, stage, scenario).
 * This hook delegates to three specialized hooks for better separation of concerns.
 *
 * @deprecated Consider using individual modal hooks directly for new code.
 * This hook exists primarily for backward compatibility with existing components.
 */
export const useDesignerModals = ({
  selectedTreeId,
  selectedSector,
  sectors,
  refreshTreeData
}: UseDesignerModalsArgs) => {
  const sectorModal = useSectorModal({
    selectedTreeId,
    defaultCategory: selectedSector?.category,
    onSuccess: refreshTreeData
  })

  const stageModal = useStageModal({
    selectedTreeId,
    selectedSector,
    sectors,
    onSuccess: refreshTreeData
  })

  const scenarioModal = useScenarioModal({
    selectedTreeId,
    onSuccess: refreshTreeData
  })

  return {
    sectorModal,
    stageModal,
    scenarioModal
  }
}

export type DesignerModalHook = ReturnType<typeof useDesignerModals>
