import React from "react";
import { act, cleanup } from "@testing-library/react";
import * as Icons from "lucide-react";
import { afterEach, describe, expect, it } from "vitest";

import { jsdomEnv, runStory } from "../api/handlers/preview/assets/story-evaluator.js";
import { setLocale } from "../ui/src/i18n";
import { renderWithProviders as render } from "../ui/src/test-utils";
import { UndoManagerProvider } from "@vrooli/react-component-library/UndoManager/1.0.0";
import { ToastManagerProvider } from "@vrooli/react-component-library/ToastManager/1.0.0";

type StoryContract = {
  kind?: string;
  title?: string;
  stories?: Array<Record<string, unknown>>;
};
type StoryModule = Record<string, unknown>;

const previewHarnessModules = import.meta.glob(
  "./preview-harnesses/**/versions/**/*.tsx",
  { eager: true },
) as Record<string, Record<string, unknown>>;

const previewHarness = (directory: string, exportName: string) => {
  const entry = Object.entries(previewHarnessModules).find(([path]) =>
    path.endsWith(`/${directory}/versions/1.0.0/${exportName}.tsx`),
  );
  return entry?.[1][exportName] as React.ComponentType<Record<string, unknown>> | undefined;
};

const ControlledState = previewHarness("controlled-state", "ControlledState");
const Direct = previewHarness("direct", "Direct");
const Showcase = previewHarness("showcase", "Showcase");

const contracts = import.meta.glob("./**/story.json", { eager: true, import: "default" }) as Record<string, StoryContract>;
const modules = import.meta.glob("./**/versions/**/*.{ts,tsx}") as Record<string, () => Promise<StoryModule>>;

const isRenderableComponent = (value: unknown): value is React.ElementType => (
  typeof value === "function" ||
  Boolean(value && typeof value === "object" && typeof (value as { $$typeof?: unknown }).$$typeof === "symbol")
);

// Contracts use the same JSON-safe value language as the browser harness.
const resolveStoryValue = (value: unknown, log: (...args: unknown[]) => void): unknown => {
  if (Array.isArray(value)) return value.map((item) => resolveStoryValue(item, log));
  if (!value || typeof value !== "object") return value;
  const structured = value as Record<string, unknown>;
  if (Object.prototype.hasOwnProperty.call(structured, "$text")) return String(structured.$text ?? "");
  if (Object.prototype.hasOwnProperty.call(structured, "$handler")) {
    const name = String(structured.$handler || "handler");
    return (...args: unknown[]) => log(name, ...args);
  }
  if (Object.prototype.hasOwnProperty.call(structured, "$rowKey")) {
    const field = String(structured.$rowKey || "id");
    return (row: Record<string, unknown> | null | undefined, index: number) => String(row?.[field] ?? index);
  }
  if (Object.prototype.hasOwnProperty.call(structured, "$icon")) {
    const icon = (Icons as Record<string, unknown>)[String(structured.$icon)] ?? Icons.Circle;
    return React.createElement(icon as React.ElementType, {
      "aria-hidden": true,
      className: typeof structured.className === "string" ? structured.className : "h-4 w-4",
    });
  }
  if (Object.prototype.hasOwnProperty.call(structured, "$node")) {
    const type = String(structured.$node || "span");
    const props = resolveStoryValue(structured.props || {}, log) as Record<string, unknown>;
    const children = resolveStoryValue(structured.children || [], log);
    return React.createElement(type, props, ...(Array.isArray(children) ? children : [children]));
  }
  if (Object.prototype.hasOwnProperty.call(structured, "$columns")) {
    return (Array.isArray(structured.$columns) ? structured.$columns : []).map((column) => {
      const definition = column as Record<string, unknown>;
      const field = String(definition.field || "");
      return {
        id: definition.id,
        header: definition.header,
        className: definition.className,
        accessor: (row: Record<string, unknown>) => definition.badge
          ? React.createElement("span", { className: "inline-flex rounded-pill border border-app-border px-2 py-1 text-xs" }, row[field])
          : row[field],
        sortValue: definition.sortable ? (row: Record<string, unknown>) => row[field] : undefined,
        searchValue: definition.searchable ? (row: Record<string, unknown>) => String(row[field] ?? "") : undefined,
      };
    });
  }
  if (Object.prototype.hasOwnProperty.call(structured, "$filters")) {
    return (Array.isArray(structured.$filters) ? structured.$filters : []).map((filter) => {
      const definition = filter as Record<string, unknown>;
      const field = String(definition.field || "");
      return { id: definition.id, label: definition.label, predicate: (row: Record<string, unknown>) => row[field] === definition.equals };
    });
  }
  return Object.fromEntries(Object.entries(structured).map(([key, child]) => [key, resolveStoryValue(child, log)]));
};

