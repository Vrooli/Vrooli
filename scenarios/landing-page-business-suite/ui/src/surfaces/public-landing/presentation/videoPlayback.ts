import { safeHref } from './links';
import type { Block } from './types';

/** No document-supplied embed options, HTML, SDKs or provider thumbnail requests. */
export function videoEmbedUrl(provider: string, source: string): string {
  if (!safeHref(source) || !source.startsWith('https://') || /[%#]/.test(source)) throw new Error('Unsafe video URL');
  // URL normalizes explicit :443 and an empty colon away; inspect raw authority first.
  const authority = source.slice('https://'.length).split(/[/?#]/)[0];
  if (!authority || authority.includes(':')) throw new Error('Unsafe video authority');
  const url = new URL(source);
  if (url.port || url.username || url.password) throw new Error('Unsafe video authority');
  let id: string | undefined;
  if (provider === 'youtube') {
    if (['youtube.com', 'www.youtube.com'].includes(url.hostname) && url.pathname === '/watch') {
      if ([...url.searchParams].length === 1 && url.searchParams.has('v')) id = url.searchParams.get('v') ?? undefined;
    } else if (!url.search) {
      if (url.hostname === 'youtu.be') id = url.pathname.slice(1);
      if (['youtube.com', 'www.youtube.com', 'www.youtube-nocookie.com'].includes(url.hostname)) id = /^\/embed\/([^/]+)$/.exec(url.pathname)?.[1];
    }
    if (id && /^[A-Za-z0-9_-]{11}$/.test(id)) return `https://www.youtube-nocookie.com/embed/${id}?autoplay=0&controls=1&playsinline=1`;
  } else if (provider === 'vimeo' && !url.search) {
    if (['vimeo.com', 'www.vimeo.com'].includes(url.hostname)) id = /^\/([1-9][0-9]{0,19})$/.exec(url.pathname)?.[1];
    if (url.hostname === 'player.vimeo.com') id = /^\/video\/([1-9][0-9]{0,19})$/.exec(url.pathname)?.[1];
    if (id) return `https://player.vimeo.com/video/${id}?autoplay=0&controls=1&dnt=1`;
  }
  throw new Error('Unsupported video provider or URL');
}

export function assertProductDemo(block: Block<'product-demo'>): void {
  const content = block.content;
  if (block.variant === 'recorded') {
    const p = content.playback;
    if (content.renderer_ref !== 'video' || content.fixture_ref || content.media_ref || !content.poster_ref || !p) throw new Error('Invalid recorded demo sources');
    if (!['stacked', 'split'].includes(p.layout)) throw new Error('Unsupported video layout');
    if (![content.alt_text, p.play_label, p.caption, p.unavailable_label].every(value => typeof value === 'string' && value.trim())) throw new Error('Missing video copy');
    videoEmbedUrl(p.provider, p.external_url);
  } else {
    if (!content.fixture_ref || content.playback || content.media_ref || content.poster_ref || !['workspace', 'backdrop', 'workflow'].includes(content.renderer_ref)) throw new Error('Invalid fixture demo sources');
  }
}
