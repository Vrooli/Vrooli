import clsx from 'clsx';
import {
  formatCurrency,
  formatRelativeTime,
  SaasTypeLabels,
  confidenceBand,
  scenarioLabel,
  type SaaSScenario,
} from '../utils/formatters';

export interface ScenarioTableProps {
  scenarios: SaaSScenario[];
}

export type { SaaSScenario };

export const ScenarioTable = ({ scenarios }: ScenarioTableProps) => (
  <div className="table-wrapper">
    <table className="scenario-table">
      <thead>
        <tr>
          <th scope="col">Scenario</th>
          <th scope="col">Type</th>
          <th scope="col">Industry</th>
          <th scope="col">Confidence</th>
          <th scope="col">Opportunity</th>
          <th scope="col">Landing page</th>
          <th scope="col">Last scan</th>
        </tr>
      </thead>
      <tbody>
        {scenarios.length === 0 && (
          <tr>
            <td colSpan={7} className="empty-state">
              No scenarios match your filters yet.
            </td>
          </tr>
        )}
        {scenarios.map((scenario) => {
          const confidence = confidenceBand(scenario.confidence_score);
          return (
            <tr key={scenario.id}>
              <td data-label="Scenario">
                <div className="scenario-name">{scenarioLabel(scenario)}</div>
                <div className="scenario-id">{scenario.scenario_name}</div>
              </td>
              <td data-label="Type">{(SaasTypeLabels[scenario.saas_type] ?? scenario.saas_type) || 'SaaS'}</td>
              <td data-label="Industry">{scenario.industry || 'Unspecified'}</td>
              <td data-label="Confidence">
                <span className={clsx('confidence-pill', confidence.toLowerCase())}>{confidence}</span>
              </td>
              <td data-label="Opportunity">{formatCurrency(scenario.revenue_potential)}</td>
              <td data-label="Landing page">
                {scenario.has_landing_page ? (
                  <a href={scenario.landing_page_url || '#'} target="_blank" rel="noreferrer">
                    View
                  </a>
                ) : (
                  <span className="status-pill">Pending</span>
                )}
              </td>
              <td data-label="Last scan">{formatRelativeTime(scenario.last_scan)}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  </div>
);
