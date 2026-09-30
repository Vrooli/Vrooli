import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import type { PublicBranding } from '../../../shared/api/types';
import { createFeedback } from '../../../shared/api/feedback';
import { loadSiteBranding } from './siteBrandingSource';
import { ContactPage } from './ContactPage';
import { LegalPage, NotFoundPage, ThankYouPage } from './SitePages';
import { Markdown } from './Markdown';
import { DEFAULT_PRIVACY_MARKDOWN, DEFAULT_TERMS_MARKDOWN, LEGAL_TOKENS, renderLegalTokens } from './legalTemplates';
import { validateContact } from './contactValidation';

vi.mock('../../../shared/api/feedback', () => ({ createFeedback: vi.fn() }));

const branding: PublicBranding = {
  site_name: 'Northwind', legal_name: 'Northwind Studio LLC', support_email: 'hello@northwind.test',
  contact_address: '100 Main Street\nSuite 4\nSpringfield, ST 00000', canonical_base_url: 'https://northwind.test',
  privacy_effective_date: '2026-09-16',
};

function renderAt(path: string) {
  return render(<MemoryRouter initialEntries={[path]}>
    <Routes>
      <Route path="/contact" element={<ContactPage />} />
      <Route path="/thank-you" element={<ThankYouPage />} />
      <Route path="/privacy" element={<LegalPage kind="privacy" />} />
      <Route path="/terms" element={<LegalPage kind="terms" />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  </MemoryRouter>);
}

async function settle() {
  await act(async () => { await Promise.resolve(); });
}

beforeEach(() => {
  vi.mocked(createFeedback).mockReset();
  vi.mocked(loadSiteBranding).mockResolvedValue(branding);
  window.sessionStorage.clear();
  document.head.innerHTML = '';
});

describe('site chrome', () => {
  it.each([
    ['/contact', 'Contact · Northwind'],
    ['/privacy', 'Privacy Policy · Northwind'],
    ['/terms', 'Terms and Conditions · Northwind'],
    ['/thank-you', 'Thank you · Northwind'],
    ['/nope', 'Page not found · Northwind'],
  ])('gives %s its own title, description, social image and alt text on every image', async (path, title) => {
    renderAt(path);
    await settle();
    await waitFor(() => { expect(document.title).toBe(title); });
    expect(document.querySelector('meta[name="description"]')?.getAttribute('content')).toBeTruthy();
    expect(document.querySelector('meta[property="og:image"]')?.getAttribute('content')).toBe('https://northwind.test/public/og-image.jpg');
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    for (const image of document.querySelectorAll('img')) expect(image.hasAttribute('alt')).toBe(true);
  });

  it('shows the configured contact email and postal address in the footer', async () => {
    renderAt('/privacy');
    await settle();
    const footer = await screen.findByRole('contentinfo');
    expect(footer).toHaveTextContent('hello@northwind.test');
    expect(footer).toHaveTextContent('100 Main Street');
    expect(footer).toHaveTextContent('Springfield, ST 00000');
    expect(screen.getAllByRole('link', { name: 'Privacy' })[0]).toHaveAttribute('href', '/privacy');
    expect(screen.getAllByRole('link', { name: 'Terms' })[0]).toHaveAttribute('href', '/terms');
  });

  it('never shows an address or email that Branding does not configure', async () => {
    vi.mocked(loadSiteBranding).mockResolvedValue({ site_name: 'Northwind' });
    renderAt('/contact');
    await settle();
    await screen.findByRole('heading', { name: 'How can we help?' });
    expect(screen.queryByText(/@/)).toBeNull();
    expect(screen.queryByText('Mailing address')).toBeNull();
  });

  it('marks not-found and thank-you pages noindex but lets legal pages be indexed', async () => {
    const view = renderAt('/nope');
    await settle();
    await waitFor(() => { expect(document.querySelector('meta[name="robots"]')?.getAttribute('content')).toBe('noindex, nofollow'); });
    view.unmount();
    renderAt('/terms');
    await settle();
    await waitFor(() => { expect(document.querySelector('link[rel="canonical"]')?.getAttribute('href')).toBe('https://northwind.test/terms'); });
    expect(document.querySelector('meta[name="robots"]')).toBeNull();
  });
});

describe('contact page', () => {
  function fill(values: { email?: string; subject?: string; message?: string }) {
    if (values.email !== undefined) fireEvent.change(screen.getByLabelText('Your email'), { target: { value: values.email } });
    if (values.subject !== undefined) fireEvent.change(screen.getByLabelText('Subject'), { target: { value: values.subject } });
    if (values.message !== undefined) fireEvent.change(screen.getByLabelText('Message'), { target: { value: values.message } });
  }

  it('explains every invalid field, links each error to its input, and focuses the first one without sending', async () => {
    renderAt('/contact');
    await settle();
    fill({ email: 'not-an-email', message: 'short' });
    fireEvent.click(screen.getByRole('button', { name: /send message/i }));

    const email = screen.getByLabelText('Your email');
    expect(email).toHaveAttribute('aria-invalid', 'true');
    expect(email).toHaveAccessibleDescription(/We only use this to reply\. Enter an email address like name@example\.com\./);
    expect(screen.getByLabelText('Subject')).toHaveAccessibleDescription('Add a short subject.');
    expect(screen.getByLabelText('Message')).toHaveAccessibleDescription(/at least 10 characters/);
    expect(email).toHaveFocus();
    expect(createFeedback).not.toHaveBeenCalled();

    fill({ email: 'buyer@example.com' });
    expect(email).toHaveAttribute('aria-invalid', 'false');
  });

  it('shows a sending state, then lands on the refund thank-you page', async () => {
    let resolve: (value: { success: boolean; id: number }) => void = () => undefined;
    vi.mocked(createFeedback).mockImplementation(() => new Promise(done => { resolve = done; }));
    renderAt('/contact?topic=refund');
    await settle();
    expect(screen.getByRole('radio', { name: /request a refund/i })).toBeChecked();
    fill({ email: ' buyer@example.com ', subject: 'Refund please', message: 'It was not the right fit for me.' });
    fireEvent.change(screen.getByLabelText(/order or subscription id/i), { target: { value: 'sub_123' } });
    fireEvent.click(screen.getByRole('button', { name: /send message/i }));

    const sending = await screen.findByRole('button', { name: /sending/i });
    expect(sending).toBeDisabled();
    expect(screen.getByLabelText('Your email')).toBeDisabled();
    expect(createFeedback).toHaveBeenCalledWith({ type: 'refund', email: 'buyer@example.com', subject: 'Refund please', message: 'It was not the right fit for me.', orderId: 'sub_123' }, expect.any(AbortSignal));

    await act(async () => { resolve({ success: true, id: 9 }); await Promise.resolve(); });
    expect(await screen.findByRole('heading', { name: 'Refund request received.' })).toBeInTheDocument();
  });

  it.each([
    [new ConnectError('email address is required', Code.InvalidArgument), 'Email address is required.'],
    [new ConnectError('down', Code.Unavailable), 'We could not reach the server. Check your connection and try again.'],
    [new ConnectError('boom', Code.Internal), 'Something went wrong on our side. Your message was not sent — please try again.'],
  ])('keeps the message and explains a failed send: %s', async (error, message) => {
    vi.mocked(createFeedback).mockRejectedValue(error);
    renderAt('/contact');
    await settle();
    fill({ email: 'buyer@example.com', subject: 'Hello', message: 'A question about plans.' });
    fireEvent.click(screen.getByRole('button', { name: /send message/i }));
    expect(await screen.findByRole('alert')).toHaveTextContent(message);
    expect(screen.getByLabelText('Message')).toHaveValue('A question about plans.');
    expect(screen.getByRole('button', { name: /send message/i })).toBeEnabled();
  });

  it('validates independently of the rendered form', () => {
    expect(validateContact({ email: 'a@b.co', subject: 'Hi', message: 'Long enough message' })).toEqual({});
    expect(Object.keys(validateContact({ email: '', subject: 'x'.repeat(201), message: '' }))).toEqual(['email', 'subject', 'message']);
  });
});

describe('thank-you page', () => {
  it.each([
    ['/thank-you?type=checkout', 'You are all set.'],
    ['/thank-you?type=message', 'Message received.'],
    ['/thank-you?type=unknown', 'Message received.'],
  ])('confirms %s', async (path, heading) => {
    renderAt(path);
    await settle();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(heading);
    const links = await screen.findAllByRole('link', { name: 'hello@northwind.test' });
    for (const link of links) expect(link).toHaveAttribute('href', 'mailto:hello@northwind.test');
  });
});

describe('legal pages', () => {
  it('fills the default privacy policy with the configured business identity and effective date', async () => {
    renderAt('/privacy');
    await settle();
    const article = await screen.findByRole('article');
    await waitFor(() => { expect(article).toHaveTextContent('Northwind Studio LLC ("we", "us", or "our")'); });
    expect(article).toHaveTextContent('northwind.test');
    expect(screen.getByText('September 16, 2026')).toHaveAttribute('dateTime', '2026-09-16');
    expect(article.textContent).not.toMatch(/\{\{/);
    expect(screen.getByRole('navigation', { name: 'On this page' })).toBeInTheDocument();
  });

  it('renders an operator-authored document instead of the template, as text rather than markup', async () => {
    vi.mocked(loadSiteBranding).mockResolvedValue({ ...branding, terms_markdown: '## Our terms\n\nWritten by {{business_name}}. <img src=x onerror=alert(1)>' });
    renderAt('/terms');
    await settle();
    expect(await screen.findByRole('heading', { name: 'Our terms', level: 2 })).toBeInTheDocument();
    expect(screen.getByText(/Written by Northwind Studio LLC\./)).toHaveTextContent('<img src=x onerror=alert(1)>');
    expect(document.querySelector('article img')).toBeNull();
  });
});

describe('markdown and templates', () => {
  it('renders headings, lists, emphasis, safe links and email addresses', () => {
    render(<Markdown source={'# Title\n\n- **Bold** item\n- [Site](https://example.test) and [bad](javascript:alert(1))\n\nWrite to help@example.test\nor call.'} />);
    expect(screen.getByRole('heading', { level: 2, name: 'Title' })).toHaveAttribute('id', 'title');
    expect(screen.getByText('Bold').tagName).toBe('STRONG');
    expect(screen.getByRole('link', { name: 'Site' })).toHaveAttribute('rel', 'noopener noreferrer');
    expect(screen.queryByRole('link', { name: 'bad' })).toBeNull();
    expect(screen.getByRole('link', { name: 'help@example.test' })).toHaveAttribute('href', 'mailto:help@example.test');
  });

  it('uses only known tokens and never leaks a placeholder when details are missing', () => {
    for (const template of [DEFAULT_PRIVACY_MARKDOWN, DEFAULT_TERMS_MARKDOWN]) {
      const tokens = [...template.matchAll(/\{\{\s*([a-z_]+)\s*\}\}/g)].map(match => match[1]);
      for (const token of tokens) expect(LEGAL_TOKENS).toContain(token);
      const rendered = renderLegalTokens(template, { businessName: 'Northwind', siteName: 'Northwind' });
      expect(rendered).not.toMatch(/\{\{|\n{3,}/);
      expect(rendered).toContain('the address on our contact page');
    }
  });
});
