import { describe, expect, it } from 'vitest';
import App from './App';
import { renderWithProviders } from './test-utils/renderWithProviders';
import { expectNoA11yViolations } from './test-utils/a11y';

describe('application shell accessibility', () => {
  it('has no detectable Axe violations on its initial route', async () => {
    const { container } = renderWithProviders(<App />);
    expect(container).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
