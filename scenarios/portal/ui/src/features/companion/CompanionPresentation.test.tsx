// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { useEffect, useState } from "react";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { contextImagePoint, contextImageSource, CompanionPresentation, CompanionToolbar, useCompanionStop, type PresentationMode } from "./CompanionPresentation";
import { strings } from "../../consts/strings";

afterEach(() => { cleanup(); delete window.desktopPresentation; });
function sourceFixture(activation:{contextId:string},bounds={x:0,y:0,width:1,height:1}) {
 return {surface:{ownerScenario:"device-control",surfaceId:"desktop",target:{ownerScenario:"vrooli-bridge",resourceId:"host",hostNodeId:"host"}},captureId:activation.contextId,capturedAt:new Date().toISOString(),displayId:"display",geometryRevision:"geometry",bounds};
}
function fixture() {
  let mode: PresentationMode = "expanded";
  window.desktopPresentation = {
    get: vi.fn(() => Promise.resolve({ version: 1, mode, canHide: true })),
    set: vi.fn(value => { mode = value; return Promise.resolve({ version: 1, mode, canHide: true }); }),
  };
  const stopped = vi.fn(); const unmounted = vi.fn();
  function Workspace() {
    const [draft, setDraft] = useState("");
    useEffect(() => () => { unmounted(); }, []);
    useCompanionStop("lease-1", "desktop", true, stopped);
    return <><output data-testid="conversation-branch">branch-7</output><output data-testid="task-run">run-42</output><input aria-label={strings.chat.title} value={draft} onChange={event => setDraft(event.target.value)} /></>;
  }
  const view = render(<CompanionPresentation><CompanionToolbar /><Workspace /></CompanionPresentation>);
  return { ...view, stopped, unmounted };
}
it("preserves mounted workspace state and its Stop action across all views", async () => { // UI-01
  const { container, stopped, unmounted } = fixture();
  const selector = await screen.findByLabelText(strings.companion.presentation);
  const draft = screen.getByLabelText(strings.chat.title);
  fireEvent.change(draft, { target: { value: "same conversation branch and draft" } });
  for (const mode of ["palette", "pill", "hidden", "expanded"]) {
    fireEvent.change(selector, { target: { value: mode } });
    await waitFor(() => expect(container.firstChild).toHaveAttribute("data-companion-mode", mode));
    expect(screen.getByLabelText(strings.chat.title)).toBe(draft);
    expect(draft).toHaveValue("same conversation branch and draft");
    expect(screen.getByTestId("conversation-branch")).toHaveTextContent("branch-7");
    expect(screen.getByTestId("task-run")).toHaveTextContent("run-42");
    expect(unmounted).not.toHaveBeenCalled();
  }
  fireEvent.click(screen.getByRole("button", { name: strings.companion.stopDesktop }));
  expect(stopped).toHaveBeenCalledTimes(1);
});
it("returns focus to the existing workspace when the palette is dismissed", async () => { // UI-05
  let receive: (snapshot: { version: number; mode: PresentationMode; revision: number }) => void = () => {};
  window.desktopPresentation = {
    get: () => Promise.resolve({ version: 1, mode: "expanded" as const, revision: 0 }),
    set: mode => Promise.resolve({ version: 1, mode, revision: 1 }),
    subscribe: listener => { receive = listener; return () => {}; },
  };
  const { container } = render(<CompanionPresentation><CompanionToolbar /><input aria-label={strings.chat.title} /></CompanionPresentation>);
  await screen.findByLabelText(strings.companion.presentation);
  const draft = screen.getByLabelText(strings.chat.title);
  draft.focus();
  act(() => receive({ version: 1, mode: "palette", revision: 1 }));
  expect(container.firstChild).toHaveAttribute("data-companion-mode", "palette");
  act(() => receive({ version: 1, mode: "expanded", revision: 2 }));
  await waitFor(() => expect(container.firstChild).toHaveAttribute("data-companion-mode", "expanded"));
  await waitFor(() => expect(draft).toHaveFocus());
});
it("reads authoritative state after a lost reply without repeating the transition", async () => {
  const { container } = fixture();
  const selector = await screen.findByLabelText(strings.companion.presentation);
  const bridge = window.desktopPresentation!;
  vi.mocked(bridge.set).mockRejectedValueOnce(new Error("lost reply"));
  vi.mocked(bridge.get).mockResolvedValue({ version: 1, mode: "pill" });
  fireEvent.change(selector, { target: { value: "pill" } });
  await waitFor(() => expect(container.firstChild).toHaveAttribute("data-companion-mode", "pill"));
  expect(bridge.set).toHaveBeenCalledTimes(1);
  expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.failed);
});
it("serializes transitions while keeping Stop available", async () => {
  const { stopped } = fixture();
  const selector = await screen.findByLabelText(strings.companion.presentation);
  let release!: () => void;
  vi.mocked(window.desktopPresentation!.set).mockImplementation(() => new Promise(resolve => { release = () => resolve({ version: 1, mode: "pill" }); }));
  selector.focus();
  fireEvent.change(selector, { target: { value: "pill" } });
  expect(selector).toHaveFocus();
  expect(selector).toHaveAttribute("aria-disabled", "true");
  expect(selector).not.toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: strings.companion.stopDesktop }));
  expect(stopped).toHaveBeenCalledTimes(1);
  await act(async () => { release(); await Promise.resolve(); });
  expect(selector).toHaveFocus();
});
it("leaves the web workspace expanded without a native bridge", () => { // UI-02
  const { container } = render(<CompanionPresentation><CompanionToolbar /><input aria-label="web draft" /></CompanionPresentation>);
  expect(container.firstChild).toHaveAttribute("data-companion-mode", "expanded");
  expect(screen.queryByLabelText(strings.companion.presentation)).not.toBeInTheDocument();
});

