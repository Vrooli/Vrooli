import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MetricsPanel, type MetricsPanelProps } from "./MetricsPanel";
import type { MetricsViewModel } from "../../../shared/controllers/knowledgeController";
import type { CollectionMaintenanceResponse } from "../../../shared/services/api";

const baseViewModel: MetricsViewModel = {
  metricCards: [
    {
      label: "Coherence",
      description: "Topical consistency",
      percentageLabel: "70.0%",
      tone: "good",
    },
  ],
  collections: [
    {
      name: "alpha",
      sizeLabel: "12 vectors",
      metrics: [{ label: "Coherence", percentageLabel: "70%" }],
    },
  ],
  overallHealth: "steady",
  lastUpdated: "10:00 AM",
  totalEntriesLabel: "42",
  hasMetrics: true,
};

const preview: CollectionMaintenanceResponse = {
  collection: "alpha",
  action: "prune_stale_chunks",
  dry_run: true,
  analyzed_points: 12,
  candidate_delete_count: 2,
  deleted_count: 0,
  took_ms: 8,
};

const createProps = (overrides: Partial<MetricsPanelProps> = {}): MetricsPanelProps => ({
  isLoading: false,
  hasError: false,
  errorMessage: "",
  hasData: true,
  viewModel: baseViewModel,
  selectedCollection: "",
  diagnostics: null,
  diagnosticsError: "",
  diagnosticsLoading: false,
  diagnosticsMode: "sample",
  diagnosticsLimit: 1200,
  drilldownTab: "integrity",
  maintenanceInFlight: false,
  maintenanceNotice: "",
  maintenanceMaxDeletes: 500,
  getMaintenancePreview: () => preview,
  getCollectionInventory: () => null,
  collectionRecords: null,
  recordsLoading: false,
  recordsError: "",
  recordsSearch: "",
  recordsNamespaceFilter: "",
  recordsDocumentFilter: "",
  onSelectCollection: vi.fn(),
  onDrilldownTabChange: vi.fn(),
  onUseSampleDiagnostics: vi.fn(),
  onUseFullDiagnostics: vi.fn(),
  onMaintenanceMaxDeletesChange: vi.fn(),
  onRecordsSearchChange: vi.fn(),
  onRecordsNamespaceFilterChange: vi.fn(),
  onRecordsDocumentFilterChange: vi.fn(),
  onRecordsNextPage: vi.fn(),
  onRecordsPreviousPage: vi.fn(),
  onPreviewMaintenance: vi.fn(),
  onApplyMaintenance: vi.fn(),
  collectionDeleteInFlight: false,
  onDeleteCollection: vi.fn(),
  onRetry: vi.fn(),
  ...overrides,
});

describe("MetricsPanel", () => {
  it("renders a loading state", () => {
    render(<MetricsPanel {...createProps({ isLoading: true })} />);

    expect(screen.getByText(/Loading metrics/i)).toBeDefined();
  });

  it("renders metrics and collections when data is available", () => {
    render(<MetricsPanel {...createProps()} />);

    expect(screen.getByText(/Overall Health/i)).toBeDefined();
    expect(screen.getAllByText(/Coherence/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/alpha/i)).toBeDefined();
    expect(screen.getByText(/12 vectors/i)).toBeDefined();
    expect(screen.getByText(/Embeddings stored in this collection/i)).toBeDefined();
  });

  it("falls back safely when the view model is missing", () => {
    const unsafeProps = {
      ...createProps(),
      viewModel: undefined as unknown as MetricsViewModel,
    };

    expect(() => render(<MetricsPanel {...unsafeProps} />)).not.toThrow();
    expect(screen.getByText(/unknown condition/i)).toBeDefined();
  });
});
