// provider-free-exception: PresentationPage consumes configured props only; this suite proves it needs no auth, router, assignment or commerce providers.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { PresentationPage } from './PresentationPage';
import { PLAYER_LOAD_TIMEOUT_MS } from './ProductDemo';
import { videoFixture } from './videoTestFixtures';
import { assertProductDemo, videoEmbedUrl } from './videoPlayback';
import { resolveResources } from './resolvedResources';
import { withPresentationBase } from './publicIntegration';
import playbackCases from '../../../../../api/internal/presentation/testdata/playback-url-cases.json';

afterEach(() => { cleanup(); vi.useRealTimers(); });
describe('configured external product-demo', () => {
  it.each(playbackCases.cases)('matches shared Go/UI playback URL contract: $name', testCase => {
    if (testCase.accepted) expect(videoEmbedUrl(testCase.provider, testCase.external_url)).toBe(testCase.canonical_url);
    else expect(() => videoEmbedUrl(testCase.provider, testCase.external_url)).toThrow();
  });
  it.each(['youtube', 'vimeo'] as const)('loads only the canonical %s player on activation and keeps its caption', provider => {
    const { presentation, block } = videoFixture(provider);
    const { container } = render(<PresentationPage presentation={presentation} />);
    expect(container.querySelector('iframe,video,script,link[rel="preconnect"]')).toBeNull();
    expect(screen.getByRole('img')).toHaveAttribute('src', '/presentation/survey-relief.png');
    const button = screen.getByRole('button', { name: 'Load configured player' });
    button.focus(); expect(button).toHaveFocus();
    fireEvent.click(button);
    const frame = screen.getByTitle('Configured demonstration player');
    expect(frame).toHaveAttribute('src', videoEmbedUrl(provider, block.content.playback!.external_url));
    expect(frame).toHaveAttribute('referrerpolicy', 'strict-origin');
    expect(frame).toHaveAttribute('sandbox', 'allow-scripts allow-same-origin allow-presentation');
    expect(frame.getAttribute('allow')).not.toContain('autoplay');
    expect(frame).toHaveFocus();
    fireEvent.load(frame);
    expect(container.querySelector('[data-player-state]')).toHaveAttribute('data-player-state', 'embedded');
    expect(screen.getByText('Configured caption remains visible.')).toBeVisible();
    expect(container.textContent).not.toMatch(/Aquila|Browser Automation Studio/);
  });
  it('never loads a provider after poster failure', () => {
    const { presentation } = videoFixture(); const { container } = render(<PresentationPage presentation={presentation} />);
    fireEvent.error(screen.getByRole('img'));
    expect(screen.getByRole('alert')).toHaveTextContent('Configured player is unavailable.');
    expect(container.querySelector('iframe')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Load configured player' })).toBeNull();
  });
  it('shows configured player errors and moves focus without hiding its caption', () => {
    const { presentation } = videoFixture(); const { container } = render(<PresentationPage presentation={presentation} />);
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    fireEvent.error(screen.getByTitle('Configured demonstration player'));
    expect(screen.getByRole('alert')).toHaveFocus();
    expect(screen.getByText('Configured caption remains visible.')).toBeVisible();
    expect(container.querySelector('iframe')).toBeNull();
  });
  it.each(['revision', 'source'] as const)('tears down active players on %s changes', change => {
    const { presentation, block } = videoFixture(); const view = render(<PresentationPage presentation={presentation} />);
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    const previous = screen.getByTitle('Configured demonstration player');
    if (change === 'revision') presentation.diagnostics.resolved_revision = 'other-revision';
    else block.content.playback!.external_url = 'https://youtu.be/abcdefghijk';
    view.rerender(<PresentationPage presentation={presentation} />);
    expect(previous).not.toBeInTheDocument();
    expect(view.container.querySelector('iframe')).toBeNull();
    expect(screen.getByRole('button', { name: 'Load configured player' })).toBeVisible();
  });
  it('bounds a stalled frame load and cleans its timer on unmount', async () => {
    vi.useFakeTimers();
    const { presentation } = videoFixture(); const view = render(<PresentationPage presentation={presentation} />);
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    await act(async () => { await vi.advanceTimersByTimeAsync(PLAYER_LOAD_TIMEOUT_MS); });
    expect(screen.getByRole('alert')).toHaveTextContent('Configured player is unavailable.');
    view.unmount();
    // jsdom schedules iframe focus/blur work independently of our load timer.
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(vi.getTimerCount()).toBe(0);
  });
  it('cancels a pending player timeout when a revision replaces the source', async () => {
    vi.useFakeTimers();
    const { presentation } = videoFixture(); const view = render(<PresentationPage presentation={presentation} />);
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    presentation.diagnostics.resolved_revision = 'replacement'; view.rerender(<PresentationPage presentation={presentation} />);
    await act(async () => { await vi.advanceTimersByTimeAsync(15000); });
    expect(screen.getByRole('button', { name: 'Load configured player' })).toBeVisible();
    expect(screen.queryByRole('alert')).toBeNull();
  });
  it('honors a player error after its frame has loaded without claiming playback', () => {
    const { presentation } = videoFixture(); const { container } = render(<PresentationPage presentation={presentation} />);
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    const frame = screen.getByTitle('Configured demonstration player'); fireEvent.load(frame); fireEvent.error(frame);
    expect(screen.getByRole('alert')).toHaveTextContent('Configured player is unavailable.');
    expect(container.querySelector('iframe')).toBeNull();
  });
  it.each(['playback', 'media_ref', 'poster_ref', 'renderer_ref', 'fixture_ref'] as const)('rejects ignored/conflicting %s on fixture demos', field => {
    const { block } = videoFixture();
    const playback = block.content.playback;
    block.variant = 'interactive'; block.content.renderer_ref = 'workflow'; block.content.fixture_ref = 'workflow-fixture';
    delete block.content.playback; delete block.content.poster_ref;
    if (field === 'playback') block.content.playback = playback;
    else block.content[field] = field === 'fixture_ref' ? '' : 'invalid';
    expect(() => { assertProductDemo(block); }).toThrow();
  });
  it('preserves proxy poster/srcset resolution without rewriting provider URLs', () => {
    const { presentation } = videoFixture();
    const { container } = render(<PresentationPage presentation={withPresentationBase(presentation, '/proxy/')} />);
    expect(screen.getByRole('img')).toHaveAttribute('src', '/proxy/presentation/survey-relief.png');
    fireEvent.click(screen.getByRole('button', { name: 'Load configured player' }));
    expect(container.querySelector('iframe')?.getAttribute('src')).toMatch(/^https:\/\/www.youtube-nocookie.com\//);
  });
  it.each(['fixture_ref', 'media_ref', 'poster_ref', 'play_label', 'caption', 'unavailable_label', 'layout', 'provider'] as const)('rejects invalid recorded %s without silent fallback', field => {
    const { block } = videoFixture();
    if (field === 'fixture_ref' || field === 'media_ref') block.content[field] = 'conflicting';
    else if (field === 'poster_ref') block.content.poster_ref = '';
    else Object.assign(block.content.playback!, { [field]: field === 'layout' || field === 'provider' ? 'unknown' : '' });
    expect(() => { assertProductDemo(block); }).toThrow();
  });
  it.each(['missing', 'external', 'mime', 'release'] as const)('rejects %s poster resources before provider contact', kind => {
    const { presentation } = videoFixture(); const asset = presentation.assets![0]!;
    if (kind === 'missing') presentation.assets = [];
    if (kind === 'external') asset.public_url = 'https://img.youtube.com/vi/abcdefghijk/default.jpg';
    if (kind === 'mime') asset.mime = 'video/mp4';
    if (kind === 'release') asset.release_ref = '';
    expect(() => resolveResources(presentation)).toThrow();
  });
});

describe('finite external video URL policy', () => {
  it.each(['https://www.youtube-nocookie.com/watch?v=abcdefghijk', 'https://youtu.be/abcdefghijk#'])('rejects agreed-policy edge outside the current shared corpus: %s', url => {
    expect(() => videoEmbedUrl('youtube', url)).toThrow();
  });
  it.each(['https://youtu.be/abcdefghijk', 'https://youtube.com/watch?v=abcdefghijk', 'https://www.youtube.com/embed/abcdefghijk', 'https://www.youtube-nocookie.com/embed/abcdefghijk'])('accepts exact YouTube form %s', url => {
    expect(videoEmbedUrl('youtube', url)).toBe('https://www.youtube-nocookie.com/embed/abcdefghijk?autoplay=0&controls=1&playsinline=1');
  });
  it.each(['https://vimeo.com/12345', 'https://www.vimeo.com/12345', 'https://player.vimeo.com/video/12345'])('accepts exact Vimeo form %s', url => {
    expect(videoEmbedUrl('vimeo', url)).toBe('https://player.vimeo.com/video/12345?autoplay=0&controls=1&dnt=1');
  });
  it.each(['https://player.vimeo.com/video/12345?autoplay=1', 'https://vimeo.com/12345?dnt=1', 'https://vimeo.com.evil.test/12345', 'https://vimeo.com/0', 'https://vimeo.com/abc', 'https://vimeo.com/12345/private', 'https://youtu.be/abcdefghijk'])('rejects unsafe or unsupported Vimeo URL %s', url => {
    expect(() => videoEmbedUrl('vimeo', url)).toThrow();
  });
  it.each(['//youtube.com/embed/abcdefghijk', 'http://youtu.be/abcdefghijk', 'https://youtube.com.evil.test/watch?v=abcdefghijk', 'https://user:pass@youtube.com/watch?v=abcdefghijk', 'https://youtube.com:444/watch?v=abcdefghijk', 'https://youtube.com/watch?v=abcdefghijk&autoplay=1', 'https://youtube.com/watch?v=abcdefghijk&v=abcdefghijk', 'https://youtu.be/abcdefghijk#x', 'https://youtu.be/%61bcdefghijk', 'https://youtu.be/abcdefghijk\\', 'https://youtu.be/../abcdefghijk', 'https://youtube.com/embed/abcdefghijk?origin=https://private.test', 'https://vimeo.com/12345', 'https://youtu.be/short'])('rejects unsafe or unsupported YouTube URL %s', url => {
    expect(() => videoEmbedUrl('youtube', url)).toThrow();
  });
});
