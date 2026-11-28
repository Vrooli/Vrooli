/**
 * Vitest environment: jsdom, with Node's own AbortController and AbortSignal.
 *
 * jsdom replaces both with its DOM-flavoured versions, but `Request` and
 * `fetch` come from Node and reject a jsdom signal ("Expected signal to be an
 * instance of AbortSignal"). React Router builds `new Request(url, { signal })`
 * on every data-router navigation, so any test that navigates a memory router
 * failed for a reason unrelated to the code under test. The app only passes
 * signals to `fetch`, so Node's classes are the right ones here.
 */
import { builtinEnvironments, type Environment } from "vitest/environments";

const NodeAbortController = globalThis.AbortController;
const NodeAbortSignal = globalThis.AbortSignal;

const environment: Environment = {
  name: "jsdom-node-abort",
  transformMode: "web",
  async setup(global, options) {
    const jsdom = await builtinEnvironments.jsdom.setup(global, options);
    Object.assign(global, { AbortController: NodeAbortController, AbortSignal: NodeAbortSignal });
    return jsdom;
  },
};

export default environment;
