import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { Button } from "../../components/ui/button";
import { exploreCandidates } from "../../api/candidates";
import { listBrands } from "../../api/brands";

interface LogoExploreFormProps {
  brandId: string;
}

/**
 * LogoExploreForm fans out N concepts × M variations in one request. It is an
 * explicit operator action (billed generations), so it stays behind a submit
 * button and reports warnings from the backend when a concept fails.
 *
 * Concepts render with an illustration model by default. "Match the style of"
 * conditions every concept on another brand's approved mark, which is how a
 * product line (Aquila → Vega → …) stays one family. The flat vector style is
 * opt-in because vector models draw simple geometric marks.
 */
export function LogoExploreForm({ brandId }: LogoExploreFormProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [brief, setBrief] = useState("");
  const [conceptsText, setConceptsText] = useState("");
  const [variations, setVariations] = useState(2);
  const [preferVector, setPreferVector] = useState(false);
  const [styleReferenceBrand, setStyleReferenceBrand] = useState("");

  // Only brands with a picked mark can be a style reference.
  const references = useQuery({
    queryKey: ["brands", "style-references"],
    queryFn: () => listBrands(),
    select: (brands) => brands.filter((brand) => brand.id !== brandId && brand.markAssetId !== ""),
  });

  const explore = useMutation({
    mutationFn: () =>
      exploreCandidates({
        brandId,
        brief,
        concepts: conceptsText
          .split("\n")
          .map((line) => line.trim())
          .filter((line) => line !== ""),
        variations,
        preferVector,
        styleReferenceBrand,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["candidates", brandId] });
    },
  });

  return (
    <form
      data-testid={selectors.logo.exploreForm}
      aria-label={t(strings.logo.exploreTitle)}
      className="flex flex-col gap-3 rounded-xl border border-white/10 bg-black/20 p-3"
      onSubmit={(event) => {
        event.preventDefault();
        explore.mutate();
      }}
    >
      <h3 className="text-sm font-medium text-slate-400">{t(strings.logo.exploreTitle)}</h3>
      <label className="flex flex-col gap-1 text-xs text-slate-400">
        {t(strings.logo.briefLabel)}
        <input
          data-testid={selectors.logo.briefInput}
          value={brief}
          onChange={(event) => setBrief(event.target.value)}
          placeholder={t(strings.logo.briefPlaceholder)}
          className="rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
        />
      </label>
      <label className="flex flex-col gap-1 text-xs text-slate-400">
        {t(strings.logo.conceptsLabel)}
        <textarea
          data-testid={selectors.logo.conceptsInput}
          value={conceptsText}
          onChange={(event) => setConceptsText(event.target.value)}
          placeholder={t(strings.logo.conceptsPlaceholder)}
          rows={3}
          className="rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
        />
      </label>
      <label className="flex flex-col gap-1 text-xs text-slate-400">
        {t(strings.logo.styleReferenceLabel)}
        <select
          data-testid={selectors.logo.styleReferenceSelect}
          value={styleReferenceBrand}
          onChange={(event) => setStyleReferenceBrand(event.target.value)}
          className="rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
        >
          <option value="">{t(strings.logo.styleReferenceNone)}</option>
          {(references.data ?? []).map((brand) => (
            <option key={brand.id} value={brand.id}>
              {brand.identity?.displayName || brand.name}
            </option>
          ))}
        </select>
      </label>
      <div className="flex flex-wrap items-center gap-4">
        <label className="flex items-center gap-2 text-xs text-slate-400">
          {t(strings.logo.variationsLabel)}
          <input
            data-testid={selectors.logo.variationsInput}
            type="number"
            min={1}
            max={4}
            value={variations}
            onChange={(event) => setVariations(Math.min(4, Math.max(1, Number(event.target.value))))}
            className="w-16 rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
          />
        </label>
        <label className="flex items-center gap-2 text-xs text-slate-400">
          <input
            type="checkbox"
            checked={preferVector}
            disabled={styleReferenceBrand !== ""}
            onChange={(event) => setPreferVector(event.target.checked)}
          />
          {t(strings.logo.preferVectorLabel)}
        </label>
      </div>
      {explore.error && <p className="text-sm text-red-400">{errorMessage(explore.error, t)}</p>}
      {(explore.data?.warnings ?? []).map((warning) => (
        <p key={warning} className="text-xs text-amber-300">
          {warning}
        </p>
      ))}
      <Button
        type="submit"
        data-testid={selectors.logo.exploreSubmit}
        disabled={explore.isPending || conceptsText.trim() === ""}
      >
        {explore.isPending ? t(strings.logo.exploring) : t(strings.logo.exploreSubmit)}
      </Button>
    </form>
  );
}