it("keeps a newer native activation when an older transition reply arrives", async () => {
 let receive: (snapshot: {version: number; mode: PresentationMode; revision: number}) => void = () => {};
 let release!: () => void;
 const unsubscribe = vi.fn();
 window.desktopPresentation = {
  get: () => Promise.resolve({version:1,mode:"expanded",revision:0}),
  set: () => new Promise(resolve => { release = () => resolve({version:1,mode:"pill",revision:1}); }),
  subscribe: listener => { receive = listener; return unsubscribe; },
 };
 const {container, unmount} = render(<CompanionPresentation><CompanionToolbar /></CompanionPresentation>);
 const selector = await screen.findByLabelText(strings.companion.presentation);
 fireEvent.change(selector,{target:{value:"pill"}});
 act(() => { receive({version:1,mode:"palette",revision:2}); });
 await act(async () => { release(); await Promise.resolve(); });
 expect(container.firstChild).toHaveAttribute("data-companion-mode","palette");
 unmount(); expect(unsubscribe).toHaveBeenCalledOnce();
});
it("shows native shortcut refusal while keeping manual presentation available", async () => { // UI-04
 window.desktopPresentation = {
  get: () => Promise.resolve({version:1,mode:"expanded",revision:0,shortcut:{accelerator:"Control+Shift+Space",status:"unavailable"}}),
  set: mode => Promise.resolve({version:1,mode,revision:1}),
 };
 render(<CompanionPresentation><CompanionToolbar /></CompanionPresentation>);
 expect(await screen.findByRole("status")).toHaveTextContent(strings.companion.shortcutUnavailable);
 expect(screen.getByLabelText(strings.companion.presentation)).not.toBeDisabled();
 expect(screen.queryByRole("option", { name: strings.companion.hidden })).not.toBeInTheDocument();
});

