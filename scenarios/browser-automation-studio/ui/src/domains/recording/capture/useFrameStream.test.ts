import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useFrameStream } from './useFrameStream';
import { useSessionStore } from '../stores/sessionStore';

const mocks = vi.hoisted(() => ({
  config: vi.fn(), recordFrame: vi.fn(), reset: vi.fn(),
  stats: {currentFps: 0, avgFps: 0, lastFrameSize: 0, avgFrameSize: 0, totalFrames: 0, totalBytes: 0, bytesPerSecond: 0},
}));
vi.mock('@/config', () => ({getConfig: mocks.config, getWsBase: () => 'ws://fixture.test/ws'}));
vi.mock('@/contexts/WebSocketContext', () => ({useWebSocket: () => ({})}));
vi.mock('../hooks/useFrameStats', () => ({useFrameStats: () => ({stats: mocks.stats, recordFrame: mocks.recordFrame, reset: mocks.reset})}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => {resolve = yes; reject = no;});
  return {promise, resolve, reject};
}
const sockets: Socket[] = [];
class Socket {
  static OPEN = 1;
  readyState = 0;
  binaryType = '';
  onopen: (() => void) | null = null;
  onmessage: ((event: {data: ArrayBuffer | string}) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => {this.readyState = 3;});
  constructor(readonly url: string) {sockets.push(this);}
  open() {this.readyState = 1; this.onopen?.();}
  frame(session_id = 'session-a', page_id = 'page-a') {
    this.onmessage?.({data: framePacket({session_id, page_id})});
  }

}
function framePacket(overrides: Record<string, unknown> = {}) {
  const header = new TextEncoder().encode(JSON.stringify({version:1, session_id:'session-a', page_id:'page-a', captured_at:'2026-09-23T03:00:00Z', ...overrides}));
  const packet = new Uint8Array(4 + header.length + 4);
  new DataView(packet.buffer).setUint32(0, header.length);
  packet.set(header, 4); packet.set([255,216,255,217], 4 + header.length);
  return packet.buffer;
}
const decodes: Array<ReturnType<typeof deferred<ImageBitmap>>> = [];
const paints = new Map<number, FrameRequestCallback>();
const draw = vi.fn(), clear = vi.fn();
let fetchMock: ReturnType<typeof vi.fn>;
let rafId = 0;
const config = {API_URL: 'http://fixture.test/api/v1', WS_URL: 'ws://fixture.test/ws', playwrightDriverPort: 24485};
const payload = {session_id:'session-a',page_id:'page-a',image: 'data:image/jpeg;base64,/9j/2Q==', width: 640, height: 480, captured_at: '2026-09-23T03:00:00Z', page_title: 'Current', page_url: 'https://fixture.test'};
const drain = async () => {for(let n=0;n<20;n++)await Promise.resolve();};
const advance = async (ms: number) => {await act(async () => {await vi.advanceTimersByTimeAsync(ms);});};
function bitmap(id: number) {return {width:640,height:480,id,close:vi.fn()} as unknown as ImageBitmap & {id: number; close: ReturnType<typeof vi.fn>};}
async function resolveDecode(index: number) {
  const image = bitmap(index + 1);
  await act(async () => {decodes[index]!.resolve(image); await drain();});
  return image;
}
function paint() {act(() => {for(const [id, callback] of [...paints]){paints.delete(id);callback(performance.now());}});}
async function mount() {
  const hook = renderHook(({sessionId,pageId}: {sessionId: string;pageId?: string | null}) => useFrameStream({sessionId,pageId}),
    {initialProps:{sessionId:'session-a',pageId:'page-a'} as {sessionId:string;pageId?:string | null}});
  const canvas=document.createElement('canvas'); canvas.width=640;canvas.height=480;
  (hook.result.current.canvasRef as {current:HTMLCanvasElement | null}).current=canvas;
  await act(drain);
  return hook;
}
const polls = () => fetchMock.mock.calls.filter(([url]) => url !== '/config');

