import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { createElement } from 'react';
import { WebSocketContext, type WebSocketMessageCallback } from '@/contexts/WebSocketContext';
import { useFrameStream } from './useFrameStream';
import { useSessionStore } from '../stores/sessionStore';

const mocks = vi.hoisted(() => ({
  config: vi.fn(), recordFrame: vi.fn(), reset: vi.fn(),
  stats: {currentFps:0,avgFps:0,lastFrameSize:0,avgFrameSize:0,totalFrames:0,totalBytes:0,bytesPerSecond:0},
}));
vi.mock('@/config', () => ({getConfig:mocks.config}));
vi.mock('../hooks/useFrameStats', () => ({useFrameStats:() => ({stats:mocks.stats,recordFrame:mocks.recordFrame,reset:mocks.reset})}));

function deferred<T>() {
  let resolve!: (value:T) => void;
  let reject!: (reason:Error) => void;
  const promise = new Promise<T>((yes,no) => {resolve=yes;reject=no;});
  return {promise,resolve,reject};
}
const decodes:Array<ReturnType<typeof deferred<ImageBitmap>>> = [];
const sockets:Socket[] = [];
const draw = vi.fn();
const config = {WS_URL:'ws://fixture.test/ws'};

class Socket {
  readyState = 0;
  binaryType = '';
  onopen:(()=>void)|null = null;
  onmessage:((event:{data:ArrayBuffer|string})=>void)|null = null;
  onclose:(()=>void)|null = null;
  onerror:(()=>void)|null = null;
  send = vi.fn();
  close = vi.fn(() => {this.readyState=3;});
  constructor(readonly url:string) {sockets.push(this);}
  open() {this.readyState=1;this.onopen?.();}
  frame(session_id='session-a',page_id='page-a') {this.onmessage?.({data:framePacket({session_id,page_id})});}
}
function framePacket(overrides:Record<string,unknown>={}) {
  const header = new TextEncoder().encode(JSON.stringify({version:1,session_id:'session-a',page_id:'page-a',captured_at:'2026-09-23T03:00:00Z',page_title:'Current',page_url:'https://fixture.test',...overrides}));
  const packet = new Uint8Array(4+header.length+4);
  new DataView(packet.buffer).setUint32(0,header.length);packet.set(header,4);packet.set([255,216,255,217],4+header.length);
  return packet.buffer;
}
function bitmap(id:number,width=640,height=480) {return {id,width,height,close:vi.fn()} as unknown as ImageBitmap & {id:number;close:ReturnType<typeof vi.fn>};}
async function drain() {for(let i=0;i<20;i++)await Promise.resolve();}
async function resolveDecode(index:number) {
  const image=bitmap(index+1);
  await act(async()=>{decodes[index]!.resolve(image);await drain();});
  return image;
}
async function mount() {
  const hook=renderHook(({sessionId,pageId}:{sessionId:string;pageId?:string|null})=>useFrameStream({sessionId,pageId}),
    {initialProps:{sessionId:'session-a',pageId:'page-a'} as {sessionId:string;pageId?:string|null}});
  const canvas=document.createElement('canvas');
  (hook.result.current.canvasRef as {current:HTMLCanvasElement|null}).current=canvas;
  await act(drain);
  return hook;
}

