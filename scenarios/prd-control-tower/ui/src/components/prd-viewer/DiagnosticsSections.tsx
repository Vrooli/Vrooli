import { ShieldAlert, ShieldCheck } from 'lucide-react'
import { DiagnosticsMetric } from './DiagnosticsMetric'
import { DiagnosticsViolationList } from './DiagnosticsViolationList'
import { DiagnosticsJsonPreview } from './DiagnosticsJsonPreview'
import { isRecord, getSectionResult, toNumber, toString } from './diagnosticsHelpers'

interface DiagnosticsSectionsProps {
  data: Record<string, unknown>
}

export function DiagnosticsSections({ data }: DiagnosticsSectionsProps) {
  const sections: Array<{
    id: 'security' | 'standards'
    section: Record<string, unknown>
  }> = []

  const securitySection = data['security']
  if (isRecord(securitySection)) {
    sections.push({ id: 'security', section: securitySection })
  }

  const standardsSection = data['standards']
  if (isRecord(standardsSection)) {
    sections.push({ id: 'standards', section: standardsSection })
  }

  if (sections.length === 0) {
    return <DiagnosticsJsonPreview value={data} />
  }

  return (
    <div className="diagnostics-cards">
      {sections.map(({ id, section }) => {
        const result = getSectionResult(section)
        const message = isRecord(section.status) ? toString(section.status.message) : toString(section.message)
        const icon = id === 'security' ? <ShieldCheck size={20} aria-hidden="true" /> : <ShieldAlert size={20} aria-hidden="true" />

        const statistics = result && isRecord(result.statistics) ? (result.statistics as Record<string, unknown>) : null
        const vulnerabilities = result && isRecord(result.vulnerabilities) ? (result.vulnerabilities as Record<string, unknown>) : null
        const violationsList = Array.isArray(result?.violations) ? (result?.violations as unknown[]) : []

        const metrics: Array<{ label: string; value: number | undefined }> = []

        if (id === 'security') {
          const total = vulnerabilities ? toNumber(vulnerabilities.total) : toNumber(statistics?.total_findings)
          metrics.push({ label: 'Total', value: total })

          const severitySource = vulnerabilities ?? (isRecord(statistics?.by_severity) ? (statistics?.by_severity as Record<string, unknown>) : null)
          const severities = ['critical', 'high', 'medium', 'low', 'info']
          severities.forEach((severity) => {
            const value = severitySource ? toNumber(severitySource[severity]) : undefined
            if (value !== undefined) {
              metrics.push({ label: severity.charAt(0).toUpperCase() + severity.slice(1), value })
            }
          })
        } else {
          const total = statistics ? toNumber(statistics.total) : undefined
          metrics.push({ label: 'Total', value: total })

          const severities = ['critical', 'high', 'medium', 'low']
          severities.forEach((severity) => {
            const value = statistics ? toNumber(statistics[severity]) : undefined
            if (value !== undefined) {
              metrics.push({ label: severity.charAt(0).toUpperCase() + severity.slice(1), value })
            }
          })
        }

        return (
          <div key={id} className="diagnostics-card">
            <div className="diagnostics-card__header">
              <span className={`diagnostics-card__icon diagnostics-card__icon--${id}`}>{icon}</span>
              <div>
                <h3 className="diagnostics-card__title">{id === 'security' ? 'Security Scan' : 'Standards Check'}</h3>
                {message && <p className="diagnostics-card__message">{message}</p>}
              </div>
            </div>

            {metrics.length > 0 && (
              <div className="diagnostics-card__metrics">
                {metrics.map((metric) => (
                  <DiagnosticsMetric key={`${id}-${metric.label}`} label={metric.label} value={metric.value} />
                ))}
              </div>
            )}

            {id === 'standards' && violationsList.length > 0 && (
              <div className="diagnostics-card__violations" aria-live="polite">
                <h4 className="diagnostics-card__subtitle">Most recent findings</h4>
                <DiagnosticsViolationList violations={violationsList} limit={5} />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
