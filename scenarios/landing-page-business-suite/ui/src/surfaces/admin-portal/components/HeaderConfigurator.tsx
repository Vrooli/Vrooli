import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import type {
  ContentSection,
  LandingHeaderConfig,
  HeaderCTAMode,
} from '../../../shared/api';
import { cloneHeaderConfig } from '../../../shared/lib/headerConfig';
import { DOWNLOAD_ANCHOR_ID } from '../../../shared/lib/sections';
import {
  createNavLinkFromSection,
  createDownloadsNavLink,
  createMenuNavLink,
  createMenuChildLink,
  generateNavLinkId,
  findSectionByTarget,
  parseNavTarget,
} from '../services/variant.service';

interface HeaderConfiguratorProps {
  config: LandingHeaderConfig;
  sections: ContentSection[];
  onChange: React.Dispatch<React.SetStateAction<LandingHeaderConfig>>;
  variantName: string;
}

export function HeaderConfigurator({ config, sections, onChange, variantName }: HeaderConfiguratorProps) {
  const [navTarget, setNavTarget] = useState('');
  const downloadsSection = sections.some((section) => section.section_type === 'downloads');

  const updateConfig = (updater: (draft: LandingHeaderConfig) => void) => {
    onChange((prev) => {
      const next = cloneHeaderConfig(prev);
      updater(next);
      return next;
    });
  };

  const handleAddLink = () => {
    if (!navTarget) return;

    const parsed = parseNavTarget(navTarget);
    if (!parsed) return;

    if (parsed.type === 'downloads') {
      updateConfig((draft) => {
        draft.nav.links.push(createDownloadsNavLink());
      });
    } else if (parsed.type === 'section') {
      const targetSection = findSectionByTarget(sections, parsed);
      if (targetSection) {
        updateConfig((draft) => {
          draft.nav.links.push(createNavLinkFromSection(targetSection));
        });
      }
    }
    setNavTarget('');
  };

  const handleNavLabelChange = (index: number, value: string) => {
    updateConfig((draft) => {
      const link = draft.nav.links[index];
      if (!link) return;
      link.label = value;
    });
  };

  const handleMenuChildChange = (
    linkIndex: number,
    childIndex: number,
    field: 'label' | 'href',
    value: string,
  ) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu') return;
      if (!Array.isArray(link.children)) {
        link.children = [];
      }
      if (!link.children[childIndex]) {
        link.children[childIndex] = createMenuChildLink();
      }
      if (field === 'label') {
        link.children[childIndex].label = value;
      } else {
        link.children[childIndex].href = value;
      }
    });
  };

  const handleAddMenuChild = (linkIndex: number) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu') return;
      if (!Array.isArray(link.children)) {
        link.children = [];
      }
      link.children.push(createMenuChildLink());
    });
  };

  const handleRemoveMenuChild = (linkIndex: number, childIndex: number) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu' || !Array.isArray(link.children)) return;
      link.children.splice(childIndex, 1);
    });
  };

  const handleVisibilityToggle = (index: number, key: 'desktop' | 'mobile', value: boolean) => {
    updateConfig((draft) => {
      const link = draft.nav.links[index];
      if (!link) return;
      link.visible_on = {
        desktop: key === 'desktop' ? value : link.visible_on?.desktop ?? true,
        mobile: key === 'mobile' ? value : link.visible_on?.mobile ?? true,
      };
    });
  };

  const handleRemoveLink = (index: number) => {
    updateConfig((draft) => {
      if (index < 0 || index >= draft.nav.links.length) return;
      draft.nav.links.splice(index, 1);
    });
  };

  const handleAddMenu = () => {
    updateConfig((draft) => {
      draft.nav.links.push(createMenuNavLink());
    });
  };

  const handleMoveLink = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= config.nav.links.length) return;
    updateConfig((draft) => {
      const [link] = draft.nav.links.splice(index, 1);
      draft.nav.links.splice(nextIndex, 0, link);
    });
  };

  const handleCTAModeChange = (
    target: 'primary' | 'secondary',
    updates: { mode?: HeaderCTAMode; label?: string; href?: string; variant?: 'solid' | 'ghost' },
  ) => {
    updateConfig((draft) => {
      draft.ctas[target] = {
        ...draft.ctas[target],
        ...updates,
      };
    });
  };

  return (
    <Card className="bg-white/5 border-white/10 mb-6">
      <CardHeader>
        <CardTitle>Header Presentation</CardTitle>
        <CardDescription className="text-slate-400">Branding, navigation, and CTA controls</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Branding Section */}
        <div className="space-y-2">
          <h3 className="text-slate-200 font-medium">Branding</h3>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Display mode</label>
              <select
                value={config.branding?.mode ?? 'logo_and_name'}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.mode = e.target.value as LandingHeaderConfig['branding']['mode'];
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="logo_and_name">Logo + Name</option>
                <option value="logo">Logo only</option>
                <option value="name">Name only</option>
                <option value="none">Minimal</option>
              </select>
            </div>
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Mobile emphasis</label>
              <select
                value={config.branding?.mobile_preference ?? 'auto'}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.mobile_preference = e.target.value as LandingHeaderConfig['branding']['mobile_preference'];
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="auto">Show both</option>
                <option value="logo">Logo on mobile</option>
                <option value="name">Name on mobile</option>
                <option value="stacked">Stacked</option>
              </select>
            </div>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Brand label</label>
              <input
                type="text"
                value={config.branding?.label ?? variantName}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.label = e.target.value;
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
                placeholder="Header title"
              />
            </div>
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Subtitle</label>
              <input
                type="text"
                value={config.branding?.subtitle ?? ''}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.subtitle = e.target.value;
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
                placeholder="Optional tagline"
              />
            </div>
          </div>
        </div>

        {/* Navigation Links Section */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-slate-200 font-medium">Navigation Links</h3>
            <div className="flex gap-2">
              <select
                value={navTarget}
                onChange={(e) => setNavTarget(e.target.value)}
                className="bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="">Select target</option>
                {sections.map((section, index) => (
                  <option
                    key={`${section.section_type}-${section.id ?? index}`}
                    value={JSON.stringify({
                      type: 'section',
                      id: section.id ?? null,
                      section_type: section.section_type,
                      order: section.order,
                    })}
                  >
                    Section · {section.section_type} #{section.order}
                  </option>
                ))}
                <option
                  value={JSON.stringify({ type: 'downloads' })}
                  disabled={!downloadsSection}
                >
                  Downloads anchor
                </option>
              </select>
              <Button variant="secondary" size="sm" onClick={handleAddLink}>
                Add link
              </Button>
              <Button variant="outline" size="sm" onClick={handleAddMenu}>
                Add menu
              </Button>
            </div>
          </div>
          {config.nav.links.length === 0 ? (
            <p className="text-sm text-slate-400">
              No manual links added. The header will mirror section order automatically.
            </p>
          ) : (
            <div className="space-y-3">
              {config.nav.links.map((link, index) => (
                <div key={link.id} className="rounded-lg border border-white/5 bg-slate-900/40 p-3 space-y-2">
                  <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                    <div className="flex-1">
                      <label className="text-xs text-slate-400 block mb-1">Label</label>
                      <input
                        type="text"
                        value={link.label}
                        onChange={(e) => handleNavLabelChange(index, e.target.value)}
                        className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-2 text-white"
                      />
                    </div>
                    <div className="flex items-center gap-3 text-xs text-slate-400">
                      <label className="flex items-center gap-1">
                        <input
                          type="checkbox"
                          checked={link.visible_on?.desktop ?? true}
                          onChange={(e) => handleVisibilityToggle(index, 'desktop', e.target.checked)}
                        />
                        Desktop
                      </label>
                      <label className="flex items-center gap-1">
                        <input
                          type="checkbox"
                          checked={link.visible_on?.mobile ?? true}
                          onChange={(e) => handleVisibilityToggle(index, 'mobile', e.target.checked)}
                        />
                        Mobile
                      </label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="ghost" size="icon" onClick={() => handleMoveLink(index, -1)} disabled={index === 0}>
                        ↑
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleMoveLink(index, 1)}
                        disabled={index === config.nav.links.length - 1}
                      >
                        ↓
                      </Button>
                      <Button variant="destructive" size="icon" onClick={() => handleRemoveLink(index)}>
                        ×
                      </Button>
                    </div>
                  </div>
                  <p className="text-xs text-slate-500">
                    {link.type === 'downloads'
                      ? 'Downloads anchor'
                      : link.type === 'menu'
                        ? 'Dropdown menu'
                        : `Link to ${link.section_type ?? 'custom target'}`}
                  </p>
                  {link.type === 'menu' && (
                    <div className="space-y-2 rounded-md border border-white/10 bg-slate-900/60 p-3">
                      <div className="flex items-center justify-between text-xs text-slate-400">
                        <span>Menu items</span>
                        <Button size="sm" variant="secondary" onClick={() => handleAddMenuChild(index)}>
                          Add item
                        </Button>
                      </div>
                      {(!link.children || link.children.length === 0) && (
                        <p className="text-xs text-slate-500">No items yet.</p>
                      )}
                      {link.children?.map((child, childIndex) => (
                        <div key={child.id} className="flex flex-col gap-2 rounded border border-white/5 bg-slate-900/50 p-2 md:flex-row md:items-center md:gap-3">
                          <div className="flex-1">
                            <label className="text-[11px] text-slate-400 block mb-1">Item label</label>
                            <input
                              type="text"
                              value={child.label}
                              onChange={(e) => handleMenuChildChange(index, childIndex, 'label', e.target.value)}
                              className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-1.5 text-white"
                            />
                          </div>
                          <div className="flex-1">
                            <label className="text-[11px] text-slate-400 block mb-1">URL or anchor</label>
                            <input
                              type="text"
                              value={child.href ?? ''}
                              onChange={(e) => handleMenuChildChange(index, childIndex, 'href', e.target.value)}
                              className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-1.5 text-white"
                            />
                          </div>
                          <Button variant="destructive" size="sm" onClick={() => handleRemoveMenuChild(index, childIndex)}>
                            Remove
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* CTA Buttons Section */}
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <h3 className="text-slate-200 font-medium">Primary CTA</h3>
            <select
              value={config.ctas.primary.mode ?? 'inherit_hero'}
              onChange={(e) => handleCTAModeChange('primary', { mode: e.target.value as HeaderCTAMode })}
              className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
            >
              <option value="inherit_hero">Use hero CTA</option>
              <option value="downloads">Downloads anchor</option>
              <option value="custom">Custom link</option>
              <option value="hidden">Hidden</option>
            </select>
            {config.ctas.primary.mode === 'custom' && (
              <div className="space-y-2">
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="Button label"
                  value={config.ctas.primary.label ?? ''}
                  onChange={(e) => handleCTAModeChange('primary', { label: e.target.value })}
                />
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="https://example.com"
                  value={config.ctas.primary.href ?? ''}
                  onChange={(e) => handleCTAModeChange('primary', { href: e.target.value })}
                />
              </div>
            )}
          </div>
          <div className="space-y-2">
            <h3 className="text-slate-200 font-medium">Secondary CTA</h3>
            <select
              value={config.ctas.secondary.mode ?? 'downloads'}
              onChange={(e) => handleCTAModeChange('secondary', { mode: e.target.value as HeaderCTAMode })}
              className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
            >
              <option value="downloads">Downloads anchor</option>
              <option value="inherit_hero">Use hero CTA</option>
              <option value="custom">Custom link</option>
              <option value="hidden">Hidden</option>
            </select>
            {(config.ctas.secondary.mode === 'custom' || config.ctas.secondary.mode === 'downloads') && (
              <div className="space-y-2">
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="Button label"
                  value={config.ctas.secondary.label ?? ''}
                  onChange={(e) => handleCTAModeChange('secondary', { label: e.target.value })}
                />
                {config.ctas.secondary.mode === 'custom' && (
                  <input
                    type="text"
                    className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                    placeholder="https://example.com"
                    value={config.ctas.secondary.href ?? ''}
                    onChange={(e) => handleCTAModeChange('secondary', { href: e.target.value })}
                  />
                )}
              </div>
            )}
          </div>
        </div>

        {/* Behavior Section */}
        <div className="space-y-2">
          <h3 className="text-slate-200 font-medium">Behavior</h3>
          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={config.behavior.sticky}
              onChange={(e) =>
                updateConfig((draft) => {
                  draft.behavior.sticky = e.target.checked;
                  if (!e.target.checked) {
                    draft.behavior.hide_on_scroll = false;
                  }
                })
              }
            />
            Sticky header
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={config.behavior.hide_on_scroll}
              disabled={!config.behavior.sticky}
              onChange={(e) =>
                updateConfig((draft) => {
                  draft.behavior.hide_on_scroll = e.target.checked;
                })
              }
            />
            Hide on downward scroll
          </label>
        </div>
      </CardContent>
    </Card>
  );
}
