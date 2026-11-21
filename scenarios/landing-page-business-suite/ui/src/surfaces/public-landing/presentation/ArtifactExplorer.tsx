import { useId, useRef, useState, type KeyboardEvent } from 'react';
import type { ArtifactExample, Block } from './types';
import type { PresentationResources } from './resources';
import { AssetImage, Intro, ProductMark } from './primitives';

function Preview({ fixture: x, resources }: { fixture: ArtifactExample; resources: PresentationResources }) {
  switch (x.kind) {
    case 'plan': return <div className="plan-paper"><p className="exhibit-kicker">{x.type}</p><h3>{x.title}</h3><p>{x.body}</p><PlanDiagram steps={x.steps} checks={x.checks} /></div>;
    case 'image': return <div className="evidence-image">{x.asset_ref && <AssetImage assetRef={x.asset_ref} resources={resources} />}<div><p className="exhibit-kicker">{x.type}</p><h3>{x.title}</h3><p>{x.body}</p></div></div>;
    case 'html-preview': return <div className="html-exhibit"><div className="exhibit-browser"><i aria-hidden="true">● ● ●</i><span>{x.type}</span></div><div className="html-composition"><p className="exhibit-kicker">{x.brand}</p><h3>{x.title}<em>{x.accent}</em></h3><p>{x.body}</p>{x.asset_ref && <div className="html-art"><AssetImage assetRef={x.asset_ref} resources={resources} /></div>}</div></div>;
    case 'video': return <div className="video-exhibit"><p className="exhibit-kicker">{x.type}</p><h3>{x.title}</h3><p>{x.body}</p><div className="video-filmstrip">{x.frames?.map((frame, index) => <figure key={index}><AssetImage assetRef={frame.asset_ref} resources={resources} /><figcaption><span>{frame.time}</span>{frame.label}</figcaption></figure>)}</div></div>;
    case 'audio': case 'code': case 'pdf': return <div className="plan-paper"><p className="exhibit-kicker">{x.type}</p><h3>{x.title}</h3><p>{x.body}</p><pre className="artifact-text">{x.source.text_excerpt}</pre></div>;
  }
}

/**
 * The plan exhibit claims a rendered Markdown document with Mermaid diagrams, so
 * the flow is drawn as a flowchart rather than listed as chips: process nodes on
 * a diagram canvas, real edges with arrowheads, and each acceptance check hanging
 * off the step it belongs to. Every string is page-owned; only the shape is ours.
 */
function CheckGlyph() {
  // Drawn, not a character: none of the presentation faces carries U+2713.
  return <svg className="flow-tick" viewBox="0 0 12 12" aria-hidden="true" focusable="false"><path d="M1.6 6.2 4.4 9 10.4 3" /></svg>;
}

function PlanDiagram({ steps, checks }: { steps?: string[]; checks?: string[] }) {
  if (!steps?.length) return checks?.length ? <PlanChecks checks={checks} /> : null;
  // Checks pair with steps only when the document supplies one for each.
  const paired = checks?.length === steps.length ? checks : undefined;
  return <>
    <div className="plan-diagram">
      <ol className="plan-flow">
        {steps.map((step, index) => <li key={index}>
          <div className="flow-node"><span>{String(index + 1).padStart(2, '0')}</span><b>{step}</b></div>
          {paired && <>
            <span className="flow-drop" aria-hidden="true" />
            <p className="flow-check"><CheckGlyph />{paired[index]}</p>
          </>}
        </li>)}
      </ol>
    </div>
    {!paired && checks?.length ? <PlanChecks checks={checks} /> : null}
  </>;
}

function PlanChecks({ checks }: { checks: string[] }) {
  return <ul>{checks.map((check, index) => <li key={index}><CheckGlyph />{check}</li>)}</ul>;
}

export function ArtifactExplorer({ block, resources }: { block: Block<'artifact-explorer'>; resources: PresentationResources }) {
  const content = block.content;
  const display = resources.blocks[block.id] ?? {};
  const [selected, setSelected] = useState(content.selected_example_id);
  const active = content.examples.some(example => example.id === selected) ? selected : content.selected_example_id;
  const prefix = useId();
  const refs = useRef<(HTMLButtonElement | null)[]>([]);
  function navigate(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    const count = content.examples.length;
    let next: number;
    switch (event.key) {
      case 'ArrowDown': case 'ArrowRight': next = (index + 1) % count; break;
      case 'ArrowUp': case 'ArrowLeft': next = (index - 1 + count) % count; break;
      case 'Home': next = 0; break;
      case 'End': next = count - 1; break;
      default: return;
    }
    event.preventDefault();
    const example = content.examples[next];
    if (example) { setSelected(example.id); refs.current[next]?.focus(); }
  }
  return <section className="artifact-story wrap" id={block.id} data-block={block.kind}><Intro heading={content.heading} body={display.description} eyebrow={display.eyebrow} breaks={display.heading_breaks} />
    <div className="artifact-workbench"><div className="artifact-topline">{display.mark && <ProductMark kind={display.mark} />}<span>{display.badge}</span><small>{display.note}</small></div>
      <div className="artifact-layout"><div className="artifact-tabs" role="tablist" aria-label={display.accessibility_label ?? content.heading}>
        {content.examples.map((example, index) => <button key={example.id} ref={element => { refs.current[index] = element; }} type="button" role="tab" id={`${prefix}-tab-${example.id}`} aria-controls={`${prefix}-panel-${example.id}`} aria-selected={active === example.id} tabIndex={active === example.id ? 0 : -1} onClick={() => { setSelected(example.id); }} onKeyDown={event => { navigate(event, index); }}><span className="file-icon" aria-hidden="true">{example.icon}</span><span><b>{example.label}</b><small>{example.filename}</small></span></button>)}
      </div><div className="artifact-panels">{content.examples.map(example => {
        return <div key={example.id} role="tabpanel" tabIndex={0} id={`${prefix}-panel-${example.id}`} aria-labelledby={`${prefix}-tab-${example.id}`} hidden={active !== example.id}><Preview fixture={example} resources={resources} /><p className="artifact-caption">{example.caption}</p></div>;
      })}</div></div><div className="artifact-formats">{display.formats?.map((format, index) => <span key={index}>{format}</span>)}</div>
    </div>
  </section>;
}
