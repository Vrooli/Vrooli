import { EventEmitter } from 'node:events';
import { createServer } from 'node:http';
import { dirname, join } from 'node:path';
import { readFile, rm } from 'node:fs/promises';
import { cleanupSession } from '../../src/infra/session-cleanup-registry';
import {
  chromium,
  type Browser,
  type BrowserContext,
  type Page,
  type CDPSession,
} from 'rebrowser-playwright';
import { create, toJson } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  ClickParamsSchema,
  SelectParamsSchema,
  CookieStorageParamsSchema,
  CookieOperation,
  StorageType,
  DragDropParamsSchema,
  DownloadParamsSchema,
  FrameSwitchParamsSchema,
  TabSwitchParamsSchema,
  TabSwitchAction,
  FrameSwitchAction,
  KeyboardParamsSchema,
  InputParamsSchema,
  ScrollParamsSchema,
  KeyboardModifier,
  MouseButton,
  KeyAction,
  ScrollBehavior,
  type ActionDefinition,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import {
  WaitHandler,
  AssertionHandler,
  ExtractionHandler,
  SelectHandler,
  UploadHandler,
  TabHandler,
  CookieStorageHandler,
  GestureHandler,
  DownloadHandler,
} from '../../src/handlers';
import { createTypedInstruction } from '../helpers/instruction-factory';
import { AssertionMode } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { FrameHandler } from '../../src/handlers/frame';
import { InteractionHandler } from '../../src/handlers/interaction';
import { KeyboardHandler } from '../../src/handlers/keyboard';
import { ScrollHandler } from '../../src/handlers/scroll';
import type { HandlerContext } from '../../src/handlers/base';
import type { HandlerInstruction } from '../../src/types';
import { createTestConfig, createMockHttpRequest, createMockHttpResponse } from '../helpers';
import { SessionManager } from '../../src/session/manager';
import { HandlerRegistry } from '../../src/handlers/registry';
import { handleSessionRun } from '../../src/routes/session-run';
import type { SessionState } from '../../src/types';
import { BEHAVIOR_SETTINGS_KEY } from '../../src/browser-profile';
import { logger, metrics } from '../../src/utils';

const fixture = `<!doctype html><style>
#pane {width:180px;height:100px;overflow:scroll} #space {width:1400px;height:1000px}
body {margin:0} #outside {width:2400px;height:1800px}
</style><button id="button">Event target</button>
<form id="form"><input id="input" value="original"><button>Submit</button></form>
<textarea id="multiline">one\ntwo</textarea>
<div id="pane"><div id="space">Scroll target</div></div><div id="outside"></div>
<script>
window.fixtureEvents=[];
for(const type of ['click','dblclick','contextmenu','mousedown','mouseup','keydown','keyup','input','submit']) {
 document.addEventListener(type,event=>{
  window.fixtureEvents.push({type,target:event.target.id,key:event.key,detail:event.detail,
   ctrl:event.ctrlKey,shift:event.shiftKey,button:event.button,at:performance.now()});
  if(type==='submit'||type==='contextmenu')event.preventDefault();
 });
}
</script>`;