it("waits for task removal after Stop completes before authorizing Quit", async () => {
 let requestQuit: (id:number)=>void = () => {};
 let finish!:()=>void;const draftLabel="retained draft";
 const stop=vi.fn().mockResolvedValue(undefined);
 const decide=vi.fn().mockResolvedValue(undefined);
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"expanded",canHide:true}),set:mode=>Promise.resolve({version:1,mode}),onQuit:listener=>{requestQuit=listener;return()=>{};},decideQuit:decide};
 function Workspace(){const[active,setActive]=useState(true);finish=()=>setActive(false);useCompanionStop("lease","desktop",active,stop);return <input aria-label={draftLabel} defaultValue="same draft"/>;}
 render(<CompanionPresentation><CompanionToolbar/><Workspace/></CompanionPresentation>);
 await screen.findByLabelText(strings.companion.presentation);
 act(()=>requestQuit(7));
 fireEvent.click(await screen.findByRole("button",{name:strings.companion.stopAndQuit}));
 await waitFor(()=>expect(stop).toHaveBeenCalledOnce());
 expect(decide).not.toHaveBeenCalled();
 expect(screen.getByRole("dialog")).toBeInTheDocument();
 expect(screen.getByLabelText(draftLabel)).toHaveValue("same draft");
 act(()=>finish());
 await waitFor(()=>expect(decide).toHaveBeenCalledWith(7,"quit"));
 expect(decide).toHaveBeenCalledTimes(1);
});
it.each([['cancelQuit','cancel'],['keepRunning','background']] as const)("supports %s without stopping the task",async(label,decision)=>{
 let requestQuit:(id:number)=>void=()=>{};const decide=vi.fn().mockResolvedValue(undefined);const stop=vi.fn();
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"expanded",canHide:true}),set:mode=>Promise.resolve({version:1,mode}),onQuit:listener=>{requestQuit=listener;return()=>{};},decideQuit:decide};
 function Workspace(){useCompanionStop("task","chat",true,stop);return null;}
 render(<CompanionPresentation><CompanionToolbar/><Workspace/></CompanionPresentation>);
 await screen.findByLabelText(strings.companion.presentation);act(()=>requestQuit(2));
 fireEvent.click(await screen.findByRole("button",{name:strings.companion[label]}));
 await waitFor(()=>expect(decide).toHaveBeenCalledWith(2,decision));expect(stop).not.toHaveBeenCalled();
});
it("keeps keyboard focus in the Quit decision and restores the draft on Escape", async () => { // UI-10
 const draftLabel="draft focus";
 let requestQuit:(id:number)=>void=()=>{};const decide=vi.fn().mockResolvedValue(undefined);
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"expanded",canHide:true}),set:mode=>Promise.resolve({version:1,mode}),onQuit:listener=>{requestQuit=listener;return()=>{};},decideQuit:decide};
 function Workspace(){useCompanionStop("task","chat",true,vi.fn());return <input aria-label={draftLabel}/>;}
 render(<CompanionPresentation><Workspace/></CompanionPresentation>);
 const draft=screen.getByLabelText(draftLabel);draft.focus();act(()=>requestQuit(3));
 const dialog=await screen.findByRole("dialog");
 expect(screen.getByRole("button",{name:strings.companion.cancelQuit})).toHaveFocus();
 const last=screen.getByRole("button",{name:strings.companion.stopAndQuit});last.focus();
 fireEvent.keyDown(last,{key:"Tab"});
 const first=screen.getByRole("button",{name:strings.companion.stopChat});expect(first).toHaveFocus();
 fireEvent.keyDown(first,{key:"Tab",shiftKey:true});expect(last).toHaveFocus();
 fireEvent.keyDown(dialog,{key:"Escape"});
 await waitFor(()=>expect(decide).toHaveBeenCalledWith(3,"cancel"));
 await waitFor(()=>expect(draft).toHaveFocus());
});

it("reconciles a lost shortcut reply without repeating registration", async () => {
 let registered = false;
 const setShortcut = vi.fn(() => { registered = true; return Promise.reject(new Error("lost reply")); });
 window.desktopPresentation = {
  get: () => Promise.resolve({ version: 1, mode: "expanded", revision: registered ? 1 : 0, canHide: registered,
   shortcut: { accelerator: registered ? "Control+Alt+P" : "Control+Shift+Space", status: registered ? "registered" : "unavailable" } }),
  set: mode => Promise.resolve({ version: 1, mode }), setShortcut,
 };
 render(<CompanionPresentation><CompanionToolbar /></CompanionPresentation>);
 const input = await screen.findByLabelText(strings.companion.shortcutSetting);
 fireEvent.change(input, { target: { value: "Control+Alt+P" } });
 fireEvent.submit(input.closest("form")!);
 await waitFor(() => expect(screen.getByRole("option", { name: strings.companion.hidden })).toBeInTheDocument());
 expect(setShortcut).toHaveBeenCalledOnce();
 expect(setShortcut).toHaveBeenCalledWith("Control+Alt+P");
 expect(screen.queryByText(strings.companion.shortcutUnavailable)).not.toBeInTheDocument();
 expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.failed);
});

