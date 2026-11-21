import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { FooterSection } from './FooterSection';

describe('FooterSection', () => {
  it('renders default company info and the three default link columns', () => {
    render(<FooterSection content={{}} />);
    expect(screen.getByRole('heading', { name: 'Landing Page' })).toBeInTheDocument();
    expect(screen.getByText('Product')).toBeInTheDocument();
    expect(screen.getByText('Company')).toBeInTheDocument();
    expect(screen.getByText('Legal')).toBeInTheDocument();
    // No social links by default.
    expect(screen.queryByRole('link', { name: 'GitHub' })).not.toBeInTheDocument();
  });

  it('renders custom company name, tagline, columns, and copyright', () => {
    render(
      <FooterSection
        content={{
          company_name: 'Acme',
          tagline: 'We ship',
          copyright: '© 2099 Acme',
          columns: [{ title: 'Resources', links: [{ label: 'Docs', url: '/docs' }] }],
        }}
      />,
    );
    expect(screen.getByRole('heading', { name: 'Acme' })).toBeInTheDocument();
    expect(screen.getByText('We ship')).toBeInTheDocument();
    expect(screen.getByText('Resources')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Docs' })).toHaveAttribute('href', '/docs');
    expect(screen.getByText('© 2099 Acme')).toBeInTheDocument();
  });

  it('renders every social icon and opens external links in a new tab', () => {
    render(
      <FooterSection
        content={{
          social_links: {
            github: 'https://github.com/acme',
            twitter: 'https://x.com/acme',
            linkedin: 'https://linkedin.com/acme',
            email: 'hi@acme.com',
          },
        }}
      />,
    );
    const github = screen.getByRole('link', { name: 'GitHub' });
    expect(github).toHaveAttribute('target', '_blank');
    expect(github).toHaveAttribute('href', 'https://github.com/acme');
    // Email uses a mailto: link and therefore no target.
    const email = screen.getByRole('link', { name: 'Email' });
    expect(email).toHaveAttribute('href', 'mailto:hi@acme.com');
    expect(email).not.toHaveAttribute('target');
  });
});
