import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useViewportEnvironment, useViewportEnvironmentStyle } from "@vrooli/react-component-library/useViewportEnvironment/1.1.2";

class VisualViewport extends EventTarget { width=393; height=793; offsetLeft=0; offsetTop=0; scale=1; }
describe("installed client drawable canvas", () => {
 let visual: VisualViewport;
 let frames: FrameRequestCallback[];
 let meta: HTMLMetaElement;
 const computed = window.getComputedStyle.bind(window);
 beforeEach(() => {
  visual = new VisualViewport(); frames=[];
  Object.defineProperty(navigator,"maxTouchPoints",{configurable:true,value:1});
  Object.defineProperty(window,"innerWidth",{configurable:true,value:393});
  Object.defineProperty(window,"innerHeight",{configurable:true,value:793});
  Object.defineProperty(window,"screen",{configurable:true,value:{width:393,height:852}});
  Object.defineProperty(window,"visualViewport",{configurable:true,value:visual});
  Object.defineProperty(navigator,"standalone",{configurable:true,value:true});
  meta = document.createElement("meta"); meta.name="apple-mobile-web-app-status-bar-style"; meta.content="black-translucent"; document.head.append(meta);
  vi.spyOn(window,"getComputedStyle").mockImplementation(element => element instanceof HTMLElement && element.style.visibility==="hidden" ? {paddingTop:"59px",paddingBottom:"34px"} as CSSStyleDeclaration : computed(element));
  vi.spyOn(HTMLElement.prototype,"getBoundingClientRect").mockImplementation(function(this:HTMLElement) { return {height:this.style.height==="100lvh"?852:0} as DOMRect; });
  vi.spyOn(window,"requestAnimationFrame").mockImplementation(callback=>{frames.push(callback);return frames.length;});
  vi.spyOn(window,"cancelAnimationFrame").mockImplementation(()=>undefined);
 });
 afterEach(()=>{cleanup();meta.remove();vi.restoreAllMocks();document.body.replaceChildren();});
 it("recognizes a keyboard when layout and visual viewports contract together",()=>{
  const {result}=renderHook(()=>({env:useViewportEnvironment(),style:useViewportEnvironmentStyle()}));
  const input=document.createElement('input');document.body.append(input);input.focus();
  act(()=>{Object.defineProperty(window,"innerHeight",{configurable:true,value:500});visual.height=500;window.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current.env.keyboardVisible).toBe(true);
  expect(result.current.env.keyboardInset).toBe(293);
  expect(result.current.style["--rcl-safe-bottom"]).toBe("0px");
  act(()=>{input.blur();Object.defineProperty(window,"innerHeight",{configurable:true,value:793});visual.height=793;window.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current.env.keyboardVisible).toBe(false);
  expect(result.current.style["--rcl-full-screen-height"]).toBe("852px");
 });
 it("does not mistake rotation with hardware focus for keyboard contraction",()=>{
  const {result}=renderHook(()=>useViewportEnvironment());
  const input=document.createElement('input');document.body.append(input);input.focus();
  act(()=>{Object.defineProperty(window,"innerWidth",{configurable:true,value:852});Object.defineProperty(window,"innerHeight",{configurable:true,value:393});visual.width=852;visual.height=393;window.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current.keyboardVisible).toBe(false);
 });
 it("keeps hardware-keyboard focus and desktop layout resizes out of virtual-keyboard state",()=>{
  Object.defineProperty(navigator,"maxTouchPoints",{configurable:true,value:0});
  const {result}=renderHook(()=>useViewportEnvironment());
  const input=document.createElement('input');document.body.append(input);input.focus();
  act(()=>{Object.defineProperty(window,"innerHeight",{configurable:true,value:500});visual.height=500;window.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current.keyboardVisible).toBe(false);
 });
 it("does not treat layout contraction without editable focus as a keyboard",()=>{
  const {result}=renderHook(()=>useViewportEnvironment());
  act(()=>{Object.defineProperty(window,"innerHeight",{configurable:true,value:500});visual.height=500;window.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current.keyboardVisible).toBe(false);
 });
 it("fills the measured 852px canvas instead of stranding navigation at innerHeight 793",()=>{
  const {result}=renderHook(()=>({env:useViewportEnvironment(),style:useViewportEnvironmentStyle()}));
  expect(result.current.env.layoutHeight).toBe(793);
  expect(result.current.env.frameHeight).toBe(852);
  expect(result.current.style["--rcl-viewport-height"]).toBe("852px");
  expect(result.current.style["--rcl-full-screen-height"]).toBe("852px");
  expect(result.current.style["--rcl-safe-bottom"]).toBe("env(safe-area-inset-bottom, 0px)");
 });
 it("leaves a browser tab's allocated frame alone",()=>{
  Object.defineProperty(navigator,"standalone",{configurable:true,value:false});
  const {result}=renderHook(()=>useViewportEnvironment()); expect(result.current.frameHeight).toBe(793);
 });
 it("requires the measured large viewport to corroborate the screen, rather than stretching to guessed screen dimensions",()=>{
  vi.spyOn(HTMLElement.prototype,"getBoundingClientRect").mockReturnValue({height:793} as DOMRect);
  const {result}=renderHook(()=>useViewportEnvironment());expect(result.current.frameHeight).toBe(793);
 });
 it("shrinks above the keyboard and restores the drawable canvas afterward",()=>{
  const input=document.createElement('input');document.body.append(input);input.focus();
  const {result}=renderHook(()=>useViewportEnvironmentStyle());
  act(()=>{visual.height=500;visual.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current["--rcl-viewport-height"]).toBe("500px");expect(result.current["--rcl-safe-bottom"]).toBe("0px");expect(result.current["--rcl-full-screen-height"]).toBeUndefined();
  act(()=>{input.blur();visual.height=793;visual.dispatchEvent(new Event('resize'));while(frames.length) frames.splice(0).forEach(cb=>cb(performance.now()));});
  expect(result.current["--rcl-full-screen-height"]).toBe("852px");
 });
});