const sourceForContract = (contractPath: string) => {
  const versionDirectory = contractPath.slice(0, contractPath.lastIndexOf("/"));
  const candidates = Object.entries(modules).filter(([path]) => path.startsWith(`${versionDirectory}/`));
  const preferred = componentNameForContract(contractPath)
    .split(/[-_]/)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join("");
  return candidates.find(([path]) => path.endsWith(`/${preferred}.tsx`))?.[1]
    ?? candidates.find(([path]) => !path.endsWith("/story.tsx"))?.[1];
};

const storyModuleForContract = (contractPath: string) => {
  const versionDirectory = contractPath.slice(0, contractPath.lastIndexOf("/"));
  return Object.entries(modules).find(([path]) => path === `${versionDirectory}/story.tsx`)?.[1];
};

const componentNameForContract = (contractPath: string) => {
  const segments = contractPath.split("/");
  const versionsIndex = segments.lastIndexOf("versions");
  return versionsIndex > 0 ? segments[versionsIndex - 1] : "";
};

const componentFromModule = (module: StoryModule, contractPath: string) => {
  const preferredName = componentNameForContract(contractPath);
  if (isRenderableComponent(module[preferredName])) return module[preferredName];
  if (isRenderableComponent(module.default)) return module.default;
  return Object.values(module).find(isRenderableComponent);
};

const harnessForStory = (story: Record<string, unknown>) => {
  const name = (story.composition as { harness?: { export?: string } } | undefined)?.harness?.export;
  if (name === "Showcase") return Showcase;
  if (name === "ControlledState") return ControlledState;
  if (name === "Direct") return Direct;
  return undefined;
};

const hookFixture = (hook: (...args: any[]) => any, args: Record<string, unknown>, environment: unknown) => {
  return function HookFixture() {
    const fixture = (environment as Record<string, unknown>)?.voiceInput || "idle";
    const media = {
      acquire: () => fixture === "permission-denied"
        ? Promise.reject(new Error("permission denied"))
        : Promise.resolve({ stop() {}, onEnded() { return () => undefined; } }),
    };
    const adapter = { connect: async () => undefined, stop: () => undefined };
    const voice = hook({
      adapter,
      media,
      mode: args.mode || (fixture === "timeout" ? "timeout" : "always-on"),
      timeoutMs: 1,
    });
    return React.createElement(
      "div",
      { role: "status", "data-rcl-hook-root": true },
      React.createElement("button", { type: "button", "data-rcl-hook-action": "start", onClick: () => void voice.start?.() }, "Start"),
      React.createElement("button", { type: "button", "data-rcl-hook-action": "stop", onClick: () => void voice.stop?.() }, "Stop"),
      React.createElement("output", null, voice.state || voice.voiceState || "idle"),
    );
  };
};

if (typeof window.matchMedia !== "function") {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: (query: string): MediaQueryList => ({
      matches: false, media: query, onchange: null,
      addListener: () => undefined, removeListener: () => undefined,
      addEventListener: () => undefined, removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  });
}

