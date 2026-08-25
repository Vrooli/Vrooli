import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { categoriesClient } from "../../api/categories";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const categoriesKey = ["categories"] as const;
const classificationKey = (signalID: string) => ["classification", signalID] as const;

// A proposal is visible but not treated as a decision. Confirming (including
// choosing a different category) appends an operator judgment on the API.
export function SignalClassificationControl({ signalID }: { signalID: string }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const categories = useQuery({ queryKey: categoriesKey, queryFn: () => categoriesClient.listCategories({}) });
  const classification = useQuery({ queryKey: classificationKey(signalID), queryFn: () => categoriesClient.getClassification({ signalId: signalID }) });
  const current = classification.data?.classification;
  const [categoryID, setCategoryID] = useState("");
  useEffect(() => {
    setCategoryID(current?.confirmedCategoryId || current?.proposedCategoryId || "");
  }, [current?.confirmedCategoryId, current?.proposedCategoryId]);
  const confirm = useMutation({
    mutationFn: () => categoriesClient.confirmClassification({ signalId: signalID, categoryId: categoryID }),
    onSuccess: () => void client.invalidateQueries({ queryKey: classificationKey(signalID) }),
  });

  if (classification.isLoading) return <p className="text-xs text-app-muted-foreground">{t(strings.categories.classificationLoading)}</p>;
  if (classification.error || !current) return <p className="text-xs text-app-muted-foreground">{t(strings.categories.classificationPending)}</p>;

  const label = current.confirmedCategoryId ? t(strings.categories.confirmed) : t(strings.categories.proposed);
  return (
    <div className="mt-2 flex flex-col gap-2 border-t border-app-border pt-2">
      <p className="text-xs text-app-muted-foreground">{label}{current.proposedCategoryId ? ` · ${t(strings.categories.confidence, { value: (current.proposedConfidence * 100).toFixed(0) })}` : ""}{current.reason ? ` · ${current.reason}` : ""}</p>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Select
          aria-label={t(strings.categories.categoryFor, { id: signalID })}
          value={categoryID}
          onChange={(event) => setCategoryID(event.target.value)}
          placeholder={t(strings.categories.choose)}
          options={(categories.data?.categories ?? []).map((category) => ({ value: category.id, label: category.name }))}
          disabled={categories.isLoading || confirm.isPending}
        />
        <Button size="sm" onClick={() => confirm.mutate()} disabled={!categoryID || confirm.isPending}>{current.confirmedCategoryId ? t(strings.categories.change) : t(strings.categories.confirm)}</Button>
      </div>
      {confirm.error && <p className="text-xs text-app-danger">{t(strings.categories.confirmError)}</p>}
    </div>
  );
}
