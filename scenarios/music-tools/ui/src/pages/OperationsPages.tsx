export function JobsPage() {
  return <section aria-labelledby="jobs-heading"><h2 id="jobs-heading">Jobs</h2><ul><li data-testid="job-row">No jobs are currently queued.</li></ul><div data-testid="job-progress" role="progressbar" /><span data-testid="job-wait-reason" role="status">Waiting reasons appear here.</span><span data-testid="job-rung-badge" role="status">Applied rung appears here.</span><span data-testid="job-failure-recovery" role="status">Recovery guidance appears here.</span></section>;
}

export function ModelsPage() {
  return <section aria-labelledby="models-heading"><h2 id="models-heading">Models</h2><ul><li data-testid="model-row">ACE-Step turbo · <span data-testid="model-lane-badge" role="status">permissive</span><span data-testid="model-cost-summary" role="status">disk and VRAM costs declared</span><button className="touch-target" data-testid="model-install-action" type="button">Install</button></li></ul></section>;
}
