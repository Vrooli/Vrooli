/**
 * ObjectPalette - Panel for selecting objects to place in the world.
 * Shows furniture and decoration options organized by category.
 */

import { useCallback } from 'react'
import {
  Armchair,
  Lamp,
  Flower2,
  Frame,
  TreePine,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { FURNITURE_CONFIGS, type FurnitureType } from '@/types/furniture'
import { DECORATION_CONFIGS, type DecorationType } from '@/types/decoration'

interface ObjectPaletteProps {
  className?: string
}

// Furniture items with icons
const FURNITURE_ITEMS: { type: FurnitureType; icon: string }[] = [
  { type: 'chair', icon: '🪑' },
  { type: 'bench', icon: '🛋️' },
  { type: 'stool', icon: '🔵' },
  { type: 'armchair', icon: '🛋️' },
  { type: 'desk', icon: '🗄️' },
  { type: 'table', icon: '🪵' },
  { type: 'picnic-table', icon: '🏕️' },
  { type: 'coffee-table', icon: '☕' },
  { type: 'campfire', icon: '🔥' },
]

// Decoration items grouped by category
const DECORATION_CATEGORIES = {
  plants: ['potted-plant', 'tall-plant', 'cactus', 'flowers'] as DecorationType[],
  trees: ['oak-tree', 'pine-tree', 'birch-tree'] as DecorationType[],
  lighting: ['floor-lamp', 'desk-lamp', 'hanging-lamp'] as DecorationType[],
  furniture: ['bookshelf', 'rug'] as DecorationType[],
  decor: ['painting', 'vase', 'globe', 'clock', 'trophy'] as DecorationType[],
}

const DECORATION_ICONS: Record<string, string> = {
  'potted-plant': '🪴',
  'tall-plant': '🌿',
  cactus: '🌵',
  flowers: '🌸',
  'oak-tree': '🌳',
  'pine-tree': '🌲',
  'birch-tree': '🍃',
  'floor-lamp': '🪔',
  'desk-lamp': '💡',
  'hanging-lamp': '🔦',
  bookshelf: '📚',
  rug: '🟫',
  painting: '🖼️',
  vase: '🏺',
  globe: '🌍',
  clock: '🕐',
  trophy: '🏆',
}

/**
 * Panel for selecting and placing world objects.
 */
export function ObjectPalette({ className }: ObjectPaletteProps) {
  const isPaletteOpen = useWorldEditorStore((state) => state.isPaletteOpen)
  const setPaletteOpen = useWorldEditorStore((state) => state.setPaletteOpen)
  const paletteTab = useWorldEditorStore((state) => state.paletteTab)
  const setPaletteTab = useWorldEditorStore((state) => state.setPaletteTab)
  const placingObject = useWorldEditorStore((state) => state.placingObject)
  const cancelPlacing = useWorldEditorStore((state) => state.cancelPlacing)
  const startPlacing = useWorldEditorStore((state) => state.startPlacing)

  const handleFurnitureClick = useCallback(
    (type: FurnitureType) => {
      startPlacing({ type: 'furniture', subtype: type })
    },
    [startPlacing]
  )

  const handleDecorationClick = useCallback(
    (type: DecorationType) => {
      startPlacing({ type: 'decoration', subtype: type })
    },
    [startPlacing]
  )

  const handleClose = useCallback(() => {
    setPaletteOpen(false)
    if (placingObject) {
      cancelPlacing()
    }
  }, [setPaletteOpen, placingObject, cancelPlacing])

  if (!isPaletteOpen) {
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
        <h3 className="text-sm font-medium text-slate-200">Add Objects</h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleClose}
          className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setPaletteTab('furniture')}
          className={`flex-1 h-8 gap-1.5 ${
            paletteTab === 'furniture'
              ? 'bg-indigo-500/30 text-indigo-300'
              : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <Armchair className="h-3.5 w-3.5" />
          <span className="text-xs">Furniture</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setPaletteTab('decorations')}
          className={`flex-1 h-8 gap-1.5 ${
            paletteTab === 'decorations'
              ? 'bg-indigo-500/30 text-indigo-300'
              : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <Flower2 className="h-3.5 w-3.5" />
          <span className="text-xs">Decor</span>
        </Button>
      </div>

      {/* Furniture Tab */}
      {paletteTab === 'furniture' && (
        <div className="grid grid-cols-4 gap-1.5">
          {FURNITURE_ITEMS.map(({ type, icon }) => {
            const config = FURNITURE_CONFIGS[type]
            return (
              <button
                key={type}
                onClick={() => handleFurnitureClick(type)}
                className="
                  flex flex-col items-center justify-center
                  p-2 h-14
                  bg-slate-700/50 hover:bg-slate-600/50
                  border border-slate-600 hover:border-slate-500
                  rounded-md
                  transition-colors
                  group
                "
                title={config.displayName}
              >
                <span className="text-lg">{icon}</span>
                <span className="text-[10px] text-slate-400 group-hover:text-slate-300 truncate w-full text-center">
                  {config.displayName.split(' ')[0]}
                </span>
              </button>
            )
          })}
        </div>
      )}

      {/* Decorations Tab */}
      {paletteTab === 'decorations' && (
        <div className="space-y-3">
          {/* Plants */}
          <div>
            <div className="flex items-center gap-1.5 mb-1.5">
              <Flower2 className="h-3 w-3 text-green-400" />
              <span className="text-xs text-slate-400">Plants</span>
            </div>
            <div className="grid grid-cols-4 gap-1.5">
              {DECORATION_CATEGORIES.plants.map((type) => {
                const config = DECORATION_CONFIGS[type]
                return (
                  <button
                    key={type}
                    onClick={() => handleDecorationClick(type)}
                    className="
                      flex flex-col items-center justify-center
                      p-1.5 h-12
                      bg-slate-700/50 hover:bg-slate-600/50
                      border border-slate-600 hover:border-slate-500
                      rounded-md
                      transition-colors
                    "
                    title={config.displayName}
                  >
                    <span className="text-base">{DECORATION_ICONS[type]}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Trees */}
          <div>
            <div className="flex items-center gap-1.5 mb-1.5">
              <TreePine className="h-3 w-3 text-green-500" />
              <span className="text-xs text-slate-400">Trees</span>
            </div>
            <div className="grid grid-cols-4 gap-1.5">
              {DECORATION_CATEGORIES.trees.map((type) => {
                const config = DECORATION_CONFIGS[type]
                return (
                  <button
                    key={type}
                    onClick={() => handleDecorationClick(type)}
                    className="
                      flex flex-col items-center justify-center
                      p-1.5 h-12
                      bg-slate-700/50 hover:bg-slate-600/50
                      border border-slate-600 hover:border-slate-500
                      rounded-md
                      transition-colors
                    "
                    title={config.displayName}
                  >
                    <span className="text-base">{DECORATION_ICONS[type]}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Lighting */}
          <div>
            <div className="flex items-center gap-1.5 mb-1.5">
              <Lamp className="h-3 w-3 text-yellow-400" />
              <span className="text-xs text-slate-400">Lighting</span>
            </div>
            <div className="grid grid-cols-4 gap-1.5">
              {DECORATION_CATEGORIES.lighting.map((type) => {
                const config = DECORATION_CONFIGS[type]
                return (
                  <button
                    key={type}
                    onClick={() => handleDecorationClick(type)}
                    className="
                      flex flex-col items-center justify-center
                      p-1.5 h-12
                      bg-slate-700/50 hover:bg-slate-600/50
                      border border-slate-600 hover:border-slate-500
                      rounded-md
                      transition-colors
                    "
                    title={config.displayName}
                  >
                    <span className="text-base">{DECORATION_ICONS[type]}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Decor Items */}
          <div>
            <div className="flex items-center gap-1.5 mb-1.5">
              <Frame className="h-3 w-3 text-blue-400" />
              <span className="text-xs text-slate-400">Decor</span>
            </div>
            <div className="grid grid-cols-4 gap-1.5">
              {[...DECORATION_CATEGORIES.furniture, ...DECORATION_CATEGORIES.decor].map((type) => {
                const config = DECORATION_CONFIGS[type]
                return (
                  <button
                    key={type}
                    onClick={() => handleDecorationClick(type)}
                    className="
                      flex flex-col items-center justify-center
                      p-1.5 h-12
                      bg-slate-700/50 hover:bg-slate-600/50
                      border border-slate-600 hover:border-slate-500
                      rounded-md
                      transition-colors
                    "
                    title={config.displayName}
                  >
                    <span className="text-base">{DECORATION_ICONS[type]}</span>
                  </button>
                )
              })}
            </div>
          </div>
        </div>
      )}

      {/* Placement hint */}
      {placingObject && (
        <div className="mt-3 p-2 bg-green-500/10 border border-green-500/30 rounded text-xs text-green-300">
          Click in the world to place. Press Escape to cancel.
        </div>
      )}
    </div>
  )
}