// [REQ:BAS-RH-J05] [REQ:BAS-RH-J22] The viewer owns bounded work for one session/page.
describe('live viewer transport and lifetime', () => {
  beforeEach(() => {
    vi.useFakeTimers(); vi.setSystemTime(new Date('2026-09-23T03:00:00Z'));
    sockets.length=0;decodes.length=0;paints.clear();rafId=0;
    draw.mockReset();clear.mockReset();mocks.recordFrame.mockReset();mocks.reset.mockReset();
    mocks.config.mockReset().mockResolvedValue(config);
    useSessionStore.setState({sessionId:null,isValidated:false});
    fetchMock=vi.fn(async (url: string) => url === '/config'
      ? new Response(JSON.stringify(config),{status:200}) : new Response(null,{status:304}));
    vi.stubGlobal('fetch',fetchMock);vi.stubGlobal('WebSocket',Socket);
    vi.stubGlobal('createImageBitmap',vi.fn(() => {const d=deferred<ImageBitmap>();decodes.push(d);return d.promise;}));
    vi.stubGlobal('requestAnimationFrame',(callback: FrameRequestCallback) => {const id=++rafId;paints.set(id,callback);return id;});
    vi.stubGlobal('cancelAnimationFrame',(id:number)=>paints.delete(id));
    vi.spyOn(HTMLCanvasElement.prototype,'getContext').mockReturnValue({drawImage:draw,clearRect:clear} as unknown as CanvasRenderingContext2D);
  });
  afterEach(() => {vi.useRealTimers();vi.unstubAllGlobals();vi.restoreAllMocks();});

  it('subscribes through the configured API route without guessing a driver port',async()=>{
    await mount();expect(sockets[0]?.url).toBe(config.WS_URL);
    act(()=>sockets[0]!.open());
    expect(sockets[0]!.send).toHaveBeenCalledWith(JSON.stringify({type:'subscribe_recording',session_id:'session-a'}));
  });
  it('keeps fallback polling until a usable socket frame is painted',async()=>{
    await mount();act(()=>sockets[0]!.open());const before=polls().length;
    await advance(1000);expect(polls().length).toBeGreaterThan(before);
  });
  it('bounds a hundred-frame decode burst and retains the newest pending image',async()=>{
    await mount();act(()=>{sockets[0]!.open();for(let i=0;i<100;i++)sockets[0]!.frame();});
    expect(decodes.length).toBe(1);
    await resolveDecode(0);expect(decodes.length).toBe(2);
    await resolveDecode(1);paint();expect(draw.mock.calls.at(-1)?.[0].id).toBe(2);
  });
  it('cannot reorder same-millisecond frames when decodes complete in reverse order',async()=>{
    await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();sockets[0]!.frame();});
    if(decodes.length>1){await resolveDecode(1);paint();await resolveDecode(0);paint();}
    else {await resolveDecode(0);paint();await resolveDecode(1);paint();}
    expect(draw.mock.calls.at(-1)?.[0].id).toBe(2);
  });
  it.each(['session','page','unmount'] as const)('disposes a decode completing after %s ownership ends',async(kind)=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});
    if(kind==='unmount')h.unmount();
    else h.rerender({sessionId:kind==='session'?'session-b':'session-a',pageId:kind==='page'?'page-b':'page-a'});
    await act(drain);const old=await resolveDecode(0);paint();
    expect(old.close).toHaveBeenCalled();expect(draw).not.toHaveBeenCalled();expect(paints.size).toBe(0);
  });
  it('bounds decoding across replacement sessions while an old decode is still pending',async()=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});
    h.rerender({sessionId:'session-b',pageId:'page-b'});await act(drain);
    act(()=>{sockets.at(-1)!.open();sockets.at(-1)!.frame('session-b','page-b');});
    expect(decodes).toHaveLength(1);
    const old=await resolveDecode(0);expect(old.close).toHaveBeenCalled();
    expect(decodes).toHaveLength(2);await resolveDecode(1);paint();
    expect(draw.mock.calls.at(-1)?.[0].id).toBe(2);
  });
  it('does not open a socket after a deferred configuration read outlives unmount',async()=>{
    const pending=deferred<typeof config>();mocks.config.mockReturnValue(pending.promise);
    fetchMock.mockImplementation(async(url:string)=>url==='/config'?new Response(JSON.stringify(await pending.promise)):new Response(null,{status:304}));
    const h=await mount();h.unmount();await act(async()=>{pending.resolve(config);await drain();});
    expect(sockets.every(s=>s.close.mock.calls.length>0)).toBe(true);
  });
  it('does not reconnect a disposed socket after switching pages',async()=>{
    const h=await mount();act(()=>sockets[0]!.open());const old=sockets[0]!;
    h.rerender({sessionId:'session-a',pageId:'page-b'});await act(drain);
    const admitted=sockets.length;act(()=>old.onclose?.());await advance(1000);
    expect(sockets).toHaveLength(admitted);
  });
  it('keeps fallback active after an invalid binary image',async()=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});
    await act(async()=>{decodes[0]!.reject(new Error('invalid JPEG'));await drain();});
    const before=polls().length;await advance(1000);
    expect(h.result.current.isWsFrameActive).toBe(false);expect(polls().length).toBeGreaterThan(before);
  });
  it('resumes fallback when a connected stream stops delivering usable frames',async()=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});await resolveDecode(0);paint();
    expect(h.result.current.isWsFrameActive).toBe(true);const before=polls().length;
    await advance(2500);expect(polls().length).toBeGreaterThan(before);expect(h.result.current.isWsFrameActive).toBe(false);
  });
  it('discards a polling response admitted before a newer socket frame',async()=>{
    const pending=deferred<Response>();fetchMock.mockImplementation(async(url:string)=>url==='/config'?new Response(JSON.stringify(config)):pending.promise);
    await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});await resolveDecode(0);paint();
    await act(async()=>{pending.resolve(new Response(JSON.stringify(payload)));await drain();});
    expect(decodes).toHaveLength(1);expect(draw.mock.calls.at(-1)?.[0].id).toBe(1);
  });
  it('does not cache an ETag before its image has decoded successfully',async()=>{
    fetchMock.mockImplementation(async(url:string)=>url==='/config'?new Response(JSON.stringify(config)):
      new Response(JSON.stringify(payload),{headers:{ETag:'"bad-image"'}}));
    await mount();await act(async()=>{decodes[0]!.reject(new Error('bad image'));await drain();});
    await advance(1000);expect(polls().length).toBeGreaterThan(1);
    expect(new Headers(polls().at(-1)?.[1]?.headers).has('If-None-Match')).toBe(false);
  });
  it('paints a source-bearing API frame and counts only JPEG bytes',async()=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});
    await resolveDecode(0);paint();expect(h.result.current.hasFrame).toBe(true);
    expect(h.result.current.isWsFrameActive).toBe(true);expect(mocks.recordFrame).toHaveBeenCalledWith(4);
  });
  it.each(['page','session'] as const)('rejects a valid envelope from a different %s before decoding',async(kind)=>{
    const h=await mount();act(()=>sockets[0]!.frame(kind==='session'?'retired':'session-a',kind==='page'?'retired':'page-a'));
    expect(decodes).toHaveLength(0);expect(h.result.current.error).toBeNull();
    act(()=>sockets[0]!.frame());await resolveDecode(0);paint();expect(h.result.current.hasFrame).toBe(true);
  });
  it.each(['anonymous','timestamp','version','length','identity','date'] as const)('rejects malformed or legacy %s frames and keeps fallback available',async(kind)=>{
    const h=await mount();let data: ArrayBuffer;
    if(kind==='anonymous')data=new Uint8Array([255,216,255,217]).buffer;
    else if(kind==='timestamp'){data=new ArrayBuffer(12);new DataView(data).setBigInt64(0,BigInt(Date.now()));new Uint8Array(data).set([255,216,255,217],8);}
    else {data=framePacket(kind==='version'?{version:2}:kind==='identity'?{page_id:''}:kind==='date'?{captured_at:'invalid'}:{});if(kind==='length')new DataView(data).setUint32(0,65536);}
    act(()=>sockets[0]!.onmessage?.({data}));expect(decodes).toHaveLength(0);
    const before=polls().length;await advance(1000);expect(polls().length).toBeGreaterThan(before);
    expect(h.result.current.isWsFrameActive).toBe(false);
  });
  it.each(['page','session'] as const)('rejects an HTTP response from a different %s before decoding or metadata publication',async(kind)=>{
    fetchMock.mockImplementation(async(url:string)=>url==='/config'?new Response(JSON.stringify(config)):
      new Response(JSON.stringify({...payload,[kind+'_id']:'retired'})));
    const h=await mount();expect(decodes).toHaveLength(0);expect(h.result.current.hasFrame).toBe(false);
  });
  it('uses conditional polling only after a successful image paint',async()=>{
    fetchMock.mockImplementation(async(url:string,init?:RequestInit)=>url==='/config'?new Response(JSON.stringify(config)):
      new Headers(init?.headers).has('If-None-Match')?new Response(null,{status:304}):
      new Response(JSON.stringify(payload),{headers:{ETag:'"good-image"'}}));
    const h=await mount();await resolveDecode(0);paint();await advance(700);
    expect(new Headers(polls().at(-1)?.[1]?.headers).get('If-None-Match')).toBe('"good-image"');
    expect(decodes).toHaveLength(1);expect(h.result.current.hasFrame).toBe(true);
    expect(h.result.current.isWsFrameActive).toBe(false);
  });
  it('ignores data delivered by a socket after its viewer is disposed',async()=>{
    const h=await mount();const socket=sockets[0]!;act(()=>socket.open());h.unmount();
    act(()=>socket.frame());expect(decodes).toHaveLength(0);expect(socket.close).toHaveBeenCalledTimes(1);
  });
  it('does not publish an old HTTP failure into a replacement session',async()=>{
    const pending=deferred<Response>();fetchMock.mockImplementation(async(url:string)=>url==='/config'?new Response(JSON.stringify(config)):
      url.includes('session-a')?pending.promise:new Response(null,{status:304}));
    const h=await mount();h.rerender({sessionId:'session-b',pageId:'page-b'});await act(drain);
    await act(async()=>{pending.resolve(new Response('{}',{status:503}));await drain();});
    expect(h.result.current.error).toBeNull();expect(decodes).toHaveLength(0);
  });
  it('stops the viewer when the final tab closes and resumes for a fresh tab',async()=>{
    const h=await mount();act(()=>{sockets[0]!.open();sockets[0]!.frame();});await resolveDecode(0);paint();
    expect(h.result.current.hasFrame).toBe(true);clear.mockClear();
    const old=sockets[0]!;const count=polls().length;
    await act(async()=>{h.rerender({sessionId:'session-a',pageId:null});await drain();});
    expect(h.result.current.hasFrame).toBe(false);expect(clear).toHaveBeenCalled();
    expect(old.close).toHaveBeenCalledTimes(1);
    act(()=>{old.frame();old.onclose?.();});await advance(15000);
    expect(polls()).toHaveLength(count);expect(sockets).toHaveLength(1);
    expect(h.result.current.error).toBeNull();
    h.rerender({sessionId:'session-a',pageId:'page-new'});await act(drain);
    expect(sockets).toHaveLength(2);expect(polls().at(-1)?.[0]).toContain('page_id=page-new');
    act(()=>{sockets[1]!.open();sockets[1]!.frame('session-a','page-new');});await resolveDecode(1);paint();
    expect(h.result.current.hasFrame).toBe(true);
  });
  it('aborts in-flight polling and discards late decode after the final tab closes',async()=>{
    const pending=deferred<Response>();fetchMock.mockReturnValue(pending.promise);
    const h=await mount();const signal=polls()[0]?.[1]?.signal as AbortSignal;
    act(()=>{sockets[0]!.open();sockets[0]!.frame();});
    h.rerender({sessionId:'session-a',pageId:null});await act(drain);
    expect(signal.aborted).toBe(true);
    await act(async()=>{pending.resolve(new Response(JSON.stringify(payload)));await drain();});
    const old=await resolveDecode(0);paint();await advance(15000);
    expect(old.close).toHaveBeenCalled();expect(draw).not.toHaveBeenCalled();
    expect(polls()).toHaveLength(1);expect(sockets).toHaveLength(1);
    expect(h.result.current.hasFrame).toBe(false);
  });
  it('keeps implicit active-page preview available when page ID is omitted',async()=>{
    const h=await mount();h.rerender({sessionId:'session-a',pageId:undefined});await act(drain);
    expect(sockets).toHaveLength(2);expect(polls().at(-1)?.[0]).not.toContain('page_id=');
    act(()=>{sockets[1]!.open();sockets[1]!.frame();});await resolveDecode(0);paint();
    expect(h.result.current.hasFrame).toBe(true);
  });
});
