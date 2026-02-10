/**
 * FurnitureContextMenu - Context menu for furniture interactions.
 * Shows options to sit agents, move, or delete furniture.
 */

import { useCallback, useMemo } from 'react'
import { Move, Trash2, X, UserPlus, Sun, Lightbulb, LightbulbOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useFurnitureStore, useFurnitureList } from '@/stores/furnitureStore'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { FURNITURE_CONFIGS, type FurnitureInstance } from '@/types/furniture'
import type { LightMode } from '@/types/decoration'
import type { Agent } from '@/types/agent'

const LIGHT_MODES: Array<{ mode: LightMode; label: string; Icon: typeof Sun }> = [
  { mode: 'auto', label: 'Auto', Icon: Sun },
  { mode: 'on', label: 'On', Icon: Lightbulb },
  { mode: 'off', label: 'Off', Icon: LightbulbOff },
]

interface FurnitureContextMenuProps {
  furniture: FurnitureInstance | null
  agents: Agent[]
  onClose: () => void
  onSitAgent: (agentId: string, furnitureId: string, seatIndex: number) => void
  onUnsitAgent: (agentId: string) => void
  seatedAgents: Map<string, { furnitureId: string; seatIndex: number }>
  className?: string
}

/**
 * Context menu for furniture interactions.
 */
