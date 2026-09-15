import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
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
