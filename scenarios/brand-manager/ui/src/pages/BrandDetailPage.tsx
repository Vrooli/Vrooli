import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Edit, Trash2, History } from "lucide-react";
import { fetchBrand, fetchVersions, deleteBrand, ApiRequestError } from "../lib/api";
import { formatDate, formatDateTime } from "../lib/utils";
import { Button } from "../components/ui/button";
import { Section } from "../components/ui/section";
import { ErrorAlert } from "../components/ui/error-alert";
import { ColorSwatch } from "../components/color-swatch";
import { ThemePreview } from "../components/theme-preview";
import { ApplyPreview } from "../components/apply-preview";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-UI-THEME] [REQ:BM-REQ-UI-APPLY]

/** Renders a label-value pair, skipping falsy values. */
function FieldRow({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div>
      <span className="text-slate-500">{label}:</span>{" "}
      <span className="text-slate-200">{value}</span>
    </div>
  );
}

interface BrandDetailPageProps {
  brandId: string;
  onNavigate: (path: string) => void;
}

export default function BrandDetailPage({ brandId, onNavigate }: BrandDetailPageProps) {
  const queryClient = useQueryClient();

  const { data: brand, isLoading, error, refetch } = useQuery({
    queryKey: ["brand", brandId],
    queryFn: () => fetchBrand(brandId),
  });

  const { data: versions } = useQuery({
    queryKey: ["versions", brandId],
    queryFn: () => fetchVersions(brandId),
    enabled: !!brand,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteBrand(brandId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["brands"] });
      onNavigate("/brands");
    },
  });

  if (isLoading) return <div className="text-slate-400 py-12 text-center">Loading brand...</div>;

  if (error || !brand) {
    const isNotFound = error instanceof ApiRequestError && error.apiError?.code === "not_found";
    const isRetryable = !isNotFound && (!(error instanceof ApiRequestError) || error.isRetryable);

    return (
      <div className="text-center py-12" data-testid="brand-detail-error">
        <ErrorAlert
          error={error}
          fallbackMessage="Failed to load brand."
          onRetry={isRetryable ? () => refetch() : undefined}
          className="text-left mb-4"
        />
        <Button variant="outline" onClick={() => onNavigate("/brands")}>
          <ArrowLeft className="mr-2 h-4 w-4" /> Back to Library
        </Button>
      </div>
    );
  }

  return (
    <div data-testid="brand-detail-page">
      <button
        onClick={() => onNavigate("/brands")}
        className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-50 mb-4 transition-colors"
        data-testid="back-to-brands"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Library
      </button>

      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-50">{brand.name}</h1>
          {brand.description && <p className="mt-1 text-slate-400">{brand.description}</p>}
          <p className="mt-2 text-xs text-slate-500">
            Version {brand.version} · Created {formatDate(brand.created_at)}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => onNavigate(`/brands/${brandId}/edit`)} data-testid="edit-brand-btn">
            <Edit className="mr-1 h-3 w-3" /> Edit
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={deleteMutation.isPending}
            onClick={() => {
              if (confirm("Delete this brand?")) deleteMutation.mutate();
            }}
            data-testid="delete-brand-btn"
          >
            <Trash2 className="mr-1 h-3 w-3" /> {deleteMutation.isPending ? "Deleting..." : "Delete"}
          </Button>
        </div>
      </div>

      {/* Delete error */}
      {deleteMutation.error && (
        <ErrorAlert
          error={deleteMutation.error}
          fallbackMessage="Failed to delete brand."
          fallbackRecovery="Check your connection and try again."
          className="mb-4"
          testId="delete-error"
        />
      )}

      {/* Colors */}
      {brand.colors && (
        <Section title="Colors" testId="brand-colors-section" className="mb-4">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            <ColorSwatch color={brand.colors.primary} label="Primary" />
            <ColorSwatch color={brand.colors.secondary} label="Secondary" />
            <ColorSwatch color={brand.colors.accent} label="Accent" />
            <ColorSwatch color={brand.colors.background} label="Background" />
            <ColorSwatch color={brand.colors.surface} label="Surface" />
            <ColorSwatch color={brand.colors.text} label="Text" />
            <ColorSwatch color={brand.colors.error} label="Error" />
          </div>
        </Section>
      )}

      {/* Identity */}
      {brand.identity && (
        <Section title="Identity" testId="brand-identity-section" className="mb-4">
          <div className="grid grid-cols-2 gap-3 text-sm">
            <FieldRow label="Display Name" value={brand.identity.display_name} />
            <FieldRow label="Tagline" value={brand.identity.tagline} />
          </div>
        </Section>
      )}

      {/* Typography */}
      {brand.typography && (
        <Section title="Typography" testId="brand-typography-section" className="mb-4">
          <div className="grid grid-cols-2 gap-3 text-sm">
            <FieldRow label="Heading" value={brand.typography.heading_font} />
            <FieldRow label="Body" value={brand.typography.body_font} />
            <FieldRow label="Mono" value={brand.typography.mono_font} />
          </div>
        </Section>
      )}

      {/* Voice */}
      {brand.voice && (
        <Section title="Voice" testId="brand-voice-section" className="mb-4">
          <div className="text-sm">
            <FieldRow label="Tone" value={brand.voice.tone} />
            <FieldRow label="Style" value={brand.voice.style} />
            {brand.voice.keywords && brand.voice.keywords.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {brand.voice.keywords.map((kw) => (
                  <span key={kw} className="rounded-full bg-white/10 px-2 py-0.5 text-xs text-slate-300">{kw}</span>
                ))}
              </div>
            )}
          </div>
        </Section>
      )}

      {/* Theme Preview [REQ:BM-REQ-UI-THEME] */}
      <div className="mb-4">
        <ThemePreview brandId={brandId} />
      </div>

      {/* Apply Preview [REQ:BM-REQ-UI-APPLY] */}
      <div className="mb-4">
        <ApplyPreview brandId={brandId} />
      </div>

      {/* Version History */}
      {versions && versions.length > 0 && (
        <Section testId="brand-versions-section">
          <h2 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
            <History className="h-4 w-4" /> Version History
          </h2>
          <div className="space-y-2">
            {versions.map((v) => (
              <div key={v.id} className="flex items-center justify-between text-sm border-b border-white/5 pb-2">
                <span className="text-slate-200">Version {v.version}</span>
                <span className="text-slate-500">{formatDateTime(v.created_at)}</span>
              </div>
            ))}
          </div>
        </Section>
      )}
    </div>
  );
}
