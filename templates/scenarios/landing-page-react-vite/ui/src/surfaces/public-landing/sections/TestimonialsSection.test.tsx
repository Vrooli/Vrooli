import { describe, it, expect } from 'vitest';
import { screen, within } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { TestimonialsSection } from './TestimonialsSection';

describe('TestimonialsSection', () => {
  it('renders the three default testimonials with rating stars', () => {
    render(<TestimonialsSection content={{}} />);
    expect(screen.getByText('Proof from the factory floor')).toBeInTheDocument();
    expect(screen.getAllByTestId(/^testimonial-/)).toHaveLength(3);
    expect(screen.getByText('Sarah Johnson')).toBeInTheDocument();
    expect(screen.getByText(/Marketing Director at TechCorp/)).toBeInTheDocument();
  });

  it('renders a custom testimonial with an avatar when provided', () => {
    render(
      <TestimonialsSection
        content={{
          title: 'What they say',
          testimonials: [
            {
              name: 'Ada Lovelace',
              role: 'Engineer',
              company: 'Analytical',
              content: 'Superb tooling.',
              avatar_url: 'https://cdn/ada.png',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('What they say')).toBeInTheDocument();
    const card = screen.getByTestId('testimonial-0');
    expect(within(card).getByRole('img', { name: 'Ada Lovelace' })).toHaveAttribute('src', 'https://cdn/ada.png');
  });

  it('falls back to an initial avatar when no image is given', () => {
    render(
      <TestimonialsSection
        content={{ testimonials: [{ name: 'Zoe', role: 'PM', company: 'Co', content: 'Nice' }] }}
      />,
    );
    const card = screen.getByTestId('testimonial-0');
    expect(within(card).queryByRole('img')).not.toBeInTheDocument();
    expect(within(card).getByText('Z')).toBeInTheDocument();
  });
});
