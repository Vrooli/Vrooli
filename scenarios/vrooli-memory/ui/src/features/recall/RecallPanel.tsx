import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { recallMemory } from "../../api/recall";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

export function RecallPanel() {
  const { t } = useTranslation();
  const [input, setInput] = useState("");
  const [query, setQuery] = useState("");
  const result = useQuery({ queryKey: ["recall", query], queryFn: () => recallMemory(query, 10), enabled: query.length > 0 });
  const experienceState: ExperienceSurfaceState = result.isLoading ? "loading" : result.error ? "error" : query && result.data?.length === 0 ? "empty" : "ready";
  return <ExperienceSurface surfaceId="results" state={experienceState} data-testid={selectors.recall.surface} aria-label={t(strings.recall.title)}>
    <Card>
    <CardHeader><CardTitle>{t(strings.recall.title)}</CardTitle></CardHeader>
    <CardContent className="space-y-4">
      <form className="flex gap-2" onSubmit={(event) => { event.preventDefault(); setQuery(input.trim()); }}>
        <Input value={input} onChange={(event) => setInput(event.target.value)} aria-label={t(strings.recall.queryLabel)} placeholder={t(strings.recall.queryPlaceholder)} />
        <Button type="submit" disabled={!input.trim()}>{t(strings.recall.submit)}</Button>
      </form>
      {result.isLoading && <p data-testid={selectors.recall.loading}>{t(strings.recall.loading)}</p>}
      {result.error && <p data-testid={selectors.recall.error} className="text-app-danger">{errorMessage(result.error, t)}</p>}
      {query && result.data?.length === 0 && <div data-testid={selectors.recall.empty}><EmptyState title={t(strings.recall.empty)} /></div>}
      {result.data && result.data.length > 0 && <ol data-testid={selectors.recall.list} className="space-y-3">
        {result.data.map((hit) => <li key={hit.nodeId} className="rounded-control border border-app-border p-3"><p className="whitespace-pre-wrap">{hit.text}</p><p className="mt-2 text-sm text-app-muted-foreground">{hit.facetId} · {t(strings.recall.score, { score: hit.score.toFixed(2) })}{hit.summary ? ` · ${t(strings.recall.summary, { count: hit.span })}` : ""}</p></li>)}
      </ol>}
    </CardContent>
    </Card>
  </ExperienceSurface>;
}
