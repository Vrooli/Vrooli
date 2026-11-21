import { describe, it, expect } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { FAQSection } from './FAQSection';

describe('FAQSection', () => {
  it('renders the default FAQ list with the first item open', () => {
    render(<FAQSection content={{}} />);
    expect(screen.getByText('Guardrails teams ask about before launch')).toBeInTheDocument();
    expect(screen.getByTestId('faq-0')).toBeInTheDocument();
    // Five default questions.
    expect(screen.getAllByTestId(/^faq-question-/)).toHaveLength(5);
  });

  it('renders custom title, subtitle, and provided FAQs', () => {
    render(
      <FAQSection
        content={{
          title: 'Custom FAQ',
          subtitle: 'Answers here',
          faqs: [{ question: 'Q1?', answer: 'A1.' }],
        }}
      />,
    );
    expect(screen.getByText('Custom FAQ')).toBeInTheDocument();
    expect(screen.getByText('Answers here')).toBeInTheDocument();
    expect(screen.getByText('Q1?')).toBeInTheDocument();
    expect(screen.getAllByTestId(/^faq-question-/)).toHaveLength(1);
  });

  it('toggles an answer open and closed when its question is clicked', () => {
    render(<FAQSection content={{ faqs: [{ question: 'Open me', answer: 'Body' }] }} />);
    const answer = screen.getByTestId('faq-answer-0');
    // First item starts open.
    expect(answer.className).toContain('max-h-96');
    fireEvent.click(screen.getByTestId('faq-question-0'));
    expect(answer.className).toContain('max-h-0');
    fireEvent.click(screen.getByTestId('faq-question-0'));
    expect(answer.className).toContain('max-h-96');
  });
});
