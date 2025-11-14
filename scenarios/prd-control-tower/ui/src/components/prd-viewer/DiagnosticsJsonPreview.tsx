interface DiagnosticsJsonPreviewProps {
  value: Record<string, unknown>
}

export function DiagnosticsJsonPreview({ value }: DiagnosticsJsonPreviewProps) {
  return (
    <pre className="diagnostics-pre" role="region" aria-label="Diagnostics output">
      {JSON.stringify(value, null, 2)}
    </pre>
  )
}
