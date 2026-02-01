/**
 * AgentCustomizeModal - Modal for customizing agent appearance.
 *
 * Features:
 * - Name editing
 * - Color pickers for body, head, and accent colors
 * - Live preview
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { X, Palette, Shirt, Crown, Briefcase } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Agent, UpdateAgentRequest } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { useAccessoryStore } from '@/stores/accessoryStore'
import type {
  HeadAccessoryType,
  ClothingTopType,
  ClothingBottomType,
  FootwearType,
} from '@/types/accessory'

interface AgentCustomizeModalProps {
  isOpen: boolean
  onClose: () => void
  agent: Agent | null
  onSave: (updates: UpdateAgentRequest) => Promise<void>
  isLoading?: boolean
}

/**
 * Color preset palette for quick selection.
 */
const COLOR_PRESETS = [
  '#6366f1', // indigo
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#06b6d4', // cyan
  '#3b82f6', // blue
  '#64748b', // slate
]

// Accessory options
const HEAD_ACCESSORIES: { type: HeadAccessoryType; icon: string; label: string }[] = [
  { type: 'none', icon: '❌', label: 'None' },
  { type: 'hat', icon: '🎩', label: 'Hat' },
  { type: 'glasses', icon: '👓', label: 'Glasses' },
  { type: 'crown', icon: '👑', label: 'Crown' },
  { type: 'headphones', icon: '🎧', label: 'Headphones' },
  { type: 'halo', icon: '😇', label: 'Halo' },
]

const CLOTHING_TOPS: { type: ClothingTopType; icon: string; label: string }[] = [
  { type: 'none', icon: '❌', label: 'None' },
  { type: 'tshirt', icon: '👕', label: 'T-Shirt' },
  { type: 'hoodie', icon: '🧥', label: 'Hoodie' },
  { type: 'jacket', icon: '🧥', label: 'Jacket' },
  { type: 'vest', icon: '🦺', label: 'Vest' },
  { type: 'dress', icon: '👗', label: 'Dress' },
]

const CLOTHING_BOTTOMS: { type: ClothingBottomType; icon: string; label: string }[] = [
  { type: 'none', icon: '❌', label: 'None' },
  { type: 'pants', icon: '👖', label: 'Pants' },
  { type: 'shorts', icon: '🩳', label: 'Shorts' },
  { type: 'skirt', icon: '👗', label: 'Skirt' },
]

const FOOTWEAR: { type: FootwearType; icon: string; label: string }[] = [
  { type: 'none', icon: '❌', label: 'None' },
  { type: 'shoes', icon: '👞', label: 'Shoes' },
  { type: 'boots', icon: '🥾', label: 'Boots' },
  { type: 'sneakers', icon: '👟', label: 'Sneakers' },
  { type: 'sandals', icon: '🩴', label: 'Sandals' },
]

/**
 * Agent customization modal component.
 */
