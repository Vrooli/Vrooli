import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { CTASection } from './CTASection';
import { FAQSection } from './FAQSection';
import { FooterSection } from './FooterSection';
import { TestimonialsSection } from './TestimonialsSection';

vi.mock('../../../shared/hooks/useMetricsHook', () => ({
  useMetrics: () => ({ trackCTAClick: vi.fn() }),
}));

describe('public landing conversion sections', () => {
  it('renders configured CTA copy while preserving a usable no-URL state', () => {
    renderWithProviders(<CTASection content={{ title: 'Ship today', subtitle: 'A reliable launch path', cta_text: 'Get started' }} />);
    expect(screen.getByRole('heading', { name: 'Ship today' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /get started/i })).toBeEnabled();
  });

  it('toggles FAQ answers and exposes the configured support destination safely', () => {
    renderWithProviders(<FAQSection content={{ faqs: [{ question: 'Can I cancel?', answer: 'Yes, whenever you need.' }, { question: 'Is support available?', answer: 'Yes, asynchronously.' }] }} supportChatUrl="https://support.example.test/chat" />);
    expect(screen.getByTestId('faq-answer-0')).toHaveClass('max-h-96');
    fireEvent.click(screen.getByTestId('faq-question-0'));
    expect(screen.getByTestId('faq-answer-0')).toHaveClass('max-h-0');
    fireEvent.click(screen.getByTestId('faq-question-1'));
    expect(screen.getByTestId('faq-answer-1')).toHaveClass('max-h-96');
    expect(screen.getByRole('link', { name: /ask our ai assistant/i })).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('renders custom footer links and external social links with safe opener isolation', () => {
    renderWithProviders(<FooterSection content={{ company_name: 'Business Suite', columns: [{ title: 'Product', links: [{ label: 'Pricing', url: '#pricing' }] }], social_links: { github: 'https://github.com/vrooli', email: 'support@example.com' } }} />);
    expect(screen.getByRole('heading', { name: 'Business Suite' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Pricing' })).toHaveAttribute('href', '#pricing');
    expect(screen.getByRole('link', { name: 'GitHub' })).toHaveAttribute('rel', 'noopener noreferrer');
    expect(screen.getByRole('link', { name: 'Email' })).toHaveAttribute('href', 'mailto:support@example.com');
  });

  it('renders testimonial ratings and accessible avatar fallbacks', () => {
    renderWithProviders(<TestimonialsSection content={{ testimonials: [{ name: 'Alex', role: 'Founder', company: 'Acme', content: 'The billing path is clear.', rating: 3, avatar_url: 'https://example.test/alex.png' }, { name: 'Bea', role: 'Operator', company: 'Beta', content: 'Reliable support.' }] }} />);
    expect(screen.getByTestId('testimonial-0').querySelectorAll('svg')).toHaveLength(4);
    expect(screen.getByRole('img', { name: 'Alex' })).toHaveAttribute('src', 'https://example.test/alex.png');
    expect(screen.getByText('B')).toBeInTheDocument();
  });
});
