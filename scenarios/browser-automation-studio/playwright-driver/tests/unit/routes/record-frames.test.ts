import {
  handleRecordFrame,
  handleRecordScreenshot,
  clearFrameCache,
  clearAllFrameCaches,
} from '../../../src/routes/record-mode/recording-frames';
import { createMockHttpRequest, createMockHttpResponse, createMockPage, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session';
import { RECORDING_FRAME_CACHE_TTL_MS } from '../../../src/constants';

describe('recording frame routes', () => {
  const config = createTestConfig();
  let mockPage: ReturnType<typeof createMockPage>;
  let sessionManager: Pick<SessionManager, 'getSession'>;
  let nowSpy: jest.SpyInstance<number, []>;

  beforeEach(() => {
    mockPage = createMockPage({
      screenshot: jest.fn().mockResolvedValue(Buffer.from('frame-1')),
      viewportSize: jest.fn().mockReturnValue({ width: 1024, height: 768 }),
      title: jest.fn().mockResolvedValue('Test Page'),
      url: jest.fn().mockReturnValue('https://example.com'),
    });
    sessionManager = {
      getSession: () => ({ id:'session-1',ownerExecutionId:'execution-a',leaseId:'lease-a',page:mockPage,
        spec:{execution_id:'execution-a',workflow_id:'fixture',reuse_mode:'fresh',viewport:{width:1024,height:768}},
        pageToIdMap:new WeakMap([[mockPage,'page-a']]) } as ReturnType<SessionManager['getSession']>),
    };
    nowSpy = jest.spyOn(Date, 'now').mockReturnValue(1000);
  });

  afterEach(() => {
    nowSpy.mockRestore();
    clearFrameCache('session-1');
    clearAllFrameCaches();
  });

  it('captures screenshot for recording screenshot endpoint', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/session-1/record/screenshot',
      body: { full_page: false, quality: 70 },
    });
    const res = createMockHttpResponse();

    await handleRecordScreenshot(req, res, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledWith({
      fullPage: false,
      type: 'jpeg',
      quality: 70,
    });
    expect(res.statusCode).toBe(200);
  });

  it('shares one in-flight capture across eight identical preview reads [REQ:BAS-RH-J05]', async () => {
    let finish!: (buffer: Buffer) => void;
    const pending = new Promise<Buffer>(resolve => { finish = resolve; });
    mockPage.screenshot.mockReturnValue(pending);
    const responses = Array.from({ length: 8 }, () => createMockHttpResponse());
    const work = responses.map(res => handleRecordFrame(createMockHttpRequest({
      method: 'GET', url: '/session/session-1/record/frame?quality=60',
    }), res, 'session-1', sessionManager as SessionManager, config));
    await new Promise<void>(resolve => setImmediate(resolve));
    const admitted = mockPage.screenshot.mock.calls.length;
    finish(Buffer.from('one-independent-capture'));
    await Promise.all(work);
    expect(responses.map(res => res.statusCode)).toEqual(Array(8).fill(200));
    expect(responses.map(res => res.getJSON().image)).toEqual(Array(8).fill('data:image/jpeg;base64,' + Buffer.from('one-independent-capture').toString('base64')));
    expect(admitted).toBe(1);
  });

  it('does not reuse cached dimensions after the viewport changes within the TTL [REQ:BAS-RH-J05]', async () => {
    const request = () => createMockHttpRequest({ method: 'GET', url: '/session/session-1/record/frame?quality=60' });
    const first = createMockHttpResponse();
    await handleRecordFrame(request(), first, 'session-1', sessionManager as SessionManager, config);
    mockPage.viewportSize.mockReturnValue({ width: 800, height: 600 });
    mockPage.screenshot.mockResolvedValue(Buffer.from('resized-document'));
    const resized = createMockHttpResponse();
    await handleRecordFrame(request(), resized, 'session-1', sessionManager as SessionManager, config);
    expect(resized.statusCode).toBe(200);
    expect(resized.getJSON()).toMatchObject({ width: 800, height: 600 });
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
  });

  it('cannot repopulate a cleared cache from an older pending capture [REQ:BAS-RH-J22]', async () => {
    let finish!: (buffer: Buffer) => void;
    mockPage.screenshot.mockImplementationOnce(() => new Promise<Buffer>(resolve => { finish = resolve; }));
    const request = () => createMockHttpRequest({ method: 'GET', url: '/session/session-1/record/frame' });
    const old = createMockHttpResponse();
    const pending = handleRecordFrame(request(), old, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    clearFrameCache('session-1');
    finish(Buffer.from('retired-capture'));
    await pending;
    const next = createMockHttpResponse();
    await handleRecordFrame(request(), next, 'session-1', sessionManager as SessionManager, config);
    expect(old.statusCode).toBe(409);
    expect(next.getJSON().image).toBe('data:image/jpeg;base64,' + Buffer.from('frame-1').toString('base64'));
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
  });

  it.each(['viewport', 'url', 'clearAll'] as const)('rejects capture retired by %s before completion', async (boundary) => {
    let finish!: (buffer: Buffer) => void;
    mockPage.screenshot.mockImplementationOnce(() => new Promise<Buffer>(resolve => { finish = resolve; }));
    const request = () => createMockHttpRequest({ method: 'GET', url: '/session/session-1/record/frame' });
    const old = createMockHttpResponse();
    const pending = handleRecordFrame(request(), old, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    if (boundary === 'viewport') mockPage.viewportSize.mockReturnValue({ width: 800, height: 600 });
    if (boundary === 'url') mockPage.url.mockReturnValue('https://example.com/new-document');
    if (boundary === 'clearAll') clearAllFrameCaches();
    finish(Buffer.from('retired-capture'));
    await pending;
    const next = createMockHttpResponse();
    await handleRecordFrame(request(), next, 'session-1', sessionManager as SessionManager, config);
    expect(old.statusCode).toBe(409);
    expect(next.statusCode).toBe(200);
    expect(next.getJSON().image).toBe('data:image/jpeg;base64,' + Buffer.from('frame-1').toString('base64'));
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
  });

  it.each(['rejection', 'throw'] as const)('does not cache a capture %s', async (failure) => {
    mockPage.screenshot.mockImplementationOnce(() => {
      if (failure === 'throw') throw new Error('synthetic capture failure');
      return Promise.reject(new Error('synthetic capture failure'));
    });
    const request = () => createMockHttpRequest({ method: 'GET', url: '/session/session-1/record/frame' });
    const failed = createMockHttpResponse();
    await handleRecordFrame(request(), failed, 'session-1', sessionManager as SessionManager, config);
    expect(failed.statusCode).toBe(500);
    const next = createMockHttpResponse();
    await handleRecordFrame(request(), next, 'session-1', sessionManager as SessionManager, config);
    expect(next.statusCode).toBe(200);
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
  });

  it('keeps different fidelity results distinct and retains the newer result after out-of-order completion', async () => {
    const finish: Array<(buffer: Buffer) => void> = [];
    mockPage.screenshot.mockImplementation(() => new Promise<Buffer>(resolve => { finish.push(resolve); }));
    const request = (quality: number) => createMockHttpRequest({ method: 'GET', url: `/session/session-1/record/frame?quality=${quality}` });
    const older = createMockHttpResponse(), newer = createMockHttpResponse();
    const low = handleRecordFrame(request(60), older, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    const high = handleRecordFrame(request(80), newer, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    expect(mockPage.screenshot.mock.calls.map(([options]) => options.quality)).toEqual([60, 80]);
    finish[1](Buffer.from('high-fidelity')); await high;
    finish[0](Buffer.from('low-fidelity')); await low;
    const retained = createMockHttpResponse();
    await handleRecordFrame(request(80), retained, 'session-1', sessionManager as SessionManager, config);
    expect(older.getJSON().image).not.toEqual(newer.getJSON().image);
    expect(retained.getJSON().image).toEqual(newer.getJSON().image);
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
  });

  it('an older failure cannot evict a newer pending capture', async () => {
    let fail!: (error: Error) => void, finish!: (buffer: Buffer) => void;
    mockPage.screenshot.mockImplementationOnce(() => new Promise<Buffer>((_, reject) => { fail = reject; }))
      .mockImplementationOnce(() => new Promise<Buffer>(resolve => { finish = resolve; }));
    const request = (quality: number) => createMockHttpRequest({ method: 'GET', url: `/session/session-1/record/frame?quality=${quality}` });
    const old = createMockHttpResponse(), current = createMockHttpResponse(), joined = createMockHttpResponse();
    const older = handleRecordFrame(request(60), old, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    const newer = handleRecordFrame(request(80), current, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    fail(new Error('retired failure')); await older;
    const follower = handleRecordFrame(request(80), joined, 'session-1', sessionManager as SessionManager, config);
    await new Promise<void>(resolve => setImmediate(resolve));
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
    finish(Buffer.from('current-capture')); await Promise.all([newer, follower]);
    expect(old.statusCode).toBe(500);
    expect(current.statusCode).toBe(200); expect(joined.statusCode).toBe(200);
    expect(current.getJSON().image).toEqual(joined.getJSON().image);
  });

  it('caches frame responses within TTL', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=false',
    });
    const res1 = createMockHttpResponse();
    const res2 = createMockHttpResponse();

    await handleRecordFrame(req, res1, 'session-1', sessionManager as SessionManager, config);
    nowSpy.mockReturnValue(1000 + RECORDING_FRAME_CACHE_TTL_MS - 1);
    await handleRecordFrame(req, res2, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledTimes(1);
    expect(res2.getJSON().content_hash).toBe(res1.getJSON().content_hash);
  });

  it('reuses cached data when content hash is unchanged after TTL', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=false',
    });
    const res1 = createMockHttpResponse();
    const res2 = createMockHttpResponse();

    await handleRecordFrame(req, res1, 'session-1', sessionManager as SessionManager, config);

    nowSpy.mockReturnValue(1000 + RECORDING_FRAME_CACHE_TTL_MS + 1);
    await handleRecordFrame(req, res2, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
    expect(res2.getJSON().image).toBe(res1.getJSON().image);
  });

  it('does not reuse a same-page cached frame across lease replacement', async () => {
    const session=sessionManager.getSession('session-1');sessionManager.getSession=()=>session;
    const req=createMockHttpRequest({method:'GET',url:'/session/session-1/record/frame'});
    const first=createMockHttpResponse();await handleRecordFrame(req,first,'session-1',sessionManager as SessionManager,config);
    session.leaseId='lease-b';mockPage.screenshot.mockResolvedValue(Buffer.from('new-owner-frame'));
    const next=createMockHttpResponse();await handleRecordFrame(req,next,'session-1',sessionManager as SessionManager,config);
    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);expect(next.getJSON().source.lease_id).toBe('lease-b');
    expect(next.getJSON().image).not.toBe(first.getJSON().image);
  });

  it.each([undefined, 'css', 'device'] as const)(
    'captures HTTP preview with the admitted %s scale [REQ:BAS-RH-J23]',
    async (scale) => {
      const session = sessionManager.getSession('session-1');
      session.spec.frame_scale = scale;
      sessionManager.getSession = () => session;
      const response = createMockHttpResponse();
      await handleRecordFrame(createMockHttpRequest({
        method: 'GET', url: '/session/session-1/record/frame',
      }), response, 'session-1', sessionManager as SessionManager, config);
      expect(response.statusCode).toBe(200);
      expect(mockPage.screenshot).toHaveBeenCalledWith(expect.objectContaining({ scale: scale ?? 'css' }));
      expect(response.getJSON()).toMatchObject({ width: 1024, height: 768 });
    },
  );

  it('uses fullPage capture when requested', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=true',
    });
    const res = createMockHttpResponse();

    await handleRecordFrame(req, res, 'session-1', sessionManager as SessionManager, config);

    const [options] = mockPage.screenshot.mock.calls[0] ?? [];
    expect(options.fullPage).toBe(true);
    expect(options.clip).toBeUndefined();
  });

  // [REQ:BAS-RH-J05] [REQ:BAS-RH-J22] A preview belongs to its producing page.
  it('rejects a requested page that is no longer active before taking a screenshot', async () => {
    const session = {id:'session-1',ownerExecutionId:'execution-a',leaseId:'lease-a',page:mockPage,
        spec:{execution_id:'execution-a',workflow_id:'fixture',reuse_mode:'fresh',viewport:{width:1024,height:768}},
      pageToIdMap:new WeakMap([[mockPage,'page-a']])} as ReturnType<SessionManager['getSession']>;
    sessionManager.getSession=()=>session;
    const req=createMockHttpRequest({method:'GET',url:'/session/session-1/record/frame?page_id=old-page'});
    const res=createMockHttpResponse();
    await handleRecordFrame(req,res,'session-1',sessionManager as SessionManager,config);
    expect(res.statusCode).toBe(409);expect(mockPage.screenshot).not.toHaveBeenCalled();
  });

  it.each(['response','cache'])('fences the %s when capture completes after switching pages', async (boundary) => {
    let finish!: (value:Buffer)=>void;
    let admitted!: ()=>void;
    const started=new Promise<void>(resolve=>{admitted=resolve;});
    mockPage.screenshot.mockImplementation(()=>{admitted();return new Promise<Buffer>(resolve=>{finish=resolve;});});
    const blue=createMockPage({screenshot:jest.fn().mockResolvedValue(Buffer.from('blue-frame')),
      title:jest.fn().mockResolvedValue('Blue'),url:jest.fn().mockReturnValue('https://fixture.test/blue'),
      viewportSize:jest.fn().mockReturnValue({width:1024,height:768})});
    const session={id:'session-1',ownerExecutionId:'execution-a',leaseId:'lease-a',page:mockPage,
        spec:{execution_id:'execution-a',workflow_id:'fixture',reuse_mode:'fresh',viewport:{width:1024,height:768}},
      pageToIdMap:new WeakMap([[mockPage,'page-a'],[blue,'page-b']])} as ReturnType<SessionManager['getSession']>;
    sessionManager.getSession=()=>session;
    const request=(pageId:string)=>createMockHttpRequest({method:'GET',url:`/session/session-1/record/frame?page_id=${pageId}`});
    const old=createMockHttpResponse();const pending=handleRecordFrame(request('page-a'),old,'session-1',sessionManager as SessionManager,config);
    await started;session.page=blue;clearFrameCache('session-1');finish(Buffer.from('red-frame'));await pending;
    if (boundary === 'response') { expect(old.statusCode).toBe(409); return; }
    const current=createMockHttpResponse();await handleRecordFrame(request('page-b'),current,'session-1',sessionManager as SessionManager,config);
    expect(current.statusCode).toBe(200);expect(blue.screenshot).toHaveBeenCalledTimes(1);
    expect(current.getJSON().image).toBe('data:image/jpeg;base64,'+Buffer.from('blue-frame').toString('base64'));
    expect(current.getJSON().page_url).toBe('https://fixture.test/blue');
  });
});
