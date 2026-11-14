// provider-free-exception: finite PresentationPage receives all content/actions as props and must render without public assignment, auth, router or API providers.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { renderToStaticMarkup } from 'react-dom/server';
import { PresentationPage, type PresentationPageProps } from './PresentationPage';
import { actionKey } from './types';
import { safeHref } from './links';
import { assertRendererContract } from './contract';
import { resolveResources } from './resolvedResources';
import { parseProductPresentation } from './decode';
import signalData from './fixtures/signal.json';
import studioData from './fixtures/studio.json';

// Provider-free by contract: this isolated renderer receives every owner join as
// props and must not depend on LandingVariant, a router, commerce, or API context.
const fixture = (studio = false): PresentationPageProps =>
  ({ presentation: parseProductPresentation(JSON.stringify((studio ? studioData : signalData).presentation)) });
afterEach(cleanup);

describe('Signal and Studio presentation', () => {
  it.each(['ArrowUp', 'ArrowLeft', 'ArrowDown'])('wraps artifact selection with %s while keeping one keyboard tab stop and linked visible panel', key => {
    render(<PresentationPage {...fixture()} />);
    const tabs = screen.getAllByRole('tab');
    const start = key === 'ArrowDown' ? tabs.length - 1 : 0;
    const end = key === 'ArrowDown' ? 0 : tabs.length - 1;
    const from = tabs[start]; const to = tabs[end];
    if (!from || !to) throw new Error('Fixture tabs missing');
    fireEvent.click(from); from.focus(); fireEvent.keyDown(from, { key });
    expect(to).toHaveFocus(); expect(to).toHaveAttribute('aria-selected', 'true');
    expect(tabs.filter(tab => tab.tabIndex === 0)).toEqual([to]);
    expect(screen.getByRole('tabpanel')).toHaveAttribute('id', to.getAttribute('aria-controls'));
    expect(screen.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', to.id);
  });
  it('leaves unrelated artifact keys to the browser without changing selection', () => {
    render(<PresentationPage {...fixture()} />);
    const tab = screen.getAllByRole('tab')[0]; if (!tab) throw new Error('Fixture tab missing');
    tab.focus();
    expect(fireEvent.keyDown(tab, { key: 'Tab', cancelable: true })).toBe(true);
    expect(tab).toHaveFocus(); expect(tab).toHaveAttribute('aria-selected', 'true');
  });
  it('returns to the configured default when an edited document removes the currently selected artifact', () => {
    const props = fixture(); const view = render(<PresentationPage {...props} />);
    const block = props.presentation.page.blocks.find(b => b.kind === 'artifact-explorer');
    if (!block) throw new Error('Fixture artifacts missing');
    fireEvent.click(screen.getByRole('tab', { name: /The HTML/ }));
    block.content.examples = block.content.examples.filter(example => example.id !== 'html');
    view.rerender(<PresentationPage {...props} />);
    expect(screen.queryByRole('tab', { name: /The HTML/ })).toBeNull();
    const selected = screen.getAllByRole('tab').filter(tab => tab.getAttribute('aria-selected') === 'true');
    expect(selected).toHaveLength(1);
    expect(screen.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', selected[0]?.id);
    expect(screen.getByRole('tabpanel')).toHaveTextContent('Shape the story');
  });
  it.each(['audio', 'code', 'pdf'] as const)('renders configured %s excerpts as text, never executable HTML or an inferred player', kind => {
    const props = fixture(); const block = props.presentation.page.blocks.find(b => b.kind === 'artifact-explorer');
    const example = block?.content.examples[0]; if (!block || !example) throw new Error('Fixture artifact missing');
    block.content.examples = [{ ...example, id: kind, kind, label: `Configured ${kind}`, filename: `configured.${kind}`, title: 'Configured excerpt', source: { ...example.source, text_excerpt: '<img src="/private" onerror="alert(1)">' } }];
    block.content.selected_example_id = kind;
    render(<PresentationPage {...props} />);
    const panel = screen.getByRole('tabpanel');
    expect(panel).toHaveTextContent('<img src="/private" onerror="alert(1)">');
    expect(panel.querySelector('pre')).toHaveClass('artifact-text');
    expect(panel.querySelector('img,iframe,audio,video,script')).toBeNull();
  });
  it('renders released hero, story and desktop device images from explicit asset slots', () => {
    const props = fixture(); const p = props.presentation;
    const hero = p.page.blocks.find(b => b.kind === 'product-hero');
    const story = p.page.blocks.find(b => b.kind === 'product-story');
    const device = p.page.blocks.find(b => b.kind === 'device-story');
    if (!hero || !story?.content.items[0] || !device) throw new Error('Fixture blocks missing');
    hero.content.fixture_ref = undefined; hero.content.visual_ref = 'survey-relief'; delete p.page.display.blocks.hero;
    story.content.items[0].visual_ref = 'pale-moon'; story.content.items[0].alt_text = 'Configured story art';
    device.variant = 'desktop'; device.content.visual_ref = 'tidal-halftone'; Reflect.deleteProperty(p.page.display.blocks, device.id);
    const { container } = render(<PresentationPage {...props} />);
    expect(container.querySelector('.hero-stage img')).toHaveAttribute('src', '/presentation/survey-relief.png');
    expect(container.querySelector('.hero-stage img')).toHaveAttribute('loading', 'eager');
    expect(screen.getByRole('img', { name: 'Configured story art' })).toHaveAttribute('src', '/presentation/pale-moon.png');
    expect(container.querySelector('.mobile-story img')).toHaveAttribute('src', '/presentation/tidal-halftone.png');
    expect(container.querySelector('.mobile-story .phone')).toBeNull();
  });
  it('renders bundle hero and catalog assets without requiring native demo fixtures in their slots', () => {
    const props = fixture(true); const p = props.presentation;
    const hero = p.page.blocks.find(b => b.kind === 'bundle-hero'); if (!hero) throw new Error('Fixture hero missing');
    p.page.display.blocks.hero = { hero_fixture_refs: {} };
    hero.content.hero_items.forEach((item, index) => {
      item.visual_ref = index === 0 ? 'pale-moon' : 'survey-relief'; item.exhibit_kind = 'visual';
      const app = p.page.display.apps[item.app_key]; if (!app) throw new Error('Fixture app missing');
      app.fixture_ref = undefined; app.visual_ref = item.visual_ref;
    });
    const { container } = render(<PresentationPage {...props} />);
    expect([...container.querySelectorAll('.hero-app-group img')].map(img => img.getAttribute('src'))).toEqual(['/presentation/pale-moon.png', '/presentation/survey-relief.png']);
    expect(container.querySelectorAll('.catalog .product-card-visual img')).toHaveLength(2);
    expect(container.querySelector('.catalog .workspace, .catalog .backdrop-app')).toBeNull();
  });
  it('renders capability claims without inventing anchor destinations and avoids duplicate provider qualifications', () => {
    const props = fixture(); const p = props.presentation;
    const strip = p.page.blocks.find(b => b.kind === 'capability-strip');
    const voice = p.page.blocks.find(b => b.kind === 'voice-story'); if (!strip || !voice) throw new Error('Fixture blocks missing');
    p.page.display.blocks[strip.id] = {};
    voice.content.provider_qualification = voice.content.note;
    const { container } = render(<PresentationPage {...props} />);
    const claims = container.querySelector('.principles');
    expect(claims?.querySelector('a')).toBeNull();
    for (const item of strip.content.items) expect(claims).toHaveTextContent(item.label);
    expect(container.querySelector('.voice-copy .voice-note')).toHaveTextContent(voice.content.note);
    expect(container.querySelector('.voice-qualification')).toBeNull();
  });
  it.each(['hero', 'catalog'])('fails closed for an unsafe resolved app detail link in the %s', surface => {
    const props = fixture(true); const p = props.presentation;
    const app = p.spotlights?.[0]; if (!app) throw new Error('Fixture app missing'); app.detail_route = 'javascript:alert(1)';
    if (surface === 'catalog') {
      p.page.blocks = p.page.blocks.filter(block => block.kind !== 'bundle-hero');
      delete p.page.display.blocks.hero;
    } else {
      p.page.blocks = p.page.blocks.filter(block => block.kind !== 'app-spotlights');
      delete p.page.display.blocks.apps;
    }
    // The prop-only renderer must reject unsafe destinations before producing
    // markup, without depending on a browser/provider error handler to sanitize.
    expect(() => renderToStaticMarkup(<PresentationPage {...props} />)).toThrow(surface === 'hero' ? 'Unresolved hero detail route' : 'Unresolved app spotlight');
  });
  it.each([[false, false], [true, false], [false, true], [true, true]])('exposes finite diagnostic capture boundaries for preview=%s fallback=%s', (preview, fallback) => {
    const props = fixture();
    Object.assign(props.presentation.diagnostics, { preview, fallback });
    const { container } = render(<PresentationPage {...props} />);
    const root = container.querySelector('.presentation-page[data-presentation-mode]');
    expect(root).toHaveAttribute('data-presentation-preview', String(preview));
    expect(root).toHaveAttribute('data-presentation-fallback', String(fallback));
    expect(container.querySelectorAll('main')).toHaveLength(preview ? 0 : 1);
    const skip = container.querySelector<HTMLAnchorElement>('.skip-link');
    expect(skip).not.toBeNull();
    expect(document.getElementById(skip!.hash.slice(1))).toHaveAttribute('tabindex', '-1');
  });
  it('reads native demo content and asset URLs only from canonical resolved arrays', () => {
    const props = fixture();
    const workspace = props.presentation.fixtures?.find(item => item.kind === 'workspace');
    if (!workspace) throw new Error('Fixture workspace missing');
    workspace.workspace.prompt = 'A different configured instruction';
    workspace.workspace.files = ['Replacement.tsx'];
    const asset = props.presentation.assets?.find(item => item.id === 'survey-relief');
    if (!asset) throw new Error('Fixture asset missing');
    asset.public_url = '/presentation/replaced.png';
    const { container } = render(<PresentationPage {...props} />);
    expect(screen.getAllByText('A different configured instruction').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Replacement.tsx').length).toBeGreaterThan(0);
    expect(container.querySelector('img[src="/presentation/replaced.png"]')).not.toBeNull();
    expect(container.querySelector('.file-diff')).toBeEmptyDOMElement();
    props.presentation.fixtures = [];
    expect(() => resolveResources(props.presentation)).toThrow('Unresolved fixture');
  });
  it('uses canonical backdrop selection and rejects a mismatched selection', () => {
    const props = fixture(true);
    const backdrop = props.presentation.fixtures?.find(item => item.id === 'art-studies');
    if (backdrop?.kind !== 'backdrop') throw new Error('Fixture backdrop missing');
    backdrop.backdrop.selected = 'Tidal Halftone';
    const { container } = render(<PresentationPage {...props} />);
    expect(container.querySelector('.product-card .backdrop-canvas img')).toHaveAttribute('src', '/presentation/tidal-halftone.png');
    backdrop.backdrop.selected = 'missing';
    expect(() => { resolveResources(props.presentation); }).toThrow('Invalid backdrop selection');
  });
  it('resets configured artifact selection when the configuration changes', () => {
    const props = fixture();
    const view = render(<PresentationPage {...props} />);
    const artifacts = props.presentation.page.blocks.find(block => block.kind === 'artifact-explorer');
    if (!artifacts) throw new Error('Fixture artifacts missing');
    artifacts.content.selected_example_id = 'html';
    view.rerender(<PresentationPage {...props} />);
    expect(screen.getByRole('tab', { name: /The HTML/ })).toHaveAttribute('aria-selected', 'true');
  });
  it('uses explicitly configured copy and has no product-name fallback', () => {
    const props = { presentation: parseProductPresentation(JSON.stringify(signalData.presentation).replace(/Aquila/g, 'Example Product').replace(/AQUILA/g, 'EXAMPLE PRODUCT')) };
    const { container } = render(<PresentationPage {...props} />);
    expect(container).not.toHaveTextContent(/aquila/i);
    expect(screen.getAllByText('Example Product').length).toBeGreaterThan(0);
  });
  it('renders configured block order, rich exhibits, and coming-soon claims', () => {
    const props = fixture();
    props.presentation.diagnostics.preview = false;
    const { container } = render(<PresentationPage {...props} />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('All your agents. One clear view.');
    expect([...container.querySelectorAll('main > [data-block]')].map(node => node.id)).toEqual(['hero', 'principles', 'artifacts', 'voice', 'details', 'mobile', 'roadmap', 'get-started']);
    expect(screen.getByText('Local speech options. AI summaries use your configured model and provider.')).toBeInTheDocument();
    const roadmap = container.querySelector('#roadmap') as HTMLElement;
    expect(within(roadmap).getAllByText('COMING SOON')).toHaveLength(3);
    expect(within(roadmap).queryByRole('button')).toBeNull();
    expect(within(roadmap).queryByRole('link')).toBeNull();
  });
  it('supports tab keyboard navigation, focus, panel relationships, Home and End', () => {
    render(<PresentationPage {...fixture()} />);
    const tabs = screen.getAllByRole('tab');
    tabs[0]?.focus();
    fireEvent.keyDown(tabs[0]!, { key: 'ArrowRight' });
    expect(tabs[1]).toHaveFocus();
    expect(tabs[1]).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', tabs[1]?.id);
    fireEvent.keyDown(tabs[1]!, { key: 'End' });
    expect(tabs[3]).toHaveFocus();
    expect(screen.getByRole('tabpanel')).toHaveTextContent('Illustrative video storyboard.');
    fireEvent.keyDown(tabs[3]!, { key: 'Home' });
    expect(tabs[0]).toHaveFocus();
    expect(screen.getByRole('tabpanel')).toHaveTextContent('Shape the story');
    expect(tabs.filter(tab => tab.tabIndex === 0)).toHaveLength(1);
  });
  it('keeps artwork panels in one identified app group and preserves resolved catalog order', () => {
    const props = fixture(true);
    const catalog = props.presentation.page.blocks.find(block => block.kind === 'app-spotlights');
    if (!catalog) throw new Error('Fixture catalog missing');
    catalog.content.app_keys.reverse();
    const { container } = render(<PresentationPage {...props} />);
    expect(container.querySelectorAll('[data-hero-app-key]')).toHaveLength(2);
    expect(container.querySelector('[data-hero-app-key="backdrop-studio"]')?.querySelectorAll('.art-print')).toHaveLength(2);
    expect([...container.querySelectorAll('[data-app-key]')].map(node => node.getAttribute('data-app-key'))).toEqual(['backdrop-studio', 'web-console']);
    expect(container.querySelectorAll('.product-card-visual button')).toHaveLength(0);
    expect(screen.queryByText('Browser Automation Studio')).toBeNull();
  });
  it('renders zero hero groups without inventing an exhibit', () => {
    const props = fixture(true);
    const hero = props.presentation.page.blocks[0];
    if (hero?.kind !== 'bundle-hero') throw new Error('Fixture hero missing');
    hero.content.hero_items = [];
    props.presentation.page.display.blocks.hero!.hero_fixture_refs = {};
    const { container } = render(<PresentationPage {...props} />);
    expect(container.querySelectorAll('[data-hero-app-key]')).toHaveLength(0);
    expect(container.querySelector('.art-seal')).toBeNull();
  });
  it('only enables commerce with an owner-resolved action and preserves unavailable reasons', () => {
    const props = fixture();
    const action = props.presentation.page.display.shell.header_action!;
    const activate = vi.fn();
    const view = render(<PresentationPage {...props} />);
    expect(screen.getAllByRole('button', { name: 'Get Aquila' }).every(button => button.hasAttribute('disabled'))).toBe(true);
    expect(screen.getAllByText(props.presentation.page.display.shell.unavailable_reason).length).toBeGreaterThan(0);
    view.rerender(<PresentationPage {...props} resolvedActions={{ [actionKey(action)]: { status: 'ready', onActivate: activate } }} />);
    fireEvent.click(screen.getAllByRole('button', { name: 'Get Aquila' })[0]!);
    expect(activate).toHaveBeenCalledTimes(1);
  });
  it('switches terminal/conversation views and closes mobile navigation with Escape', () => {
    const { container } = render(<PresentationPage {...fixture()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Conversation' }));
    expect(container.querySelector('.messages-view')).not.toHaveAttribute('hidden');
    expect(container.querySelector('.terminal-view')).toHaveAttribute('hidden');
    const menu = screen.getByRole('button', { name: 'Toggle navigation' });
    fireEvent.click(menu);
    expect(menu).toHaveAttribute('aria-expanded', 'true');
    fireEvent.keyDown(menu, { key: 'Escape' });
    expect(menu).toHaveAttribute('aria-expanded', 'false');
    expect(menu).toHaveFocus();
  });
  it('renders edited content in app_detail without a page-name heuristic or executing HTML', () => {
    const props = fixture();
    props.presentation.mode = 'app_detail';
    const hero = props.presentation.page.blocks[0];
    if (hero?.kind !== 'product-hero') throw new Error('Fixture hero missing');
    hero.content.title = 'A different product\nA different story';
    const artifacts = props.presentation.page.blocks.find(block => block.kind === 'artifact-explorer');
    if (artifacts?.kind !== 'artifact-explorer') throw new Error('Fixture artifacts missing');
    const html = artifacts.content.examples.find(example => example.id === 'html');
    if (!html) throw new Error('Fixture HTML missing');
    html.title = '<script>window.secret=1</script>';
    const { container } = render(<PresentationPage {...props} />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('A different product A different story');
    fireEvent.click(screen.getByRole('tab', { name: /The HTML/ }));
    expect(screen.getByRole('tabpanel')).toHaveTextContent('<script>window.secret=1</script>');
    expect(container.querySelector('iframe,script')).toBeNull();
  });
  it('fails unsupported versions, variants, duplicate groups, and invalid selection', () => {
    const props = fixture(true);
    const hero = props.presentation.page.blocks[0];
    if (hero?.kind !== 'bundle-hero') throw new Error('Fixture hero missing');
    hero.variant = 'arbitrary-html';
    expect(() => { assertRendererContract(props.presentation); }).toThrow('Unsupported presentation block');
    hero.variant = 'editorial';
    hero.content.hero_items.push(hero.content.hero_items[0]!);
    expect(() => { assertRendererContract(props.presentation); }).toThrow('Duplicate hero app');
    const single = fixture();
    const artifacts = single.presentation.page.blocks.find(block => block.kind === 'artifact-explorer');
    if (artifacts?.kind !== 'artifact-explorer') throw new Error('Fixture artifacts missing');
    artifacts.content.selected_example_id = 'missing';
    expect(() => { assertRendererContract(single.presentation); }).toThrow('Invalid artifact selection');
  });
  it.each(['javascript:alert(1)', 'data:text/html,unsafe', '//third-party.test', '/\\evil.test', ' https://example.test'])('rejects unsafe target %s', value => {
    expect(safeHref(value)).toBeUndefined();
  });
});
