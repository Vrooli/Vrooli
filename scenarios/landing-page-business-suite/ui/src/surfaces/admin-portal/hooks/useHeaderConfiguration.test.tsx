import { act, renderHook } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { ContentSection, LandingHeaderConfig } from '../../../shared/api';
import { buildDefaultHeaderConfig } from '../../../shared/lib/headerConfig';
import { useHeaderConfiguration } from './useHeaderConfiguration';

const sections: ContentSection[] = [
  { id: 7, variant_id: 1, section_type: 'features', content: {}, order: 2, enabled: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 8, variant_id: 1, section_type: 'downloads', content: {}, order: 3, enabled: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
];

function renderConfiguration(initial = buildDefaultHeaderConfig('Acme')) {
  return renderHook(() => {
    const [config, onChange] = useState<LandingHeaderConfig>(initial);
    return { config, hook: useHeaderConfiguration({ config, sections, onChange }) };
  });
}

describe('useHeaderConfiguration', () => {
  it('adds real section and download navigation targets, while ignoring malformed targets', () => {
    const { result } = renderConfiguration();
    expect(result.current.hook.downloadsSection).toBe(true);

    act(() => { result.current.hook.setNavTarget(JSON.stringify({ type: 'section', id: 7 })); });
    act(() => { result.current.hook.handleAddLink(); });
    expect(result.current.config.nav.links[0]).toMatchObject({ type: 'section', section_id: 7, label: 'features' });

    act(() => { result.current.hook.setNavTarget(JSON.stringify({ type: 'downloads' })); });
    act(() => { result.current.hook.handleAddLink(); });
    expect(result.current.config.nav.links[1]).toMatchObject({ type: 'downloads', label: 'Downloads' });

    act(() => { result.current.hook.setNavTarget('{not-json'); });
    act(() => { result.current.hook.handleAddLink(); });
    expect(result.current.config.nav.links).toHaveLength(2);
  });

  it('edits and reorders navigation without mutating the prior configuration', () => {
    const initial = buildDefaultHeaderConfig('Acme');
    initial.nav.links = [
      { id: 'first', type: 'custom', label: 'First', href: '#first', visible_on: { desktop: true, mobile: true } },
      { id: 'second', type: 'custom', label: 'Second', href: '#second', visible_on: { desktop: true, mobile: true } },
    ];
    const { result } = renderConfiguration(initial);

    act(() => { result.current.hook.handleNavLabelChange(0, 'Overview'); });
    act(() => { result.current.hook.handleVisibilityToggle(0, 'mobile', false); });
    act(() => { result.current.hook.handleMoveLink(0, 1); });

    expect(initial.nav.links[0]?.label).toBe('First');
    expect(result.current.config.nav.links.map((link) => link.label)).toEqual(['Second', 'Overview']);
    expect(result.current.config.nav.links[1]?.visible_on).toEqual({ desktop: true, mobile: false });
  });

  it('manages menu children and CTA modes defensively', () => {
    const { result } = renderConfiguration();
    act(() => { result.current.hook.handleAddMenu(); });
    expect(result.current.config.nav.links[0]?.children).toHaveLength(2);

    act(() => { result.current.hook.handleAddMenuChild(0); });
    act(() => { result.current.hook.handleMenuChildChange(0, 2, 'label', 'Documentation'); });
    act(() => { result.current.hook.handleMenuChildChange(0, 2, 'href', '/docs'); });
    expect(result.current.config.nav.links[0]?.children?.[2]).toMatchObject({ label: 'Documentation', href: '/docs' });

    act(() => { result.current.hook.handleRemoveMenuChild(0, 1); });
    act(() => { result.current.hook.handleCTAModeChange('primary', { mode: 'custom', label: 'Start trial', href: '/start' }); });
    expect(result.current.config.nav.links[0]?.children).toHaveLength(2);
    expect(result.current.config.ctas.primary).toMatchObject({ mode: 'custom', label: 'Start trial', href: '/start' });
  });
});
