import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { MessageInitShape } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { CreateBrandRequestSchema } from "@vrooli/proto-types/brand-manager/v1/brands/brands_pb";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { brandsClient } from "../../api/brands";
import { errorMessage } from "../../lib/errorMessage";

const BRANDS_QUERY_KEY = ["brands"] as const;

/**
 * BrandsCard is the brands-domain CRUD surface: it lists brands (newest-updated
 * first) and creates new ones via mutation, wired to the BrandsService Connect
 * client. It is the canonical reference card the other domain cards mirror.
 */
export function BrandsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const brandsQuery = useQuery({
    queryKey: BRANDS_QUERY_KEY,
    queryFn: () => brandsClient.listBrands({}),
  });

  const createBrandMutation = useMutation({
    mutationFn: (input: MessageInitShape<typeof CreateBrandRequestSchema>) => brandsClient.createBrand(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: BRANDS_QUERY_KEY });
    },
  });

  const handleCreateBrand = () => {
    createBrandMutation.mutate({
      name: `Brand ${(brandsQuery.data?.brands.length ?? 0) + 1}`,
    });
  };

  return (
    <section
      data-testid={selectors.brands.card}
      aria-label={t(strings.brands.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.brands.title)}</h2>
      {brandsQuery.isLoading && (
        <p data-testid={selectors.brands.loading} className="mt-2 text-slate-200">
          {t(strings.brands.loading)}
        </p>
      )}
      {brandsQuery.error && (
        <p data-testid={selectors.brands.error} className="mt-2 text-red-400">
          {errorMessage(brandsQuery.error, t)}
        </p>
      )}
      {brandsQuery.data && brandsQuery.data.brands.length === 0 && (
        <p data-testid={selectors.brands.empty} className="mt-2 text-slate-200">
          {t(strings.brands.empty)}
        </p>
      )}
      {brandsQuery.data && brandsQuery.data.brands.length > 0 && (
        <ul data-testid={selectors.brands.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {brandsQuery.data.brands.map((brand) => (
            <li key={brand.id} className="rounded-lg border border-white/10 p-3">
              <div className="flex items-center justify-between">
                <span className="font-medium">{brand.name}</span>
                <span data-testid={selectors.brands.version} className="text-xs text-slate-400">
                  {`v${brand.version}`}
                </span>
              </div>
              {brand.colors?.primary && (
                <div className="mt-1 flex items-center gap-2 text-xs text-slate-400">
                  <span>{t(strings.brands.primaryLabel)}</span>
                  <span
                    aria-hidden="true"
                    className="inline-block h-3 w-3 rounded-full border border-white/20"
                    style={{ backgroundColor: brand.colors.primary }}
                  />
                  <span>{brand.colors.primary}</span>
                </div>
              )}
              {brand.updatedAt && (
                <div data-testid={selectors.brands.updatedAt} className="mt-1 text-xs text-slate-400">
                  {formatDate(timestampDate(brand.updatedAt), {
                    dateStyle: "medium",
                    timeStyle: "short",
                  })}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
      <Button
        data-testid={selectors.brands.createButton}
        className="mt-4"
        onClick={handleCreateBrand}
        disabled={createBrandMutation.isPending}
      >
        {t(strings.brands.create)}
      </Button>
      {createBrandMutation.error && (
        <p data-testid={selectors.brands.error} className="mt-2 text-red-400">
          {errorMessage(createBrandMutation.error, t)}
        </p>
      )}
    </section>
  );
}
