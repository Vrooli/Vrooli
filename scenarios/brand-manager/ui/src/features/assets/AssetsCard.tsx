import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { assetsClient } from "../../api/assets";
import { errorMessage } from "../../lib/errorMessage";

const ASSETS_QUERY_KEY = ["assets"] as const;

/** formatBytes renders a byte count as a compact, locale-neutral size. */
function formatBytes(bytes: bigint): string {
  const n = Number(bytes);
  if (n < 1024) {
    return `${n} B`;
  }
  if (n < 1024 * 1024) {
    return `${(n / 1024).toFixed(1)} KB`;
  }
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * AssetsCard lists brand asset files (newest-uploaded first). It is a read
 * surface: assets are uploaded from the CLI/wizard where a brand id and a local
 * file are supplied. Mirrors the canonical AssignmentsCard structure but wired
 * to the AssetsService Connect client.
 */
export function AssetsCard() {
  const { t } = useTranslation();

  const assetsQuery = useQuery({
    queryKey: ASSETS_QUERY_KEY,
    queryFn: () => assetsClient.listAssets({}),
  });

  return (
    <section
      data-testid={selectors.assets.card}
      aria-label={t(strings.assets.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.assets.title)}</h2>
      {assetsQuery.isLoading && (
        <p data-testid={selectors.assets.loading} className="mt-2 text-slate-200">
          {t(strings.assets.loading)}
        </p>
      )}
      {assetsQuery.error && (
        <p data-testid={selectors.assets.error} className="mt-2 text-red-400">
          {errorMessage(assetsQuery.error, t)}
        </p>
      )}
      {assetsQuery.data && assetsQuery.data.assets.length === 0 && (
        <p data-testid={selectors.assets.empty} className="mt-2 text-slate-200">
          {t(strings.assets.empty)}
        </p>
      )}
      {assetsQuery.data && assetsQuery.data.assets.length > 0 && (
        <ul data-testid={selectors.assets.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {assetsQuery.data.assets.map((asset) => (
            <li key={asset.id} className="rounded-lg border border-white/10 p-3">
              <div className="flex items-center justify-between">
                <span data-testid={selectors.assets.filename} className="font-medium">
                  {asset.filename}
                </span>
                <span data-testid={selectors.assets.size} className="text-xs text-slate-400">
                  {formatBytes(asset.size)}
                </span>
              </div>
              <div className="mt-1 text-xs text-slate-400">
                <span>{t(strings.assets.brandLabel)}</span>{" "}
                <span data-testid={selectors.assets.brand}>{asset.brandId}</span>
              </div>
              <div className="mt-1 text-xs text-slate-400">
                <span>{t(strings.assets.typeLabel)}</span>{" "}
                <span data-testid={selectors.assets.mimeType}>{asset.mimeType}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
