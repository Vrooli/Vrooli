import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, screen } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { RevisionPreview } from './RevisionPreview';
import { previewFixture } from './testFixtures';

afterEach(cleanup);
describe('authorized native revision preview', () => {
  it('decodes inline display into the same native page and leaves owner transactions disconnected', () => {
    const value = previewFixture();
    value.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: 'purchase', status: 'ready', href: '/purchase', reason: '', appKey: '', planRef: 'configured-plan' }];
    const { container } = render(<RevisionPreview value={value} />);
    expect(screen.getByRole('heading', { name: 'Configured preview heading' })).toBeInTheDocument();
    expect(container.querySelector('.presentation-page')).toHaveAttribute('data-presentation-mode', 'empty');
    expect(screen.getByRole('button', { name: 'Configured action' })).toBeDisabled();
    expect(container.querySelector('iframe')).toBeNull();
  });
  it('fails closed without display and does not disclose configured private source', () => {
    const value = previewFixture();
    if (!value.page) throw new Error('Missing fixture page');
    value.page.display = undefined; value.page.title = 'PRIVATE-EXCEPTION-DETAIL';
    render(<RevisionPreview value={value} />);
    expect(screen.getByRole('alert')).toHaveTextContent('Check its configured display mapping');
    expect(document.body).not.toHaveTextContent('PRIVATE-EXCEPTION-DETAIL');
    expect(document.querySelector('.presentation-page')).toBeNull();
  });
  it('rejects public responses at the private preview boundary', () => {
    const value = previewFixture();
    if (!value.diagnostics) throw new Error('Missing fixture diagnostics');
    value.diagnostics.preview = false;
    render(<RevisionPreview value={value} />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(document.querySelector('.presentation-page')).toBeNull();
  });
});
