import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { componentsClient } from "../../api/components";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

export function IngestComponentForm() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");
  const [sourceFile, setSourceFile] = useState("");
  const [slug, setSlug] = useState("");
  const [tags, setTags] = useState("");
  const ingest = useMutation({
    mutationFn: () => componentsClient.ingestComponent({
      scenario,
      sourceFile,
      slug,
      tags: tags.split(",").map((tag) => tag.trim()).filter(Boolean),
    }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["components"] }),
  });

  return (
    <details className="rounded-xl border border-app-border bg-app-surface p-4">
      <summary className="cursor-pointer text-sm font-medium text-app-foreground">{t(strings.components.ingest.title)}</summary>
      <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.components.ingest.description)}</p>
      <form
        className="mt-3 grid gap-2 md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          ingest.mutate();
        }}
      >
        <Input value={scenario} onChange={(event) => setScenario(event.target.value)} placeholder={t(strings.components.ingest.scenario)} required />
        <Input value={sourceFile} onChange={(event) => setSourceFile(event.target.value)} placeholder={t(strings.components.ingest.sourceFile)} required />
        <Input value={slug} onChange={(event) => setSlug(event.target.value)} placeholder={t(strings.components.ingest.slug)} required />
        <Input value={tags} onChange={(event) => setTags(event.target.value)} placeholder={t(strings.components.ingest.tags)} />
        <div className="md:col-span-2">
          <Button type="submit" disabled={ingest.isPending}>{ingest.isPending ? t(strings.components.ingest.running) : t(strings.components.ingest.submit)}</Button>
        </div>
      </form>
      {ingest.error && <p className="mt-2 text-xs text-app-danger">{errorMessage(ingest.error, t)}</p>}
      {ingest.data && (
        <p className="mt-2 text-xs text-app-success">
          {t(strings.components.ingest.success, { version: ingest.data.draftVersion, findings: ingest.data.findings.length })}
        </p>
      )}
    </details>
  );
}
