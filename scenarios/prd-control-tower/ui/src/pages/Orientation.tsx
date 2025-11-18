import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertCircle, BookOpen, Compass, FileText, Layers, ShieldAlert, Sparkles, Target, Workflow } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Separator } from '../components/ui/separator'
import { TopNav } from '../components/ui/top-nav'
import { buildApiUrl } from '../utils/apiClient'
import { fetchQualitySummary } from '../utils/quality'
import type { CatalogResponse, QualitySummary } from '../types'

const TEMPLATE_SECTIONS = [
  {
    title: 'Overview',
    subtitle: '## 🎯 Overview',
    description:
      'Set the context: what permanent capability are we adding, who benefits, and where does it run (CLI/API/UI/automation). Keep it non-technical.',
  },
  {
    title: 'Operational Targets',
    subtitle: '## 🎯 Operational Targets',
    description:
      'Define P0/P1/P2 outcomes as single-line checklists. IDs stay stable forever; requirements link to these via `prd_ref`.',
  },
  {
    title: 'Tech Direction Snapshot',
    subtitle: '## 🧱 Tech Direction Snapshot',
    description:
      'Document preferred stacks, storage expectations, and non-goals without dropping into implementation detail.',
  },
  {
    title: 'Dependencies & Launch Plan',
    subtitle: '## 🤝 Dependencies & Launch Plan',
    description:
      'List required resources, upstream scenarios, risks, and launch sequencing so ops knows how to deploy the capability.',
  },
  {
    title: 'UX & Branding',
    subtitle: '## 🎨 UX & Branding',
    description:
      'Describe intended look/feel, accessibility bar, and product voice so designers know what “done” looks like.',
  },
]

const PRD_EXAMPLES: Record<'skeleton' | 'full', string> = {
  skeleton: `# Scenario Name

## 🎯 Overview
- **Purpose**: Describe the permanent capability
- **Users**: Operators, admins
- **Surfaces**: CLI, API, UI

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Outcome title | One-line description

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Outcome title | One-line description

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Outcome title | One-line description

## 🧱 Tech Direction Snapshot
- Preferred stacks, storage, integration strategy, non-goals

## 🤝 Dependencies & Launch Plan
- Required resources, scenario dependencies, risks, sequencing

## 🎨 UX & Branding
- Look/feel, accessibility, voice, branding hooks
`,
  full: `# PRD Control Tower

## 🎯 Overview
- **Purpose**: Keep every scenario/resource stocked with a compliant, machine-readable PRD and surface drift instantly.
- **Users**: Product ops, scenario maintainers, ecosystem-manager agents.
- **Surfaces**: React UI (draft workspace + catalog), Go API, Bash CLI, automation hooks.

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Catalog coverage analytics | Track total entities, PRD coverage, draft backlog in real time.
- [ ] OT-P0-002 | Draft validation autopilot | Run template + requirements checks in < 8s for 95th percentile drafts.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Diff-aware publish workflow | Human confirms rendered diff before PRD.md is replaced.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Tenant-specific PRD templates | Org-level overrides that stay machine-readable.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go (API), React/TypeScript (UI), Bun/Vite tooling, Postgres for catalog metadata.
- Data/storage: filesystem for drafts, Postgres for metadata, qdrant optional for semantic search.
- Integration: reuse scenario-auditor for validation, ecosystem-manager for generation tasks, resource-openrouter for AI assists.
- Non-goals: editing downstream requirements, generating business logic, or bypassing lifecycle commands.

## 🤝 Dependencies & Launch Plan
- Required resources: postgres, scenario-auditor, resource-openrouter (optional AI), filesystem storage.
- Scenario dependencies: ecosystem-manager (task dispatch), app-monitor (status ingest).
- Risks: stale legacy PRDs, draft noise, AI hallucinations.
- Launch sequencing: (1) bootstrap catalog scan, (2) ship draft workspace GA, (3) enable publish guardrails, (4) migrate legacy PRDs.

## 🎨 UX & Branding
- Look & feel: bright, card-based workspace mirroring core Vrooli brand (violet/orange palette, rounded corners, generous whitespace).
- Accessibility: keyboard-first editing, WCAG AA contrast for chips, screen-reader friendly descriptions for validation states.
- Voice: confident operations voice with short, actionable copy (“Fix 3 missing sections”).
- Branding hooks: sparkles + control tower iconography to reinforce guidance narrative.
`,
}

