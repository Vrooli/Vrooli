interface DiagnosticsMetricProps {
  label: string
  value: number | undefined
}

export function DiagnosticsMetric({ label, value }: DiagnosticsMetricProps) {
  return (
    <div className="diagnostics-metric">
      <span className="diagnostics-metric__label">{label}</span>
      <span className="diagnostics-metric__value">{value ?? 0}</span>
    </div>
  )
}
