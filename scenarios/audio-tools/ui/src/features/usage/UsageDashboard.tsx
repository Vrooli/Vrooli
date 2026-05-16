/**
 * Usage Dashboard — operator accounting surface.
 *
 * Renders recent operations, credits charged, provider distribution, and
 * fallback reasons. Backed by UsageService.ListRecent + GetSummary once those
 * handlers ship past their Unimplemented stubs.
 */
export function UsageDashboard(): JSX.Element {
  return (
    <section aria-label="Usage Dashboard" className="audio-tools-feature-usage">
      <header>
        <h2>Usage Dashboard</h2>
        <p>Recent operations, charged credits, provider distribution, fallback reasons.</p>
      </header>
      <CreditsCard />
      <ProviderDistributionChart />
      <RecentOperationsTable />
      <FallbackReasonsList />
    </section>
  );
}

function CreditsCard(): JSX.Element {
  return (
    <article aria-label="Credits">
      <h3>Credits charged</h3>
      <strong>0</strong>
      <span> last 24 h</span>
    </article>
  );
}

function ProviderDistributionChart(): JSX.Element {
  return (
    <article aria-label="Provider distribution">
      <h3>Provider distribution</h3>
      <ul>
        <li>local: —</li>
        <li>byok: —</li>
        <li>vrooli: — (flag-off)</li>
      </ul>
    </article>
  );
}

function RecentOperationsTable(): JSX.Element {
  return (
    <article aria-label="Recent operations">
      <h3>Recent operations</h3>
      <table>
        <thead>
          <tr>
            <th>When</th>
            <th>Capability</th>
            <th>Operation</th>
            <th>Tier</th>
            <th>Provider</th>
            <th>Latency</th>
            <th>Credits</th>
          </tr>
        </thead>
        <tbody>
          <tr><td colSpan={7}><em>No operations yet.</em></td></tr>
        </tbody>
      </table>
    </article>
  );
}

function FallbackReasonsList(): JSX.Element {
  return (
    <article aria-label="Fallback reasons">
      <h3>Fallback reasons</h3>
      <p>None recorded in this window.</p>
    </article>
  );
}
