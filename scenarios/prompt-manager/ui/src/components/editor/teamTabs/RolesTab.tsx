/**
 * RolesTab - Team role management tab.
 *
 * Features:
 * - List defined roles
 * - Add/edit/delete roles
 * - Role name and description editing
 */

import { useState, useCallback } from 'react'
import { Plus, X, Edit2, Check, Shield } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, TeamRole } from '@/types/team'

interface RolesTabProps {
  team: TeamDetails
  onSetRoles: (roles: TeamRole[]) => Promise<TeamRole[]>
}

/**
 * Roles management tab component.
 */
export function RolesTab({ team, onSetRoles }: RolesTabProps) {
  const [editingRoleId, setEditingRoleId] = useState<string | null>(null)
  const [newRole, setNewRole] = useState<{ name: string; description: string } | null>(null)
  const [editedName, setEditedName] = useState('')
  const [editedDescription, setEditedDescription] = useState('')

  // Start editing a role
  const handleStartEdit = useCallback((role: TeamRole) => {
    setEditingRoleId(role.id)
    setEditedName(role.name)
    setEditedDescription(role.description ?? '')
    setNewRole(null)
  }, [])

  // Save edited role
  const handleSaveEdit = useCallback(async () => {
    if (!editingRoleId || !editedName.trim()) return

    const updatedRoles = team.roles.map((role) =>
      role.id === editingRoleId
        ? { ...role, name: editedName.trim(), description: editedDescription.trim() || undefined }
        : role
    )
    await onSetRoles(updatedRoles)
    setEditingRoleId(null)
  }, [editingRoleId, editedName, editedDescription, team.roles, onSetRoles])

  // Cancel editing
  const handleCancelEdit = useCallback(() => {
    setEditingRoleId(null)
    setNewRole(null)
  }, [])

  // Delete a role
  const handleDeleteRole = useCallback(
    async (roleId: string) => {
      const updatedRoles = team.roles.filter((role) => role.id !== roleId)
      await onSetRoles(updatedRoles)
    },
    [team.roles, onSetRoles]
  )

  // Start creating a new role
  const handleStartNewRole = useCallback(() => {
    setEditingRoleId(null)
    setNewRole({ name: '', description: '' })
  }, [])

  // Save new role
  const handleSaveNewRole = useCallback(async () => {
    if (!newRole || !newRole.name.trim()) return

    const roleId = newRole.name.toLowerCase().replace(/\s+/g, '-')
    const updatedRoles = [
      ...team.roles,
      {
        id: roleId,
        name: newRole.name.trim(),
        description: newRole.description.trim() || undefined,
      },
    ]
    await onSetRoles(updatedRoles)
    setNewRole(null)
  }, [newRole, team.roles, onSetRoles])

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">
          Team Roles ({team.roles.length})
        </h3>
        <button
          type="button"
          onClick={handleStartNewRole}
          disabled={newRole !== null}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
            'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          <Plus className="h-3.5 w-3.5" />
          Add Role
        </button>
      </div>

      {/* Roles List */}
      {team.roles.length === 0 && !newRole ? (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <Shield className="h-10 w-10 text-muted-foreground/50 mb-3" />
          <p className="text-sm text-muted-foreground">No roles defined</p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Create roles to organize team member permissions
          </p>
        </div>
      ) : (
        <ul className="space-y-2">
          {/* Existing roles */}
          {team.roles.map((role) => (
            <li
              key={role.id}
              className={cn(
                'flex items-start gap-3 px-3 py-3',
                'bg-muted rounded-lg group',
                'hover:bg-muted/80 transition-colors'
              )}
            >
              {editingRoleId === role.id ? (
                /* Edit mode */
                <div className="flex-1 space-y-2">
                  <input
                    type="text"
                    value={editedName}
                    onChange={(e) => setEditedName(e.target.value)}
                    placeholder="Role name"
                    className={cn(
                      'w-full px-2 py-1 text-sm',
                      'bg-background border border-border rounded',
                      'focus:outline-none focus:ring-2 focus:ring-primary'
                    )}
                    autoFocus
                  />
                  <input
                    type="text"
                    value={editedDescription}
                    onChange={(e) => setEditedDescription(e.target.value)}
                    placeholder="Description (optional)"
                    className={cn(
                      'w-full px-2 py-1 text-xs',
                      'bg-background border border-border rounded',
                      'focus:outline-none focus:ring-2 focus:ring-primary'
                    )}
                  />
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => void handleSaveEdit()}
                      disabled={!editedName.trim()}
                      className={cn(
                        'flex items-center gap-1 px-2 py-1 text-xs rounded',
                        'bg-primary text-primary-foreground hover:bg-primary/90',
                        'disabled:opacity-50 disabled:cursor-not-allowed'
                      )}
                    >
                      <Check className="h-3 w-3" />
                      Save
                    </button>
                    <button
                      type="button"
                      onClick={handleCancelEdit}
                      className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                /* View mode */
                <>
                  <Shield className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{role.name}</p>
                    {role.description && (
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {role.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      type="button"
                      onClick={() => handleStartEdit(role)}
                      className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-background"
                      title="Edit role"
                    >
                      <Edit2 className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleDeleteRole(role.id)}
                      className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                      title="Delete role"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </>
              )}
            </li>
          ))}

          {/* New role form */}
          {newRole && (
            <li className="flex items-start gap-3 px-3 py-3 bg-primary/5 border border-primary/20 rounded-lg">
              <Shield className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
              <div className="flex-1 space-y-2">
                <input
                  type="text"
                  value={newRole.name}
                  onChange={(e) => setNewRole({ ...newRole, name: e.target.value })}
                  placeholder="Role name"
                  className={cn(
                    'w-full px-2 py-1 text-sm',
                    'bg-background border border-border rounded',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                  autoFocus
                />
                <input
                  type="text"
                  value={newRole.description}
                  onChange={(e) => setNewRole({ ...newRole, description: e.target.value })}
                  placeholder="Description (optional)"
                  className={cn(
                    'w-full px-2 py-1 text-xs',
                    'bg-background border border-border rounded',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                />
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => void handleSaveNewRole()}
                    disabled={!newRole.name.trim()}
                    className={cn(
                      'flex items-center gap-1 px-2 py-1 text-xs rounded',
                      'bg-primary text-primary-foreground hover:bg-primary/90',
                      'disabled:opacity-50 disabled:cursor-not-allowed'
                    )}
                  >
                    <Check className="h-3 w-3" />
                    Create
                  </button>
                  <button
                    type="button"
                    onClick={handleCancelEdit}
                    className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </li>
          )}
        </ul>
      )}
    </div>
  )
}
