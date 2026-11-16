/**
 * Vitest's jsdom environment replaces AbortController and AbortSignal with
 * DOM-realm versions, while Node's Request remains installed globally. React
 * Router creates a Request for every data-router navigation, and Node rejects
 * the jsdom signal by realm identity even though the signal is valid.
 *
 * Keep jsdom for the DOM, but restore Node's matching abort classes so router
 * navigation and fetch cancellation use the same realm as Request.
 */
import { builtinEnvironments, type Environment } from "vitest/environments";

const NodeAbortController = globalThis.AbortController;
const NodeAbortSignal = globalThis.AbortSignal;

const environment: Environment = {
  name: "jsdom-node-abort",
  transformMode: "web",
  async setup(global, options) {
    const jsdom = await builtinEnvironments.jsdom.setup(global, options);
    Object.assign(global, {
      AbortController: NodeAbortController,
      AbortSignal: NodeAbortSignal,
    });
    return jsdom;
  },
};

export default environment;
