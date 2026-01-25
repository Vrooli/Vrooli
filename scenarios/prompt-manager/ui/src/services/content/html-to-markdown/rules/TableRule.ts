/**
 * TableRule - Custom Turndown rule for GFM table support.
 *
 * Converts HTML tables back to GitHub Flavored Markdown tables.
 * Handles alignment, cell content, and edge cases.
 */

import TurndownService from 'turndown'

// Create a minimal Turndown instance for processing cell content
// This preserves inline formatting like `code`, **bold**, *italic*, etc.
let cellTurndown: TurndownService | null = null

function getCellTurndown(): TurndownService {
  if (!cellTurndown) {
    cellTurndown = new TurndownService({
      headingStyle: 'atx',
      codeBlockStyle: 'fenced',
    })
    // Disable escaping to prevent double-escaping
    cellTurndown.escape = (text: string): string => text
  }
  return cellTurndown
}

/**
 * Extract content from an element, preserving inline formatting.
 */
function getCellContent(cell: Element): string {
  // Use innerHTML to preserve formatting, then convert back to markdown
  const html = cell.innerHTML
  if (!html) return ''

  // Convert cell HTML to markdown
  const markdown = getCellTurndown().turndown(html)

  // Clean up for table cell: escape pipes, remove newlines
  return markdown
    .replace(/\|/g, '\\|')
    .replace(/\n/g, ' ')
    .trim()
}

/**
 * Get text alignment from a cell element.
 */
function getCellAlignment(cell: Element): 'left' | 'center' | 'right' | null {
  const style = cell.getAttribute('style') || ''
  const align = cell.getAttribute('align')

  if (style.includes('text-align: center') || align === 'center') return 'center'
  if (style.includes('text-align: right') || align === 'right') return 'right'
  if (style.includes('text-align: left') || align === 'left') return 'left'
  return null
}

/**
 * Create the separator row for a markdown table.
 */
function createSeparatorRow(
  columnCount: number,
  alignments: (string | null)[]
): string {
  const separators: string[] = []

  for (let i = 0; i < columnCount; i++) {
    const align = alignments[i]
    if (align === 'center') {
      separators.push(':---:')
    } else if (align === 'right') {
      separators.push('---:')
    } else if (align === 'left') {
      separators.push(':---')
    } else {
      separators.push('---')
    }
  }

  return '| ' + separators.join(' | ') + ' |'
}

/**
 * Create the table rule for Turndown.
 */
export function createTableRule(): TurndownService.Rule {
  return {
    filter: 'table',
    replacement: function (_content: string, node: Node): string {
      const table = node as HTMLTableElement
      const rows: string[] = []
      let columnCount = 0
      const alignments: (string | null)[] = []
      let hasHeader = false

      // Process thead for header row
      const thead = table.querySelector('thead')
      if (thead) {
        const headerRow = thead.querySelector('tr')
        if (headerRow) {
          const cells = Array.from(headerRow.querySelectorAll('th, td'))
          const headerCells: string[] = []

          cells.forEach((cell) => {
            headerCells.push(getCellContent(cell))
            alignments.push(getCellAlignment(cell))
          })

          columnCount = headerCells.length
          rows.push('| ' + headerCells.join(' | ') + ' |')
          rows.push(createSeparatorRow(columnCount, alignments))
          hasHeader = true
        }
      }

      // Process tbody for data rows
      const tbody = table.querySelector('tbody') || table
      const dataRows = Array.from(tbody.querySelectorAll('tr'))

      dataRows.forEach((row, index) => {
        // Skip if this row is in thead (already processed)
        if (row.closest('thead')) return

        const cells = Array.from(row.querySelectorAll('td, th'))
        const rowCells: string[] = []

        cells.forEach((cell) => {
          rowCells.push(getCellContent(cell))
          // If no header, get alignments from first row
          if (!hasHeader && index === 0) {
            alignments.push(getCellAlignment(cell))
          }
        })

        // If this is the first row and no header, create a pseudo-header
        if (!hasHeader && index === 0) {
          columnCount = rowCells.length
          rows.push('| ' + rowCells.join(' | ') + ' |')
          rows.push(createSeparatorRow(columnCount, alignments))
          hasHeader = true
        } else {
          rows.push('| ' + rowCells.join(' | ') + ' |')
        }
      })

      // If we have no rows, return empty
      if (rows.length === 0) return ''

      // Return the table with surrounding newlines for proper markdown block
      return '\n\n' + rows.join('\n') + '\n\n'
    },
  }
}
