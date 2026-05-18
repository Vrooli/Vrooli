import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { goldenClient, type Golden } from "../../api/golden";
import { errorMessage } from "../../lib/errorMessage";

const GOLDENS_QUERY_KEY = ["goldens"] as const;

interface RegisterFormState {
  slug: string;
  template: string;
  version: string;
  path: string;
}

const EMPTY_FORM: RegisterFormState = {
  slug: "",
  template: "",
  version: "",
  path: "",
};

/**
 * GoldensCard is the OT-P0-001 vertical-slice UI: lists registered
 * goldens, drives RegisterGolden via a small form, and reveals a
 * detail panel with regenerate/delete actions when a row is selected.
 *
 * Pattern is the same as NotesCard — replace this surface when the
 * dashboard grid (OT-P0-010) lands, but until then this is enough for
 * an operator to drive the golden registry end-to-end.
 */
export function GoldensCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [form, setForm] = useState<RegisterFormState>(EMPTY_FORM);
  const [selectedSlug, setSelectedSlug] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const listQuery = useQuery({
    queryKey: GOLDENS_QUERY_KEY,
    queryFn: () => goldenClient.listGoldens({}),
  });

  const registerMutation = useMutation({
    mutationFn: (input: RegisterFormState) => goldenClient.registerGolden(input),
    onSuccess: () => {
      setForm(EMPTY_FORM);
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
    },
  });

  const regenerateMutation = useMutation({
    mutationFn: (slug: string) => goldenClient.regenerateGolden({ slug }),
    onSuccess: (resp) => {
      const slug = resp.golden?.slug ?? "";
      setStatusMessage(t(strings.goldens.regenerateSuccess, { slug }));
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (slug: string) => goldenClient.deleteGolden({ slug }),
    onSuccess: (_, slug) => {
      setStatusMessage(t(strings.goldens.deleteSuccess, { slug }));
      setSelectedSlug(null);
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
    },
  });

  const handleRegister = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setStatusMessage(null);
    registerMutation.mutate(form);
  };

  const handleRegenerate = (slug: string) => {
    if (!window.confirm(t(strings.goldens.confirmRegenerate, { slug }))) return;
    setStatusMessage(null);
    regenerateMutation.mutate(slug);
  };

  const handleDelete = (slug: string) => {
    if (!window.confirm(t(strings.goldens.confirmDelete, { slug }))) return;
    setStatusMessage(null);
    deleteMutation.mutate(slug);
  };

  const goldens = listQuery.data?.goldens ?? [];
  const selected = goldens.find((g) => g.slug === selectedSlug) ?? null;

  return (
    <section
      data-testid={selectors.goldens.card}
      aria-label={t(strings.goldens.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <header>
        <h2 className="text-sm font-medium text-slate-400">{t(strings.goldens.title)}</h2>
        <p className="text-xs text-slate-500">{t(strings.goldens.subtitle)}</p>
      </header>

      {listQuery.isLoading && (
        <p data-testid={selectors.goldens.loading} className="mt-2 text-slate-200">
          {t(strings.goldens.loading)}
        </p>
      )}
      {listQuery.error && (
        <p data-testid={selectors.goldens.error} className="mt-2 text-red-400">
          {errorMessage(listQuery.error, t)}
        </p>
      )}
      {!listQuery.isLoading && goldens.length === 0 && (
        <p data-testid={selectors.goldens.empty} className="mt-2 text-slate-200">
          {t(strings.goldens.empty)}
        </p>
      )}
      {goldens.length > 0 && (
        <ul data-testid={selectors.goldens.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {goldens.map((g) => (
            <li key={g.slug}>
              <button
                type="button"
                data-testid={selectors.goldens.row}
                onClick={() => setSelectedSlug(g.slug)}
                className="w-full rounded-lg border border-white/10 p-3 text-left hover:border-white/30"
              >
                <div className="font-medium">{g.slug}</div>
                <div className="text-xs text-slate-400">
                  {t(strings.goldens.rowLabel, {
                    slug: g.slug,
                    template: g.templateId,
                    version: g.templateVersionPinned,
                  })}
                </div>
                <div className="text-xs text-slate-500">{g.path}</div>
              </button>
            </li>
          ))}
        </ul>
      )}

      <form
        data-testid={selectors.goldens.registerForm}
        className="mt-4 grid gap-2"
        onSubmit={handleRegister}
      >
        <h3 className="text-xs font-medium text-slate-400">{t(strings.goldens.registerHeading)}</h3>
        <label className="block text-xs text-slate-400">
          {t(strings.goldens.slugLabel)}
          <Input
            data-testid={selectors.goldens.registerSlug}
            value={form.slug}
            onChange={(e) => setForm({ ...form, slug: e.target.value })}
            required
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.goldens.templateLabel)}
          <Input
            data-testid={selectors.goldens.registerTemplate}
            value={form.template}
            onChange={(e) => setForm({ ...form, template: e.target.value })}
            required
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.goldens.versionLabel)}
          <Input
            data-testid={selectors.goldens.registerVersion}
            value={form.version}
            onChange={(e) => setForm({ ...form, version: e.target.value })}
            required
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.goldens.pathLabel)}
          <Input
            data-testid={selectors.goldens.registerPath}
            value={form.path}
            onChange={(e) => setForm({ ...form, path: e.target.value })}
            required
          />
        </label>
        <Button
          data-testid={selectors.goldens.registerSubmit}
          type="submit"
          disabled={registerMutation.isPending}
        >
          {registerMutation.isPending
            ? t(strings.goldens.registering)
            : t(strings.goldens.registerSubmit)}
        </Button>
        {registerMutation.error && (
          <p data-testid={selectors.goldens.registerError} className="text-red-400">
            {errorMessage(registerMutation.error, t)}
          </p>
        )}
      </form>

      {selected && (
        <GoldenDetail
          golden={selected}
          onClose={() => setSelectedSlug(null)}
          onRegenerate={() => handleRegenerate(selected.slug)}
          onDelete={() => handleDelete(selected.slug)}
          isRegenerating={regenerateMutation.isPending}
          isDeleting={deleteMutation.isPending}
        />
      )}

      {statusMessage && (
        <p data-testid={selectors.goldens.detailStatus} className="mt-2 text-xs text-emerald-300">
          {statusMessage}
        </p>
      )}
    </section>
  );
}

