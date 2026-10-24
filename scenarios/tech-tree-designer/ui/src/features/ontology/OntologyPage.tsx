import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, CircleDot, Layers3, Link2, Network, RefreshCw, Search, Target, X } from "lucide-react";
import {
  CapabilityKind,
  CoverageState,
  FocusReason,
  OverlayEvidenceSource,
  OverlayNodeKind,
  type Capability,
  type CoverageClassification,
  type Fulfillment,
  type OverlayEdge,
  type OverlayNode,
} from "@vrooli/proto-types/tech-tree-designer/v1/ontology/ontology_pb";

import { ontologyClient, describeOverlayGraph, getOntologyCoverage, listCapabilities, listFulfillments, listOntologyFocus, unlinkFulfillment } from "../../api/techTree";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const percent = (value: number) => `${Math.round(value * 100)}%`;

const capabilityKindLabel = (kind: CapabilityKind) => {
  switch (kind) {
    case CapabilityKind.SECTOR:
      return "Sector";
    case CapabilityKind.CAPABILITY:
      return "Capability";
    case CapabilityKind.COMPONENT:
      return "Component";
    case CapabilityKind.CAPSTONE:
      return "Capstone";
    case CapabilityKind.SIMULATION:
      return "Simulation";
    default:
      return "Capability";
  }
};

const coverageStateLabel = (state: CoverageState) => {
  switch (state) {
    case CoverageState.BUILT:
      return "Built";
    case CoverageState.IN_FLIGHT:
      return "In-flight";
    case CoverageState.GAP:
      return "Gap";
    case CoverageState.UNMAPPED:
      return "Unmapped";
    default:
      return "Unknown";
  }
};

const focusReasonLabel = (reason: FocusReason) => {
  switch (reason) {
    case FocusReason.GAP:
      return "Gap";
    case FocusReason.CLOSEST_TO_DONE:
      return "Closest";
    case FocusReason.UNMAPPED_SCENARIO:
      return "Unmapped";
    default:
      return "Focus";
  }
};

const evidenceLabel = (edge: OverlayEdge) => {
  const evidence = edge.evidence[0];
  switch (evidence?.source) {
    case OverlayEvidenceSource.DECOMPOSES:
      return "decomposes";
    case OverlayEvidenceSource.FULFILLS:
      return "fulfills";
    case OverlayEvidenceSource.PLANNED_PROTO_IMPORT:
      return "planned import";
    case OverlayEvidenceSource.PROTO_IMPORT:
      return "proto import";
    case OverlayEvidenceSource.GO_IMPORT:
      return "go import";
    default:
      return "edge";
  }
};

const overlayNodeLabel = (node: OverlayNode) => node.displayName || node.scenario;

const overlayNodeTypeLabel = (kind: OverlayNodeKind) => {
  if (kind === OverlayNodeKind.CAPABILITY) return "capability";
  if (kind === OverlayNodeKind.PLANNED) return "planned";
  if (kind === OverlayNodeKind.LIVE) return "live";
  return "node";
};

function buildChildren(capabilities: Capability[]) {
  const byParent = new Map<string, Capability[]>();
  for (const capability of capabilities) {
    const key = capability.parentId || "";
    byParent.set(key, [...(byParent.get(key) ?? []), capability]);
  }
  for (const entries of byParent.values()) {
    entries.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
  }
  return byParent;
}

