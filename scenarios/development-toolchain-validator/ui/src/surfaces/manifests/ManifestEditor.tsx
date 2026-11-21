import { useEffect, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  ConvergenceTarget,
  manifestClient,
  type ContentRule,
} from "../../api/manifest";
import { errorMessage } from "../../lib/errorMessage";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { Textarea } from "../../shared/ui/primitives/Textarea";
import { Card, CardHeader, CardTitle } from "../../shared/ui/primitives/Card";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";

interface EditorState {
  allowedPaths: string;
  wildcardAllowed: boolean;
  convergenceTarget: ConvergenceTarget;
  contentRulesJson: string;
}

const EMPTY: EditorState = {
  allowedPaths: "",
  wildcardAllowed: false,
  convergenceTarget: ConvergenceTarget.NONE,
  contentRulesJson: "[]",
};

function safeStringifyContentRules(rules: readonly ContentRule[]): string {
  try {
    // Strip $typeName etc. from proto messages before serializing.
    const plain = rules.map((r) => ({
      pathGlob: r.pathGlob,
      mustContain: r.mustContain,
      mustNotContain: r.mustNotContain,
    }));
    return JSON.stringify(plain, null, 2);
  } catch {
    return "[]";
  }
}

function parseContentRulesJson(raw: string): ContentRule[] | { error: string } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    return { error: err instanceof Error ? err.message : "invalid JSON" };
  }
  if (!Array.isArray(parsed)) return { error: "Expected an array" };
  const out: ContentRule[] = [];
  for (const item of parsed) {
    if (typeof item !== "object" || item === null) {
      return { error: "Each rule must be an object" };
    }
    const obj = item as Record<string, unknown>;
    const pathGlob = typeof obj.pathGlob === "string" ? obj.pathGlob : "";
    const mustContain = Array.isArray(obj.mustContain)
      ? obj.mustContain.filter((v): v is string => typeof v === "string")
      : [];
    const mustNotContain = Array.isArray(obj.mustNotContain)
      ? obj.mustNotContain.filter((v): v is string => typeof v === "string")
      : [];
    const rule: ContentRule = {
      $typeName: "vrooli.development_toolchain_validator.v1.manifest.ContentRule",
      pathGlob,
      mustContain,
      mustNotContain,
    };
    out.push(rule);
  }
  return out;
}

/**
 * Manifest editor — upsert allowed paths, wildcard flag, convergence
 * target, and content rules for a (skill, golden) pair.
 */
