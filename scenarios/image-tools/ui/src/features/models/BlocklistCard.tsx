import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient, type BlocklistEntry } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";

const BLOCKLIST_QUERY_KEY = ["models", "blocklist"] as const;

/**
 * BlocklistCard is a read-only surface listing license-encumbered models the
 * catalog deliberately excludes (ListBlocklist). Each entry documents the
 * reason and the ONNX/GGUF export trap so the exclusion is auditable.
 */
export function BlocklistCard() {
  const { t } = useTranslation();

  const blocklistQuery = useQuery({
    queryKey: BLOCKLIST_QUERY_KEY,
    queryFn: () => modelsClient.listBlocklist({}),
  });

  const entries: BlocklistEntry[] = blocklistQuery.data?.entries ?? [];

  return (
    <section
      data-testid={selectors.models.blocklist.card}
      aria-label={t(strings.models.blocklist.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.models.blocklist.title)}</h2>
      {blocklistQuery.isLoading && (
        <p data-testid={selectors.models.blocklist.loading} className="mt-2 text-slate-200">
          {t(strings.models.blocklist.loading)}
        </p>
      )}
      {blocklistQuery.error && (
        <p data-testid={selectors.models.blocklist.error} className="mt-2 text-red-400">
          {errorMessage(blocklistQuery.error, t)}
        </p>
      )}
      {blocklistQuery.data && entries.length === 0 && (
        <p data-testid={selectors.models.blocklist.empty} className="mt-2 text-slate-200">
          {t(strings.models.blocklist.empty)}
        </p>
      )}
      {entries.length > 0 && (
        <ul
          data-testid={selectors.models.blocklist.list}
          className="mt-2 space-y-2 text-sm text-slate-200"
        >
          {entries.map((entry) => (
            <li
              key={entry.id}
              data-testid={selectors.models.blocklist.entry}
              className="rounded-lg border border-white/10 p-3"
            >
              <div className="font-medium">{entry.id}</div>
              {entry.operations.length > 0 && (
                <div className="mt-1 text-xs text-slate-400">
                  {t(strings.models.blocklist.operationsLabel)} {entry.operations.join(", ")}
                </div>
              )}
              <div className="mt-1 text-xs text-slate-400">
                {t(strings.models.blocklist.licenseLabel)} {entry.license || "—"}
              </div>
              <div className="mt-1 text-xs text-slate-400">
                {t(strings.models.blocklist.reasonLabel)} {entry.reason}
              </div>
              {entry.exportingOnnxRemovesRestriction && (
                <p
                  data-testid={selectors.models.blocklist.onnxWarning}
                  className="mt-1 text-xs text-amber-300"
                >
                  {t(strings.models.blocklist.onnxWarning)}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
