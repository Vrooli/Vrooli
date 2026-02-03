/**
 * TeamCodeView - YAML code editor for team structure.
 *
 * Features:
 * - Monaco editor with YAML syntax highlighting
 * - Converts team data to human-readable YAML format
 * - Bi-directional sync: YAML edits update team state
 * - Schema validation for YAML structure
 */

import { useCallback, useMemo, useRef, useEffect } from 'react'
import Editor, { type OnMount, type OnChange } from '@monaco-editor/react'
import YAML from 'yaml'
import { Network } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails } from '@/types/team'
import type { OrgEdge } from '@/types/orgChart'

// ============================================================================
// Types
// ============================================================================

interface TeamCodeViewProps {
  team: TeamDetails
  edges: OrgEdge[]
  onTeamChange?: (updates: TeamYamlData) => void
  readOnly?: boolean
  onSwitchToGraph?: () => void
  className?: string
}

/**
 * YAML representation of team data for editing.
 */
export interface TeamYamlData {
  name: string
  mission?: string
  roles: Array<{
    id: string
    name: string
    description?: string
  }>
  members: Array<{
    id: string
    name: string
    roles: string[]
    status: string
    reportsTo?: string
  }>
}

// ============================================================================
// Conversion Functions
// ============================================================================

/**
 * Convert team data to YAML-friendly structure.
 */
function teamToYaml(team: TeamDetails, edges: OrgEdge[]): TeamYamlData {
  // Build manager lookup: reportId -> managerId
  const managerMap = new Map<string, string>()
  edges.forEach((edge) => {
    managerMap.set(edge.reportId, edge.managerId)
  })

  // Build member name lookup for readable reportsTo
  const memberNames = new Map<string, string>()
  team.members.forEach((m) => {
    memberNames.set(m.agentId, m.displayName)
  })

  // Build role name lookup
  const roleNames = new Map<string, string>()
  team.roles.forEach((r) => {
    roleNames.set(r.id, r.name)
  })

  return {
    name: team.displayName,
    mission: team.mission ?? undefined,
    roles: team.roles.map((role) => ({
      id: role.id,
      name: role.name,
      description: role.description,
    })),
    members: team.members.map((member) => {
      const managerId = managerMap.get(member.agentId)
      const managerName = managerId ? memberNames.get(managerId) : undefined

      return {
        id: member.agentId,
        name: member.displayName,
        roles: member.roles.map((roleId) => roleNames.get(roleId) ?? roleId),
        status: member.status,
        reportsTo: managerName,
      }
    }),
  }
}

/**
 * Convert YAML data back to team updates.
 * Returns null if parsing fails.
 */
function yamlToTeamUpdates(yamlContent: string): TeamYamlData | null {
  try {
    const parsed = YAML.parse(yamlContent) as unknown

    // Basic validation
    if (!parsed || typeof parsed !== 'object') {
      return null
    }

    const data = parsed as Record<string, unknown>

    // Validate required fields
    if (typeof data.name !== 'string') {
      return null
    }

    return {
      name: data.name,
      mission: typeof data.mission === 'string' ? data.mission : undefined,
      roles: Array.isArray(data.roles)
        ? data.roles.map((r: unknown) => {
            const role = r as Record<string, unknown>
            return {
              id: typeof role.id === 'string' ? role.id : '',
              name: typeof role.name === 'string' ? role.name : '',
              description: typeof role.description === 'string' ? role.description : undefined,
            }
          })
        : [],
      members: Array.isArray(data.members)
        ? data.members.map((m: unknown) => {
            const member = m as Record<string, unknown>
            return {
              id: typeof member.id === 'string' ? member.id : '',
              name: typeof member.name === 'string' ? member.name : '',
              roles: Array.isArray(member.roles)
                ? member.roles
                    .filter((r: unknown) => typeof r === 'string')
                    .map((r: unknown) => r as string)
                : [],
              status: typeof member.status === 'string' ? member.status : 'active',
              reportsTo: typeof member.reportsTo === 'string' ? member.reportsTo : undefined,
            }
          })
        : [],
    }
  } catch {
    return null
  }
}

// ============================================================================
// Component
// ============================================================================

export function TeamCodeView({
  team,
  edges,
  onTeamChange,
  readOnly = false,
  onSwitchToGraph,
  className,
}: TeamCodeViewProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null)

  // Convert team to YAML string
  const yamlContent = useMemo(() => {
    const yamlData = teamToYaml(team, edges)
    return YAML.stringify(yamlData, {
      indent: 2,
      lineWidth: 0, // Disable line wrapping
    })
  }, [team, edges])

  // Track if content was externally updated
  const lastExternalContent = useRef(yamlContent)

  // Update editor when external content changes
  useEffect(() => {
    if (editorRef.current && yamlContent !== lastExternalContent.current) {
      const model = editorRef.current.getModel()
      if (model) {
        const currentValue = model.getValue()
        // Only update if the YAML structure changed, not just formatting
        if (currentValue !== yamlContent) {
          const parsed = yamlToTeamUpdates(currentValue)
          const newParsed = yamlToTeamUpdates(yamlContent)
          // Compare parsed structures to avoid unnecessary updates
          if (JSON.stringify(parsed) !== JSON.stringify(newParsed)) {
            model.setValue(yamlContent)
          }
        }
      }
      lastExternalContent.current = yamlContent
    }
  }, [yamlContent])

  // Handle editor mount
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
  }, [])

  // Handle content change
  const handleChange: OnChange = useCallback(
    (value) => {
      if (!value || !onTeamChange) return

      const updates = yamlToTeamUpdates(value)
      if (updates) {
        onTeamChange(updates)
      }
    },
    [onTeamChange]
  )

  return (
    <div className={cn('h-full flex flex-col', className)}>
      {/* Header */}
      <div className="flex-shrink-0 flex items-center gap-2 px-3 py-1.5 bg-[#1e1e1e] border-b border-[#3c3c3c]">
        <span className="text-xs text-slate-400 font-medium">team.yaml</span>
        {readOnly && (
          <span className="text-xs text-slate-500 bg-slate-800 px-1.5 py-0.5 rounded">
            Read-only
          </span>
        )}
        {onSwitchToGraph && (
          <button
            type="button"
            onClick={onSwitchToGraph}
            className={cn(
              'ml-auto flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-md',
              'bg-slate-800 text-slate-200 border border-slate-700',
              'hover:bg-slate-700 transition-colors'
            )}
            title="Switch to graph view"
          >
            <Network className="h-3.5 w-3.5" />
            Graph View
          </button>
        )}
      </div>

      {/* Editor */}
      <div className="flex-1">
        <Editor
          height="100%"
          defaultLanguage="yaml"
          value={yamlContent}
          onChange={handleChange}
          onMount={handleEditorMount}
          theme="vs-dark"
          options={{
            readOnly,
            minimap: { enabled: false },
            wordWrap: 'on',
            lineNumbers: 'on',
            fontSize: 13,
            fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
            tabSize: 2,
            scrollBeyondLastLine: false,
            padding: { top: 12, bottom: 12 },
            renderLineHighlight: 'line',
            cursorBlinking: 'smooth',
            smoothScrolling: true,
            scrollbar: {
              vertical: 'auto',
              horizontal: 'auto',
              verticalScrollbarSize: 8,
              horizontalScrollbarSize: 8,
            },
            overviewRulerBorder: false,
            hideCursorInOverviewRuler: true,
            folding: true,
            foldingStrategy: 'indentation',
            automaticLayout: true,
          }}
        />
      </div>
    </div>
  )
}