export function OntologyPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState("");
  const [scenarioSlug, setScenarioSlug] = useState("");
  const [filter, setFilter] = useState("");
  const [view, setView] = useState<"ontology" | "implementation" | "overlay">("overlay");

  const capabilitiesQuery = useQuery({ queryKey: ["ontology-capabilities"], queryFn: listCapabilities });
  const coverageQuery = useQuery({ queryKey: ["ontology-coverage"], queryFn: getOntologyCoverage });
  const focusQuery = useQuery({ queryKey: ["ontology-focus"], queryFn: listOntologyFocus });
  const fulfillmentsQuery = useQuery({ queryKey: ["ontology-fulfillments"], queryFn: listFulfillments });
  const overlayQuery = useQuery({ queryKey: ["ontology-overlay"], queryFn: describeOverlayGraph });

  const capabilities = useMemo(() => capabilitiesQuery.data ?? [], [capabilitiesQuery.data]);
  const selectedCapability = capabilities.find((capability) => capability.id === selectedId) ?? capabilities[0];
  const fulfillmentByCapability = useMemo(() => {
    const next = new Map<string, Fulfillment[]>();
    for (const fulfillment of fulfillmentsQuery.data ?? []) {
      next.set(fulfillment.capabilityId, [...(next.get(fulfillment.capabilityId) ?? []), fulfillment]);
    }
    return next;
  }, [fulfillmentsQuery.data]);
  const classifications = useMemo(() => {
    const next = new Map<string, CoverageClassification>();
    for (const item of coverageQuery.data?.classifications ?? []) {
      next.set(item.capabilityId, item);
    }
    return next;
  }, [coverageQuery.data?.classifications]);
  const children = useMemo(() => buildChildren(capabilities), [capabilities]);
  const filteredRoots = useMemo(() => {
    const roots = children.get("") ?? [];
    if (!filter.trim()) return roots;
    const needle = filter.trim().toLowerCase();
    const matchingIds = new Set(
      capabilities
        .filter((capability) =>
          [capability.slug, capability.name, capability.description].some((value) => value.toLowerCase().includes(needle)),
        )
        .map((capability) => capability.id),
    );
    return roots.filter((root) => subtreeHasMatch(root, children, matchingIds));
  }, [capabilities, children, filter]);

  const linkMutation = useMutation({
    mutationFn: async () => {
      if (!selectedCapability || !scenarioSlug.trim()) return undefined;
      return ontologyClient.linkFulfillment({
        fulfillment: {
          capabilityId: selectedCapability.id,
          scenarioSlug: scenarioSlug.trim(),
          note: "",
          createdAt: "",
        },
      });
    },
    onSuccess: async () => {
      setScenarioSlug("");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["ontology-coverage"] }),
        queryClient.invalidateQueries({ queryKey: ["ontology-fulfillments"] }),
        queryClient.invalidateQueries({ queryKey: ["ontology-overlay"] }),
      ]);
    },
  });
  const unlinkMutation = useMutation({
    mutationFn: unlinkFulfillment,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["ontology-coverage"] }),
        queryClient.invalidateQueries({ queryKey: ["ontology-fulfillments"] }),
        queryClient.invalidateQueries({ queryKey: ["ontology-overlay"] }),
      ]);
    },
  });

  const refreshAll = async () => {
    await Promise.all([
      capabilitiesQuery.refetch(),
      coverageQuery.refetch(),
      focusQuery.refetch(),
      fulfillmentsQuery.refetch(),
      overlayQuery.refetch(),
    ]);
  };

  const overlay = overlayQuery.data;
  const visibleOverlayNodes = useMemo(() => overlay?.nodes.filter((node) => {
    if (view === "ontology") return node.kind === OverlayNodeKind.CAPABILITY;
    if (view === "implementation") return node.kind !== OverlayNodeKind.CAPABILITY;
    return true;
  }) ?? [], [overlay?.nodes, view]);
  const visibleNodeIds = useMemo(() => new Set(visibleOverlayNodes.map((node) => node.scenario)), [visibleOverlayNodes]);
  const visibleOverlayEdges = useMemo(
    () => overlay?.edges.filter((edge) => visibleNodeIds.has(edge.fromScenario) && visibleNodeIds.has(edge.toScenario)) ?? [],
    [overlay?.edges, visibleNodeIds],
  );
  const overlayNeighborhood = useMemo(() => {
    if (!selectedCapability || view === "implementation") return undefined;
    const ids = new Set<string>([selectedCapability.id]);
    for (const node of visibleOverlayNodes) {
      if (node.parent === selectedCapability.id) ids.add(node.scenario);
      if (node.scenario === selectedCapability.id && node.parent) ids.add(node.parent);
    }
    for (const edge of visibleOverlayEdges) {
      if (ids.has(edge.fromScenario)) ids.add(edge.toScenario);
      if (ids.has(edge.toScenario)) ids.add(edge.fromScenario);
    }
    return ids;
  }, [selectedCapability, view, visibleOverlayEdges, visibleOverlayNodes]);
  const overlayMapNodes = (overlayNeighborhood
    ? visibleOverlayNodes.filter((node) => overlayNeighborhood.has(node.scenario))
    : visibleOverlayNodes).slice(0, 72);
  const overlayMapNodeIds = new Set(overlayMapNodes.map((node) => node.scenario));
  const overlayMapEdges = visibleOverlayEdges
    .filter((edge) => overlayMapNodeIds.has(edge.fromScenario) && overlayMapNodeIds.has(edge.toScenario))
    .slice(0, 120);

  return (
    <section data-testid={selectors.pages.ontology} className="flex flex-col gap-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-sm font-medium uppercase text-app-muted-foreground">{t(strings.ontology.eyebrow)}</p>
          <h2 className="text-3xl font-semibold">{t(strings.ontology.title)}</h2>
          <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">{t(strings.ontology.description)}</p>
        </div>
        <Button variant="outline" onClick={() => void refreshAll()}>
          <RefreshCw aria-hidden className="mr-2 h-4 w-4" />
          {t(strings.ontology.actions.refresh)}
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <Metric icon={<Layers3 aria-hidden className="h-5 w-5" />} label={t(strings.ontology.metrics.capabilities)} value={coverageQuery.data?.totalCapabilities ?? capabilities.length} />
        <Metric icon={<Check aria-hidden className="h-5 w-5" />} label={t(strings.ontology.metrics.ontologyCompleteness)} value={percent(coverageQuery.data?.ontologyCompleteness ?? 0)} />
        <Metric icon={<Target aria-hidden className="h-5 w-5" />} label={t(strings.ontology.metrics.implementationSituatedness)} value={percent(coverageQuery.data?.implementationSituatedness ?? 0)} />
        <Metric icon={<CircleDot aria-hidden className="h-5 w-5" />} label={t(strings.ontology.metrics.unmapped)} value={coverageQuery.data?.unmappedScenarios ?? 0} />
      </div>

      {(capabilitiesQuery.isLoading || coverageQuery.isLoading) && <StatePanel label={t(strings.ontology.states.loading)} />}
      {(capabilitiesQuery.error || coverageQuery.error) && <StatePanel label={t(strings.ontology.states.error)} tone="error" />}

      <div className="grid gap-5 xl:grid-cols-[minmax(300px,0.9fr)_minmax(0,1.3fr)]">
        <div className="flex flex-col gap-4">
          <section className="rounded-lg border border-app-border bg-app-surface p-4">
            <div className="flex items-center gap-2">
              <Search aria-hidden className="h-4 w-4 text-app-muted-foreground" />
              <input
                value={filter}
                onChange={(event) => setFilter(event.target.value)}
                placeholder={t(strings.ontology.tree.filterPlaceholder)}
                className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-app-muted-foreground"
              />
            </div>
            <div className="mt-4 max-h-[520px] overflow-auto pr-1">
              {filteredRoots.length === 0 && !capabilitiesQuery.isLoading ? (
                <p className="text-sm text-app-muted-foreground">{t(strings.ontology.states.empty)}</p>
              ) : (
                filteredRoots.map((capability) => (
                  <CapabilityBranch
                    key={capability.id}
                    capability={capability}
                    childrenByParent={children}
                    selectedId={selectedCapability?.id ?? ""}
                    onSelect={setSelectedId}
                    classifications={classifications}
                    depth={0}
                    matchingFilter={filter}
                  />
                ))
              )}
            </div>
          </section>

          <section className="rounded-lg border border-app-border bg-app-surface p-4">
            <h3 className="text-sm font-semibold">{t(strings.ontology.focus.title)}</h3>
            <div className="mt-3 space-y-2">
              {(focusQuery.data?.items ?? []).map((item) => (
                <button
                  key={`${item.reason}-${item.capabilityId}-${item.capabilitySlug}`}
                  type="button"
                  onClick={() => item.capabilityId && setSelectedId(item.capabilityId)}
                  className="w-full rounded-md border border-app-border bg-app-background px-3 py-2 text-left text-sm transition hover:border-app-primary"
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-medium">{item.capabilityName || item.capabilitySlug}</span>
                    <span className="rounded-full bg-app-primary/15 px-2 py-0.5 text-xs text-app-primary">{focusReasonLabel(item.reason)}</span>
                  </div>
                  <p className="mt-1 text-xs text-app-muted-foreground">
                    {t(strings.ontology.focus.score, { score: item.score.toFixed(1), dependents: item.downstreamDependents })}
                  </p>
                </button>
              ))}
              {!focusQuery.isLoading && (focusQuery.data?.items.length ?? 0) === 0 && (
                <p className="text-sm text-app-muted-foreground">{t(strings.ontology.focus.empty)}</p>
              )}
            </div>
          </section>
        </div>

        <div className="flex flex-col gap-4">
          <section className="rounded-lg border border-app-border bg-app-surface p-4">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <p className="text-xs uppercase text-app-muted-foreground">{selectedCapability ? capabilityKindLabel(selectedCapability.kind) : t(strings.ontology.detail.none)}</p>
                <h3 className="mt-1 text-xl font-semibold">{selectedCapability?.name || selectedCapability?.slug || t(strings.ontology.detail.none)}</h3>
                <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">
                  {selectedCapability?.description || t(strings.ontology.detail.noDescription)}
                </p>
              </div>
              {selectedCapability ? (
                <span className="rounded-full bg-app-primary/15 px-3 py-1 text-xs font-medium text-app-primary">
                  {coverageStateLabel(classifications.get(selectedCapability.id)?.state ?? CoverageState.UNSPECIFIED)}
                </span>
              ) : null}
            </div>

            <div className="mt-4 grid gap-3 md:grid-cols-3">
              <DetailStat label={t(strings.ontology.detail.direct)} value={classifications.get(selectedCapability?.id ?? "")?.directlyFulfilled ? t(strings.ontology.detail.yes) : t(strings.ontology.detail.no)} />
              <DetailStat label={t(strings.ontology.detail.subtree)} value={classifications.get(selectedCapability?.id ?? "")?.subtreeCovered ? t(strings.ontology.detail.yes) : t(strings.ontology.detail.no)} />
              <DetailStat label={t(strings.ontology.detail.fulfillments)} value={fulfillmentByCapability.get(selectedCapability?.id ?? "")?.length ?? 0} />
            </div>

            <form
              className="mt-4 flex flex-col gap-2 sm:flex-row"
              onSubmit={(event) => {
                event.preventDefault();
                void linkMutation.mutateAsync();
              }}
            >
              <input
                value={scenarioSlug}
                onChange={(event) => setScenarioSlug(event.target.value)}
                placeholder={t(strings.ontology.fulfillment.placeholder)}
                className="h-10 min-w-0 flex-1 rounded-md border border-app-border bg-app-background px-3 text-sm"
              />
              <Button type="submit" disabled={!selectedCapability || !scenarioSlug.trim() || linkMutation.isPending}>
                <Link2 aria-hidden className="mr-2 h-4 w-4" />
                {t(strings.ontology.fulfillment.link)}
              </Button>
            </form>

            <div className="mt-4 flex flex-wrap gap-2">
              {(fulfillmentByCapability.get(selectedCapability?.id ?? "") ?? []).map((fulfillment) => (
                <span key={`${fulfillment.capabilityId}-${fulfillment.scenarioSlug}`} className="inline-flex items-center gap-2 rounded-full bg-app-background px-3 py-1 text-xs text-app-muted-foreground">
                  <button
                    type="button"
                    onClick={() => setScenarioSlug(fulfillment.scenarioSlug)}
                    className="max-w-52 truncate text-left hover:text-app-foreground"
                  >
                    {fulfillment.scenarioSlug}
                  </button>
                  <button
                    type="button"
                    onClick={() => void unlinkMutation.mutateAsync({
                      capabilityId: fulfillment.capabilityId,
                      scenarioSlug: fulfillment.scenarioSlug,
                    })}
                    className="rounded-full p-0.5 hover:bg-app-border hover:text-app-foreground"
                    aria-label={`Remove ${fulfillment.scenarioSlug}`}
                    disabled={unlinkMutation.isPending}
                  >
                    <X aria-hidden className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-app-border bg-app-surface p-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <h3 className="text-sm font-semibold">{t(strings.ontology.coverage.title)}</h3>
              <div className="flex rounded-md border border-app-border p-1">
                {(["ontology", "implementation", "overlay"] as const).map((mode) => (
                  <button
                    key={mode}
                    type="button"
                    onClick={() => setView(mode)}
                    className={[
                      "rounded px-3 py-1 text-xs font-medium capitalize",
                      view === mode ? "bg-app-primary text-white" : "text-app-muted-foreground hover:text-app-foreground",
                    ].join(" ")}
                  >
                    {mode}
                  </button>
                ))}
              </div>
            </div>
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              {(coverageQuery.data?.sectors ?? []).slice(0, 8).map((sector) => (
                <div key={sector.sectorId} className="rounded-md border border-app-border bg-app-background p-3">
                  <div className="flex items-center justify-between gap-3 text-sm">
                    <span className="font-medium">{sector.sectorName || sector.sectorSlug}</span>
                    <span className="text-app-muted-foreground">{percent(sector.ontologyCompleteness)}</span>
                  </div>
                  <div className="mt-2 h-2 overflow-hidden rounded-full bg-app-border">
                    <div className="h-full rounded-full bg-app-primary" style={{ width: percent(sector.ontologyCompleteness) }} />
                  </div>
                  <p className="mt-2 text-xs text-app-muted-foreground">
                    {t(strings.ontology.coverage.counts, { built: sector.builtCapabilities, inflight: sector.inflightCapabilities, gap: sector.gapCapabilities })}
                  </p>
                </div>
              ))}
            </div>

            <div className="mt-4 rounded-lg border border-app-border bg-app-background p-3">
              <div className="flex items-center gap-2 text-sm font-medium">
                <Network aria-hidden className="h-4 w-4" />
                {t(strings.ontology.overlay.title, { nodes: visibleOverlayNodes.length, edges: visibleOverlayEdges.length })}
              </div>
              <OverlayMap
                nodes={overlayMapNodes}
                edges={overlayMapEdges}
                ariaLabel={t(strings.ontology.overlay.ariaLabel)}
                selectedCapabilityId={selectedCapability?.id ?? ""}
                onCapabilitySelect={setSelectedId}
                onScenarioSelect={setScenarioSlug}
              />
              {!overlayQuery.isLoading && visibleOverlayNodes.length === 0 && (
                <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.ontology.overlay.empty)}</p>
              )}
            </div>
          </section>
        </div>
      </div>
    </section>
  );
}

