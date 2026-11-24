import { useMemo } from "react";
import {
  Blocks,
  Boxes,
  Command,
  Compass,
  FileText,
  FormInput,
  Layers,
  LayoutTemplate,
  Sparkles,
  Table2,
  Workflow,
} from "lucide-react";

import type { CatalogAsset } from "../../api/components";
import { SearchKind, type SearchResult } from "../../api/search";
import { WorkspaceProvenance, type WorkspaceSummary } from "../../api/workspaces";
import { appRoutes, libraryDomainPath, workspacePath } from "../../routes";
import { assetDomain, assetGroup } from "../data";

export type StartEntry = {
  id: string;
  title: string;
  countValue: number;
  pagesValue: number;
  count: string;
  examples: string;
  href: string;
  Icon: typeof Boxes;
};

const MOBILE_COMMANDS_URL = "/?commands=1";
type Translate = (key: string, options?: Record<string, string | number>) => string;

/** The intent a domain usually serves. This is presentation, not data. */
const DOMAIN_ICON: Record<string, typeof Boxes> = {
  controls: Blocks,
  data: Table2,
  forms: FormInput,
  navigation: Compass,
  layout: LayoutTemplate,
  feedback: Sparkles,
  content: FileText,
  motion: Workflow,
};

/** The filter keeps using the real domain; this is only its presentation label. */
const DOMAIN_LABEL: Record<string, string> = {
  controls: "Actions & controls",
  "data-display": "Collections & data",
  forms: "Forms & auth",
  hooks: "Hooks & runtime",
  legacy: "Legacy surface",
  navigation: "Navigation & shell",
  primitives: "Foundations",
  uncategorized: "Uncategorised",
};

function labelForDomain(domain: string): string {
  if (DOMAIN_LABEL[domain]) return DOMAIN_LABEL[domain];
  return domain.replace(/[-_]/g, " ").replace(/^\w/, (letter) => letter.toUpperCase());
}

function iconForDomain(domain: string): typeof Boxes {
  const lower = domain.toLowerCase();
  for (const [key, icon] of Object.entries(DOMAIN_ICON)) {
    if (lower.includes(key)) return icon;
  }
  return Boxes;
}