const WORKFLOW_STEPS = [
  {
    title: 'Capture backlog intent',
    description:
      'Start in Backlog. Capture loose opportunities, cluster ideas, and promote promising ones into drafts when they have signal.',
  },
  {
    title: 'Draft & lock targets',
    description:
      'Use the draft workspace, AI helpers, and validation panels to build the PRD spine, lock operational targets, then seed `requirements/` with matching entries.',
  },
  {
    title: 'Publish & scaffold scenario',
    description:
      'Run the publish wizard to choose a template, create the scenario, and copy your draft into PRD.md so it becomes the source of truth.',
  },
  {
    title: 'Wire requirements & tests',
    description:
      'Link operational targets to requirement registries and map acceptance tests so the control tower can reason over coverage in real-time.',
  },
  {
    title: 'Dispatch ecosystem-manager',
    description:
      'From the Scenario Control Center, create an ecosystem-manager task so an agent keeps iterating until all requirements and targets are green.',
  },
]

const ACTION_TILES = [
  {
    title: 'View Coverage Catalog',
    description: 'See who has published PRDs, drafts in progress, and entities missing coverage.',
    to: '/catalog',
    icon: Compass,
  },
  {
    title: 'Open Draft Workspace',
    description: 'Continue editing, validate sections, and publish when ready.',
    to: '/drafts',
    icon: Layers,
  },
  {
    title: 'Review Requirements Registry',
    description: 'Ensure every requirement has a `prd_ref` and keep operational targets aligned.',
    to: '/requirements-registry',
    icon: Target,
  },
  {
    title: 'Run Quality Scanner',
    description: 'Batch-audit PRDs for template drift, missing targets, and coverage gaps.',
    to: '/quality-scanner',
    icon: ShieldAlert,
  },
]