export function FurnitureContextMenu({
  furniture,
  agents,
  onClose,
  onSitAgent,
  onUnsitAgent,
  seatedAgents,
  className,
}: FurnitureContextMenuProps) {
  const removeFurniture = useFurnitureStore((state) => state.removeFurniture)
  const setFurnitureLightMode = useFurnitureStore((state) => state.setLightMode)
  const setEditMode = useWorldEditorStore((state) => state.setEditMode)
  const selectObject = useWorldEditorStore((state) => state.selectObject)

  // Read live furniture so light mode changes reflect immediately
  const furnitureListLive = useFurnitureList()
  const liveFurniture = furnitureListLive.find((f) => f.id === furniture?.id) ?? null
  const current = liveFurniture ?? furniture
  const currentLightMode: LightMode = current?.lightMode ?? 'auto'

  const handleSetLightMode = useCallback(
    (mode: LightMode) => {
      if (!furniture) return
      setFurnitureLightMode(furniture.id, mode)
    },
    [furniture, setFurnitureLightMode]
  )

  // Get furniture config
  const config = furniture ? FURNITURE_CONFIGS[furniture.type] : null
  const seats = useMemo(() => config?.seats ?? [], [config])
  const hasSeat = seats.length > 0

  // Find which seats are occupied
  const occupiedSeats = useMemo(() => {
    const occupied = new Set<number>()
    seatedAgents.forEach((seat) => {
      if (seat.furnitureId === furniture?.id) {
        occupied.add(seat.seatIndex)
      }
    })
    return occupied
  }, [seatedAgents, furniture?.id])

  // Find available seats
  const availableSeats = useMemo(() => {
    return seats.map((_, index) => index).filter((index) => !occupiedSeats.has(index))
  }, [seats, occupiedSeats])

  // Agents who are not seated anywhere
  const unseatedAgents = useMemo(() => {
    const seatedAgentIds = new Set(seatedAgents.keys())
    return agents.filter((a) => !seatedAgentIds.has(a.id))
  }, [agents, seatedAgents])

  // Agents seated on this furniture
  const agentsOnThisFurniture = useMemo(() => {
    const result: Array<{ agent: Agent; seatIndex: number }> = []
    seatedAgents.forEach((seat, agentId) => {
      if (seat.furnitureId === furniture?.id) {
        const agent = agents.find((a) => a.id === agentId)
        if (agent) {
          result.push({ agent, seatIndex: seat.seatIndex })
        }
      }
    })
    return result
  }, [seatedAgents, furniture?.id, agents])

  const handleMove = useCallback(() => {
    if (!furniture) return
    setEditMode(true)
    selectObject({ id: furniture.id, type: 'furniture' })
    onClose()
  }, [furniture, setEditMode, selectObject, onClose])

  const handleDelete = useCallback(() => {
    if (!furniture) return
    // First unsit any agents on this furniture
    agentsOnThisFurniture.forEach(({ agent }) => {
      onUnsitAgent(agent.id)
    })
    removeFurniture(furniture.id)
    onClose()
  }, [furniture, agentsOnThisFurniture, onUnsitAgent, removeFurniture, onClose])

  const handleSitAgent = useCallback(
    (agentId: string) => {
      const firstSeat = availableSeats[0]
      if (!furniture || firstSeat === undefined) return
      // Sit on the first available seat
      onSitAgent(agentId, furniture.id, firstSeat)
    },
    [furniture, availableSeats, onSitAgent]
  )

  const handleUnsitAgent = useCallback(
    (agentId: string) => {
      onUnsitAgent(agentId)
    },
    [onUnsitAgent]
  )

  if (!furniture) {
    return null
  }

  return (
    <div
      className={`
        w-64 p-3
        bg-slate-800/95 backdrop-blur-sm
        border border-slate-700 rounded-lg
        shadow-xl
        ${className ?? ''}
      `}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-slate-200">
          {config?.displayName ?? 'Furniture'}
        </h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Light Mode Controls - only for light-emitting furniture */}
      {config?.emitsLight && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5">Light Mode</div>
          <div className="flex gap-1">
            {LIGHT_MODES.map(({ mode, label, Icon }) => (
              <button
                key={mode}
                onClick={() => handleSetLightMode(mode)}
                className={`
                  flex-1 flex items-center justify-center gap-1 py-1.5 rounded text-xs
                  transition-colors
                  ${currentLightMode === mode
                    ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                    : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/50 hover:text-slate-300 border border-transparent'
                  }
                `}
              >
                <Icon className="h-3 w-3" />
                {label}
              </button>
            ))}
          </div>
          <div className="text-[10px] text-slate-500 mt-1">
            {currentLightMode === 'auto' && 'Follows day/night cycle'}
            {currentLightMode === 'on' && 'Always on'}
            {currentLightMode === 'off' && 'Always off'}
          </div>
        </div>
      )}

      {/* Seated Agents */}
      {agentsOnThisFurniture.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5">Seated</div>
          <div className="space-y-1">
            {agentsOnThisFurniture.map(({ agent, seatIndex }) => (
              <div
                key={agent.id}
                className="flex items-center justify-between p-1.5 bg-slate-700/50 rounded"
              >
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: agent.appearance?.body ?? '#6366f1' }}
                  />
                  <span className="text-xs text-slate-300">{agent.displayName}</span>
                  <span className="text-xs text-slate-500">Seat {seatIndex + 1}</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleUnsitAgent(agent.id)}
                  className="h-5 px-1 text-xs text-slate-400 hover:text-slate-200"
                >
                  Stand
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Sit Agent Section */}
      {hasSeat && availableSeats.length > 0 && unseatedAgents.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5 flex items-center gap-1">
            <UserPlus className="h-3 w-3" />
            Sit Agent ({availableSeats.length} seat{availableSeats.length !== 1 ? 's' : ''} available)
          </div>
          <div className="max-h-32 overflow-y-auto space-y-1">
            {unseatedAgents.map((agent) => (
              <button
                key={agent.id}
                onClick={() => handleSitAgent(agent.id)}
                className="
                  w-full flex items-center gap-2 p-1.5
                  bg-slate-700/30 hover:bg-slate-600/50
                  rounded transition-colors
                "
              >
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: agent.appearance?.body ?? '#6366f1' }}
                />
                <span className="text-xs text-slate-300">{agent.displayName}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* No seats message */}
      {!hasSeat && (
        <div className="mb-3 p-2 bg-slate-700/30 rounded text-xs text-slate-400">
          This furniture doesn&apos;t have seats
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleMove}
          className="flex-1 h-8 gap-1.5 text-blue-400 hover:text-blue-300 hover:bg-blue-500/20"
        >
          <Move className="h-3.5 w-3.5" />
          <span className="text-xs">Move</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleDelete}
          className="flex-1 h-8 gap-1.5 text-red-400 hover:text-red-300 hover:bg-red-500/20"
        >
          <Trash2 className="h-3.5 w-3.5" />
          <span className="text-xs">Delete</span>
        </Button>
      </div>
    </div>
  )
}