export function useStartPageEntries(
  assets: CatalogAsset[],
  summaries: WorkspaceSummary[],
  commands: {
    response: { total: number } | null;
    results: SearchResult[];
    loading: boolean;
    error: unknown;
  },
  pageTypes: {
    response: { total: number } | null;
    results: SearchResult[];
    loading: boolean;
    error: unknown;
  },
  t: Translate,
) {
  const curated = summaries.filter(
    (workspace) => workspace.provenance === WorkspaceProvenance.CURATED,
  );
  const recent = summaries
    .filter((workspace) => workspace.provenance !== WorkspaceProvenance.CURATED)
    .slice(0, 3);

  const domainEntries = useMemo(() => {
    const byDomain = new Map<string, { count: number; examples: string[]; group: string }>();
    for (const asset of assets) {
      const domain = assetDomain(asset);
      const group = assetGroup(asset);
      const key = `${group}:${domain}`;
      const entry = byDomain.get(key) ?? { count: 0, examples: [], group };
      entry.count += 1;
      if (entry.examples.length < 3) entry.examples.push(asset.displayName);
      byDomain.set(key, entry);
    }
    return [...byDomain.entries()]
      .map(([key, value]) => ({ domain: key.slice(key.indexOf(":") + 1), ...value }))
      .sort((left, right) =>
        right.count === left.count
          ? left.domain.localeCompare(right.domain)
          : right.count - left.count,
      );
  }, [assets]);

  // These first cards project existing catalog and workspace data. Counts and
  // examples always come from the current index response.
  const browseEntries = useMemo<StartEntry[]>(() => {
    const landingTypeResults = pageTypes.results.filter(
      (result) => result.kind === SearchKind.PAGE_TYPE,
    );
    const landingTypeCount = pageTypes.response?.total ?? landingTypeResults.length;
    const onboardingWorkspaces = summaries.filter(
      (workspace) =>
        Boolean(workspace.id) && workspace.tags.some((tag) => tag.toLowerCase() === "onboarding"),
    );
    const onboardingAssets = assets.filter((asset) =>
      /onboarding/i.test(`${asset.catalogId} ${asset.displayName}`),
    );
    const landingEntry: StartEntry[] =
      landingTypeCount > 0 || pageTypes.loading || Boolean(pageTypes.error)
        ? [
            {
              id: "landing-pages",
              title: t("workbench.start.landingPages", { defaultValue: "Landing pages" }),
              countValue: landingTypeCount,
              pagesValue: 0,
              count: landingTypeCount
                ? t("workbench.start.pageTypesCount", {
                    count: landingTypeCount,
                    defaultValue: "{{count}} page types",
                  })
                : pageTypes.loading
                  ? t("workbench.start.pageTypesLoading", { defaultValue: "Searching page types…" })
                  : pageTypes.error
                    ? t("workbench.start.pageTypesUnavailable", {
                        defaultValue: "Page-type index unavailable",
                      })
                    : t("workbench.start.noLandingTypes", {
                        defaultValue: "No page types indexed",
                      }),
              examples: landingTypeResults.length
                ? landingTypeResults
                    .slice(0, 3)
                    .map((result) => result.title)
                    .join(" · ")
                : t("workbench.start.pageTypeCatalog", { defaultValue: "Page-type catalog" }),
              href:
                landingTypeResults[0]?.href ??
                libraryDomainPath("templates", "components", "LandingPage"),
              Icon: LayoutTemplate,
            },
          ]
        : [];
    const onboardingPages = onboardingWorkspaces.reduce(
      (total, workspace) => total + workspace.pageCount,
      0,
    );
    const onboardingEntry: StartEntry[] =
      onboardingWorkspaces.length || onboardingAssets.length
        ? [
            {
              id: "onboarding-flows",
              title: t("workbench.start.onboardingFlows", { defaultValue: "Onboarding flows" }),
              countValue: onboardingWorkspaces.length || onboardingAssets.length,
              pagesValue: onboardingPages,
              count: onboardingWorkspaces.length
                ? t("workbench.start.onboardingWorkspaceCount", {
                    count: onboardingWorkspaces.length,
                    pages: onboardingPages,
                    defaultValue: "{{count}} workspace · {{pages}} pages",
                  })
                : t("workbench.start.onboardingAssetsCount", {
                    count: onboardingAssets.length,
                    defaultValue: "{{count}} catalog assets",
                  }),
              examples: [
                ...onboardingWorkspaces.map((workspace) => workspace.name),
                ...onboardingAssets.map((asset) => asset.displayName),
              ]
                .filter((value, index, all) => all.indexOf(value) === index)
                .slice(0, 3)
                .join(" · "),
              href:
                onboardingWorkspaces.length === 1
                  ? workspacePath(onboardingWorkspaces[0]?.id ?? "")
                  : appRoutes.design,
              Icon: Workflow,
            },
          ]
        : [];
    const workspaceEntry: StartEntry[] = summaries.length
      ? [
          {
            id: "workspaces",
            title: t("workbench.start.workspaces", { defaultValue: "Workspaces" }),
            countValue: summaries.length,
            pagesValue: 0,
            count: t("workbench.start.workspacesCount", {
              count: summaries.length,
              defaultValue: "{{count}} workspaces",
            }),
            examples: summaries
              .slice(0, 3)
              .map((workspace) => workspace.name)
              .join(" · "),
            href: appRoutes.design,
            Icon: Layers,
          },
        ]
      : [];
    const catalogEntries = domainEntries
      .filter((entry) => entry.domain !== "templates")
      .slice(0, 5)
      .map((entry) => ({
        id: `${entry.group}:${entry.domain}`,
        title: labelForDomain(entry.domain),
        countValue: entry.count,
        pagesValue: 0,
        count: t("workbench.start.categoryCount", {
          count: entry.count,
          domain: entry.domain,
          defaultValue: "{{count}} assets · {{domain}}",
        }),
        examples: entry.examples.join(" · "),
        href: libraryDomainPath(entry.domain, entry.group),
        Icon: iconForDomain(entry.domain),
      }));
    return [...landingEntry, ...onboardingEntry, ...workspaceEntry, ...catalogEntries];
  }, [assets, domainEntries, pageTypes, summaries, t]);

  const mobilePrimaryIntents = useMemo<StartEntry[]>(() => {
    const byID = new Map(browseEntries.map((entry) => [entry.id, entry]));
    const landing = byID.get("landing-pages") ?? {
      id: "landing-pages",
      title: t("workbench.start.landingPages", { defaultValue: "Landing pages" }),
      countValue: 0,
      pagesValue: 0,
      count: t("workbench.start.noLandingTypes", { defaultValue: "No page types indexed" }),
      examples: t("workbench.start.pageTypeCatalog", { defaultValue: "Page-type catalog" }),
      href: appRoutes.library,
      Icon: LayoutTemplate,
    };
    const onboarding = byID.get("onboarding-flows") ?? {
      id: "onboarding-flows",
      title: t("workbench.start.onboardingFlows", { defaultValue: "Onboarding flows" }),
      countValue: 0,
      pagesValue: 0,
      count: t("workbench.start.noOnboardingFlows", {
        defaultValue: "No onboarding flows indexed",
      }),
      examples: t("workbench.start.startOnboarding", { defaultValue: "Start a workspace" }),
      href: appRoutes.design,
      Icon: Workflow,
    };
    const workspaceEntry = byID.get("workspaces") ?? {
      id: "workspaces",
      title: t("workbench.start.workspaces", { defaultValue: "Workspaces" }),
      countValue: 0,
      pagesValue: 0,
      count: t("workbench.start.workspacesCount", {
        count: 0,
        defaultValue: "{{count}} workspaces",
      }),
      examples: t("workbench.start.noWorkspaceExamples", {
        defaultValue: "Create or import a workspace",
      }),
      href: appRoutes.design,
      Icon: Layers,
    };
    const componentAssets = assets.filter((asset) => assetGroup(asset) === "components");
    const commandCount = commands.response?.total ?? commands.results.length;
    const commandExamples = commands.results.slice(0, 2).map((command) => command.title);
    return [
      landing,
      onboarding,
      workspaceEntry,
      {
        id: "components",
        title: t("workbench.start.components", { defaultValue: "Components" }),
        countValue: componentAssets.length,
        pagesValue: 0,
        count: t("workbench.start.componentAssetsCount", {
          count: componentAssets.length,
          defaultValue: "{{count}} assets",
        }),
        examples: componentAssets
          .slice(0, 3)
          .map((asset) => asset.displayName)
          .join(" · "),
        href: `${appRoutes.library}?group=components`,
        Icon: Blocks,
      },
      {
        id: "templates",
        title: t("workbench.start.mobileTemplates", { defaultValue: "Templates" }),
        countValue: curated.length,
        pagesValue: curated.reduce((total, workspace) => total + workspace.pageCount, 0),
        count: t("workbench.start.curatedTemplateCount", {
          count: curated.length,
          defaultValue: "{{count}} curated",
        }),
        examples:
          curated
            .slice(0, 2)
            .map((workspace) => workspace.name)
            .join(" · ") ||
          t("workbench.start.curatedTemplateExamples", {
            defaultValue: "Browse the workspace shelf",
          }),
        href: `${appRoutes.design}#curated-templates`,
        Icon: LayoutTemplate,
      },
      {
        id: "actions",
        title: t("workbench.start.actions", { defaultValue: "Actions" }),
        countValue: commandCount,
        pagesValue: 0,
        count: commands.loading
          ? t("workbench.start.loadingCommands", { defaultValue: "Loading commands…" })
          : commands.error
            ? t("workbench.start.commandsUnavailable", { defaultValue: "Commands unavailable" })
            : t("workbench.start.commandCount", {
                count: commandCount,
                defaultValue: "{{count}} commands",
              }),
        examples:
          commandExamples.join(" · ") ||
          t("workbench.start.commandExamples", {
            defaultValue: "Browse indexed workbench actions",
          }),
        href: MOBILE_COMMANDS_URL,
        Icon: Command,
      },
    ];
  }, [assets, browseEntries, commands, curated, t]);

  return { curated, recent, browseEntries, mobilePrimaryIntents };
}
