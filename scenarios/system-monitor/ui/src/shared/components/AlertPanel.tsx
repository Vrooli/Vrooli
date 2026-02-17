import type { Alert } from '../../types';

interface AlertPanelProps {
  alerts: Alert[];
}

const severityColor = (severity: string): string => {
  switch (severity) {
    case 'critical': return 'var(--color-error)';
    case 'high': return 'var(--color-warning)';
    default: return 'var(--color-info)';
  }
};

export const AlertPanel = ({ alerts }: AlertPanelProps) => {
  return (
    <section className="alert-panel card">
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">ACTIVE ALERTS</h2>
        <span
          className="alert-badge"
          style={{
            background: alerts.length > 0 ? 'var(--color-error)' : 'var(--color-success)',
            fontSize: 'var(--font-size-sm)'
          }}
        >
          {alerts.length}
        </span>
      </div>

      <div className="alert-list">
        {alerts.length === 0 ? (
          <div className="text-center text-muted p-lg text-lg">
            NO ACTIVE ALERTS
          </div>
        ) : (
          alerts.map(alert => (
            <div key={alert.id} className="alert-item pool-item mb-sm" style={{
              border: '1px solid var(--color-accent)',
              flexDirection: 'column',
              alignItems: 'stretch'
            }}>
              <div className="flex-row-between">
                <span style={{ color: severityColor(alert.severity) }}>
                  [{alert.severity.toUpperCase()}] {alert.category}
                </span>
                <span className="text-muted text-sm">
                  {new Date(alert.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div style={{ marginTop: 'var(--spacing-xs)' }}>
                {alert.message}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
};