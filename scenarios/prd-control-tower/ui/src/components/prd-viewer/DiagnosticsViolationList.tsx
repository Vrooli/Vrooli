import { isRecord, toString } from './diagnosticsHelpers'

interface DiagnosticsViolationListProps {
  violations: unknown[]
  limit?: number
}

export function DiagnosticsViolationList({ violations, limit }: DiagnosticsViolationListProps) {
  const items = violations
    .filter((violation): violation is Record<string, unknown> => isRecord(violation))
    .slice(0, typeof limit === 'number' ? limit : violations.length)

  if (items.length === 0) {
    return <p className="diagnostics-empty">No violations reported in this category.</p>
  }

  return (
    <div className="diagnostics-list">
      {items.map((violation, index) => {
        const key = toString(violation.id) ?? `${index}`
        const title = toString(violation.title) ?? toString(violation.rule) ?? 'Unnamed rule'
        const message = toString(violation.description) ?? toString(violation.message)
        const filePath = toString(violation.file_path)
        const standard = toString(violation.standard)
        const severity = (toString(violation.severity) ?? toString(violation.level))?.toLowerCase()

        return (
          <div key={key} className="diagnostics-list__item">
            <div className="diagnostics-list__header">
              <h3 className="diagnostics-list__title">{title}</h3>
              {severity && (
                <span className={`severity-badge severity-badge--${severity}`}>{severity}</span>
              )}
            </div>
            {message && <p className="diagnostics-list__message">{message}</p>}
            {(filePath || standard) && (
              <div className="diagnostics-list__details">
                {filePath && (
                  <span className="diagnostics-list__path" aria-label="File path">
                    {filePath}
                  </span>
                )}
                {standard && (
                  <span className="diagnostics-list__standard">Standard: {standard}</span>
                )}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