// [REQ:BAS-RH-J13] Browser effects are observed independently of handler output.
describe('typed browser action semantics', () => {
  let browser: Browser;
  let browserContext: BrowserContext;
  let page: Page;
  let context: HandlerContext;
  let interaction: InteractionHandler;
  let keyboard: KeyboardHandler;
  const instruction = (action: ActionDefinition): HandlerInstruction => ({
    index: 0,
    nodeId: 'fixture',
    action,
  });
  const events = () => page.evaluate('window.fixtureEvents');

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
  });
  beforeEach(async () => {
    browserContext = await browser.newContext({ viewport: { width: 800, height: 600 } });
    page = await browserContext.newPage();
    await page.goto(`data:text/html,${encodeURIComponent(fixture)}`);
    context = {
      page,
      browserContext,
      config: createTestConfig({ execution: { defaultTimeoutMs: 1000 } }),
      logger,
      metrics,
      sessionId: 'synthetic-semantics',
    };
    interaction = new InteractionHandler();
    keyboard = new KeyboardHandler();
  });
  afterEach(async () => {
    await browserContext?.close();
  });
  afterAll(async () => {
    await browser?.close();
  });

  it('persists an attachment downloaded directly by URL', async () => {
    let requests = 0;
    const bytes = 'independent URL download bytes';
    await page.route('http://download-fixture.test/file', async (route) => {
      requests++;
      await route.fulfill({
        contentType: 'application/octet-stream',
        headers: { 'content-disposition': 'attachment; filename="url-fixture.txt"' },
        body: bytes,
      });
    });
    const result = await new DownloadHandler().execute(
      createTypedInstruction('download', {
        url: 'http://download-fixture.test/file',
        timeoutMs: 2000,
      }),
      context
    );
    const saved = result.extracted_data?.download_path as string | undefined;
    try {
      expect(result.success).toBe(true);
      expect(requests).toBe(1);
      expect(result.extracted_data?.filename).toBe('url-fixture.txt');
      expect(await readFile(saved!, 'utf8')).toBe(bytes);
    } finally {
      if (saved) await rm(saved, { force: true });
    }
  });

  it.each(['html', 'aborted'])(
    'requires a download event after URL navigation: %s',
    async (kind) => {
      await page.route('http://download-fixture.test/no-file', async (route) => {
        if (kind === 'aborted') await route.abort('aborted');
        else
          await route.fulfill({ contentType: 'text/html', body: '<title>No attachment</title>' });
      });
      const result = await new DownloadHandler().execute(
        createTypedInstruction('download', {
          url: 'http://download-fixture.test/no-file',
          timeoutMs: 200,
        }),
        context
      );
      expect(result.success).toBe(false);
      expect(result.extracted_data?.download_path).toBeUndefined();
    }
  );

  // [REQ:BAS-RH-J07] [REQ:BAS-RH-J17] The network fixture owns the effect log.
  it('dispatches arbitrary JavaScript once when its effect is followed by navigation', async () => {
    const effects: string[] = [];
    await page.route('http://evaluate-fixture.test/**', async (route) => {
      if (route.request().method() === 'POST') {
        effects.push(route.request().url());
        await route.fulfill({ status: 204 });
      } else {
        await route.fulfill({
          contentType: 'text/html',
          body: '<!doctype html><title>After navigation</title>',
        });
      }
    });
    await page.goto('http://evaluate-fixture.test/first');
    const handler = new ExtractionHandler();
    const result = await handler.execute(
      createTypedInstruction('evaluate', {
        expression: `(async () => {
          await fetch('/effect', { method: 'POST' });
          location.href = '/next';
          await new Promise(() => {});
        })()`,
      }),
      context
    );
    expect({ effects, error: result.error }).toEqual({
      effects: ['http://evaluate-fixture.test/effect'],
      error: expect.objectContaining({ retryable: false }),
    });
    expect(result.success).toBe(false);
    expect(result.error?.retryable).toBe(false);
    await page.waitForURL('http://evaluate-fixture.test/next');
    const next = await handler.execute(
      createTypedInstruction('evaluate', { expression: 'document.title' }),
      context
    );
    expect(next.success).toBe(true);
    expect(next.extracted_data?.result).toBe('After navigation');
  });

  it('delivers a Control double-click and releases its temporary modifier', async () => {
    const result = await interaction.execute(
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.CLICK,
          params: {
            case: 'click',
            value: create(ClickParamsSchema, {
              selector: '#button',
              clickCount: 2,
              modifiers: [KeyboardModifier.CTRL],
              delayMs: 40,
            }),
          },
        })
      ),
      context
    );
    expect(result.success).toBe(true);
    const log = await events();
    expect(
      log
        .filter((e: { type: string }) => e.type === 'click')
        .map((e: { detail: number; ctrl: boolean }) => [e.detail, e.ctrl])
    ).toEqual([
      [1, true],
      [2, true],
    ]);
    expect(log.filter((e: { type: string }) => e.type === 'dblclick')).toHaveLength(1);
    const down = log.find((e: { type: string }) => e.type === 'mousedown');
    const up = log.find((e: { type: string }) => e.type === 'mouseup');
    expect(up.at - down.at).toBeGreaterThanOrEqual(20);
    await page.click('#button');
    expect((await events()).filter((e: { type: string }) => e.type === 'click').at(-1).ctrl).toBe(
      false
    );
  });

  it('uses the declared right mouse button', async () => {
    const result = await interaction.execute(
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.CLICK,
          params: {
            case: 'click',
            value: create(ClickParamsSchema, { selector: '#button', button: MouseButton.RIGHT }),
          },
        })
      ),
      context
    );
    expect(result.success).toBe(true);
    expect(
      (await events())
        .filter((e: { type: string }) => e.type === 'contextmenu')
        .map((e: { button: number }) => e.button)
    ).toEqual([2]);
  });

  it('applies Control-A and releases Control after keyboard failure', async () => {
    await page.focus('#input');
    const action = create(ActionDefinitionSchema, {
      type: ActionType.KEYBOARD,
      params: {
        case: 'keyboard',
        value: create(KeyboardParamsSchema, { key: 'a', modifiers: [KeyboardModifier.CTRL] }),
      },
    });
    expect((await keyboard.execute(instruction(action), context)).success).toBe(true);
    expect(
      await page
        .locator('#input')
        .evaluate((element: HTMLInputElement) => [element.selectionStart, element.selectionEnd])
    ).toEqual([0, 8]);
    action.params = {
      case: 'keyboard',
      value: create(KeyboardParamsSchema, {
        key: 'NotARealKey',
        modifiers: [KeyboardModifier.CTRL],
      }),
    };
    expect((await keyboard.execute(instruction(action), context)).success).toBe(false);
    await page.keyboard.press('x');
    expect(
      (await events())
        .filter((e: { type: string; key: string }) => e.type === 'keydown' && e.key === 'x')
        .at(-1).ctrl
    ).toBe(false);
  });

  it('keeps an explicitly held modifier through a temporary chord', async () => {
    await page.focus('#input');
    const key = (value: string, action: KeyAction, modifiers: KeyboardModifier[] = []) =>
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.KEYBOARD,
          params: {
            case: 'keyboard',
            value: create(KeyboardParamsSchema, { key: value, action, modifiers }),
          },
        })
      );
    expect((await keyboard.execute(key('Control', KeyAction.DOWN), context)).success).toBe(true);
    expect(
      (await keyboard.execute(key('a', KeyAction.PRESS, [KeyboardModifier.CTRL]), context)).success
    ).toBe(true);
    await page.keyboard.press('x');
    expect(
      (await events())
        .filter((e: { type: string; key: string }) => e.type === 'keydown' && e.key === 'x')
        .at(-1).ctrl
    ).toBe(true);
    expect((await keyboard.execute(key('Control', KeyAction.UP), context)).success).toBe(true);
  });

  it('appends when requested, replaces with empty input, and submits once', async () => {
    const input = (value: string, clearFirst: boolean, submit = false) =>
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.INPUT,
          params: {
            case: 'input',
            value: create(InputParamsSchema, {
              selector: '#input',
              value,
              clearFirst,
              submit,
              delayMs: 5,
            }),
          },
        })
      );
    expect((await interaction.execute(input('-suffix', false), context)).success).toBe(true);
    expect(await page.inputValue('#input')).toBe('original-suffix');
    expect((await interaction.execute(input('', true, true), context)).success).toBe(true);
    expect(await page.inputValue('#input')).toBe('');
    expect((await events()).filter((e: { type: string }) => e.type === 'submit')).toHaveLength(1);
  });

  it('scrolls the target element and keeps omitted axes during relative updates', async () => {
    const scroll = new ScrollHandler();
    const first = instruction(
      create(ActionDefinitionSchema, {
        type: ActionType.SCROLL,
        params: {
          case: 'scroll',
          value: create(ScrollParamsSchema, { selector: '#pane', x: 200, y: 300 }),
        },
      })
    );
    const firstResult = await scroll.execute(first, context);
    expect(firstResult.error).toBeUndefined();
    expect(firstResult.success).toBe(true);
    const read = () =>
      page.locator('#pane').evaluate((element) => [element.scrollLeft, element.scrollTop]);
    expect(await read()).toEqual([200, 300]);
    const next = instruction(
      create(ActionDefinitionSchema, {
        type: ActionType.SCROLL,
        params: {
          case: 'scroll',
          value: create(ScrollParamsSchema, { selector: '#pane', deltaY: 50 }),
        },
      })
    );
    expect((await scroll.execute(next, context)).success).toBe(true);
    expect(await read()).toEqual([200, 350]);
    expect(await page.evaluate(() => [window.scrollX, window.scrollY])).toEqual([0, 0]);
  });
  it.each(['smooth', 'stepped'] as const)(
    'reaches the exact bounded element destination with %s scrolling',
    async (style) => {
      if (style === 'stepped')
        Object.assign(browserContext, {
          [BEHAVIOR_SETTINGS_KEY]: {
            scroll_style: 'stepped',
            scroll_speed_min: 500,
            scroll_speed_max: 500,
            click_delay_min: 0,
            click_delay_max: 0,
            micro_pause_enabled: false,
          },
        });
      context.config.execution.defaultTimeoutMs = 3000;
      const action = create(ActionDefinitionSchema, {
        type: ActionType.SCROLL,
        params: {
          case: 'scroll',
          value: create(ScrollParamsSchema, {
            selector: '#pane',
            x: 99999,
            y: 99999,
            behavior: style === 'smooth' ? ScrollBehavior.SMOOTH : undefined,
          }),
        },
      });
      const result = await new ScrollHandler().execute(instruction(action), context);
      expect(result.error).toBeUndefined();
      expect(result.success).toBe(true);
      expect(await page.locator('#pane').evaluate((e) => [e.scrollLeft, e.scrollTop])).toEqual([
        1220, 900,
      ]);
    }
  );

  it('updates viewport deltas without resetting the other axis', async () => {
    await page.evaluate(() => window.scrollTo(100, 200));
    const action = create(ActionDefinitionSchema, {
      type: ActionType.SCROLL,
      params: { case: 'scroll', value: create(ScrollParamsSchema, { deltaY: 75 }) },
    });
    const result = await new ScrollHandler().execute(instruction(action), context);
    expect(result.error).toBeUndefined();
    expect(result.success).toBe(true);
    expect(await page.evaluate(() => [window.scrollX, window.scrollY])).toEqual([100, 275]);
  });

  it('appends after the whole multiline value', async () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.INPUT,
      params: {
        case: 'input',
        value: create(InputParamsSchema, {
          selector: '#multiline',
          value: '-suffix',
          clearFirst: false,
        }),
      },
    });
    const result = await interaction.execute(instruction(action), context);
    expect(result.error).toBeUndefined();
    expect(await page.inputValue('#multiline')).toBe('one\ntwo-suffix');
  });
  // [REQ:BAS-RH-J03] The parent fixture receives a document-specific effect
  // marker; response metadata is not the action oracle.
  it('frame-switch makes later public instructions affect only the selected document', async () => {
    await page.evaluate(() => {
      (window as unknown as { frameEffects: string[] }).frameEffects = [];
      window.addEventListener('message', (event) => {
        if (typeof event.data?.fixtureFrame === 'string') {
          (window as unknown as { frameEffects: string[] }).frameEffects.push(
            event.data.fixtureFrame
          );
        }
      });
      for (const id of ['left', 'right']) {
        const frame = document.createElement('iframe');
        frame.id = id;
        frame.srcdoc = `<button id="button" onclick="parent.postMessage({fixtureFrame:'${id}'},'*')">Frame ${id}</button>`;
        document.body.prepend(frame);
      }
    });
    await page.frameLocator('#left').locator('#button').waitFor();
    await page.frameLocator('#right').locator('#button').waitFor();
    const config = createTestConfig({ execution: { defaultTimeoutMs: 1000 } });
    const manager = new SessionManager(config);
    const session = {
      id: 'frame-effect-fixture',
      phase: 'ready',
      browser,
      context: browserContext,
      page,
      pages: [page],
      currentPageIndex: 0,
      frameStack: [],
      ownerExecutionId: 'owner',
      leaseId: 'lease',
      spec: { execution_id: 'owner', reuse_mode: 'fresh' },
      createdAt: new Date(),
      lastUsedAt: new Date(),
      instructionCount: 0,
      instructionReceipts: new Map(),
      lastInstructionSequence: 0,
    } as unknown as SessionState;
    Reflect.set(manager, 'sessions', new Map([[session.id, session]]));
    const registry = new HandlerRegistry();
    for (const handler of [
      new FrameHandler(),
      interaction,
      keyboard,
      new ScrollHandler(),
      new WaitHandler(),
      new AssertionHandler(),
      new ExtractionHandler(),
      new SelectHandler(),
      new UploadHandler(),
      new TabHandler(),
    ])
      registry.register(handler);
    let sequence = 0;
    const run = async (action: ActionDefinition, expectedSuccess = true) => {
      const response = createMockHttpResponse();
      const operation = ++sequence;
      await handleSessionRun(
        createMockHttpRequest({
          body: {
            execution_id: 'owner',
            lease_id: 'lease',
            operation_sequence: operation,
            invocation_id: `frame-${operation}`,
            attempt: 1,
            instruction: {
              index: operation,
              nodeId: `frame-${operation}`,
              action: toJson(ActionDefinitionSchema, action),
            },
          },
        }),
        response,
        session.id,
        manager,
        registry,
        config,
        logger,
        metrics
      );
      expect(response.statusCode).toBe(200);
      const result = response.getJSON();
      expect({
        type: action.type,
        success: result.success ?? false,
        failure: result.failure,
      }).toMatchObject({ success: expectedSuccess });
      return response.getJSON();
    };
    const switchFrame = (action: FrameSwitchAction, selector?: string) =>
      run(
        create(ActionDefinitionSchema, {
          type: ActionType.FRAME_SWITCH,
          params: {
            case: 'frameSwitch',
            value: create(FrameSwitchParamsSchema, { action, selector, timeoutMs: 1000 }),
          },
        })
      );
    const click = () =>
      run(
        create(ActionDefinitionSchema, {
          type: ActionType.CLICK,
          params: {
            case: 'click',
            value: create(ClickParamsSchema, { selector: '#button', timeoutMs: 1000 }),
          },
        })
      );
    await switchFrame(FrameSwitchAction.ENTER, '#left');
    await click();
    expect({
      frames: await page.evaluate('window.frameEffects'),
      mainClicks: (await events()).filter((event: { type: string }) => event.type === 'click')
        .length,
    }).toEqual({ frames: ['left'], mainClicks: 0 });
    const left = session.frameStack[0]!;
    await left.evaluate(() => {
      document.body.insertAdjacentHTML(
        'beforeend',
        '<input id="input" value="child-original"><input id="file" type="file">' +
          '<select id="select"><option value="a">A</option><option value="b">B</option></select>' +
          '<p id="child-only">child sentinel</p><div id="pane" style="width:100px;height:50px;overflow:auto"><div style="height:400px">space</div></div>'
      );
      const nested = document.createElement('iframe');
      nested.id = 'nested';
      nested.srcdoc = `<button id="button" onclick="top.postMessage({fixtureFrame:'nested'},'*')">Nested</button>`;
      document.body.append(nested);
      (window as unknown as { childKeys: string[] }).childKeys = [];
      window.addEventListener('keydown', (event) =>
        (window as unknown as { childKeys: string[] }).childKeys.push(event.key)
      );
    });
    const typed = (type: string, params: Record<string, unknown>) =>
      run(createTypedInstruction(type, params).action!);
    await typed('input', { selector: '#input', value: 'child' });
    await typed('focus', { selector: '#input' });
    await page.focus('#input'); // Physical keyboard must restore the selected document's focus.
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.KEYBOARD,
        params: { case: 'keyboard', value: create(KeyboardParamsSchema, { key: 'End' }) },
      })
    );
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.KEYBOARD,
        params: { case: 'keyboard', value: create(KeyboardParamsSchema, { key: 'x' }) },
      })
    );
    expect(await left.evaluate('window.childKeys')).toEqual(['End', 'x']);
    expect(await left.inputValue('#input')).toBe('child');
    expect(await page.inputValue('#input')).toBe('original');
    await typed('blur', { selector: '#input' });
    expect(await left.evaluate('document.activeElement.id')).not.toBe('input');
    await typed('wait', { selector: '#child-only', timeoutMs: 1000 });
    await typed('assert', { selector: '#child-only', mode: AssertionMode.EXISTS, timeoutMs: 1000 });
    await typed('extract', { selector: '#child-only', timeoutMs: 1000 });
    await typed('evaluate', { expression: 'window.childEvaluation=42' });
    expect(await left.evaluate('window.childEvaluation')).toBe(42);
    expect(await page.evaluate('window.childEvaluation')).toBeUndefined();
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.SELECT,
        params: {
          case: 'selectOption',
          value: create(SelectParamsSchema, {
            selector: '#select',
            selectBy: { case: 'value', value: 'b' },
          }),
        },
      })
    );
    expect(await left.inputValue('#select')).toBe('b');
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.SCROLL,
        params: {
          case: 'scroll',
          value: create(ScrollParamsSchema, { selector: '#pane', y: 100 }),
        },
      })
    );
    expect(await left.locator('#pane').evaluate((element) => element.scrollTop)).toBe(100);
    expect(await page.locator('#pane').evaluate((element) => element.scrollTop)).toBe(0);
    await typed('uploadfile', { selector: '#file', filePaths: [__filename] });
    expect(
      await left.locator('#file').evaluate((element: HTMLInputElement) => element.files?.length)
    ).toBe(1);
    await switchFrame(FrameSwitchAction.ENTER, '#nested');
    await click();
    await switchFrame(FrameSwitchAction.PARENT);
    await click();
    await switchFrame(FrameSwitchAction.ENTER, '#nested');
    await switchFrame(FrameSwitchAction.EXIT); // EXIT skips all ancestors.
    expect(session.frameStack).toHaveLength(0);
    await switchFrame(FrameSwitchAction.ENTER, '#right');
    await click();
    await switchFrame(FrameSwitchAction.EXIT);
    await click();
    expect(await page.evaluate('window.frameEffects')).toEqual(['left', 'nested', 'left', 'right']);
    expect(
      (await events()).filter((event: { type: string }) => event.type === 'click')
    ).toHaveLength(1);
    const ambiguous = await run(
      create(ActionDefinitionSchema, {
        type: ActionType.FRAME_SWITCH,
        params: {
          case: 'frameSwitch',
          value: create(FrameSwitchParamsSchema, {
            action: FrameSwitchAction.ENTER,
            frameUrl: 'about:srcdoc',
          }),
        },
      }),
      false
    );
    expect(ambiguous.failure.message).toMatch(/ambiguous/);
    expect(session.frameStack).toHaveLength(0);
    await switchFrame(FrameSwitchAction.ENTER, '#left');
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.TAB_SWITCH,
        params: {
          case: 'tabSwitch',
          value: create(TabSwitchParamsSchema, {
            action: TabSwitchAction.OPEN,
            url: `data:text/html,${encodeURIComponent(fixture)}`,
          }),
        },
      })
    );
    expect(session.frameStack).toHaveLength(0);
    expect(session.page).not.toBe(page);
    await click();
    expect(
      await session.page.evaluate("window.fixtureEvents.filter(e=>e.type==='click').length")
    ).toBe(1);
    expect(
      (await events()).filter((event: { type: string }) => event.type === 'click')
    ).toHaveLength(1);
    await run(
      create(ActionDefinitionSchema, {
        type: ActionType.TAB_SWITCH,
        params: {
          case: 'tabSwitch',
          value: create(TabSwitchParamsSchema, { action: TabSwitchAction.CLOSE }),
        },
      })
    );
    expect(session.page).toBe(page);
    expect(session.pages).toEqual([page]);
    await switchFrame(FrameSwitchAction.ENTER, '#left');
    const detached = page.waitForEvent('framedetached', { predicate: (frame) => frame === left });
    await page.locator('#left').evaluate((element) => element.remove());
    await detached;
    const refused = await run(
      create(ActionDefinitionSchema, {
        type: ActionType.CLICK,
        params: { case: 'click', value: create(ClickParamsSchema, { selector: '#button' }) },
      }),
      false
    );
    expect(refused.failure.message).toMatch(/detached/);
    expect(
      (await events()).filter((event: { type: string }) => event.type === 'click')
    ).toHaveLength(1);
    await switchFrame(FrameSwitchAction.EXIT);
  }, 20000);

  it('keeps storage, drag effects and repeated downloads in their selected document', async () => {
    const downloads: string[] = [];
    await browserContext.route('**/*', (route) => {
      const url = new URL(route.request().url());
      if (url.pathname === '/download')
        return route.fulfill({
          contentType: 'text/plain',
          headers: { 'content-disposition': `attachment; filename="${url.hostname}.txt"` },
          body: url.hostname,
        });
      const body =
        '<a id="download" href="/download">Download</a>' +
        '<div id="source" draggable="true" style="width:80px;height:30px">Drag</div>' +
        '<div id="target" style="width:80px;height:30px" ondragover="event.preventDefault()" ondrop="window.drops=(window.drops||0)+1">Drop</div>';
      return route.fulfill({
        contentType: 'text/html',
        body:
          body +
          (url.hostname === 'parent.test'
            ? '<iframe id="child" src="http://child.test/"></iframe>'
            : ''),
      });
    });
    await page.goto('http://parent.test/');
    const child = page.frames().find((frame) => frame.url() === 'http://child.test/')!;
    context.frameStack = [child];
    const cookie = await new CookieStorageHandler().execute(
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.COOKIE_STORAGE,
          params: {
            case: 'cookieStorage',
            value: create(CookieStorageParamsSchema, {
              operation: CookieOperation.SET,
              storageType: StorageType.LOCAL_STORAGE,
              key: 'fixture-key',
              value: 'child',
            }),
          },
        })
      ),
      context
    );
    expect(cookie.success).toBe(true);
    expect(await child.evaluate('localStorage.getItem("fixture-key")')).toBe('child');
    expect(await page.evaluate('localStorage.getItem("fixture-key")')).toBeNull();
    const drag = await new GestureHandler().execute(
      instruction(
        create(ActionDefinitionSchema, {
          type: ActionType.DRAG_DROP,
          params: {
            case: 'dragDrop',
            value: create(DragDropParamsSchema, {
              sourceSelector: '#source',
              targetSelector: '#target',
              timeoutMs: 1000,
            }),
          },
        })
      ),
      context
    );
    expect(drag.error).toBeUndefined();
    expect(drag.success).toBe(true);
    expect(await child.evaluate('window.drops')).toBe(1);
    expect(await page.evaluate('window.drops')).toBeUndefined();
    const download = new DownloadHandler();
    const action = instruction(
      create(ActionDefinitionSchema, {
        type: ActionType.DOWNLOAD,
        params: {
          case: 'download',
          value: create(DownloadParamsSchema, { selector: '#download', timeoutMs: 1000 }),
        },
      })
    );
    try {
      for (const host of ['child.test', 'parent.test']) {
        if (host === 'parent.test') context.frameStack.length = 0;
        const result = await download.execute(action, context);
        expect(result.error).toBeUndefined();
        expect(result.success).toBe(true);
        const data = result.extracted_data as { download_path: string; filename: string };
        downloads.push(data.download_path);
        expect(data.filename).toBe(`${host}.txt`);
        expect(await readFile(data.download_path, 'utf8')).toBe(host);
      }
      expect(new Set(downloads).size).toBe(2);
    } finally {
      await Promise.all([...new Set(downloads)].map((path) => rm(path, { force: true })));
      await cleanupSession(context.sessionId);
    }
  });

  it('a retried start cannot admit a second real browser click while the first is settling', async () => {
    const config = createTestConfig();
    let release!: () => void;
    let entered!: () => void;
    const settling = new Promise<void>((resolve) => {
      entered = resolve;
    });
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const manager = new SessionManager(config, undefined, {
      onInstructionEnd: async () => {
        entered();
        await gate;
      },
    });
    const session = {
      id: 'browser-admission-fixture',
      phase: 'ready',
      browser,
      ownerExecutionId: 'fixture-execution',
      leaseId: 'fixture-lease',
      context: browserContext,
      page,
      pages: [page],
      spec: { execution_id: 'fixture-execution', reuse_mode: 'fresh' },
      createdAt: new Date(),
      lastUsedAt: new Date(),
      instructionCount: 0,
      instructionReceipts: new Map(),
      lastInstructionSequence: 0,
    } as unknown as SessionState;
    Reflect.set(manager, 'sessions', new Map([[session.id, session]]));
    const registry = new HandlerRegistry();
    registry.register(interaction);
    const action = toJson(
      ActionDefinitionSchema,
      create(ActionDefinitionSchema, {
        type: ActionType.CLICK,
        params: { case: 'click', value: create(ClickParamsSchema, { selector: '#button' }) },
      })
    );
    const run = (index: number) => {
      const res = createMockHttpResponse();
      const done = handleSessionRun(
        createMockHttpRequest({
          body: {
            execution_id: 'fixture-execution',
            lease_id: 'fixture-lease',
            operation_sequence: index + 1,
            invocation_id: `visit-${index}`,
            attempt: 1,
            instruction: { index, nodeId: `click-${index}`, action },
          },
        }),
        res,
        session.id,
        manager,
        registry,
        config,
        logger,
        metrics
      );
      return { res, done };
    };
    const first = run(0);
    try {
      await Promise.race([
        settling,
        first.done.then(() => {
          throw new Error('First click completed before its settlement gate');
        }),
      ]);
      await manager.startSession(session.spec);
      const competing = run(1);
      await competing.done;
      expect(competing.res.statusCode).toBe(409);
      expect((await events()).filter((e: { type: string }) => e.type === 'click')).toHaveLength(1);
      release();
      await first.done;
      expect(first.res.statusCode).toBe(200);
      expect(first.res.getJSON().success).toBe(true);
      const next = run(1);
      await next.done;
      expect(next.res.statusCode).toBe(200);
      expect(next.res.getJSON().success).toBe(true);
      expect((await events()).filter((e: { type: string }) => e.type === 'click')).toHaveLength(2);
    } finally {
      release();
      await first.done;
    }
  }, 15000);

  it('only the current unreleased lease can execute or retrieve cached browser effects', async () => {
    const config = createTestConfig();
    const manager = new SessionManager(config);
    const session = {
      id: 'lease-effect-fixture',
      phase: 'ready',
      browser,
      context: browserContext,
      page,
      pages: [page],
      ownerExecutionId: 'owner-1',
      leaseId: 'lease-1',
      leaseReleasedAt: undefined,
      spec: { execution_id: 'owner-1', reuse_mode: 'fresh' },
      createdAt: new Date(),
      lastUsedAt: new Date(),
      instructionCount: 0,
      instructionReceipts: new Map(),
      lastInstructionSequence: 0,
    } as unknown as SessionState;
    Reflect.set(manager, 'sessions', new Map([[session.id, session]]));
    const registry = new HandlerRegistry();
    registry.register(interaction);
    const action = toJson(
      ActionDefinitionSchema,
      create(ActionDefinitionSchema, {
        type: ActionType.CLICK,
        params: { case: 'click', value: create(ClickParamsSchema, { selector: '#button' }) },
      })
    );
    const run = async (execution_id: string, lease_id: string, index: number, key: string) => {
      const response = createMockHttpResponse();
      await handleSessionRun(
        createMockHttpRequest({
          headers: { 'x-idempotency-key': `${lease_id}:${index + 1}` },
          body: {
            execution_id,
            lease_id,
            operation_sequence: index + 1,
            invocation_id: key,
            attempt: 1,
            instruction: { index, nodeId: `leased-click-${index}`, action },
          },
        }),
        response,
        session.id,
        manager,
        registry,
        config,
        logger,
        metrics
      );
      return response;
    };
    const count = async () =>
      (await events()).filter((event: { type: string }) => event.type === 'click').length;
    expect((await run('owner-1', 'lease-1', 0, 'first')).statusCode).toBe(200);
    expect(await count()).toBe(1);
    expect((await run('previous-owner', 'lease-1', 0, 'first')).statusCode).toBe(404);
    expect((await run('owner-1', 'previous-lease', 1, 'new')).statusCode).toBe(404);
    expect(await count()).toBe(1);
    expect(manager.releaseExecutionLease(session.id, 'owner-1', 'lease-1')).toBe(true);
    expect((await run('owner-1', 'lease-1', 1, 'released')).statusCode).toBe(404);
    expect(await count()).toBe(1);
    session.ownerExecutionId = 'owner-2';
    session.leaseId = 'lease-2';
    session.leaseReleasedAt = undefined;
    expect((await run('owner-1', 'lease-1', 0, 'first')).statusCode).toBe(404);
    expect((await run('owner-2', 'lease-2', 1, 'second')).statusCode).toBe(200);
    expect(await count()).toBe(2);
  }, 15000);

  it('repeats logical loop effects while retransmissions remain the same operation', async () => {
    const config = createTestConfig();
    const manager = new SessionManager(config);
    const session = {
      id: 'operation-effect-fixture',
      phase: 'ready',
      browser,
      context: browserContext,
      page,
      pages: [page],
      ownerExecutionId: 'owner',
      leaseId: 'lease',
      spec: { execution_id: 'owner', reuse_mode: 'fresh' },
      createdAt: new Date(),
      lastUsedAt: new Date(),
      instructionCount: 0,
      instructionReceipts: new Map(),
      lastInstructionSequence: 0,
    } as unknown as SessionState;
    Reflect.set(manager, 'sessions', new Map([[session.id, session]]));
    const registry = new HandlerRegistry();
    registry.register(interaction);
    const action = toJson(
      ActionDefinitionSchema,
      create(ActionDefinitionSchema, {
        type: ActionType.CLICK,
        params: { case: 'click', value: create(ClickParamsSchema, { selector: '#button' }) },
      })
    );
    const run = async (operation_sequence: number, invocation_id: string, index = 0) => {
      const response = createMockHttpResponse();
      await handleSessionRun(
        createMockHttpRequest({
          body: {
            execution_id: 'owner',
            lease_id: 'lease',
            operation_sequence,
            invocation_id,
            attempt: 1,
            instruction: { index, nodeId: 'same-loop-node', action },
          },
        }),
        response,
        session.id,
        manager,
        registry,
        config,
        logger,
        metrics
      );
      return response;
    };
    const count = async () =>
      (await events()).filter((event: { type: string }) => event.type === 'click').length;
    expect((await run(1, 'iteration-one')).statusCode).toBe(200);
    const second = await run(2, 'iteration-two');
    expect(second.statusCode).toBe(200);
    expect(await count()).toBe(2);
    const replay = await run(2, 'iteration-two');
    expect(replay.statusCode).toBe(200);
    expect(replay.getJSON()).toEqual(second.getJSON());
    expect(await count()).toBe(2);
    expect((await run(2, 'iteration-two', 1)).statusCode).toBe(409);
    expect(await count()).toBe(2);
  }, 15000);

  it.each(['handler', 'audio-decoration'] as const)(
    'retains uncertainty after a real click and %s failure',
    async (failureSite) => {
      const config = createTestConfig();
      const manager = new SessionManager(config);
      const session = {
        id: 'uncertain-effect-fixture',
        phase: 'ready',
        browser,
        context: browserContext,
        page,
        pages: [page],
        ownerExecutionId: 'owner',
        leaseId: 'lease',
        spec: { execution_id: 'owner', reuse_mode: 'fresh' },
        createdAt: new Date(),
        lastUsedAt: new Date(),
        instructionCount: 0,
        instructionReceipts: new Map(),
        lastInstructionSequence: 0,
      } as unknown as SessionState;
      Reflect.set(manager, 'sessions', new Map([[session.id, session]]));
      const registry = new HandlerRegistry();
      registry.register(interaction);
      await page.evaluate(
        "document.querySelector('#button').setAttribute('onclick', `console.error('diagnostic after effect')`)"
      );
      const originalExecute = interaction.execute.bind(interaction);
      const attached: CDPSession[] = [];
      const attach = browserContext.newCDPSession.bind(browserContext);
      const attachSpy = jest
        .spyOn(browserContext, 'newCDPSession')
        .mockImplementation(async (target) => {
          const cdp = await attach(target);
          attached.push(cdp);
          return cdp;
        });
      const executeSpy = jest.spyOn(interaction, 'execute');
      if (failureSite === 'handler')
        executeSpy.mockImplementationOnce(async (...args) => {
          await originalExecute(...args);

          throw new Error('failure after click');
        });
      else
        session.audioPlaybackFailure = () => {
          throw new Error('decoration after click');
        };
      const action = toJson(
        ActionDefinitionSchema,
        create(ActionDefinitionSchema, {
          type: ActionType.CLICK,
          params: { case: 'click', value: create(ClickParamsSchema, { selector: '#button' }) },
        })
      );
      const run = async () => {
        const response = createMockHttpResponse();
        await handleSessionRun(
          createMockHttpRequest({
            body: {
              execution_id: 'owner',
              lease_id: 'lease',
              operation_sequence: 1,
              invocation_id: 'uncertain-visit',
              attempt: 2,
              instruction: { index: 0, nodeId: 'uncertain-node', action },
            },
          }),
          response,
          session.id,
          manager,
          registry,
          config,
          logger,
          metrics
        );
        return response;
      };
      try {
        const first = await run();
        const repeated = await run();
        expect(first.statusCode).toBe(200);
        expect(first.getJSON().failure.code).toBe('INSTRUCTION_OUTCOME_UNCERTAIN');
        expect(first.getJSON().failure.retryable).not.toBe(true);
        expect(first.getJSON().attempt).toBe(2);
        expect(first.getJSON().notes).toMatchObject({
          invocation_id: 'uncertain-visit',
          operation_sequence: '1',
        });
        expect(repeated.getJSON()).toEqual(first.getJSON());
        expect(
          (await events()).filter((event: { type: string }) => event.type === 'click')
        ).toHaveLength(1);
        expect(executeSpy).toHaveBeenCalledTimes(1);
        expect(attached).toHaveLength(1);
        await expect(attached[0]!.send('Runtime.evaluate', { expression: '1' })).rejects.toThrow();
        if (failureSite === 'handler') {
          expect(first.getJSON().consoleLogs).toEqual(
            expect.arrayContaining([expect.objectContaining({ text: 'diagnostic after effect' })])
          );
          expect(first.getJSON().dom_html).toContain('Event target');
          expect(
            Buffer.from(first.getJSON().screenshot_base64, 'base64').subarray(1, 4).toString()
          ).toBe('PNG');
        }
      } finally {
        executeSpy.mockRestore();
        attachSpy.mockRestore();
      }
    },
    15000
  );
});