it("expires native context without recapturing or disabling chat", async () => {
 vi.useFakeTimers();
 try {
  let receive!: Parameters<NonNullable<NonNullable<Window["desktopPresentation"]>["subscribe"]>>[0];
  const get = vi.fn(() => Promise.resolve({ version: 1, mode: "expanded" as const, revision: 0, activation: { status: "capturing" as const } }));
  window.desktopPresentation = { get, set: mode => Promise.resolve({version:1,mode}), subscribe: listener => {receive=listener;return()=>{};} };
  await act(async () => {render(<CompanionPresentation><CompanionToolbar /><input aria-label={strings.chat.title} defaultValue="draft" /></CompanionPresentation>);await Promise.resolve();});
  expect(screen.getByTestId("companion-context-status")).toHaveTextContent(strings.companion.contextCapturing);
  act(() => receive({ version:1,mode:"palette",revision:1,activation:{status:"ready",contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+1000}}));
  expect(screen.getByTestId("companion-context-status")).toHaveTextContent(strings.companion.contextReady);
  await act(async () => {await vi.advanceTimersByTimeAsync(1000);});
  expect(screen.getByTestId("companion-context-status")).toHaveTextContent(strings.companion.contextUnavailable);
  expect(screen.getByLabelText(strings.chat.title)).toHaveValue("draft");
  expect(screen.getByLabelText(strings.chat.title)).not.toBeDisabled();
  expect(get).toHaveBeenCalledOnce();
 } finally {vi.useRealTimers();}
});

it("keeps text chat usable when native capture permission is denied", async () => { // UI-12
 window.desktopPresentation = {
  get: () => Promise.resolve({ version: 1, mode: "expanded" as const, activation: { status: "unavailable" as const } }),
  set: mode => Promise.resolve({ version: 1, mode }),
 };
 render(<CompanionPresentation><CompanionToolbar /><input aria-label={strings.chat.title} defaultValue="text fallback" /></CompanionPresentation>);
 await waitFor(() => expect(screen.getByTestId("companion-context-status")).toHaveTextContent(strings.companion.contextUnavailable));
 const draft = screen.getByLabelText(strings.chat.title);
 expect(draft).not.toBeDisabled();
 expect(draft).toHaveValue("text fallback");
});

it("rejects malformed ready context and retains ordinary manual presentation", async () => {
 window.desktopPresentation = { get:()=>Promise.resolve({version:1,mode:"expanded",activation:{status:"ready",contextId:"native-window-42",expiresAt:Date.now()+10000}}),set:mode=>Promise.resolve({version:1,mode}) };
 render(<CompanionPresentation><CompanionToolbar /><input aria-label={strings.chat.title} /></CompanionPresentation>);
 expect(await screen.findByRole("alert")).toHaveTextContent(strings.companion.failed);
 expect(screen.queryByText(strings.companion.contextReady)).not.toBeInTheDocument();
 expect(screen.getByLabelText(strings.chat.title)).not.toBeDisabled();
});


it.each([false, true])("dismisses the native reference without replay after lost reply=%s", async lostReply => {
 const { stopped, unmounted } = fixture();
 const bridge = window.desktopPresentation!;
 const ready = { version:1, mode:"expanded" as const, revision:1, activation:{status:"ready" as const,contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+20000} };
 const dismissed = { version:1, mode:"expanded" as const, revision:2, activation:{status:"unavailable" as const} };
 vi.mocked(bridge.get).mockResolvedValue(ready);
 let finish!: () => void;
 bridge.dismissContext = vi.fn(() => new Promise<typeof dismissed>((resolve, reject) => { finish = () => { vi.mocked(bridge.get).mockResolvedValue(dismissed); if (lostReply) reject(new Error("lost reply")); else resolve(dismissed); }; }));
 fireEvent(window, new Event("load"));
 const button = await screen.findByRole("button", {name:strings.companion.dismissContext});
 const draft = screen.getByRole("textbox", {name:strings.chat.title});
 fireEvent.change(draft, {target:{value:"keep this draft"}});
 fireEvent.click(button); fireEvent.click(button);
 expect(bridge.dismissContext).toHaveBeenCalledOnce();
 await act(async () => {finish(); await Promise.resolve();});
 await waitFor(() => expect(screen.queryByRole("button", {name:strings.companion.dismissContext})).not.toBeInTheDocument());
 expect(screen.getByTestId("companion-context-status")).toHaveTextContent(strings.companion.contextUnavailable);
 expect(screen.getByRole("textbox", {name:strings.chat.title})).toBe(draft);
 expect(draft).toHaveValue("keep this draft");
 expect(stopped).not.toHaveBeenCalled(); expect(unmounted).not.toHaveBeenCalled();
 expect(bridge.set).not.toHaveBeenCalled(); expect(bridge.dismissContext).toHaveBeenCalledOnce();
});

