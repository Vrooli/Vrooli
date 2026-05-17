import { ChevronRight } from "lucide-react";
import { Link } from "react-router-dom";
import { Panel } from "../../components/ui/panel";
import { PageHeader } from "../../components/composites/PageHeader";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

interface DocItem {
  title: string;
  path: string;
  description: string;
  group: "PRD" | "Concepts" | "Reference" | "Internal";
}

const DOCS: DocItem[] = [
  { title: "PRD", path: "PRD.md", description: "Operational targets and scope", group: "PRD" },
  { title: "Architecture", path: "docs/concepts/ARCHITECTURE.md", description: "Surface map and load-bearing principles", group: "Concepts" },
  { title: "Domains", path: "docs/concepts/DOMAINS.md", description: "Capability domain inventory", group: "Concepts" },
  { title: "Flows", path: "docs/concepts/FLOWS.md", description: "End-to-end request flows", group: "Concepts" },
  { title: "Integrations", path: "docs/concepts/INTEGRATIONS.md", description: "Resource and scenario dependencies", group: "Concepts" },
  { title: "API endpoints", path: "docs/reference/api-endpoints.md", description: "Connect-RPC + REST exception inventory", group: "Reference" },
  { title: "CLI commands", path: "docs/reference/cli-commands.md", description: "audio-tools subcommand reference", group: "Reference" },
  { title: "Configuration", path: "docs/reference/configuration.md", description: "Env vars and runtime knobs", group: "Reference" },
  { title: "Adoption snippets", path: "docs/reference/adoption.md", description: "Copy-paste snippets for consumer scenarios", group: "Reference" },
  { title: "Invariants", path: "docs/internal/INVARIANTS.md", description: "Non-negotiable contracts", group: "Internal" },
  { title: "Seams", path: "docs/internal/SEAMS.md", description: "Test seams + cross-scenario boundaries", group: "Internal" },
  { title: "Extraction sources", path: "docs/internal/EXTRACTION-SOURCES.md", description: "Source-by-source migration provenance", group: "Internal" },
];

export function DocsPage() {
  const { t } = useTranslation();
  const groupTitle: Record<DocItem["group"], string> = {
    PRD: t(strings.docs.groupPRD),
    Concepts: t(strings.docs.groupConcepts),
    Reference: t(strings.docs.groupReference),
    Internal: t(strings.docs.groupInternal),
  };
  const groups: DocItem["group"][] = ["PRD", "Concepts", "Reference", "Internal"];
  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.docs.title)} description={t(strings.docs.description)} />
      {groups.map((g) => (
        <Panel key={g} title={groupTitle[g]}>
          <ul className="grid gap-2 md:grid-cols-2">
            {DOCS.filter((d) => d.group === g).map((d) => (
              <li key={d.path}>
                <Link
                  to={`/docs/${d.path}`}
                  className="group flex items-start justify-between gap-2 rounded-control border border-app-border bg-app-surface-muted/40 p-3 transition hover:border-app-primary hover:bg-app-surface-muted/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
                >
                  <div className="min-w-0">
                    <p className="font-medium text-app-foreground">{d.title}</p>
                    <p className="text-xs text-app-muted-foreground">{d.description}</p>
                    <p className="mt-1 font-mono text-[11px] text-app-muted-foreground">{d.path}</p>
                  </div>
                  <ChevronRight
                    className="h-4 w-4 shrink-0 text-app-muted-foreground transition group-hover:translate-x-0.5 group-hover:text-app-primary"
                    aria-hidden="true"
                  />
                </Link>
              </li>
            ))}
          </ul>
        </Panel>
      ))}
    </div>
  );
}
