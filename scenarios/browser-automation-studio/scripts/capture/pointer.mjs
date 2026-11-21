// Capture adapter for Puppeteer-compatible desktop pages. This is presentation
// only: real mouse events still drive the unmodified application. Node >=22.18.
import {REPLAY_ARROW_CURSOR_PATH} from '../../ui/src/domains/replay-style/constants.ts';

const pause = ms => new Promise(resolve => setTimeout(resolve, ms));

/** Select the BAS Classic Light arrow. Coordinates are its actual tip, not its
 * bounding-box center. Do not use on touch footage or already baked pointers. */
export async function installDesktopPointer(page, {theme, scale = 1.35, initial = {x: 420, y: 410}} = {}) {
  if (theme !== 'arrowLight') throw new Error('This capture adapter supports BAS arrowLight; select it explicitly.');
  if (!Number.isFinite(scale) || scale < 0.6 || scale > 1.8) throw new Error('BAS cursor scale must be 0.6..1.8.');
  const events = [];
  const origin = performance.now();
  let position = {...initial};
  let disposed = false;
  const mouse = page.mouse;
  const original = {move: mouse.move.bind(mouse), click: mouse.click.bind(mouse), down: mouse.down.bind(mouse), up: mouse.up.bind(mouse)};
  await page.evaluate(({path, scale, initial}) => {
    if (document.querySelector('[data-bas-capture-pointer]')) throw new Error('A capture pointer is already installed.');
    const host = document.createElement('div');
    host.setAttribute('data-bas-capture-pointer', 'arrowLight');
    host.style.cssText = 'position:fixed;inset:0;pointer-events:none;z-index:2147483647;overflow:hidden';
    const root = host.attachShadow({mode: 'open'});
    const style = document.createElement('style');
    style.textContent = `:host{pointer-events:none}#tip{position:absolute;width:${32*scale}px;height:${32*scale}px;transform-origin:0 0;filter:drop-shadow(0 3px 4px #0008)}#ring{position:absolute;width:38px;height:38px;border:2px solid #67e8f9;border-radius:50%;opacity:0;transform:translate(-50%,-50%);box-sizing:border-box}`;
    root.append(style);
    const ring = document.createElement('div'); ring.id = 'ring'; root.append(ring);
    const tip = document.createElement('div'); tip.id = 'tip';
    tip.innerHTML = `<svg viewBox="0 0 32 32" width="100%" height="100%"><path d="${path}" fill="rgba(255,255,255,0.96)" stroke="rgba(15,23,42,0.85)" stroke-width="1.4" stroke-linejoin="round"/></svg>`;
    root.append(tip); document.documentElement.append(host);
    const move = (x,y) => {tip.style.left = (x-6*scale)+'px';tip.style.top = (y-3*scale)+'px';ring.style.left=x+'px';ring.style.top=y+'px';};
    window.__basCapturePointer = {move, down(){ring.getAnimations().forEach(a=>a.cancel());ring.animate([{opacity:.75,scale:.55},{opacity:0,scale:1.2}],{duration:360,easing:'ease-out'});}, remove(){host.remove();delete window.__basCapturePointer;}};
    move(initial.x,initial.y);
  }, {path: REPLAY_ARROW_CURSOR_PATH, scale, initial});
  await original.move(initial.x, initial.y);
  function log(type, extra = {}) { events.push({type, t:(performance.now()-origin)/1000, ...position, ...extra}); }
  log('initial');
  async function setPosition(x,y) {
    await original.move(x,y);
    await page.evaluate((x,y)=>window.__basCapturePointer?.move(x,y),x,y);
    position={x,y};log('move');
  }
  mouse.move = async (x,y,options={}) => {
    if (disposed) throw new Error('Desktop pointer disposed.');
    const start={...position};const dx=x-start.x,dy=y-start.y;const distance=Math.hypot(dx,dy);
    if(distance<.25) return;
    // Bounded, deterministic travel. Elapsed wall time, rather than CDP step
    // count, controls velocity. The arc returns to exactly the native target.
    const duration=Math.min(690,Math.max(180,120+distance*.46));
    const arc=Math.min(32,distance*.045)*(dx>=0?-1:1);const length=distance||1;
    const t0=performance.now();
    while(true){const u=Math.min(1,(performance.now()-t0)/duration);const e=u*u*(3-2*u);const bend=4*e*(1-e)*arc;
      await setPosition(start.x+dx*e-dy/length*bend,start.y+dy*e+dx/length*bend);
      if(u>=1)break;await pause(12);
    }
  };
  mouse.down = async options => {await page.evaluate(()=>window.__basCapturePointer?.down());log('down',{button:options?.button??'left'});return original.down(options);};
  mouse.up = async options => {log('up',{button:options?.button??'left'});return original.up(options);};
  mouse.click = async (x,y,options={}) => {
    await mouse.move(x,y);await pause(75);
    const count=options.clickCount??1;
    for(let i=1;i<=count;i++){await mouse.down({...options,clickCount:i});await pause(options.delay??65);await mouse.up({...options,clickCount:i});if(i<count)await pause(80);}
  };
  return {
    theme, scale, baked: true, events,
    async dispose(){if(disposed)return;disposed=true;Object.assign(mouse, original);await page.evaluate(()=>window.__basCapturePointer?.remove());},
  };
}
