import { queryAllByRole, queryAllByText } from "@testing-library/react";
import { runStory } from "../../api/handlers/preview/assets/story-evaluator.js";

export type APIState = { path: string; method: "GET" | "POST"; status: number; body: unknown };
type PageStory = Record<string, unknown> & { id: string; route?: string; apiState?: APIState[] };
type PageContract = { kind: "page"; route: string; stories: PageStory[] };

export const pageStoryContracts = import.meta.glob<PageContract>("./**/*.story.json", {
  eager: true,
  import: "default",
});

export function installPageAPIState(responses: APIState[], host: Pick<Window, "fetch"> = window) {
  const original = host.fetch;
  host.fetch = async (input, init) => {
    const request = input instanceof Request ? input : undefined;
    const address = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(address, window.location.href);
    const method = (init?.method ?? request?.method ?? "GET").toUpperCase();
    const fixture = responses.find((item) => item.path === url.pathname && item.method === method);
    if (fixture) {
      return new Response(
        [204, 205, 304].includes(fixture.status) ? null : JSON.stringify(fixture.body),
        {
          status: fixture.status,
          headers: { "content-type": "application/json" },
        },
      );
    }
    // Stories must never change the running API's state through undeclared calls.
    // Connect reads also use POST and therefore need explicit API-state fixtures.
    if (method !== "GET" && method !== "HEAD") {
      return new Response(
        JSON.stringify({
          code: "unavailable",
          message: `Page story has no API fixture for ${method} ${url.pathname}`,
        }),
        {
          status: 503,
          headers: { "content-type": "application/json" },
        },
      );
    }
    return original.call(host, input, init);
  };
  return () => {
    host.fetch = original;
  };
}

// Called before App imports or renders, so even its first request sees fixtures.
export function preparePageStory(search: URLSearchParams) {
  const subject = search.get("__rcl_page_story");
  if (!subject) return undefined;
  const entry = Object.entries(pageStoryContracts).find(([path]) =>
    path.endsWith(`/${subject}.story.json`),
  );
  if (!entry) throw new Error(`Unknown page story subject: ${subject}`);
  const contract = entry[1];
  const story = contract.stories.find((item) => item.id === search.get("__rcl_story"));
  if (!story) throw new Error(`Unknown story for ${subject}`);
  const route = story.route ?? contract.route;
  if (window.location.pathname !== route.split("?")[0]) {
    throw new Error(`Page story ${subject} requires route ${route}`);
  }
  const restore = installPageAPIState(story.apiState ?? []);
  return {
    dispose: restore,
    run: async (root: HTMLElement) => {
      root.setAttribute("data-preview-sheet", "");
      const result = await runStory(
        { ...story, kind: "page" },
        { document, window },
        {
          queries: { queryAllByRole, queryAllByText },
          browser: true,
        },
      );
      const evidence = document.createElement("pre");
      evidence.id = "rcl-story-result";
      evidence.hidden = true;
      evidence.textContent = JSON.stringify(result);
      document.body.append(evidence);
      root.setAttribute("data-preview-readiness-marker", "");
      root.setAttribute("data-preview-ready", "true");
      root.setAttribute("data-rcl-story-status", result.passed ? "passed" : "failed");
    },
  };
}
