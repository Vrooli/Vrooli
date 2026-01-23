/**
 * MemberListPanel - Panel for listing and managing members.
 */

import { Plus, User, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useMemberData } from '@/hooks/useMemberData'
import { DEFAULT_MEMBER_COLORS } from '@/types/member'

interface MemberListPanelProps {
  selectedMemberId: string | null
  onSelectMember: (id: string) => void
  onCreateMember: () => void
  onDeleteMember: (id: string) => void
  className?: string
}

/**
 * Member list panel for the sidebar.
 */
export function MemberListPanel({
  selectedMemberId,
  onSelectMember,
  onCreateMember,
  onDeleteMember,
  className,
}: MemberListPanelProps) {
  const { members, isLoading, isError, createMember } = useMemberData()

  const handleCreateMember = async () => {
    const name = `Member ${members.length + 1}`
    await createMember({
      name,
      ...DEFAULT_MEMBER_COLORS,
      skills: [],
    })
    onCreateMember()
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
        <p className="text-sm text-destructive">Failed to load members</p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* Member list */}
      <div className="flex-1 overflow-y-auto py-1">
        {members.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <User className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No members yet</p>
            <button
              type="button"
              onClick={() => void handleCreateMember()}
              className="text-xs text-primary hover:underline"
            >
              Create your first member
            </button>
          </div>
        ) : (
          members.map((member) => (
            <button
              key={member.id}
              type="button"
              onClick={() => onSelectMember(member.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedMemberId === member.id && 'bg-primary/10'
              )}
            >
              {/* Member preview */}
              <div
                className="w-8 h-8 rounded-full flex-shrink-0 flex items-center justify-center"
                style={{ backgroundColor: member.bodyColor }}
              >
                <div
                  className="w-4 h-4 rounded-full"
                  style={{ backgroundColor: member.headColor }}
                />
              </div>

              {/* Member info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {member.name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {member.skills.length} skill{member.skills.length !== 1 ? 's' : ''}
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDeleteMember(member.id)
                  }}
                  className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                  title="Delete member"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer - New member button */}
      <div className="flex-shrink-0 px-3 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleCreateMember()}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
          )}
        >
          <Plus className="h-4 w-4" />
          New Member
        </button>
      </div>
    </div>
  )
}
