import { DiffEditor } from '@monaco-editor/react'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { cn } from '../../lib/utils'

interface DiffViewerProps {
  /**
   * Original content (typically the published PRD)
   */
  original: string
  /**
   * Modified content (typically the draft content)
   */
  modified: string
  /**
   * Optional title for the diff viewer
   */
  title?: string
  /**
   * Optional className for styling
   */
  className?: string
  /**
   * Height of the diff editor (default: 600px)
   */
  height?: string | number
  /**
   * Language for syntax highlighting (default: 'markdown')
   */
  language?: string
}

/**
 * DiffViewer component displays a side-by-side diff between two text contents
 * using Monaco Editor's DiffEditor.
 *
 * Used primarily for previewing changes before publishing a draft to PRD.md
 */
export function DiffViewer({
  original,
  modified,
  title = 'Changes Preview',
  className,
  height = 600,
  language = 'markdown',
}: DiffViewerProps) {
  return (
    <Card className={cn('w-full', className)}>
      <CardHeader>
        <CardTitle className="text-lg">{title}</CardTitle>
        <p className="text-sm text-muted-foreground">
          <span className="font-medium text-red-600">Red</span> indicates removed content,{' '}
          <span className="font-medium text-green-600">green</span> indicates added content.
        </p>
      </CardHeader>
      <CardContent>
        <div className="rounded-md border overflow-hidden">
          <DiffEditor
            height={height}
            language={language}
            original={original}
            modified={modified}
            theme="vs-light"
            options={{
              readOnly: true,
              renderSideBySide: true,
              scrollBeyondLastLine: false,
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: 'on',
              wordWrap: 'on',
              folding: true,
              renderWhitespace: 'selection',
              scrollbar: {
                vertical: 'auto',
                horizontal: 'auto',
              },
            }}
          />
        </div>
      </CardContent>
    </Card>
  )
}
