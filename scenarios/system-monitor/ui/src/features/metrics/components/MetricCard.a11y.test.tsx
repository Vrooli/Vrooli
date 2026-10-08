import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { expectNoA11yViolations } from '@vrooli/api-base/testing';
import { MetricCard } from './MetricCard';

describe('MetricCard accessibility', () => {
  it('has no detectable violations when a metric is unavailable', async () => {
    // provider-free-exception: MetricCard is a pure presentational component;
    // the shared provider package currently ships a conflicting React major.
    const { container } = render(
      <MetricCard
        type="cpu"
        label="CPU"
        unit="%"
        isExpanded={false}
        onToggle={() => undefined}
        alertCount={0}
      />,
    );

    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'CPU: not measured');
    await expectNoA11yViolations(container);
  });
});
