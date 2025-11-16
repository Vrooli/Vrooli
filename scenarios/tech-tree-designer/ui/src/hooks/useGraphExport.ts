import { useState, useCallback } from 'react'
import { exportGraphDOT, exportGraphJSON } from '../services/techTree'
import type { Sector, StageDependency } from '../types/techTree'

export type ExportFormat = 'dot' | 'json' | 'text'

interface ExportOptions {
  format: ExportFormat
  treeId?: string
  sectors?: Sector[]
  dependencies?: StageDependency[]
}

/**
 * Hook for exporting tech tree graphs in various formats.
 * Supports DOT (Graphviz), JSON, and plain text formats.
 */
export const useGraphExport = () => {
  const [isExporting, setIsExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

  const downloadFile = useCallback((content: string, filename: string, mimeType: string) => {
    const blob = new Blob([content], { type: mimeType })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [])

  const copyToClipboard = useCallback(async (content: string) => {
    try {
      await navigator.clipboard.writeText(content)
      return true
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
      return false
    }
  }, [])

  const generateTextExport = useCallback((sectors: Sector[], dependencies: StageDependency[]): string => {
    const lines: string[] = []
    lines.push('# Tech Tree Export')
    lines.push(`Generated: ${new Date().toISOString()}`)
    lines.push('')

    for (const sector of sectors) {
      lines.push(`## Sector: ${sector.name}`)
      lines.push(`Category: ${sector.category}`)
      lines.push(`Progress: ${(sector.progress_percentage ?? 0).toFixed(1)}%`)
      lines.push('')

      if (sector.stages && sector.stages.length > 0) {
        lines.push('### Stages:')
        for (const stage of sector.stages) {
          lines.push(`- ${stage.name} (${stage.stage_type}, ${(stage.progress_percentage ?? 0).toFixed(1)}%)`)
          if (stage.description) {
            lines.push(`  ${stage.description}`)
          }

          // Show scenario mappings
          if (stage.scenario_mappings && stage.scenario_mappings.length > 0) {
            lines.push('  Linked scenarios:')
            for (const mapping of stage.scenario_mappings) {
              lines.push(`    - ${mapping.scenario_name} (${mapping.completion_status ?? 'unknown'})`)
            }
          }
        }
        lines.push('')
      }
    }

    if (dependencies.length > 0) {
      lines.push('## Dependencies')
      for (const depPayload of dependencies) {
        const dep = depPayload.dependency
        lines.push(`- Prerequisite: ${depPayload.prerequisite_name ?? dep.prerequisite_stage_id}`)
        lines.push(`  Dependent: ${depPayload.dependent_name ?? dep.dependent_stage_id}`)
        lines.push(`  Type: ${dep.dependency_type ?? 'required'}, Strength: ${(dep.dependency_strength ?? 1).toFixed(2)}`)
        if (dep.description) {
          lines.push(`  ${dep.description}`)
        }
        lines.push('')
      }
    }

    return lines.join('\n')
  }, [])

  const exportGraph = useCallback(async (options: ExportOptions) => {
    setIsExporting(true)
    setExportError(null)

    try {
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)

      switch (options.format) {
        case 'dot': {
          const dotContent = await exportGraphDOT(options.treeId)
          downloadFile(dotContent, `tech-tree-${timestamp}.dot`, 'text/vnd.graphviz')
          break
        }

        case 'json': {
          const jsonData = await exportGraphJSON(options.treeId)
          const jsonContent = JSON.stringify(jsonData, null, 2)
          downloadFile(jsonContent, `tech-tree-${timestamp}.json`, 'application/json')
          break
        }

        case 'text': {
          // Use provided data or fetch it
          let sectors = options.sectors
          let dependencies = options.dependencies

          if (!sectors || !dependencies) {
            const jsonData = await exportGraphJSON(options.treeId)
            sectors = jsonData.sectors
            dependencies = jsonData.dependencies
          }

          const textContent = generateTextExport(sectors, dependencies)
          downloadFile(textContent, `tech-tree-${timestamp}.txt`, 'text/plain')
          break
        }

        default:
          throw new Error(`Unsupported export format: ${options.format}`)
      }

      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Export failed'
      setExportError(errorMessage)
      console.error('Export error:', err)
      return false
    } finally {
      setIsExporting(false)
    }
  }, [downloadFile, generateTextExport])

  const copyGraphAsText = useCallback(async (sectors: Sector[], dependencies: StageDependency[]) => {
    setIsExporting(true)
    setExportError(null)

    try {
      const textContent = generateTextExport(sectors, dependencies)
      const success = await copyToClipboard(textContent)

      if (!success) {
        setExportError('Failed to copy to clipboard')
      }

      return success
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Copy failed'
      setExportError(errorMessage)
      console.error('Copy error:', err)
      return false
    } finally {
      setIsExporting(false)
    }
  }, [generateTextExport, copyToClipboard])

  return {
    exportGraph,
    copyGraphAsText,
    isExporting,
    exportError,
    clearError: () => setExportError(null)
  }
}
