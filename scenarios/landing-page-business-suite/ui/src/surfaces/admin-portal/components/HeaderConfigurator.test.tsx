import { cleanup, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { useState } from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import type { ContentSection, LandingHeaderConfig } from '../../../shared/api';
import { buildDefaultHeaderConfig } from '../../../shared/lib/headerConfig';
import { HeaderConfigurator } from './HeaderConfigurator';

const sections: ContentSection[] = [
  { id: 1, variant_id: 1, section_type: 'features', content: {}, order: 2, enabled: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 2, variant_id: 1, section_type: 'downloads', content: {}, order: 3, enabled: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
];

function Configurator({ initial = buildDefaultHeaderConfig('Acme') }: { initial?: LandingHeaderConfig }) {
  const [config, onChange] = useState(initial);
  return <HeaderConfigurator config={config} sections={sections} onChange={onChange} variantName="Acme" />;
}

afterEach(cleanup);

describe('HeaderConfigurator', () => {
  it('renders safe defaults and updates branding plus sticky behavior', () => {
    render(<Configurator />);
    expect(screen.getByText('No manual links added. The header will mirror section order automatically.')).toBeInTheDocument();
    const selects = screen.getAllByRole('combobox');
    fireEvent.change(selects[0]!, { target: { value: 'name' } });
    fireEvent.change(selects[1]!, { target: { value: 'stacked' } });
    fireEvent.change(screen.getByPlaceholderText('Header title'), { target: { value: 'Acme Desktop' } });
    fireEvent.change(screen.getByPlaceholderText('Optional tagline'), { target: { value: 'Ship faster' } });
    expect(screen.getByPlaceholderText('Header title')).toHaveValue('Acme Desktop');
    expect(screen.getByPlaceholderText('Optional tagline')).toHaveValue('Ship faster');

    const [sticky, hideOnScroll] = screen.getAllByRole('checkbox');
    expect(hideOnScroll).toBeEnabled();
    fireEvent.click(sticky!);
    expect(sticky).not.toBeChecked();
    expect(hideOnScroll).toBeDisabled();
  });

  it('adds section and menu navigation items, including editable child links', () => {
    render(<Configurator />);
    const selects = screen.getAllByRole('combobox');
    fireEvent.change(selects[2]!, { target: { value: JSON.stringify({ type: 'section', id: 1, section_type: 'features', order: 2 }) } });
    fireEvent.click(screen.getByRole('button', { name: 'Add link' }));
    expect(screen.getByDisplayValue('features')).toBeInTheDocument();
    expect(screen.getByText('Link to features')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add menu' }));
    expect(screen.getByText('Dropdown menu')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add item' }));
    const itemLabels = screen.getAllByDisplayValue('Menu item');
    fireEvent.change(itemLabels[itemLabels.length - 1]!, { target: { value: 'Documentation' } });
    expect(screen.getByDisplayValue('Documentation')).toBeInTheDocument();
  });

  it('supports downloads links, menu-child removal, and link removal', () => {
    render(<Configurator />);
    const navTarget = screen.getAllByRole('combobox')[2]!;
    fireEvent.change(navTarget, { target: { value: JSON.stringify({ type: 'downloads' }) } });
    fireEvent.click(screen.getByRole('button', { name: 'Add link' }));
    expect(screen.getByDisplayValue('Downloads')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add menu' }));
    fireEvent.click(screen.getByRole('button', { name: 'Add item' }));
    fireEvent.change(screen.getByDisplayValue('Menu item'), { target: { value: 'Support' } });
    fireEvent.change(screen.getAllByDisplayValue('#')[0]!, { target: { value: '/support' } });
    expect(screen.getAllByDisplayValue('/support')).not.toHaveLength(0);
    const removeButtons = screen.getAllByRole('button', { name: 'Remove' });
    fireEvent.click(removeButtons[0]!);
    expect(screen.getAllByRole('button', { name: 'Remove' })).toHaveLength(removeButtons.length - 1);
    fireEvent.click(screen.getAllByRole('button', { name: '×' })[0]!);
    expect(screen.queryAllByDisplayValue('Downloads')).toHaveLength(0);
  });

  it('exposes custom CTA fields and keeps mobile navigation visibility editable', () => {
    const initial = buildDefaultHeaderConfig('Acme');
    initial.nav.links = [{ id: 'docs', type: 'custom', label: 'Docs', href: '/docs', visible_on: { desktop: true, mobile: true } }];
    render(<Configurator initial={initial} />);
    const selects = screen.getAllByRole('combobox');
    fireEvent.change(selects[3]!, { target: { value: 'custom' } });
    fireEvent.change(screen.getAllByPlaceholderText('Button label')[0]!, { target: { value: 'Start now' } });
    fireEvent.change(screen.getByPlaceholderText('https://example.com'), { target: { value: '/start' } });
    expect(screen.getByDisplayValue('Start now')).toBeInTheDocument();
    expect(screen.getByDisplayValue('/start')).toBeInTheDocument();

    const checkboxes = screen.getAllByRole('checkbox');
    fireEvent.click(checkboxes[1]!);
    expect(checkboxes[1]).not.toBeChecked();
  });

  it('reorders manual links and saves a custom secondary CTA', () => {
    const initial = buildDefaultHeaderConfig('Acme');
    initial.nav.links = [
      { id: 'first', type: 'custom', label: 'First', href: '/first', visible_on: { desktop: true, mobile: true } },
      { id: 'second', type: 'custom', label: 'Second', href: '/second', visible_on: { desktop: true, mobile: true } },
    ];
    render(<Configurator initial={initial} />);

    const arrows = screen.getAllByRole('button', { name: '↓' });
    fireEvent.click(arrows[0]!);
    expect(screen.getAllByDisplayValue(/First|Second/)[0]).toHaveValue('Second');
    fireEvent.click(screen.getAllByRole('button', { name: '↑' })[1]!);
    expect(screen.getAllByDisplayValue(/First|Second/)[0]).toHaveValue('First');

    fireEvent.change(screen.getAllByRole('combobox')[4]!, { target: { value: 'custom' } });
    const labels = screen.getAllByPlaceholderText('Button label');
    const urls = screen.getAllByPlaceholderText('https://example.com');
    fireEvent.change(labels[labels.length - 1]!, { target: { value: 'Contact sales' } });
    fireEvent.change(urls[urls.length - 1]!, { target: { value: '/contact' } });
    expect(screen.getByDisplayValue('Contact sales')).toBeInTheDocument();
    expect(screen.getByDisplayValue('/contact')).toBeInTheDocument();
  });
});
