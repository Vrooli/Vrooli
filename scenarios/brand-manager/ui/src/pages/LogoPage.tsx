import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import type { LogoCandidate } from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";
import { listBrands } from "../api/brands";
import { listCandidates } from "../api/candidates";
import { listContainerStyles } from "../api/styles";
import { LogoCandidateGallery } from "../features/logo/LogoCandidateGallery";
import { LogoCompare } from "../features/logo/LogoCompare";
import { LogoExploreForm } from "../features/logo/LogoExploreForm";
import { LogoRefinePanel } from "../features/logo/LogoRefinePanel";
import { LogoTargetPreview } from "../features/logo/LogoTargetPreview";

const MAX_COMPARE = 4;
const GALLERY_LIMIT = 48;

/**
 * LogoPage is the operator surface for the brand-identity pipeline: pick a
 * brand, explore concepts, compare, pick/reject/restore, refine by instruction,
 * vectorize, or masked removal, and preview the applied icon at every declared
 * size. The selected brand is stored in the URL so a link reopens the page.
 */
export function LogoPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedBrandId = searchParams.get("brand") ?? "";

  const [compareIds, setCompareIds] = useState<string[]>([]);
  const [refineTarget, setRefineTarget] = useState<LogoCandidate | null>(null);

  const brandsQuery = useQuery({
    queryKey: ["brands", "logo-selector"],
    queryFn: () => listBrands(),
  });
  const brands = brandsQuery.data ?? [];
  const selectedBrand = useMemo(
    () => brands.find((brand) => brand.id === selectedBrandId),
    [brands, selectedBrandId],
  );

  const stylesQuery = useQuery({
    queryKey: ["container-styles"],
    queryFn: () => listContainerStyles(),
  });
  const containerStyle = useMemo(() => {
    if (!selectedBrand?.containerStyleId) {
      return undefined;
    }
    return (stylesQuery.data ?? []).find((style) => style.id === selectedBrand.containerStyleId);
  }, [stylesQuery.data, selectedBrand]);

  const toggleCompare = (candidateId: string) => {
    setCompareIds((current) => {
      if (current.includes(candidateId)) {
        return current.filter((id) => id !== candidateId);
      }
      const next = [...current, candidateId];
      return next.length > MAX_COMPARE ? next.slice(next.length - MAX_COMPARE) : next;
    });
  };

  const candidatesQuery = useQuery({
    queryKey: ["candidates", selectedBrandId, "gallery"],
    queryFn: () => listCandidates(selectedBrandId, { limit: GALLERY_LIMIT }),
    enabled: selectedBrandId !== "",
  });
  const candidates = candidatesQuery.data ?? [];
  const compareCandidates = candidates.filter((candidate) => compareIds.includes(candidate.id));

  return (
    <section
      data-testid={selectors.pages.logo}
      aria-labelledby="logo-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-2">
        <h2 id="logo-heading" className="text-2xl font-semibold">
          {t(strings.logo.title)}
        </h2>
        <label className="flex items-center gap-2 text-sm text-slate-400">
          {t(strings.logo.brandLabel)}
          <select
            data-testid={selectors.logo.brandSelect}
            value={selectedBrandId}
            onChange={(event) => {
              const value = event.target.value;
              setSearchParams(value ? { brand: value } : {}, { replace: true });
              setCompareIds([]);
              setRefineTarget(null);
            }}
            className="rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
          >
            <option value="">{t(strings.logo.brandPlaceholder)}</option>
            {brands.map((brand) => (
              <option key={brand.id} value={brand.id}>
                {brand.name}
              </option>
            ))}
          </select>
        </label>
        {brandsQuery.error && (
          <p className="text-sm text-red-400">{errorMessage(brandsQuery.error, t)}</p>
        )}
      </header>

      {selectedBrandId !== "" && (
        <div className="flex flex-col gap-4">
          <LogoExploreForm brandId={selectedBrandId} />
          <LogoCandidateGallery
            brandId={selectedBrandId}
            candidates={candidates}
            isLoading={candidatesQuery.isLoading}
            queryError={candidatesQuery.error}
            selectedForCompare={compareIds}
            onToggleCompare={toggleCompare}
            onSelectForRefine={setRefineTarget}
          />
          <LogoCompare candidates={compareCandidates} />
          {refineTarget && (
            <LogoRefinePanel brandId={selectedBrandId} candidate={refineTarget} />
          )}
          <LogoTargetPreview
            brandId={selectedBrandId}
            markAssetId={selectedBrand?.markAssetId ?? ""}
            style={containerStyle}
          />
        </div>
      )}
    </section>
  );
}