it.each(["expiry","replacement","dismissal"])("clears image pixels on %s and preserves the composer", async operation => {
 vi.useFakeTimers();
 try {
  const activation={status:"ready" as const,contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+1000,hasImage:true};
  const ready={version:1,mode:"palette" as const,revision:1,activation};
  let receive!: Parameters<NonNullable<NonNullable<Window["desktopPresentation"]>["subscribe"]>>[0];
  const readContextImage=vi.fn(()=>Promise.resolve({...activation,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",source:sourceFixture(activation),sourceBounds:{x:0,y:0,width:1,height:1}}));
  window.desktopPresentation={get:()=>Promise.resolve(ready),set:mode=>Promise.resolve({version:1,mode}),readContextImage,subscribe:listener=>{receive=listener;return()=>{};},dismissContext:()=>Promise.resolve({version:1,mode:"palette",revision:2,activation:{status:"unavailable"}})};
  await act(async()=>{await Promise.resolve();render(<CompanionPresentation><CompanionToolbar/><input aria-label={strings.chat.title} defaultValue="keep draft"/></CompanionPresentation>);});
  const draft=screen.getByLabelText(strings.chat.title);
  expect(readContextImage).not.toHaveBeenCalled();
  await act(async()=>{await Promise.resolve();fireEvent.click(screen.getByRole("button",{name:strings.companion.previewContext}));});
  expect(screen.getByRole("img",{name:strings.companion.previewAlt})).toHaveAttribute("src","data:image/png;base64,aW1hZ2U=");
  expect(screen.getByText(strings.companion.previewLocal)).toBeInTheDocument();
  if(operation==="expiry") await act(async()=>{await vi.advanceTimersByTimeAsync(1000);});
  if(operation==="replacement") act(()=>receive({...ready,revision:2,activation:{...activation,contextId:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}}));
  if(operation==="dismissal") await act(async()=>{await Promise.resolve();fireEvent.click(screen.getByRole("button",{name:strings.companion.dismissContext}));});
  expect(screen.queryByRole("img")).not.toBeInTheDocument();
  expect(readContextImage).toHaveBeenCalledOnce();expect(screen.getByLabelText(strings.chat.title)).toBe(draft);expect(draft).toHaveValue("keep draft");
 } finally {vi.useRealTimers();}
});

it("ignores a late image after dismissal and allows chat after preview failure",async()=>{
 const activation={status:"ready" as const,contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+20000,hasImage:true};
 let finish!:(value:unknown)=>void;
 const readContextImage=vi.fn(()=>new Promise<unknown>(resolve=>{finish=resolve;}));
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"palette",activation}),set:mode=>Promise.resolve({version:1,mode}),readContextImage,dismissContext:()=>Promise.resolve({version:1,mode:"palette",activation:{status:"unavailable"}})};
 render(<CompanionPresentation><CompanionToolbar/><input aria-label={strings.chat.title} defaultValue="keep draft"/></CompanionPresentation>);
 fireEvent.click(await screen.findByRole("button",{name:strings.companion.previewContext}));
 await act(async()=>{await Promise.resolve();finish({contextId:"wrong"});});
 expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.previewFailed);
 expect(screen.getByLabelText(strings.chat.title)).not.toBeDisabled();
 fireEvent.click(screen.getByRole("button",{name:strings.companion.previewContext}));
 fireEvent.click(screen.getByRole("button",{name:strings.companion.dismissContext}));
 await act(async()=>{await Promise.resolve();finish({...activation,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",source:sourceFixture(activation),sourceBounds:{x:0,y:0,width:1,height:1}});});
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
 expect(screen.getByLabelText(strings.chat.title)).toHaveValue("keep draft");
});

it("maps resized and portrait previews to original pixels without using desktop origins",()=>{ // UI-06
 expect(contextImagePoint(35,30,{left:10,top:20,width:100,height:50},{width:200,height:100})).toEqual({x:50,y:20});
 expect(contextImagePoint(22.5,25,{left:10,top:20,width:50,height:25},{width:200,height:100})).toEqual({x:50,y:20});
 expect(contextImagePoint(25,100,{left:0,top:0,width:50,height:100},{width:100,height:200})).toEqual({x:50,y:200});
 expect(contextImagePoint(-100,500,{left:0,top:0,width:100,height:50},{width:200,height:100})).toEqual({x:0,y:100});
 expect(contextImagePoint(1,2,{left:0,top:0,width:0,height:50},{width:200,height:100})).toBeUndefined();
});
it("keeps regions and annotations on their source pixels across compact views",async()=>{
 const activation={status:"ready" as const,contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+20000,hasImage:true};
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"expanded",activation}),set:mode=>Promise.resolve({version:1,mode,activation}),readContextImage:()=>Promise.resolve({...activation,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",source:sourceFixture(activation,{x:-1920,y:-100,width:200,height:100}),sourceBounds:{x:-1920,y:-100,width:200,height:100}})};
 render(<CompanionPresentation><CompanionToolbar/></CompanionPresentation>);
 fireEvent.click(await screen.findByRole("button",{name:strings.companion.previewContext}));
 const overlay=await screen.findByTestId("context-image-overlay");
 overlay.setPointerCapture=vi.fn();
 const geometry=vi.spyOn(overlay,"getBoundingClientRect").mockReturnValue(new DOMRect(10,20,100,50));
 const pointer=(type:string,x:number,y:number)=>{const event=new MouseEvent(type,{bubbles:true,button:0,clientX:x,clientY:y});Object.defineProperty(event,"pointerId",{value:1});fireEvent(overlay,event);};
 pointer("pointerdown",35,30);pointer("pointermove",85,60);pointer("pointerup",85,60);
 const region=screen.getByTestId("context-image-region");
 expect(region).toHaveAttribute("x","50");expect(region).toHaveAttribute("y","20");expect(region).toHaveAttribute("width","100");expect(region).toHaveAttribute("height","60");
 pointer("pointerdown",10,20);pointer("pointermove",20,25);pointer("pointercancel",20,25);
 expect(region).toHaveAttribute("x","50");
 fireEvent.click(screen.getByRole("button",{name:strings.companion.drawAnnotation}));
 geometry.mockReturnValue(new DOMRect(10,20,50,25));
 pointer("pointerdown",22.5,25);pointer("pointermove",35,30);pointer("pointerup",35,30);
 expect(screen.getByTestId("context-image-mark")).toHaveAttribute("points","50,20 100,40 100,40");
 const selector=screen.getByLabelText(strings.companion.presentation);
 fireEvent.change(selector,{target:{value:"pill"}});
 await waitFor(()=>expect(overlay).not.toBeVisible());
 fireEvent.change(selector,{target:{value:"expanded"}});
 await waitFor(()=>expect(overlay).toBeVisible());
 expect(screen.getByTestId("context-image-overlay")).toBe(overlay);expect(region).toHaveAttribute("x","50");expect(screen.getAllByTestId("context-image-mark")).toHaveLength(1);
 fireEvent.change(screen.getByLabelText(strings.companion.width),{target:{value:"500"}});expect(region).toHaveAttribute("width","100");
 fireEvent.change(screen.getByLabelText(strings.companion.width),{target:{value:"80"}});expect(region).toHaveAttribute("width","80");
 fireEvent.click(screen.getByRole("button",{name:strings.companion.resetAnnotations}));
 expect(screen.queryByTestId("context-image-mark")).not.toBeInTheDocument();expect(region).toHaveAttribute("x","0");expect(region).toHaveAttribute("width","200");
});

it("preserves exact source provenance and rejects missing, stale or mismatched capture metadata",()=>{
 const now=Date.now(), id="59a6140b-a9cd-43d5-992b-2506b0177424", expires=now+20000;
 const bounds={x:-100,y:-50,width:200,height:100};
 const source=sourceFixture({contextId:id},bounds);
 source.capturedAt=new Date(now).toISOString();
 const result=contextImageSource({...source,sessionId:"must-not-project"},id,expires,bounds,now);
 expect(result).toEqual(source);
 result.bounds.x=0;result.surface.target.resourceId="changed";
 expect(source.bounds.x).toBe(-100);expect(source.surface.target.resourceId).toBe("host");
 for(const value of [undefined,{...source,captureId:"other"},{...source,bounds:{...bounds,width:1}},{...source,capturedAt:new Date(now-31000).toISOString()},{...source,capturedAt:new Date(now+1).toISOString()},{...source,geometryRevision:""},{...source,surface:{...source.surface,target:undefined}}]) {
  expect(()=>contextImageSource(value,id,expires,bounds,now)).toThrow();
 }
});

it("retains reviewed native context explicitly and keeps its receipt after closing preview",async()=>{
 const {ContextRetentionProvider,ContextRetentionActions}=await import("./ContextRetention");
 const {contextCaptureClient}=await import("../../api/contextcapture");
 const {create}=await import("@bufbuild/protobuf");
 const {DocumentSchema}=await import("@vrooli/proto-types/portal/v1/contextcapture/contextcapture_pb");
 const {contextRetentionKey}=await import("./contextRetentionStore");
 const originalLocks=Object.getOwnPropertyDescriptor(navigator,"locks");
 Object.defineProperty(navigator,"locks",{configurable:true,value:{request:(_name:string,_options:unknown,run:(lock:object)=>Promise<void>)=>run({})}});
 localStorage.removeItem(contextRetentionKey);
 const activation={status:"ready" as const,contextId:"59a6140b-a9cd-43d5-992b-2506b0177424",expiresAt:Date.now()+20000,hasImage:true};
 window.desktopPresentation={get:()=>Promise.resolve({version:1,mode:"expanded",activation}),set:mode=>Promise.resolve({version:1,mode,activation}),readContextImage:()=>Promise.resolve({...activation,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",source:sourceFixture(activation),sourceBounds:{x:0,y:0,width:1,height:1}})};
 const send=vi.spyOn(contextCaptureClient,"import").mockImplementation(request=>Promise.resolve(create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId:request.requestId,source:request.source,region:request.region,strokes:request.strokes,originalSha256:"a".repeat(64),createdAt:{seconds:BigInt(Math.floor(Date.now()/1000))},expiresAt:{seconds:BigInt(Math.floor(Date.now()/1000)+3600)}})));
 try {
  render(<CompanionPresentation><ContextRetentionProvider account={{id:"alice",realm:"realm",token:"secret",expires:Date.now()+60000}}><CompanionToolbar/><ContextRetentionActions/></ContextRetentionProvider></CompanionPresentation>);
  fireEvent.click(await screen.findByRole("button",{name:strings.companion.previewContext}));
  const retain=await screen.findByRole("button",{name:strings.companion.retainContext});expect(send).not.toHaveBeenCalled();
  fireEvent.click(retain);await waitFor(()=>expect(screen.getAllByText(strings.companion.contextRetained).length).toBeGreaterThan(0));
  expect(send).toHaveBeenCalledTimes(1);expect(send.mock.calls[0]?.[0].region).toMatchObject({x:0,y:0,width:1,height:1});
  expect(screen.queryByText(strings.companion.previewLocal)).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:strings.companion.closePreview}));
  expect(screen.getByText(strings.companion.contextRetained)).toBeInTheDocument();
  expect(screen.getByRole("button",{name:strings.companion.deleteRetainedContext})).toBeInTheDocument();
 }finally{send.mockRestore();localStorage.removeItem(contextRetentionKey);if(originalLocks)Object.defineProperty(navigator,"locks",originalLocks);else Reflect.deleteProperty(navigator,"locks");}
});
