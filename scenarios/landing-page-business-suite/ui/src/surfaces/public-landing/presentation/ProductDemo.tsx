import { useEffect, useId, useRef, useState } from 'react';
import type { Block, DemoPlayback } from './types';
import type { PresentationResources } from './resources';
import { AssetImage, Intro } from './primitives';
import { Visual } from './exhibits';
import { assertProductDemo, videoEmbedUrl } from './videoPlayback';

export const PLAYER_LOAD_TIMEOUT_MS = 15000;

export function ProductDemo({ block, resources, revision }: { block: Block<'product-demo'>; resources: PresentationResources; revision: string }) {
  assertProductDemo(block);
  const content = block.content;
  if (content.playback && content.poster_ref) return <RecordedDemo key={JSON.stringify([revision, content])} block={block} playback={content.playback} poster={content.poster_ref} resources={resources} />;
  if (!content.fixture_ref) throw new Error('Missing demo fixture');
  return <section id={block.id} data-block={block.kind} className="product-demo wrap"><Intro heading={content.heading} body={content.description} eyebrow={resources.blocks[block.id]?.eyebrow} breaks={resources.blocks[block.id]?.heading_breaks} /><figure aria-label={content.alt_text}><Visual visualRef={content.fixture_ref} resources={resources} interactive={block.variant === 'interactive'} /><figcaption>{resources.blocks[block.id]?.note}</figcaption></figure></section>;
}

function RecordedDemo({ block, playback, poster, resources }: { block: Block<'product-demo'>; playback: DemoPlayback; poster: string; resources: PresentationResources }) {
  const [state, setState] = useState<'poster' | 'loading' | 'embedded' | 'unavailable'>('poster');
  const frame = useRef<HTMLIFrameElement>(null);
  const error = useRef<HTMLParagraphElement>(null);
  const activated = useRef(false);
  const id = useId();
  const src = videoEmbedUrl(playback.provider, playback.external_url);
  useEffect(() => {
    if (state === 'unavailable' && activated.current) error.current?.focus();
    if (state !== 'loading' && state !== 'embedded') return;
    const element = frame.current;
    const unavailable = () => { setState('unavailable'); };
    // Resource errors do not bubble consistently through React for iframe nodes.
    element?.addEventListener('error', unavailable);
    let timer: ReturnType<typeof setTimeout> | undefined;
    if (state === 'loading') { element?.focus(); timer = setTimeout(unavailable, PLAYER_LOAD_TIMEOUT_MS); }
    return () => { clearTimeout(timer); element?.removeEventListener('error', unavailable); };
  }, [state]);
  return <section id={block.id} data-block={block.kind} className={`product-demo recorded-demo video-${playback.layout} wrap`} data-player-state={state}>
    <Intro heading={block.content.heading} body={block.content.description} eyebrow={resources.blocks[block.id]?.eyebrow} breaks={resources.blocks[block.id]?.heading_breaks} />
    <figure aria-label={block.content.alt_text} aria-describedby={`${id}-caption`}>
      <div className="video-player-frame" onErrorCapture={event => { if (event.target instanceof HTMLImageElement) setState('unavailable'); }}>
        {state === 'poster' ? <button type="button" className="video-activate" aria-label={playback.play_label} onClick={() => { activated.current = true; setState('loading'); }}>
          <AssetImage assetRef={poster} resources={resources} />
          <span className="video-activate-label"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 5 11 7-11 7Z" fill="currentColor" /></svg>{playback.play_label}</span>
        </button> : state === 'unavailable' ? <p ref={error} role="alert" tabIndex={-1} className="video-unavailable">{playback.unavailable_label}</p> : <iframe ref={frame} src={src} title={block.content.alt_text} aria-describedby={`${id}-caption`}
          referrerPolicy="strict-origin" sandbox="allow-scripts allow-same-origin allow-presentation" allow="encrypted-media; fullscreen; picture-in-picture" allowFullScreen
          onLoad={() => { setState(current => current === 'loading' ? 'embedded' : current); }} />}
      </div>
      <figcaption id={`${id}-caption`}>{playback.caption}</figcaption>
    </figure>
  </section>;
}
