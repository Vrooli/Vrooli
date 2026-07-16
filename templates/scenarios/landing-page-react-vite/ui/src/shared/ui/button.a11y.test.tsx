/**
 * Baseline accessibility regression test.
 *
 * The Button primitive is the smallest stable render target that exercises the
 * axe harness end-to-end. UI Health treats this file as the generated a11y
 * contract; add surface- and route-level a11y tests as coverage needs grow.
 */
import { afterEach, describe, it } from 'vitest';
import { cleanup } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils';

import { expectNoA11yViolations } from '../../test-utils/a11y';
import { Button } from './button';

describe('Button accessibility', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders without axe violations', async () => {
    const { container } = render(<Button>Continue</Button>);
    await expectNoA11yViolations(container);
  });
});
