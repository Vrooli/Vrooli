export interface MetricSourceFailure {
  source: string;
  message: string;
  hasPreviousData: boolean;
}

interface MetricSourceFailureNoticeProps {
  failures: MetricSourceFailure[];
  onRetry: () => void;
  isRetrying: boolean;
}

/** Makes failed metric refreshes visible on detail routes that retain last-good data. */
export function MetricSourceFailureNotice({ failures, onRetry, isRetrying }: MetricSourceFailureNoticeProps) {
  if (failures.length === 0) return null;

  return (
    <section className="card" role="alert" aria-labelledby="metric-source-failure-title">
      <h2 id="metric-source-failure-title" className="section-heading">Some readings could not be refreshed</h2>
      <ul>
        {failures.map(({ source, message, hasPreviousData }) => (
          <li key={source}>
            <strong>{source}:</strong> {message} {hasPreviousData
              ? 'The last received values remain visible.'
              : 'No successful reading is available yet.'}
          </li>
        ))}
      </ul>
      <button type="button" className="btn btn-action" onClick={onRetry} disabled={isRetrying}>
        {isRetrying ? 'Refreshing…' : 'Retry refresh'}
      </button>
    </section>
  );
}