describe("live library story contracts", () => {
  afterEach(() => cleanup());

  for (const [contractPath, contract] of Object.entries(contracts)) {
    for (const story of contract.stories || []) {
      const storyName = String(story.name || story.id || "unnamed");
      it(`${contract.title || contractPath} > ${storyName}`, async () => {
        const loader = sourceForContract(contractPath);
        expect(loader, `missing component source for ${contractPath}`).toBeDefined();
        const component = componentFromModule(await loader!(), contractPath);
        const log = () => undefined;
        const args = resolveStoryValue(story.args && typeof story.args === "object" ? story.args : {}, log) as Record<string, unknown>;
        const componentName = componentNameForContract(contractPath);
        if (componentName === "ArrayField") {
          args.store = {
            subscribe: () => () => undefined,
            getField: () => ({ value: [] }),
            setValue: () => undefined,
          };
          args.field = "items";
          args.label = "Items";
          args.renderItem = () => React.createElement("span", null, "Item");
          args.createItem = () => ({}) as never;
        }
        if (componentName === "Avatar") args.name ??= "Ada Lovelace";
        if (componentName === "AttachmentPreview") args.name = "attachment.txt";
        if (componentName === "BottomNav") {
          args.items ??= [{ id: "home", label: "Home", icon: React.createElement(Icons.Circle), href: "#home" }];
          args.label ??= "Primary navigation";
        }
        if (componentName === "DataTable") {
          args.rows ??= [];
          args.columns ??= [];
          args.getRowKey ??= (_row: unknown, index: number) => String(index);
          args.caption ??= "Records";
        }
        if (componentName === "Chart") {
          args.data ??= [{ id: "sample", label: "Sample", value: 1 }];
          args.title ??= "Sample chart";
        }
        if (componentName === "CartesianCharts") {
          args.data ??= [{ id: "sample", label: "Sample", value: 1 }];
          args.title ??= "Sample chart";
        }
        if (componentName === "ConflictResolutionFlow") {
          args.fields ??= [{ id: "sample", label: "Sample", local: "local", remote: "remote" }];
        }
        if (componentName === "Select") args.options ??= [];
        if (componentName === "markdown-renderer") args.content ??= "# Markdown renderer";
        if (componentName === "FilePreview") args.name ??= "attachment.txt";
        if (componentName === "GeneratedForm") {
          args.fields ??= [{ name: "title", type: "text", label: "Title" }];
        }
        if (componentName === "ComputedField" || componentName === "ConditionalField" || componentName === "ObjectField") {
          args.store ??= {
            subscribe: () => () => undefined,
            getValues: () => ({ enabled: true, title: "Sample" }),
            getField: () => ({ value: "Sample", defaultValue: "Sample", dirty: false }),
            setValue: () => undefined,
          };
          args.field ??= "title";
        }
        if (componentName === "ComputedField") {
          args.compute ??= (values: Record<string, unknown>) => values.title;
          args.label ??= "Computed value";
        }
        if (componentName === "ConditionalField") {
          args.when ??= () => true;
          args.children ??= React.createElement("span", null, "Conditional content");
        }
        if (componentName === "ObjectField") {
          args.title ??= "Details";
          args.children ??= React.createElement("span", null, "Object content");
        }
        if (componentName === "GlobalCommandSystem") {
          args.commands ??= [{ id: "help", label: "Help", run: () => undefined }];
        }
        if (componentName === "MasterDetail") {
          args.items ??= [{ id: "one", title: "One", value: "one" }];
        }
        if (componentName === "Message") {
          args.actor ??= { name: "Assistant" };
          args.content ??= "A message";
        }
        if (componentName === "MonetizationAccount") {
          args.plan ??= "pro";
          args.status ??= "active";
          args.credits ??= 42;
        }
        if (componentName === "RadioGroup") {
          args.options ??= [{ value: "one", label: "One" }, { value: "two", label: "Two" }];
          args.label ??= "Choice";
        }
        if (componentName === "ResourceCollection") {
          args.title ??= "Resources";
          args.rows ??= [{ id: "one", name: "One" }];
          args.columns ??= [{ id: "name", header: "Name", accessor: (row: Record<string, unknown>) => row.name }];
          args.getRowKey ??= (row: Record<string, unknown>, index: number) => String(row.id ?? index);
        }
        if (componentName === "StoryPalette") {
          args.stories ??= [{ id: "default", label: "Default" }];
        }
        if (componentName === "Toolbar") {
          args.items ??= [{ id: "action", label: "Action" }];
          args.label ??= "Toolbar";
        }
        if (componentName === "Sortable") {
          args.items ??= [{ id: "one", value: "One", label: "One" }];
        }
        if (componentName === "MorphingIcon") {
          args.icon ??= "send";
          args.from ??= "check";
          args.label ??= "Icon";
        }
        if (componentName === "SidebarShell") {
          Object.defineProperty(window, "innerWidth", { configurable: true, value: 1280 });
        }
        const environment = story.environment && typeof story.environment === "object" ? story.environment : {};
        const fixtures = Object.fromEntries(Object.entries(environment as Record<string, unknown>).map(([key, value]) => [key, { value }]));
        const specimenExport = (story.composition as { specimen?: { export?: string } } | undefined)?.specimen?.export;
        const storyLoader = storyModuleForContract(contractPath);
        const storyModule = storyLoader ? await storyLoader() : {};
        const specimen = specimenExport ? storyModule[specimenExport] : undefined;
        const harness = harnessForStory(story);
        expect(specimen || component, `missing renderable export for ${contractPath}`).toBeDefined();
        await setLocale("en");

        const directArgs = { ...args };
        if (componentName === "Icon") directArgs.label ??= String(directArgs.name || "Icon");
        if (componentName === "Dialog") {
          directArgs.open ??= true;
          directArgs.title ??= "Dialog";
          directArgs.children ??= React.createElement("p", null, "Dialog content");
          directArgs.closeLabel ??= "Close dialog";
          directArgs.onClose ??= () => undefined;
        }
        const renderDirectComponent = () => {
          if (componentName === "UndoBanner") {
            render(React.createElement(UndoManagerProvider, null, React.createElement(component as React.ElementType, directArgs)));
          } else if (componentName === "Toast") {
            render(React.createElement(ToastManagerProvider, null, React.createElement(component as React.ElementType, directArgs)));
          } else {
            render(React.createElement(component as React.ElementType, directArgs));
          }
        };

        // A version may intentionally reuse another version's specimen. Mount
        // the contract's own source first so coverage follows the version
        // under test instead of silently measuring only the reused specimen.
        // A composed story is the contract's valid construction path. Trying
        // to mount its raw component with an empty args object first creates a
        // false failure for components whose required input is supplied by the
        // specimen (Banner is one example). When the story has real direct
        // arguments, retain the extra mount so a reused specimen still covers
        // the version named by this contract.
        if (contract.kind !== "hook" && typeof specimen === "function" && specimen !== component
          && Object.keys(directArgs).length > 0) {
          await act(async () => { renderDirectComponent(); });
          await act(async () => {
          await new Promise((resolve) => window.setTimeout(resolve, 20));
          });
          cleanup();
          document.body.replaceChildren();
        }

        await act(async () => {
          if (harness) {
            expect(component, `missing renderable subject for ${contractPath}`).toBeDefined();
            render(React.createElement(harness, {
              subject: component, args, environment, fixtures, log,
              config: (story.composition as { harness?: { config?: Record<string, unknown> } } | undefined)?.harness?.config,
            }));
          } else if (typeof specimen === "function") {
            render(React.createElement(specimen, { args, environment, fixtures, log }));
          } else if (contract.kind === "hook") {
            render(React.createElement(hookFixture(component as (...args: any[]) => any, args, environment)));
          } else if (componentName === "UndoBanner") {
            render(React.createElement(UndoManagerProvider, null, React.createElement(component as React.ElementType, args)));
          } else if (componentName === "Toast") {
            render(React.createElement(ToastManagerProvider, null, React.createElement(component as React.ElementType, args)));
          } else {
            renderDirectComponent();
          }
        });
        // The browser harness gives mounted effects a short settling window
        // before it begins interactions. Keep that window inside React's act
        // boundary so legitimate delayed effects do not become test warnings.
        await act(async () => {
          await new Promise((resolve) => window.setTimeout(resolve, 60));
        });

        const result = await runStory(
          { ...story, kind: contract.kind || "component", expect: story.expect || [] },
          { document, window },
          {
            ...jsdomEnv,
            queries: await import("@testing-library/dom"),
            actInteraction: async (dispatch: () => void) => { await act(async () => { dispatch(); }); },
            wait: async (milliseconds: number) => {
              await act(async () => {
                await new Promise((resolve) => window.setTimeout(resolve, milliseconds));
              });
            },
            flush: async () => { await act(async () => {}); },
          },
        );
        if (!Array.isArray(story.expect) || story.expect.length === 0) {
          expect(document.body.firstElementChild).not.toBeNull();
        }
        expect(result.failures, JSON.stringify(result.failures)).toEqual([]);
      });
    }
  }
});
