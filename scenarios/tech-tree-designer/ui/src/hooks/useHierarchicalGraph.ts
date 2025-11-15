import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Sector } from '../types/techTree'
import type { StageNode, DesignerNode } from '../types/graph'

interface UseHierarchicalGraphParams {
  sectors: Sector[]
  stageNodes: StageNode[]
}

interface UseHierarchicalGraphResult {
  collapsedSectors: Set<string>
  toggleSectorCollapse: (sectorId: string) => void
  filteredStageNodes: StageNode[]
  sectorGroupNodes: DesignerNode[]
}

const STORAGE_KEY = 'tech-tree-designer:collapsed-sectors'

/**
 * Custom hook for managing hierarchical graph with collapsible sectors.
 * Sectors can be collapsed to hide their child stages, improving performance and clarity.
 */
export const useHierarchicalGraph = ({
  sectors,
  stageNodes
}: UseHierarchicalGraphParams): UseHierarchicalGraphResult => {
  // Load collapsed state from localStorage and validate against current sectors
  const [collapsedSectors, setCollapsedSectors] = useState<Set<string>>(() => {
    if (typeof window === 'undefined') {
      return new Set()
    }
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        const storedIds = Array.isArray(parsed) ? parsed : []

        // Only keep collapsed sector IDs that exist in the current sector list
        // This prevents stale localStorage data from hiding the entire graph
        const validSectorIds = new Set(sectors.map(s => s.id))
        const validCollapsed = storedIds.filter((id: string) => validSectorIds.has(id))

        console.log('[useHierarchicalGraph] Loaded collapsed sectors:', validCollapsed.length, 'of', storedIds.length)
        return new Set(validCollapsed)
      }
    } catch (error) {
      console.warn('Failed to load collapsed sectors from localStorage', error)
    }
    return new Set()
  })

  // Persist collapsed state to localStorage
  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(Array.from(collapsedSectors)))
    } catch (error) {
      console.warn('Failed to save collapsed sectors to localStorage', error)
    }
  }, [collapsedSectors])

  // Validate collapsed sectors when sectors change (handles lazy-loaded sectors)
  useEffect(() => {
    if (sectors.length === 0 || collapsedSectors.size === 0) {
      return
    }

    const validSectorIds = new Set(sectors.map(s => s.id))
    const invalidCollapsed = Array.from(collapsedSectors).filter(id => !validSectorIds.has(id))

    if (invalidCollapsed.length > 0) {
      console.warn('[useHierarchicalGraph] Removing invalid collapsed sectors:', invalidCollapsed)
      setCollapsedSectors(prev => {
        const next = new Set(prev)
        invalidCollapsed.forEach(id => next.delete(id))
        return next
      })
    }
  }, [sectors, collapsedSectors])

  // Toggle sector collapse state
  const toggleSectorCollapse = useCallback((sectorId: string) => {
    setCollapsedSectors((prev) => {
      const next = new Set(prev)
      if (next.has(sectorId)) {
        next.delete(sectorId)
      } else {
        next.add(sectorId)
      }
      return next
    })
  }, [])

  // Automatically clear collapsed sectors if it would result in an empty graph
  useEffect(() => {
    // Wait for both stageNodes and sectors to be populated
    if (stageNodes.length === 0 || sectors.length === 0) {
      return
    }

    // Don't run auto-expand if there are no collapsed sectors
    if (collapsedSectors.size === 0) {
      return
    }

    const visibleNodes = stageNodes.filter((node) => {
      const stage = node.data
      const parentSector = sectors.find((s) => s.id === stage.sectorId || s.stages?.some((st) => st.id === node.id))
      if (!parentSector) {
        return true
      }
      return !collapsedSectors.has(parentSector.id)
    })

    // If all sectors are collapsed, clear the collapsed state
    if (visibleNodes.length === 0) {
      console.warn('[useHierarchicalGraph] All sectors collapsed - auto-expanding to prevent empty graph')
      setCollapsedSectors(new Set())
    }
  }, [stageNodes, sectors, collapsedSectors])

  // Filter stage nodes based on collapsed sectors
  const filteredStageNodes = useMemo(() => {
    const filtered = stageNodes.filter((node) => {
      const stage = node.data
      // Find parent sector for this stage
      const parentSector = sectors.find((s) => s.id === stage.sectorId || s.stages?.some((st) => st.id === node.id))
      if (!parentSector) {
        return true // Show orphaned stages
      }
      return !collapsedSectors.has(parentSector.id)
    })

    console.log('[useHierarchicalGraph] Filtered nodes:', filtered.length, 'from', stageNodes.length, 'collapsed:', collapsedSectors.size)
    return filtered
  }, [stageNodes, sectors, collapsedSectors])

  // Create sector group nodes (collapsed sector representations)
  const sectorGroupNodes = useMemo(() => {
    const groupNodes: DesignerNode[] = []

    sectors.forEach((sector) => {
      if (!collapsedSectors.has(sector.id)) {
        return // Only create group nodes for collapsed sectors
      }

      const stageCount = sector.stages?.length || 0
      const avgProgress = stageCount > 0
        ? sector.stages!.reduce((sum, s) => sum + (s.progress_percentage || 0), 0) / stageCount
        : 0

      // Position at the first stage's location (or sector position)
      const firstStage = sector.stages?.[0]
      const posX = firstStage?.position_x ?? sector.position_x ?? 0
      const posY = firstStage?.position_y ?? sector.position_y ?? 0

      groupNodes.push({
        id: `sector-group::${sector.id}`,
        position: { x: posX, y: posY },
        data: {
          sectorId: sector.id,
          sectorName: sector.name,
          sectorColor: sector.color,
          stageCount,
          avgProgress,
          collapsed: true
        },
        type: 'sectorGroup',
        style: {
          background: `linear-gradient(135deg, ${sector.color || '#3B82F6'}22, ${sector.color || '#3B82F6'}44)`,
          color: '#f8fafc',
          border: `2px solid ${sector.color || '#3B82F6'}`,
          borderRadius: 16,
          padding: 20,
          fontSize: 14,
          fontWeight: 600,
          boxShadow: '0 16px 32px rgba(2, 6, 23, 0.5)',
          minWidth: 240,
          minHeight: 120
        }
      })
    })

    return groupNodes
  }, [sectors, collapsedSectors])

  return {
    collapsedSectors,
    toggleSectorCollapse,
    filteredStageNodes,
    sectorGroupNodes
  }
}
