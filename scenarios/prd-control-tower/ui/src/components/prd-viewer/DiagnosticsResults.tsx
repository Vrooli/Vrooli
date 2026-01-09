import { AlertTriangle } from 'lucide-react'
import { DiagnosticsViolationList } from './DiagnosticsViolationList'
import { DiagnosticsSections } from './DiagnosticsSections'
import { isRecord, toString } from './diagnosticsHelpers'

interface DiagnosticsResultsProps {
  data: unknown
}

export function DiagnosticsResults({ data }: DiagnosticsResultsProps) {
  if (data === null || data === undefined) {
    return <p className="diagnostics-empty">Diagnostics completed without returning additional details.</p>
  }

  if (Array.isArray(data)) {
    return <DiagnosticsViolationList violations={data} />
  }

  if (isRecord(data)) {
    if (toString(data.error) || toString(data.message)) {
      return (
        <div className="diagnostics-error" role="alert">
          <AlertTriangle size={20} aria-hidden="true" />
          <div>
            <p className="diagnostics-error__title">Diagnostic tool reported an issue</p>
            <p>{toString(data.message) ?? toString(data.error)}</p>
          </div>
        </div>
      )
    }

    return <DiagnosticsSections data={data} />
  }

  return (
    <pre className="diagnostics-pre" role="region" aria-label="Diagnostics output">
      {String(data)}
    </pre>
  )
}
