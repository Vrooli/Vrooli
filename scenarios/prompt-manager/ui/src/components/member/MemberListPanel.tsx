/**
 * AvatarListPanel - Panel for listing and managing avatars.
 */

import { Plus, User, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAvatarData } from '@/hooks/useAvatarData'
import { DEFAULT_AVATAR_COLORS } from '@/types/avatar'

interface AvatarListPanelProps {
  selectedAvatarId: string | null
  onSelectAvatar: (id: string) => void
  onCreateAvatar: () => void
  onDeleteAvatar: (id: string) => void
  className?: string
}

/**
 * Avatar list panel for the sidebar.
 */
export function AvatarListPanel({
  selectedAvatarId,
  onSelectAvatar,
  onCreateAvatar,
  onDeleteAvatar,
  className,
}: AvatarListPanelProps) {
  const { avatars, isLoading, isError, createAvatar } = useAvatarData()

  const handleCreateAvatar = async () => {
    const name = `Avatar ${avatars.length + 1}`
    await createAvatar({
      name,
      ...DEFAULT_AVATAR_COLORS,
      skills: [],
    })
    onCreateAvatar()
  }

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <p className="text-sm text-destructive">Failed to load avatars</p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* Avatar list */}
      <div className="flex-1 overflow-y-auto py-1">
        {avatars.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <User className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No avatars yet</p>
            <button
              type="button"
              onClick={() => void handleCreateAvatar()}
              className="text-xs text-primary hover:underline"
            >
              Create your first avatar
            </button>
          </div>
        ) : (
          avatars.map((avatar) => (
            <button
              key={avatar.id}
              type="button"
              onClick={() => onSelectAvatar(avatar.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedAvatarId === avatar.id && 'bg-primary/10'
              )}
            >
              {/* Avatar preview */}
              <div
                className="w-8 h-8 rounded-full flex-shrink-0 flex items-center justify-center"
                style={{ backgroundColor: avatar.bodyColor }}
              >
                <div
                  className="w-4 h-4 rounded-full"
                  style={{ backgroundColor: avatar.headColor }}
                />
              </div>

              {/* Avatar info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {avatar.name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {avatar.skills.length} skill{avatar.skills.length !== 1 ? 's' : ''}
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDeleteAvatar(avatar.id)
                  }}
                  className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                  title="Delete avatar"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer - New avatar button */}
      <div className="flex-shrink-0 px-3 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleCreateAvatar()}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
          )}
        >
          <Plus className="h-4 w-4" />
          New Avatar
        </button>
      </div>
    </div>
  )
}
