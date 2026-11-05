import { render } from '@/test-utils';
import { describe, it } from 'vitest';
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { FormCheckbox } from './FormCheckbox';

describe('FormCheckbox accessibility', () => {
  it('has no axe violations for its labeled interactive control', async () => {
    const { container } = render(
      <FormCheckbox checked={false} label="Enable recording" onChange={() => {}} />
    );

    await expectNoA11yViolations(container);
  });
});