// [REQ:BAS-RH-J03] Evaluate through the installed SDK with its default runtime fix.
describe.each([false, true])('native frame contexts (site isolation: %s)', (isolated) => {
  it('preserves document identity through concurrency, nested frames and navigation', async () => {
    const browser = await chromium.launch({
      headless: true,
      args: isolated ? ['--site-per-process'] : [],
    });
    try {
      const context = await browser.newContext();
      await context.route('**/*', (route) => {
        const url = new URL(route.request().url());
        if (url.pathname === '/worker.js') {
          return route.fulfill({ contentType: 'text/javascript', body: 'self.answer=42' });
        }
        const body =
          url.pathname === '/root'
            ? '<iframe id="left" src="/child"></iframe><iframe id="right" src="/child"></iframe>' +
              '<iframe id="cross" src="http://child.test/child"></iframe>' +
              '<iframe id="opaque" sandbox="allow-scripts" src="http://opaque.test/child"></iframe>'
            : url.pathname === '/child'
              ? '<p>Child</p><iframe src="/leaf"></iframe>'
              : '<p>Leaf</p>';
        return route.fulfill({ contentType: 'text/html', body });
      });
      const page = await context.newPage();
      await page.goto(`data:text/html,${encodeURIComponent(fixture)}`);
      expect(await page.evaluate('window.fixtureEvents')).toEqual([]);
      await page.goto('http://parent.test/root');
      const cross = page.frames().find((frame) => frame.url() === 'http://child.test/child')!;
      if (isolated) {
        const session = await context.newCDPSession(cross);
        expect((await session.send('Target.getTargetInfo')).targetInfo.type).toBe('iframe');
        await session.detach();
      }
      for (const frame of page.frames()) {
        const expected = frame.url();
        expect(
          await Promise.all(Array.from({ length: 3 }, () => frame.evaluate('location.href')))
        ).toEqual([expected, expected, expected]);
        expect(
          await frame.locator('html').evaluate((element) => element.ownerDocument.location.href)
        ).toBe(expected);
      }
      const left = (await (await page.$('#left'))!.contentFrame())!;
      const right = (await (await page.$('#right'))!.contentFrame())!;
      await left.evaluate('window.localSentinel="left"');
      await right.evaluate('window.localSentinel="right"');
      expect(await left.evaluate('window.localSentinel')).toBe('left');
      expect(await right.evaluate('window.localSentinel')).toBe('right');
      for (const destination of ['http://parent.test/changed', 'http://child.test/changed']) {
        await left.goto(destination);
        expect(await left.evaluate('location.href')).toBe(destination);
      }
      const detached = page.waitForEvent('framedetached', {
        predicate: (frame) => frame === cross,
      });
      await page.locator('#cross').evaluate((element) => element.remove());
      await detached;
      await expect(cross.evaluate('location.href')).rejects.toThrow(/detached/);
      const workerCreated = page.waitForEvent('worker');
      await page.evaluate('window.worker=new Worker("/worker.js")');
      expect(await (await workerCreated).evaluate('self.answer')).toBe(42);
      expect(
        await page.evaluate(() => {
          let detected = false;
          const error = new Error();
          Object.defineProperty(error, 'stack', {
            get() {
              detected = true;
              return '';
            },
          });
          console.debug(error);
          return detected;
        })
      ).toBe(false);
    } finally {
      await browser.close();
    }
  });
});