export default function Orientation() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [summary, setSummary] = useState({ total: 0, withPrd: 0, drafts: 0 })
  const [qualitySummary, setQualitySummary] = useState<QualitySummary | null>(null)
  const [qualityError, setQualityError] = useState<string | null>(null)
  const [exampleMode, setExampleMode] = useState<'skeleton' | 'full'>('skeleton')

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        setError(null)
        const response = await fetch(buildApiUrl('/catalog'))
        if (!response.ok) {
          throw new Error(`Failed to load catalog summary: ${response.statusText}`)
        }
        const data: CatalogResponse = await response.json()
        const stats = data.entries.reduce(
          (acc, entry) => {
            acc.total += 1
            if (entry.has_prd) acc.withPrd += 1
            if (entry.has_draft) acc.drafts += 1
            return acc
          },
          { total: 0, withPrd: 0, drafts: 0 },
        )
        setSummary(stats)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [])

  const coverage = useMemo(() => {
    if (summary.total === 0) return 0
    return Math.round((summary.withPrd / summary.total) * 100)
  }, [summary])

  useEffect(() => {
    let mounted = true
    fetchQualitySummary()
      .then((data) => {
        if (mounted) {
          setQualitySummary(data)
          setQualityError(null)
        }
      })
      .catch((err) => {
        if (mounted) {
          setQualitySummary(null)
          setQualityError(err instanceof Error ? err.message : 'Quality summary unavailable')
        }
      })

    return () => {
      mounted = false
    }
  }, [])

  const qualityStats = useMemo(() => {
    if (!qualitySummary) return []
    return [
      { label: 'Tracked entities', value: qualitySummary.total_entities },
      { label: 'Missing PRDs', value: qualitySummary.missing_prd },
      { label: 'With issues', value: qualitySummary.with_issues },
    ]
  }, [qualitySummary])

  return (
    <div className="app-container space-y-8">
      <TopNav />

      <header className="rounded-3xl border bg-white/95 p-8 shadow-soft-lg">
        <div className="space-y-4">
          <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">PRD Operating Guide</span>
          <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
            <span className="rounded-2xl bg-amber-100 p-3 text-amber-600">
              <BookOpen size={28} strokeWidth={2.5} />
            </span>
            Understand the Control Tower
          </div>
          <p className="max-w-3xl text-base text-muted-foreground">
            Every scenario becomes permanent intelligence. Use this primer to understand the canonical PRD format, validation workflow, and where to take action next—including the moment to fire the scanner and keep coverage honest.
          </p>
          <div className="flex flex-wrap gap-3">
            <Button size="lg" asChild>
              <Link to="/catalog">Dive into catalog</Link>
            </Button>
            <Button variant="secondary" size="lg" asChild>
              <Link to="/drafts">Jump to drafts</Link>
            </Button>
            <Button variant="outline" size="lg" asChild>
              <Link to="/quality-scanner">Run quality scan</Link>
            </Button>
          </div>
        </div>
      </header>

      <section className="grid gap-4 lg:grid-cols-[2fr,1fr]">
        <Card className="border bg-white/95">
          <CardHeader>
            <CardTitle className="text-lg">Current Coverage</CardTitle>
            <CardDescription>Live metrics straight from the catalog service.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid gap-6 sm:grid-cols-3">
              <div>
                <p className="text-sm text-muted-foreground">Total entities</p>
                <p className="text-3xl font-semibold text-slate-900">
                  {loading ? '—' : summary.total.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Published PRDs</p>
                <p className="text-2xl font-semibold text-emerald-600">{loading ? '—' : summary.withPrd}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Active drafts</p>
                <p className="text-2xl font-semibold text-slate-900">{loading ? '—' : summary.drafts}</p>
              </div>
            </div>
            <Separator />
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <p className="text-sm text-muted-foreground">Coverage</p>
                <p className="text-4xl font-semibold text-emerald-600">{loading ? '—' : `${coverage}%`}</p>
              </div>
              <div className="flex items-center gap-3 rounded-2xl border border-dashed bg-slate-50 px-4 py-2 text-sm text-slate-700">
                <Sparkles size={18} className="text-violet-500" />
                Keep drafts moving to lift the percentage.
              </div>
            </div>
            {error && (
              <p className="text-sm text-amber-600">
                <AlertCircle className="mr-2 inline" size={14} /> {error}
              </p>
            )}
          </CardContent>
        </Card>

        <Card className="border bg-white/95">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <ShieldAlert size={18} className="text-rose-500" /> Quality health snapshot
            </CardTitle>
            <CardDescription>Quick pulse on tracked entities and issue counts.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-2 text-sm text-slate-600">
              {qualityStats.length === 0 && <span>—</span>}
              {qualityStats.map((stat) => (
                <span key={stat.label}>
                  <strong className="text-slate-900">{stat.value}</strong> {stat.label}
                </span>
              ))}
            </div>
            {qualitySummary?.last_generated && (
              <p className="text-xs text-muted-foreground">
                Updated {new Date(qualitySummary.last_generated).toLocaleString()}
              </p>
            )}
            {qualityError && <p className="text-xs text-amber-600">{qualityError}</p>}
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr),minmax(0,1fr)]">
        <Card className="border bg-white/90 min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-xl">
              <FileText size={20} /> Canonical PRD spine
            </CardTitle>
            <CardDescription>
              These sections come directly from <code>scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md</code> and are enforced by scenario-auditor.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {TEMPLATE_SECTIONS.map((section) => (
              <div key={section.title} className="rounded-2xl border bg-slate-50 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                  {section.subtitle}
                </p>
                <p className="text-lg font-semibold text-slate-900">{section.title}</p>
                <p className="text-sm text-slate-600">{section.description}</p>
              </div>
            ))}
            <div className="rounded-2xl border border-dashed bg-white p-4 min-w-0">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="text-sm font-semibold text-slate-800">Example view</p>
                <div className="flex gap-2">
                  <Button
                    variant={exampleMode === 'skeleton' ? 'default' : 'ghost'}
                    size="sm"
                    onClick={() => setExampleMode('skeleton')}
                  >
                    Skeleton
                  </Button>
                  <Button
                    variant={exampleMode === 'full' ? 'default' : 'ghost'}
                    size="sm"
                    onClick={() => setExampleMode('full')}
                  >
                    Full example
                  </Button>
                </div>
              </div>
              <div
                className="mt-3 max-h-[28rem] w-full overflow-x-auto overflow-y-auto rounded-xl border border-slate-200 bg-slate-900 p-4 text-sm shadow-inner"
                tabIndex={0}
              >
                <pre className="block min-w-full whitespace-pre text-xs leading-relaxed text-slate-100">
{PRD_EXAMPLES[exampleMode]}
                </pre>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border bg-white/90 min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-xl">
              <Workflow size={20} /> Lifecycle overview
            </CardTitle>
            <CardDescription>
              Every scenario flows backlog → draft → publish → requirements → ecosystem-manager. This loop keeps PRDs shippable and executable.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {WORKFLOW_STEPS.map((step, index) => (
              <div key={step.title} className="relative pl-10">
                <span className="absolute left-0 top-0 flex h-8 w-8 items-center justify-center rounded-full bg-violet-100 text-sm font-semibold text-violet-700">
                  {index + 1}
                </span>
                <p className="font-semibold text-slate-900">{step.title}</p>
                <p className="text-sm text-slate-600">{step.description}</p>
              </div>
            ))}
            <div className="rounded-2xl border border-dashed bg-slate-50 p-4 text-sm text-slate-600">
              🔗 Need deep dives? See <a className="text-primary underline" href="https://github.com/vrooli/Vrooli/blob/main/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md" target="_blank" rel="noreferrer">Canonical PRD Template</a> and <a className="text-primary underline" href="https://github.com/vrooli/Vrooli/blob/main/docs/testing/architecture/REQUIREMENT_FLOW.md" target="_blank" rel="noreferrer">Requirement Flow</a>.
            </div>
            <div className="rounded-2xl border border-violet-200 bg-violet-50 p-4 text-sm text-violet-900">
              Close the loop by running the <Link to="/quality-scanner" className="font-semibold underline">Quality Scanner</Link> after each publish. It flags template drift, missing targets, and stale `prd_ref` links within seconds.
            </div>
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        {ACTION_TILES.map(({ title, description, to, icon: Icon }) => (
          <Card key={title} className="border bg-white/90 transition hover:-translate-y-1 hover:border-violet-200 hover:shadow-lg">
            <CardContent className="flex h-full flex-col gap-3 p-5">
              <div className="flex items-center gap-3">
                <span className="rounded-2xl bg-violet-100 p-2 text-violet-600">
                  <Icon size={20} />
                </span>
                <p className="text-lg font-semibold text-slate-900">{title}</p>
              </div>
              <p className="text-sm text-slate-600 flex-1">{description}</p>
              <Button asChild variant="link" className="self-start px-0">
                <Link to={to} className="flex items-center gap-2">
                  Go now
                  <span aria-hidden="true">→</span>
                </Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="rounded-3xl border bg-slate-900 p-6 text-slate-100">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-sm uppercase tracking-[0.3em] text-violet-200">Need a nudge?</p>
            <p className="text-2xl font-semibold">AI assistants and validators are wired in.</p>
            <p className="text-sm text-slate-300">
              Use the draft workspace to run validation, call the AI section generator, or jump into requirements coverage dashboards.
            </p>
          </div>
          <Button variant="secondary" size="lg" asChild className="text-slate-900">
            <Link to="/drafts">Open draft workspace</Link>
          </Button>
        </div>
      </section>
    </div>
  )
}
