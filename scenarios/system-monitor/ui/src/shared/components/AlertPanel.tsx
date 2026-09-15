import type { Alert } from '../../types';
import { formatTime } from '../utils/formatters';

interface AlertPanelProps {
  alerts: Alert[];
}

export const AlertPanel = ({ alerts }: AlertPanelProps) => {
  return (
    <section className="alert-panel card">
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">ACTIVE ALERTS</h2>
        <span
          className={`alert-badge text-sm ${alerts.length > 0 ? 'alert-badge-error' : 'alert-badge-success'}`}
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
            <div key={alert.id} className="alert-item pool-item mb-sm" data-sm-style="sm-style-2f44229022">
              <div className="flex-row-between">
                <span className={`status-color-${alert.severity.toLowerCase()}`}>
                  [{alert.severity.toUpperCase()}] {alert.category}
                </span>
                <span className="text-muted text-sm">
                  {formatTime(alert.timestamp)}
                </span>
              </div>
              <div data-sm-style="sm-style-4edd7007ae">
                {alert.message}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
};