interface GoldenDetailProps {
  golden: Golden;
  onClose: () => void;
  onRegenerate: () => void;
  onDelete: () => void;
  isRegenerating: boolean;
  isDeleting: boolean;
}

function GoldenDetail({
  golden,
  onClose,
  onRegenerate,
  onDelete,
  isRegenerating,
  isDeleting,
}: GoldenDetailProps) {
  const { t } = useTranslation();
  return (
    <aside
      data-testid={selectors.goldens.detail}
      className="mt-4 rounded-lg border border-emerald-400/30 bg-emerald-950/20 p-3"
      aria-label={t(strings.goldens.detailHeading)}
    >
      <header className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-emerald-200">{golden.slug}</h3>
        <Button
          data-testid={selectors.goldens.detailClose}
          type="button"
          variant="outline"
          onClick={onClose}
        >
          {t(strings.goldens.close)}
        </Button>
      </header>
      <dl className="mt-2 grid gap-1 text-xs text-slate-200">
        <div>
          <dt className="text-slate-400">{t(strings.goldens.templateLabel)}</dt>
          <dd>
            {golden.templateId}@{golden.templateVersionPinned}
          </dd>
        </div>
        <div>
          <dt className="text-slate-400">{t(strings.goldens.pathLabel)}</dt>
          <dd className="font-mono">{golden.path}</dd>
        </div>
        {golden.lastRegeneratedAt && (
          <div>
            <dt className="text-slate-400">{t(strings.goldens.lastRegeneratedLabel)}</dt>
            <dd>
              {formatDate(timestampDate(golden.lastRegeneratedAt), {
                dateStyle: "medium",
                timeStyle: "short",
              })}
            </dd>
          </div>
        )}
      </dl>
      <div className="mt-3 flex gap-2">
        <Button
          data-testid={selectors.goldens.detailRegenerate}
          type="button"
          onClick={onRegenerate}
          disabled={isRegenerating}
        >
          {isRegenerating ? t(strings.goldens.regenerating) : t(strings.goldens.regenerate)}
        </Button>
        <Button
          data-testid={selectors.goldens.detailDelete}
          type="button"
          variant="outline"
          onClick={onDelete}
          disabled={isDeleting}
          className="border-red-400/40 text-red-300 hover:bg-red-950/30"
        >
          {isDeleting ? t(strings.goldens.deleting) : t(strings.goldens.delete)}
        </Button>
      </div>
    </aside>
  );
}