// [REQ:BAS-RH-J05] [REQ:BAS-RH-J22] The viewer owns bounded WebSocket work for one session/page.
describe('live viewer WebSocket transport and lifetime',()=>{
  beforeEach(()=>{
    vi.useFakeTimers();vi.setSystemTime(new Date('2026-09-23T03:00:00Z'));
    sockets.length=0;decodes.length=0;draw.mockReset();mocks.recordFrame.mockReset();mocks.reset.mockReset();
    mocks.config.mockReset().mockResolvedValue(config);useSessionStore.setState({sessionId:null,isValidated:false});
    vi.stubGlobal('fetch',vi.fn());
    vi.stubGlobal('WebSocket',Socket);
    vi.stubGlobal('createImageBitmap',vi.fn(()=>{const d=deferred<ImageBitmap>();decodes.push(d);return d.promise;}));
    vi.spyOn(HTMLCanvasElement.prototype,'getContext').mockReturnValue({drawImage:draw,clearRect:vi.fn()} as unknown as CanvasRenderingContext2D);
  });
  afterEach(()=>{vi.useRealTimers();vi.unstubAllGlobals();vi.restoreAllMocks();});

  it('subscribes through the configured API socket and never fetches screenshots',async()=>{
    await mount();expect(sockets[0]?.url).toBe(config.WS_URL);
    act(()=>sockets[0]!.open());
    expect(sockets[0]!.send).toHaveBeenCalledWith(JSON.stringify({type:'subscribe_recording',session_id:'session-a'}));
    expect(fetch).not.toHaveBeenCalled();
    act(()=>sockets[0]!.onclose?.());
    await act(async()=>{await vi.advanceTimersByTimeAsync(250);});
    expect(sockets).toHaveLength(2);
    expect(fetch).not.toHaveBeenCalled();
  });
  it('bounds a burst to one active decode and the newest pending frame',async()=>{
    await mount();act(()=>{sockets[0]!.open();for(let i=0;i<100;i++)sockets[0]!.frame();});
    expect(decodes).toHaveLength(1);
    const stale=await resolveDecode(0);expect(stale.close).toHaveBeenCalledTimes(1);
    expect(decodes).toHaveLength(2);await resolveDecode(1);
    expect(draw).toHaveBeenCalledTimes(1);expect(mocks.recordFrame).toHaveBeenCalledWith(4);
  });
  it('rejects frames from another session or page before decoding',async()=>{
    const hook=await mount();act(()=>sockets[0]!.frame('retired','page-a'));
    expect(decodes).toHaveLength(0);act(()=>sockets[0]!.frame());await resolveDecode(0);
    expect(hook.result.current.hasFrame).toBe(true);
  });
  it('reconnects after socket loss and ignores disposed sockets',async()=>{
    const hook=await mount();act(()=>{sockets[0]!.open();sockets[0]!.onclose?.();});
    await act(async()=>{await vi.advanceTimersByTimeAsync(250);});
    expect(sockets).toHaveLength(2);
    const old=sockets[0]!;hook.rerender({sessionId:'session-a',pageId:'page-b'});await act(drain);
    const count=sockets.length;act(()=>old.onclose?.());
    await act(async()=>{await vi.advanceTimersByTimeAsync(500);});
    expect(sockets).toHaveLength(count);
  });
  it('routes execution frames through the same bounded decode and paint path',async()=>{
    const listeners:WebSocketMessageCallback[]=[];const send=vi.fn(()=>true);
    const context={isConnected:true,send,subscribe:vi.fn(),unsubscribe:vi.fn(),reconnect:vi.fn(),
      subscribeToMessages:(callback:WebSocketMessageCallback)=>{listeners.push(callback);return ()=>{};}};
    const wrapper=({children}:{children:React.ReactNode})=>createElement(WebSocketContext.Provider,{value:context},children);
    const hook=renderHook(()=>useFrameStream({sessionId:null,executionId:'execution-a'}),{wrapper});
    expect(send).toHaveBeenCalledWith({type:'subscribe_execution_frames',execution_id:'execution-a'});
    act(()=>listeners.forEach(listener=>listener({type:'execution_frame',execution_id:'execution-a',data:'/9j/2Q==',media_type:'image/jpeg',width:640,height:480,captured_at:'2026-09-23T03:00:00Z'})));
    await resolveDecode(0);
    expect(hook.result.current.hasFrame).toBe(true);expect(hook.result.current.frameCount).toBe(1);
    expect(hook.result.current.frameUrl).toBe('data:image/jpeg;base64,/9j/2Q==');
    hook.unmount();expect(send).toHaveBeenCalledWith({type:'unsubscribe_execution_frames'});
  });
});
