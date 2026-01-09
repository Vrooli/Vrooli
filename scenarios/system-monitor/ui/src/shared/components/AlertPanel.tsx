import type { Alert } from '../../types';

interface AlertPanelProps {
  alerts: Alert[];
}

export const AlertPanel = ({ alerts }: AlertPanelProps) => {
  return (
    <section className="alert-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>ACTIVE ALERTS</h2>
        <span className="alert-count" style={{
          background: alerts.length > 0 ? 'var(--color-error)' : 'var(--color-success)',
          color: 'var(--color-background)',
          padding: 'var(--spacing-xs) var(--spacing-sm)',
          borderRadius: '50px',
          fontSize: 'var(--font-size-sm)',
          fontWeight: 'bold'
        }}>
          {alerts.length}
        </span>
      </div>
      
      <div className="alert-list">
        {alerts.length === 0 ? (
          <div className="no-alerts" style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO ACTIVE ALERTS
          </div>
        ) : (
          alerts.map(alert => (
            <div key={alert.id} className="alert-item" style={{
              padding: 'var(--spacing-md)',
              border: '1px solid var(--color-accent)',
              borderRadius: 'var(--border-radius-md)',
              marginBottom: 'var(--spacing-sm)',
              background: 'rgba(0, 0, 0, 0.5)'
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ 
                  color: alert.severity === 'critical' ? 'var(--color-error)' :
                         alert.severity === 'high' ? 'var(--color-warning)' :
                         'var(--color-info)'
                }}>
                  [{alert.severity.toUpperCase()}] {alert.category}
                </span>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  {new Date(alert.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div style={{ marginTop: 'var(--spacing-xs)', color: 'var(--color-text)' }}>
                {alert.message}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
};