export function AgentCustomizeModal({
  isOpen,
  onClose,
  agent,
  onSave,
  isLoading = false,
}: AgentCustomizeModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  // Form state
  const [displayName, setDisplayName] = useState('')
  const [bodyColor, setBodyColor] = useState<string>(DEFAULT_AGENT_COLORS.body)
  const [headColor, setHeadColor] = useState<string>(DEFAULT_AGENT_COLORS.head)
  const [accentColor, setAccentColor] = useState<string>(DEFAULT_AGENT_COLORS.accent)
  const [activeTab, setActiveTab] = useState<'colors' | 'accessories'>('colors')

  // Accessory state
  const [headAccessory, setHeadAccessory] = useState<HeadAccessoryType>('none')
  const [clothingTop, setClothingTop] = useState<ClothingTopType>('none')
  const [clothingBottom, setClothingBottom] = useState<ClothingBottomType>('none')
  const [footwear, setFootwear] = useState<FootwearType>('none')

  // Accessory store
  const getAgentAccessories = useAccessoryStore((state) => state.getAgentAccessories)
  const setAgentAccessories = useAccessoryStore((state) => state.setAgentAccessories)

  // Initialize form when agent changes
  useEffect(() => {
    if (agent) {
      setDisplayName(agent.displayName)
      setBodyColor(agent.appearance?.body ?? DEFAULT_AGENT_COLORS.body)
      setHeadColor(agent.appearance?.head ?? DEFAULT_AGENT_COLORS.head)
      setAccentColor(agent.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent)

      // Load accessories from store
      const accessories = getAgentAccessories(agent.id)
      setHeadAccessory(accessories.head?.type ?? 'none')
      setClothingTop(accessories.clothingTop?.type ?? 'none')
      setClothingBottom(accessories.clothingBottom?.type ?? 'none')
      setFootwear(accessories.footwear?.type ?? 'none')
    }
  }, [agent, getAgentAccessories])

  // Focus name input when opened
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => nameInputRef.current?.focus(), 0)
    }
  }, [isOpen])

  // Handle escape key
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isLoading) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Handle click outside
  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (
        dialogRef.current &&
        !dialogRef.current.contains(event.target as Node) &&
        !isLoading
      ) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Set up event listeners
  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown)
      document.addEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = ''
    }
  }, [isOpen, handleKeyDown, handleClickOutside])

  // Handle save
  const handleSave = async () => {
    if (!agent) return

    // Save accessories to store
    setAgentAccessories(agent.id, {
      head: { type: headAccessory },
      clothingTop: { type: clothingTop },
      clothingBottom: { type: clothingBottom },
      footwear: { type: footwear },
    })

    // Save agent data
    await onSave({
      displayName,
      appearance: {
        body: bodyColor,
        head: headColor,
        accent: accentColor,
      },
    })
    onClose()
  }

  if (!isOpen || !agent) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className={cn(
          'relative w-full max-w-lg mx-4 p-6',
          'bg-card border border-border rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="customize-dialog-title"
      >
        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-muted-foreground hover:text-foreground hover:bg-muted transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
            <Palette className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 id="customize-dialog-title" className="text-lg font-semibold text-foreground">
              Customize Agent
            </h2>
            <p className="text-sm text-muted-foreground">Personalize your agent's appearance</p>
          </div>
        </div>

        {/* Preview */}
        <div className="flex justify-center mb-4">
          <div className="relative">
            {/* Head accessory preview */}
            {headAccessory !== 'none' && (
              <span className="absolute -top-4 left-1/2 -translate-x-1/2 text-2xl">
                {HEAD_ACCESSORIES.find((a) => a.type === headAccessory)?.icon}
              </span>
            )}
            {/* Body */}
            <div
              className="w-20 h-28 rounded-full flex items-start justify-center pt-4"
              style={{ backgroundColor: bodyColor }}
            >
              {/* Head */}
              <div
                className="w-12 h-12 rounded-full flex items-center justify-center"
                style={{ backgroundColor: headColor }}
              >
                {/* Accent (antenna) */}
                <div
                  className="absolute -top-2 w-3 h-3 rounded-full"
                  style={{ backgroundColor: accentColor }}
                />
              </div>
            </div>
            {/* Arms */}
            <div
              className="absolute top-8 -left-3 w-4 h-12 rounded-full"
              style={{ backgroundColor: bodyColor }}
            />
            <div
              className="absolute top-8 -right-3 w-4 h-12 rounded-full"
              style={{ backgroundColor: bodyColor }}
            />
          </div>
        </div>

        {/* Name input */}
        <div className="mb-4">
          <label htmlFor="agent-name" className="block text-sm font-medium text-foreground mb-1">
            Name
          </label>
          <input
            ref={nameInputRef}
            id="agent-name"
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className={cn(
              'w-full px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
            placeholder="Enter agent name"
          />
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-4 border-b border-border">
          <button
            type="button"
            onClick={() => setActiveTab('colors')}
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 transition-colors',
              activeTab === 'colors'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
          >
            <Palette className="h-4 w-4" />
            Colors
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('accessories')}
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 transition-colors',
              activeTab === 'accessories'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
          >
            <Shirt className="h-4 w-4" />
            Accessories
          </button>
        </div>

        {/* Colors Tab */}
        {activeTab === 'colors' && (
          <div className="space-y-4">
            <ColorPicker
              label="Body Color"
              value={bodyColor}
              onChange={setBodyColor}
            />
            <ColorPicker
              label="Head Color"
              value={headColor}
              onChange={setHeadColor}
            />
            <ColorPicker
              label="Accent Color"
              value={accentColor}
              onChange={setAccentColor}
            />
          </div>
        )}

        {/* Accessories Tab */}
        {activeTab === 'accessories' && (
          <div className="space-y-4 max-h-[300px] overflow-y-auto pr-2">
            {/* Head Accessory */}
            <AccessoryPicker
              label="Head"
              icon={<Crown className="h-4 w-4" />}
              options={HEAD_ACCESSORIES}
              value={headAccessory}
              onChange={(type) => setHeadAccessory(type as HeadAccessoryType)}
            />

            {/* Clothing Top */}
            <AccessoryPicker
              label="Top"
              icon={<Shirt className="h-4 w-4" />}
              options={CLOTHING_TOPS}
              value={clothingTop}
              onChange={(type) => setClothingTop(type as ClothingTopType)}
            />

            {/* Clothing Bottom */}
            <AccessoryPicker
              label="Bottom"
              icon={<Briefcase className="h-4 w-4" />}
              options={CLOTHING_BOTTOMS}
              value={clothingBottom}
              onChange={(type) => setClothingBottom(type as ClothingBottomType)}
            />

            {/* Footwear */}
            <AccessoryPicker
              label="Footwear"
              icon={<span className="text-sm">👟</span>}
              options={FOOTWEAR}
              value={footwear}
              onChange={(type) => setFootwear(type as FootwearType)}
            />
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-3 mt-6">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
              'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground',
              'border border-border transition-colors',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={isLoading}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            {isLoading ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Color picker component with presets and custom input.
 */
interface ColorPickerProps {
  label: string
  value: string
  onChange: (color: string) => void
}

function ColorPicker({ label, value, onChange }: ColorPickerProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-foreground mb-2">
        {label}
      </label>
      <div className="flex items-center gap-3">
        {/* Color presets */}
        <div className="flex gap-1 flex-wrap">
          {COLOR_PRESETS.map((color) => (
            <button
              key={color}
              type="button"
              onClick={() => onChange(color)}
              className={cn(
                'w-6 h-6 rounded-full border-2 transition-all',
                value === color ? 'border-foreground scale-110' : 'border-transparent hover:scale-105'
              )}
              style={{ backgroundColor: color }}
              title={color}
            />
          ))}
        </div>

        {/* Custom color input */}
        <div className="flex items-center gap-2 ml-auto">
          <input
            type="color"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="w-8 h-8 rounded cursor-pointer border border-border"
          />
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className={cn(
              'w-20 px-2 py-1 text-xs font-mono',
              'bg-muted border border-border rounded',
              'text-foreground',
              'focus:outline-none focus:ring-1 focus:ring-primary'
            )}
          />
        </div>
      </div>
    </div>
  )
}

/**
 * Accessory picker component with emoji options.
 */
interface AccessoryPickerProps {
  label: string
  icon: React.ReactNode
  options: { type: string; icon: string; label: string }[]
  value: string
  onChange: (type: string) => void
}

function AccessoryPicker({ label, icon, options, value, onChange }: AccessoryPickerProps) {
  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <span className="text-muted-foreground">{icon}</span>
        <label className="text-sm font-medium text-foreground">{label}</label>
      </div>
      <div className="flex gap-1.5 flex-wrap">
        {options.map((option) => (
          <button
            key={option.type}
            type="button"
            onClick={() => onChange(option.type)}
            className={cn(
              'flex flex-col items-center justify-center',
              'w-12 h-12 rounded-lg border transition-all',
              value === option.type
                ? 'border-primary bg-primary/10 scale-105'
                : 'border-border hover:border-muted-foreground hover:bg-muted'
            )}
            title={option.label}
          >
            <span className="text-lg">{option.icon}</span>
            <span className="text-[10px] text-muted-foreground truncate max-w-full">
              {option.type === 'none' ? '' : option.label.slice(0, 5)}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}