function subtreeHasMatch(capability: Capability, children: Map<string, Capability[]>, matchingIds: Set<string>): boolean {
  if (matchingIds.has(capability.id)) return true;
  return (children.get(capability.id) ?? []).some((child) => subtreeHasMatch(child, children, matchingIds));
}

function OverlayMap({
  nodes,
  edges,
  ariaLabel,
  selectedCapabilityId,
  onCapabilitySelect,
  onScenarioSelect,
}: {
  nodes: OverlayNode[];
  edges: OverlayEdge[];
  ariaLabel: string;
  selectedCapabilityId: string;
  onCapabilitySelect: (id: string) => void;
  onScenarioSelect: (slug: string) => void;
}) {
  const positions = useMemo(() => {
    const capabilityNodes = nodes.filter((node) => node.kind === OverlayNodeKind.CAPABILITY);
    const implementationNodes = nodes.filter((node) => node.kind !== OverlayNodeKind.CAPABILITY);
    const rowHeight = 62;
    const next = new Map<string, { x: number; y: number }>();
    for (const [index, node] of capabilityNodes.entries()) {
      next.set(node.scenario, { x: 150, y: 46 + index * rowHeight });
    }
    for (const [index, node] of implementationNodes.entries()) {
      next.set(node.scenario, { x: 560, y: 46 + index * rowHeight });
    }
    return next;
  }, [nodes]);
  const height = Math.max(320, Math.max(1, nodes.filter((node) => node.kind === OverlayNodeKind.CAPABILITY).length, nodes.filter((node) => node.kind !== OverlayNodeKind.CAPABILITY).length) * 62 + 48);

  if (nodes.length === 0) {
    return null;
  }

  return (
    <div className="mt-3 overflow-auto rounded-md border border-app-border bg-app-surface">
      <svg role="img" aria-label={ariaLabel} viewBox={`0 0 720 ${height}`} className="min-h-[320px] w-[720px] max-w-none">
        <defs>
          <marker id="ontology-overlay-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
            <path d="M 0 0 L 8 4 L 0 8 z" fill="rgb(148 163 184)" />
          </marker>
        </defs>
        {edges.map((edge) => {
          const from = positions.get(edge.fromScenario);
          const to = positions.get(edge.toScenario);
          if (!from || !to) return null;
          const sameColumn = Math.abs(from.x - to.x) < 10;
          const controlX = sameColumn ? from.x + 120 : (from.x + to.x) / 2;
          return (
            <path
              key={`${edge.fromScenario}-${edge.toScenario}-${evidenceLabel(edge)}`}
              d={`M ${from.x + 96} ${from.y} C ${controlX} ${from.y}, ${controlX} ${to.y}, ${to.x - 96} ${to.y}`}
              fill="none"
              stroke={edge.evidence.some((item) => item.source === OverlayEvidenceSource.FULFILLS) ? "rgb(45 212 191)" : "rgb(148 163 184)"}
              strokeOpacity="0.72"
              strokeWidth="1.6"
              markerEnd="url(#ontology-overlay-arrow)"
            />
          );
        })}
        {nodes.map((node) => {
          const position = positions.get(node.scenario);
          if (!position) return null;
          const isCapability = node.kind === OverlayNodeKind.CAPABILITY;
          const selected = node.scenario === selectedCapabilityId;
          return (
            <g
              key={node.scenario}
              role="button"
              tabIndex={0}
              onClick={() => isCapability ? onCapabilitySelect(node.scenario) : onScenarioSelect(node.scenario)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  if (isCapability) onCapabilitySelect(node.scenario);
                  else onScenarioSelect(node.scenario);
                }
              }}
              className="cursor-pointer"
            >
              <rect
                x={position.x - 112}
                y={position.y - 22}
                width="224"
                height="44"
                rx="7"
                fill={isCapability ? "rgb(15 23 42)" : "rgb(8 47 73)"}
                stroke={selected ? "rgb(99 102 241)" : "rgb(51 65 85)"}
                strokeWidth={selected ? "2.4" : "1.2"}
              />
              <text x={position.x - 96} y={position.y - 3} fill="rgb(248 250 252)" fontSize="12" fontWeight="600">
                {truncateLabel(overlayNodeLabel(node), 27)}
              </text>
              <text x={position.x - 96} y={position.y + 14} fill="rgb(148 163 184)" fontSize="10">
                {truncateLabel(`${overlayNodeTypeLabel(node.kind)} · ${node.scenario}`, 34)}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function truncateLabel(value: string, limit: number) {
  if (value.length <= limit) return value;
  return `${value.slice(0, limit - 1)}…`;
}

function CapabilityBranch({
  capability,
  childrenByParent,
  selectedId,
  onSelect,
  classifications,
  depth,
  matchingFilter,
}: {
  capability: Capability;
  childrenByParent: Map<string, Capability[]>;
  selectedId: string;
  onSelect: (id: string) => void;
  classifications: Map<string, CoverageClassification>;
  depth: number;
  matchingFilter: string;
}) {
  const childNodes = childrenByParent.get(capability.id) ?? [];
  const startsOpen = depth < 1 || matchingFilter.trim().length > 0;
  const [open, setOpen] = useState(startsOpen);
  const state = classifications.get(capability.id)?.state ?? CoverageState.UNSPECIFIED;

  return (
    <div>
      <div className="flex items-center gap-2 py-1" style={{ paddingLeft: depth * 14 }}>
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-xs text-app-muted-foreground hover:bg-app-background"
          aria-label={open ? "Collapse capability" : "Expand capability"}
        >
          {childNodes.length ? (open ? "-" : "+") : ""}
        </button>
        <button
          type="button"
          onClick={() => onSelect(capability.id)}
          className={[
            "min-w-0 flex-1 rounded-md px-2 py-1 text-left text-sm transition",
            selectedId === capability.id ? "bg-app-primary/15 text-app-primary" : "hover:bg-app-background",
          ].join(" ")}
        >
          <span className="block truncate font-medium">{capability.name || capability.slug}</span>
          <span className="block truncate text-xs text-app-muted-foreground">{capability.slug}</span>
        </button>
        <span className={[
          "h-2.5 w-2.5 rounded-full",
          state === CoverageState.BUILT ? "bg-emerald-400" : state === CoverageState.IN_FLIGHT ? "bg-amber-400" : "bg-slate-500",
        ].join(" ")} />
      </div>
      {open && childNodes.map((child) => (
        <CapabilityBranch
          key={child.id}
          capability={child}
          childrenByParent={childrenByParent}
          selectedId={selectedId}
          onSelect={onSelect}
          classifications={classifications}
          depth={depth + 1}
          matchingFilter={matchingFilter}
        />
      ))}
    </div>
  );
}

function Metric({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="rounded-lg border border-app-border bg-app-surface p-4">
      <div className="flex items-center gap-2 text-app-muted-foreground">{icon}<p className="text-xs uppercase">{label}</p></div>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function DetailStat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-md border border-app-border bg-app-background p-3">
      <p className="text-xs uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-medium">{value}</p>
    </div>
  );
}

function StatePanel({ label, tone = "default" }: { label: string; tone?: "default" | "error" }) {
  return (
    <div className={[
      "rounded-lg border p-5 text-sm",
      tone === "error" ? "border-red-500/40 bg-red-950/25 text-red-100" : "border-app-border bg-app-surface text-app-muted-foreground",
    ].join(" ")}>
      {label}
    </div>
  );
}