// The patched SDK owns protocol discovery; simulate legal asynchronous delivery
// separately from the native browser matrix, without changing the installed SDK.
describe('SDK context acknowledgement', () => {
  it('keeps context discovery usable across fresh contexts and scriptless navigations', async () => {
    let effects = 0;
    const server = createServer((request, response) => {
      if (request.method === 'POST') {
        effects++;
        response.writeHead(204).end();
      } else {
        response
          .writeHead(200, { 'content-type': 'text/html' })
          .end('<title>Next document</title>');
      }
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
    const { port } = server.address() as { port: number };
    let browser: Browser | undefined;
    try {
      browser = await chromium.launch({ headless: true });
      for (let trial = 0; trial < 20; trial++) {
        const context = await browser.newContext();
        try {
          const page = await context.newPage();
          await page.goto('data:text/html,<script>window.fixtureReady=true</script>');
          await page.goto(`http://127.0.0.1:${port}/first`);
          const error = await page
            .evaluate(
              `(async () => {
            await fetch('/effect', { method: 'POST' });
            location.href = '/next';
            await new Promise(() => {});
          })()`
            )
            .then(
              () => null,
              (error: Error) => error.message
            );
          expect({ trial, effects, error }).toEqual({
            trial,
            effects: trial + 1,
            error: expect.stringMatching(/Execution context was destroyed/),
          });
          await page.waitForURL(`http://127.0.0.1:${port}/next`);
          expect(await page.evaluate('document.title')).toBe('Next document');
        } finally {
          await context.close();
        }
      }
    } finally {
      await Promise.all([
        browser?.close(),
        new Promise<void>((resolve, reject) =>
          server.close((error) => (error ? reject(error) : resolve()))
        ),
      ]);
    }
  });

  it.each([false, true])(
    'joins only the matching delayed binding receipt (initially absent: %s)',
    async (initiallyAbsent) => {
      const playwrightRoot = dirname(require.resolve('rebrowser-playwright'));
      const coreRoot = dirname(require.resolve('playwright-core', { paths: [playwrightRoot] }));
      const { CRSession } = require(join(coreRoot, 'lib/server/chromium/crConnection.js'));
      const client = new EventEmitter() as EventEmitter & {
        send: jest.Mock;
        _sendMayFail: jest.Mock;
      };
      let binding = '';
      let absent = initiallyAbsent;
      const payloads: string[] = [];
      client.send = jest.fn(
        async (method: string, params: { name?: string; expression?: string }) => {
          if (method === 'Runtime.addBinding') binding = params.name!;
          if (method === 'Runtime.evaluate') {
            const world = absent
              ? {}
              : {
                  [binding]: (payload: string) => {
                    payloads.push(payload);
                    setImmediate(() => {
                      client.emit('Runtime.bindingCalled', {
                        name: binding,
                        payload: 'wrong-receipt',
                        executionContextId: 999,
                      });
                      client.emit('Runtime.bindingCalled', {
                        name: binding,
                        payload,
                        executionContextId: 17,
                      });
                    });
                  },
                };
            absent = false;
            const value = new Function('globalThis', `return ${params.expression}`)(world);
            return { result: { value } };
          }
          return {};
        }
      );
      client._sendMayFail = jest.fn().mockResolvedValue({});
      for (let discovery = 0; discovery < 2; discovery++) {
        await expect(
          CRSession.prototype.__re__getMainWorld.call(client, {
            client,
            frameId: 'fixture-worker',
            isWorker: true,
          })
        ).resolves.toBe(17);
      }
      expect(client.listenerCount('Runtime.bindingCalled')).toBe(0);
      const registrations = client.send.mock.calls.filter(
        ([method]) => method === 'Runtime.addBinding'
      );
      expect(registrations).toHaveLength(initiallyAbsent ? 3 : 2);
      expect(new Set(registrations.map(([, params]) => params.name)).size).toBe(1);
      expect(new Set(payloads).size).toBe(2);
      expect(client._sendMayFail).not.toHaveBeenCalledWith(
        'Runtime.removeBinding',
        expect.anything()
      );
    }
  );
});
