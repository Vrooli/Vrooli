import { useMutation } from "@tanstack/react-query";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useState } from "react";

import { retrievalClient, type RetrievedSignal } from "../../api/retrieval";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";

function optionalTimestamp(value: string) {
  if (!value) return undefined;
  const date = new Date(`${value}T00:00:00Z`);
  return Number.isNaN(date.valueOf()) ? undefined : timestampFromDate(date);
}

// Search is a journal view, not an ambient queue. Every disposition and
// category is selectable as a filter but neither state removes a signal from
// the underlying index or unfiltered results.
export function SearchCard() {
  const [text, setText] = useState("");
  const [tags, setTags] = useState("");
  const [categoryID, setCategoryID] = useState("");
  const [disposition, setDisposition] = useState("");
  const [sourceKind, setSourceKind] = useState("");
  const [capturedAfter, setCapturedAfter] = useState("");
  const [capturedBefore, setCapturedBefore] = useState("");
  const [results, setResults] = useState<RetrievedSignal[]>([]);
  const [nextPageAfter, setNextPageAfter] = useState("");
  const [hasSearched, setHasSearched] = useState(false);
  const search = useMutation({
    mutationFn: ({ pageAfter }: { pageAfter: string }) => retrievalClient.search({
      filter: {
        text: text.trim(),
        tags: tags.split(",").map((tag) => tag.trim()).filter(Boolean),
        categoryId: categoryID.trim(),
        disposition,
        sourceKind: sourceKind.trim(),
        capturedAfter: optionalTimestamp(capturedAfter),
        capturedBefore: optionalTimestamp(capturedBefore),
        pageSize: 50, pageAfter,
      },
    }),
    onSuccess: (response, variables) => { setResults((current) => variables.pageAfter ? [...current, ...response.results] : response.results); setNextPageAfter(response.nextPageAfter); setHasSearched(true); },
  });

  return <Card aria-label="Search signal journal"><CardHeader><CardTitle>Search the signal journal</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
    <p className="text-sm text-app-muted-foreground">Search includes every captured signal, including items dropped from ambient review. Filters only narrow this view; they never remove journal records.</p>
    <div className="grid gap-2 md:grid-cols-2">
      <Input aria-label="Search text" value={text} onChange={(event) => setText(event.target.value)} placeholder="Keywords or semantic query" />
      <Input aria-label="Search tags" value={tags} onChange={(event) => setTags(event.target.value)} placeholder="Tags, comma separated" />
      <Input aria-label="Category ID filter" value={categoryID} onChange={(event) => setCategoryID(event.target.value)} placeholder="Category ID (optional)" />
      <Select aria-label="Disposition filter" value={disposition} onChange={(event) => setDisposition(event.target.value)} placeholder="Any disposition" options={[{ value: "unresolved", label: "Unresolved" }, { value: "triaged", label: "Triaged" }, { value: "dropped", label: "Dropped from ambient" }]} />
      <Input aria-label="Source kind filter" value={sourceKind} onChange={(event) => setSourceKind(event.target.value)} placeholder="Source kind (url, text, image)" />
      <div className="grid grid-cols-2 gap-2"><Input aria-label="Captured after" type="date" value={capturedAfter} onChange={(event) => setCapturedAfter(event.target.value)} /><Input aria-label="Captured before" type="date" value={capturedBefore} onChange={(event) => setCapturedBefore(event.target.value)} /></div>
    </div>
    <Button onClick={() => search.mutate({ pageAfter: "" })} disabled={search.isPending}>{search.isPending ? "Searching…" : "Search journal"}</Button>
    {search.error && <p className="text-app-danger">Search failed. The captured journal has not changed.</p>}
    {hasSearched && results.length === 0 && <p>No signals match these filters.</p>}
    {results.length > 0 && <ul aria-label="Search results" className="space-y-2">{results.map((result) => <li key={result.signal?.id} className="rounded border border-app-border p-3 text-sm">
      <div className="flex flex-wrap justify-between gap-2"><strong>{result.signal?.sourceUrl || result.signal?.sourceKind || "Captured signal"}</strong><span>{result.disposition || "unresolved"}{result.categoryId ? ` · ${result.categoryId}` : ""}</span></div>
      <p>{result.signal?.extractedContent || result.signal?.captureNote || "No extracted text."}</p>
      {result.signal?.tags.length ? <p className="text-app-muted-foreground">Tags: {result.signal.tags.join(", ")}</p> : null}
      {result.score > 0 ? <p className="text-app-muted-foreground">Relevance: {result.score.toFixed(2)}</p> : null}
    </li>)}</ul>}
    {nextPageAfter && <Button variant="secondary" onClick={() => search.mutate({ pageAfter: nextPageAfter })} disabled={search.isPending}>{search.isPending ? "Loading…" : "Load more results"}</Button>}
  </CardContent></Card>;
}
