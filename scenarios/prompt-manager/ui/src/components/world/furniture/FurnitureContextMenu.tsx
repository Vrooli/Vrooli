/**
 * FurnitureContextMenu - Context menu for furniture interactions.
 * Shows options to sit members, move, or delete furniture.
 */

import { useCallback, useMemo } from 'react'
import { Move, Trash2, X, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { FURNITURE_CONFIGS, type FurnitureInstance } from '@/types/furniture'
import type { Member } from '@/types/member'

interface FurnitureContextMenuProps {
  furniture: FurnitureInstance | null
  members: Member[]
  onClose: () => void
  onSitMember: (memberId: string, furnitureId: string, seatIndex: number) => void
  onUnsitMember: (memberId: string) => void
  seatedMembers: Map<string, { furnitureId: string; seatIndex: number }>
  className?: string
}

/**
 * Context menu for furniture interactions.
 */
export function FurnitureContextMenu({
  furniture,
  members,
  onClose,
  onSitMember,
  onUnsitMember,
  seatedMembers,
  className,
}: FurnitureContextMenuProps) {
  const removeFurniture = useFurnitureStore((state) => state.removeFurniture)
  const setEditMode = useWorldEditorStore((state) => state.setEditMode)
  const selectObject = useWorldEditorStore((state) => state.selectObject)

  // Get furniture config
  const config = furniture ? FURNITURE_CONFIGS[furniture.type] : null
  const seats = config?.seats ?? []
  const hasSeat = seats.length > 0

  // Find which seats are occupied
  const occupiedSeats = useMemo(() => {
    const occupied = new Set<number>()
    seatedMembers.forEach((seat) => {
      if (seat.furnitureId === furniture?.id) {
        occupied.add(seat.seatIndex)
      }
    })
    return occupied
  }, [seatedMembers, furniture?.id])

  // Find available seats
  const availableSeats = useMemo(() => {
    return seats.map((_, index) => index).filter((index) => !occupiedSeats.has(index))
  }, [seats, occupiedSeats])

  // Members who are not seated anywhere
  const unseatedMembers = useMemo(() => {
    const seatedMemberIds = new Set(seatedMembers.keys())
    return members.filter((m) => !seatedMemberIds.has(m.id))
  }, [members, seatedMembers])

  // Members seated on this furniture
  const membersOnThisFurniture = useMemo(() => {
    const result: Array<{ member: Member; seatIndex: number }> = []
    seatedMembers.forEach((seat, memberId) => {
      if (seat.furnitureId === furniture?.id) {
        const member = members.find((m) => m.id === memberId)
        if (member) {
          result.push({ member, seatIndex: seat.seatIndex })
        }
      }
    })
    return result
  }, [seatedMembers, furniture?.id, members])

  const handleMove = useCallback(() => {
    if (!furniture) return
    setEditMode(true)
    selectObject({ id: furniture.id, type: 'furniture' })
    onClose()
  }, [furniture, setEditMode, selectObject, onClose])

  const handleDelete = useCallback(() => {
    if (!furniture) return
    // First unsit any members on this furniture
    membersOnThisFurniture.forEach(({ member }) => {
      onUnsitMember(member.id)
    })
    removeFurniture(furniture.id)
    onClose()
  }, [furniture, membersOnThisFurniture, onUnsitMember, removeFurniture, onClose])

  const handleSitMember = useCallback(
    (memberId: string) => {
      const firstSeat = availableSeats[0]
      if (!furniture || firstSeat === undefined) return
      // Sit on the first available seat
      onSitMember(memberId, furniture.id, firstSeat)
    },
    [furniture, availableSeats, onSitMember]
  )

  const handleUnsitMember = useCallback(
    (memberId: string) => {
      onUnsitMember(memberId)
    },
    [onUnsitMember]
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

      {/* Seated Members */}
      {membersOnThisFurniture.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5">Seated</div>
          <div className="space-y-1">
            {membersOnThisFurniture.map(({ member, seatIndex }) => (
              <div
                key={member.id}
                className="flex items-center justify-between p-1.5 bg-slate-700/50 rounded"
              >
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: member.bodyColor }}
                  />
                  <span className="text-xs text-slate-300">{member.name}</span>
                  <span className="text-xs text-slate-500">Seat {seatIndex + 1}</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleUnsitMember(member.id)}
                  className="h-5 px-1 text-xs text-slate-400 hover:text-slate-200"
                >
                  Stand
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Sit Member Section */}
      {hasSeat && availableSeats.length > 0 && unseatedMembers.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5 flex items-center gap-1">
            <UserPlus className="h-3 w-3" />
            Sit Member ({availableSeats.length} seat{availableSeats.length !== 1 ? 's' : ''} available)
          </div>
          <div className="max-h-32 overflow-y-auto space-y-1">
            {unseatedMembers.map((member) => (
              <button
                key={member.id}
                onClick={() => handleSitMember(member.id)}
                className="
                  w-full flex items-center gap-2 p-1.5
                  bg-slate-700/30 hover:bg-slate-600/50
                  rounded transition-colors
                "
              >
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: member.bodyColor }}
                />
                <span className="text-xs text-slate-300">{member.name}</span>
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
