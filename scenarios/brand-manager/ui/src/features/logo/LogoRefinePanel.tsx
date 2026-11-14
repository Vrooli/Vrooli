import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { LogoCandidate } from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { Button } from "../../components/ui/button";
import { refineCandidate } from "../../api/candidates";
import { uploadAsset } from "../../api/assets";
import { MaskBrush } from "./MaskBrush";

interface LogoRefinePanelProps {
  brandId: string;
  candidate: LogoCandidate;
}

/**
 * LogoRefinePanel derives a NEW candidate from the selected one. Every action
 * records the parent, so the gallery's lineage stays complete: an instruction
 * edit, a vectorize pass, a background removal, or a painted masked removal
 * (which uploads the mask as an asset first).
 */
export function LogoRefinePanel({ brandId, candidate }: LogoRefinePanelProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [instruction, setInstruction] = useState("");

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["candidates", brandId] });
  };

  const refineInstruction = useMutation({
    mutationFn: () => refineCandidate(candidate.id, { instruction }),
    onSuccess: invalidate,
  });
  const vectorize = useMutation({
    mutationFn: () => refineCandidate(candidate.id, { vectorize: {} }),
    onSuccess: invalidate,
  });
  const removeBackground = useMutation({
    mutationFn: () => refineCandidate(candidate.id, { removeBackground: true }),
    onSuccess: invalidate,
  });
  const maskedRemoval = useMutation({
    mutationFn: async (mask: Blob) => {
      const bytes = new Uint8Array(await mask.arrayBuffer());
      const asset = await uploadAsset({
        brandId,
        filename: `mask-${Date.now()}.png`,
        mimeType: "image/png",
        content: bytes,
      });
      return refineCandidate(candidate.id, { maskAssetId: asset.id });
    },
    onSuccess: invalidate,
  });

  const busy =
    refineInstruction.isPending ||
    vectorize.isPending ||
    removeBackground.isPending ||
    maskedRemoval.isPending;
  const error =
    refineInstruction.error ?? vectorize.error ?? removeBackground.error ?? maskedRemoval.error;

  return (
    <section
      data-testid={selectors.logo.refinePanel}
      aria-label={t(strings.logo.refineTitle)}
      className="flex flex-col gap-3 rounded-xl border border-white/10 bg-black/20 p-3"
    >
      <h3 className="text-sm font-medium text-slate-400">
        {t(strings.logo.refineTitle)}: {candidate.concept}
      </h3>
      <form
        className="flex flex-col gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          refineInstruction.mutate();
        }}
      >
        <label className="flex flex-col gap-1 text-xs text-slate-400">
          {t(strings.logo.instructionLabel)}
          <input
            data-testid={selectors.logo.instructionInput}
            value={instruction}
            onChange={(event) => setInstruction(event.target.value)}
            placeholder={t(strings.logo.instructionPlaceholder)}
            className="rounded-control border border-white/10 bg-black/30 px-2 py-1 text-sm text-slate-200"
          />
        </label>
        <div className="flex flex-wrap gap-2">
          <Button
            type="submit"
            data-testid={selectors.logo.refineSubmit}
            disabled={busy || instruction.trim() === ""}
          >
            {refineInstruction.isPending ? t(strings.logo.refining) : t(strings.logo.refineSubmit)}
          </Button>
          <Button
            type="button"
            data-testid={selectors.logo.vectorizeSubmit}
            variant="outline"
            disabled={busy}
            onClick={() => vectorize.mutate()}
          >
            {vectorize.isPending ? t(strings.logo.vectorizing) : t(strings.logo.vectorize)}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={() => removeBackground.mutate()}
          >
            {removeBackground.isPending
              ? t(strings.logo.removingBackground)
              : t(strings.logo.removeBackground)}
          </Button>
        </div>
      </form>
      {error && <p className="text-sm text-red-400">{errorMessage(error, t)}</p>}
      <MaskBrush disabled={busy} onSubmit={(mask) => maskedRemoval.mutate(mask)} />
    </section>
  );
}
