/**
 * SeatEditorOverlay - HTML panel for precision seat editing.
 * Shows per-seat controls: X/Y/Z inputs, rotation slider, delete button.
 * Also has Add Seat and Done buttons.
 */

import { useCallback } from 'react'
import { Plus, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useWorldSeatsStore } from '@/stores/worldSeatsStore'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import type { SeatPosition } from '@/types/furniture'
import { FURNITURE_CONFIGS } from '@/types/furniture'

const SEAT_COLORS = ['#ef4444', '#3b82f6', '#22c55e', '#f59e0b', '#a855f7', '#ec4899']

export function SeatEditorOverlay() {
  const editingSeatType = useWorldEditorStore((s) => s.editingSeatType)
  const stopSeatEdit = useWorldEditorStore((s) => s.stopSeatEdit)
  const seats = useWorldSeatsStore((s) => editingSeatType ? s.seats[editingSeatType] ?? [] : [])
  const updateSeat = useWorldSeatsStore((s) => s.updateSeat)
  const addSeat = useWorldSeatsStore((s) => s.addSeat)
  const removeSeat = useWorldSeatsStore((s) => s.removeSeat)

  const handleFieldChange = useCallback(
    (index: number, field: 'x' | 'y' | 'z' | 'rotation', value: number) => {
      if (!editingSeatType) return
      const current = seats[index]
      if (!current) return

      if (field === 'rotation') {
        updateSeat(editingSeatType, index, { ...current, rotation: value })
      } else {
        const pos: [number, number, number] = [...current.position]
        const axisMap = { x: 0, y: 1, z: 2 } as const
        pos[axisMap[field]] = value
        updateSeat(editingSeatType, index, { ...current, position: pos })
      }
    },
    [editingSeatType, seats, updateSeat]
  )

  const handleAddSeat = useCallback(() => {
    if (!editingSeatType) return
    const newSeat: SeatPosition = { position: [0, 0.8, 0], rotation: 0 }
    addSeat(editingSeatType, newSeat)
  }, [editingSeatType, addSeat])

  const handleRemoveSeat = useCallback(
    (index: number) => {
      if (!editingSeatType) return
      removeSeat(editingSeatType, index)
    },
    [editingSeatType, removeSeat]
  )

  if (!editingSeatType) return null

  const config = FURNITURE_CONFIGS[editingSeatType]

  return (
    <div className="w-72 max-h-[70vh] overflow-y-auto p-3 bg-slate-800/95 backdrop-blur-sm border border-slate-700 rounded-lg shadow-xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-slate-200">
          Edit Seats - {config.displayName}
        </h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={stopSeatEdit}
          className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Seats list */}
      <div className="space-y-3">
        {seats.map((seat, index) => {
          const color = SEAT_COLORS[index % SEAT_COLORS.length]
          return (
            <div key={index} className="p-2 bg-slate-700/50 rounded border border-slate-600/50">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: color }}
                  />
                  <span className="text-xs font-medium text-slate-300">Seat {index + 1}</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleRemoveSeat(index)}
                  className="h-5 w-5 p-0 text-red-400 hover:text-red-300"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>

              {/* Position inputs */}
              <div className="grid grid-cols-3 gap-1.5 mb-2">
                {(['x', 'y', 'z'] as const).map((axis, ai) => (
                  <div key={axis}>
                    <label className="text-[10px] text-slate-500 uppercase">{axis}</label>
                    <input
                      type="number"
                      step={0.05}
                      value={seat.position[ai]}
                      onChange={(e) => handleFieldChange(index, axis, parseFloat(e.target.value) || 0)}
                      className="w-full px-1.5 py-1 text-xs bg-slate-800 border border-slate-600 rounded text-slate-200 focus:border-blue-500 focus:outline-none"
                    />
                  </div>
                ))}
              </div>

              {/* Rotation slider */}
              <div>
                <div className="flex items-center justify-between">
                  <label className="text-[10px] text-slate-500 uppercase">Rotation</label>
                  <span className="text-[10px] text-slate-400">
                    {Math.round((seat.rotation * 180) / Math.PI)}°
                  </span>
                </div>
                <input
                  type="range"
                  min={0}
                  max={6.283185}
                  step={0.01}
                  value={seat.rotation}
                  onChange={(e) => handleFieldChange(index, 'rotation', parseFloat(e.target.value))}
                  className="w-full h-1.5 accent-blue-500"
                />
              </div>
            </div>
          )
        })}
      </div>

      {/* Actions */}
      <div className="flex gap-2 mt-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleAddSeat}
          className="flex-1 h-8 gap-1.5 text-green-400 hover:text-green-300 hover:bg-green-500/20"
        >
          <Plus className="h-3.5 w-3.5" />
          <span className="text-xs">Add Seat</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={stopSeatEdit}
          className="flex-1 h-8 gap-1.5 text-blue-400 hover:text-blue-300 hover:bg-blue-500/20"
        >
          <span className="text-xs">Done</span>
        </Button>
      </div>
    </div>
  )
}
