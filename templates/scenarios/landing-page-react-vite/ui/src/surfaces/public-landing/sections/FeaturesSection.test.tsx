import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FeaturesSection } from './FeaturesSection';

describe('FeaturesSection', () => {
  it('falls back to default icon when variant specifies an unknown key', () => {
    render(
      <FeaturesSection
        content={{
          features: [
            {
              title: 'Unknown icon feature',
              description: 'Should render using fallback icon',
              icon: 'not_a_real_icon' as any,
            },
          ],
        }}
      />
    );

    expect(screen.getByText('Unknown icon feature')).toBeInTheDocument();
    expect(screen.getByText('Should render using fallback icon')).toBeInTheDocument();
  });
});