export function ManifestEditor() {
  const { t } = useTranslation();
  const params = useParams<{ skillId: string; goldenSlug: string }>();
  const skillId = params.skillId ?? "";
  const goldenSlug = params.goldenSlug ?? "";
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [form, setForm] = useState<EditorState>(EMPTY);
  const [parseError, setParseError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);

  const getQuery = useQuery({
    queryKey: ["manifest", skillId, goldenSlug] as const,
    queryFn: () => manifestClient.getManifest({ skillId, goldenSlug }),
    enabled: skillId.length > 0 && goldenSlug.length > 0,
    retry: false,
  });

  useEffect(() => {
    const m = getQuery.data?.manifest;
    if (!m) return;
    setForm({
      allowedPaths: m.allowedPaths.join("\n"),
      wildcardAllowed: m.wildcardAllowed,
      convergenceTarget: m.convergenceTarget,
      contentRulesJson: safeStringifyContentRules(m.contentRules),
    });
  }, [getQuery.data]);

  const upsertMutation = useMutation({
    mutationFn: (payload: {
      allowedPaths: string[];
      wildcardAllowed: boolean;
      convergenceTarget: ConvergenceTarget;
      contentRules: ContentRule[];
    }) =>
      manifestClient.upsertManifest({
        manifest: {
          $typeName: "vrooli.development_toolchain_validator.v1.manifest.Manifest",
          skillId,
          goldenSlug,
          allowedPaths: payload.allowedPaths,
          wildcardAllowed: payload.wildcardAllowed,
          convergenceTarget: payload.convergenceTarget,
          contentRules: payload.contentRules,
          templateVersionPinned:
            getQuery.data?.manifest?.templateVersionPinned ?? "",
          skillVersionPinned:
            getQuery.data?.manifest?.skillVersionPinned ?? "",
        },
      }),
    onSuccess: () => {
      setStatus(t(strings.manifests.saveSuccess));
      void queryClient.invalidateQueries({
        queryKey: ["manifest", skillId, goldenSlug],
      });
      void queryClient.invalidateQueries({ queryKey: ["manifests"] });
    },
  });

  const clearStaleMutation = useMutation({
    mutationFn: () => manifestClient.clearStale({ skillId, goldenSlug }),
    onSuccess: () => {
      setStatus(t(strings.manifests.clearStaleSuccess));
      void queryClient.invalidateQueries({ queryKey: ["staleness", "all"] });
    },
  });

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const parsed = parseContentRulesJson(form.contentRulesJson);
    if ("error" in parsed) {
      setParseError(parsed.error);
      return;
    }
    setParseError(null);
    const allowedPaths = form.allowedPaths
      .split("\n")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
    upsertMutation.mutate({
      allowedPaths,
      wildcardAllowed: form.wildcardAllowed,
      convergenceTarget: form.convergenceTarget,
      contentRules: parsed,
    });
  };

  if (getQuery.isLoading) {
    return (
      <LoadingSkeleton
        data-testid={selectors.manifests.loading}
        variant="card"
        count={2}
      />
    );
  }

  return (
    <section
      data-testid={selectors.manifests.editor}
      aria-labelledby={selectors.manifests.editorHeading}
      className="flex flex-col gap-5"
    >
      <PanelHeader
        title={
          <span
            data-testid={selectors.manifests.editorHeading}
            id={selectors.manifests.editorHeading}
          >
            {t(strings.manifests.editorHeading, { skillId, goldenSlug })}
          </span>
        }
        actions={
          <div className="flex gap-2">
            <Button
              data-testid={selectors.manifests.editorBack}
              size="sm"
              variant="ghost"
              onClick={() => void navigate(ROUTES.manifestsIndex)}
            >
              <ArrowLeft className="h-4 w-4" />
              {t(strings.manifests.backToIndex)}
            </Button>
            <Button
              data-testid={selectors.manifests.editorClearStale}
              size="sm"
              variant="outline"
              onClick={() => clearStaleMutation.mutate()}
              disabled={clearStaleMutation.isPending}
            >
              {clearStaleMutation.isPending
                ? t(strings.manifests.clearingStale)
                : t(strings.manifests.clearStaleButton)}
            </Button>
          </div>
        }
      />

      {getQuery.error ? (
        <p
          data-testid={selectors.manifests.error}
          className="text-sm text-status-failure"
        >
          {errorMessage(getQuery.error, t)}
        </p>
      ) : null}

      <Card surface="raised">
        <CardHeader>
          <CardTitle>
            {t(strings.manifests.pinningLabel, {
              template: getQuery.data?.manifest?.templateVersionPinned || "—",
              skill: getQuery.data?.manifest?.skillVersionPinned || "—",
            })}
          </CardTitle>
        </CardHeader>
        <form className="mt-3 flex flex-col gap-3" onSubmit={handleSubmit}>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.manifests.allowedPathsLabel)}
            <Textarea
              data-testid={selectors.manifests.editorAllowedPaths}
              rows={4}
              value={form.allowedPaths}
              onChange={(e) =>
                setForm({ ...form, allowedPaths: e.target.value })
              }
            />
          </label>
          <label className="flex items-center gap-2 text-xs text-app-foreground">
            <input
              type="checkbox"
              data-testid={selectors.manifests.editorWildcardAllowed}
              checked={form.wildcardAllowed}
              onChange={(e) =>
                setForm({ ...form, wildcardAllowed: e.target.checked })
              }
              className="h-4 w-4 rounded border-app-border bg-app-surface-input"
            />
            {t(strings.manifests.wildcardAllowedLabel)}
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.manifests.convergenceTargetLabel)}
            <select
              data-testid={selectors.manifests.editorConvergence}
              value={form.convergenceTarget}
              onChange={(e) =>
                setForm({
                  ...form,
                  convergenceTarget: Number(e.target.value),
                })
              }
              className="mt-1 block w-full rounded-control border border-app-border bg-app-surface-input px-2 py-1 text-sm text-app-foreground"
            >
              <option value={ConvergenceTarget.NONE}>
                {t(strings.manifests.convergenceNone)}
              </option>
              <option value={ConvergenceTarget.EMPTY_DIFF}>
                {t(strings.manifests.convergenceEmptyDiff)}
              </option>
            </select>
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.manifests.contentRulesLabel)}
            <Textarea
              data-testid={selectors.manifests.editorContentRules}
              rows={6}
              value={form.contentRulesJson}
              onChange={(e) =>
                setForm({ ...form, contentRulesJson: e.target.value })
              }
            />
          </label>
          {parseError ? (
            <p className="text-xs text-status-failure">{parseError}</p>
          ) : null}
          <Button
            data-testid={selectors.manifests.editorSave}
            type="submit"
            disabled={upsertMutation.isPending}
          >
            {upsertMutation.isPending
              ? t(strings.manifests.saving)
              : t(strings.manifests.saveButton)}
          </Button>
          {upsertMutation.error ? (
            <p className="text-xs text-status-failure">
              {errorMessage(upsertMutation.error, t)}
            </p>
          ) : null}
        </form>
      </Card>

      {status ? (
        <p
          data-testid={selectors.manifests.editorStatus}
          className="text-xs text-status-pass"
        >
          {status}
        </p>
      ) : null}
    </section>
  );
}
