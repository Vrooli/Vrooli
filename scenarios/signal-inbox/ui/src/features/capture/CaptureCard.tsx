import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { signalsClient, uploadSignalImage } from "../../api/signals";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "../../components/ui/input";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { selectors } from "../../consts/selectors";
import { SignalClassificationControl } from "../categories/SignalClassificationControl";

const signalsKey = ["signals"] as const;

// Capture is always reachable: source kind is inferred from this one field,
// never selected by the operator. Image upload joins this feature through the
// BlobStore transport seam rather than adding bytes to this RPC.
export function CaptureCard() {
  const client = useQueryClient();
  const [source, setSource] = useState("");
  const [note, setNote] = useState("");
  const [tags, setTags] = useState("");
  const [image, setImage] = useState<File | null>(null);
  const list = useQuery({ queryKey: signalsKey, queryFn: () => signalsClient.listSignals({}) });
  const capture = useMutation({
    mutationFn: async () => {
      const value = source.trim();
      const captureTags = tags.split(",").map((tag) => tag.trim()).filter(Boolean);
      const tagField = captureTags.length > 0 ? { tags: captureTags } : {};
      const request = image
        ? { source: { case: "imagePayloadRef" as const, value: await uploadSignalImage(image) }, captureNote: note, ...tagField }
        : /^https?:\/\//i.test(value)
        ? { source: { case: "url" as const, value }, captureNote: note, ...tagField }
        : { source: { case: "text" as const, value }, captureNote: note, ...tagField };
      return signalsClient.captureSignal(request);
    },
    onSuccess: () => {
      setSource("");
      setNote("");
      setTags("");
      setImage(null);
      void client.invalidateQueries({ queryKey: signalsKey });
    },
  });

  return (
    <Card data-testid={selectors.capture.card} aria-label="Capture signal">
      <CardHeader><CardTitle>Capture a signal</CardTitle></CardHeader>
      <CardContent className="flex flex-col gap-3">
        <Textarea
          data-testid={selectors.capture.source}
          aria-label="URL or text to capture"
          value={source}
          onChange={(event) => setSource(event.target.value)}
          placeholder="Paste a URL or text"
        />
        <Input
          data-testid={selectors.capture.note}
          aria-label="Capture note"
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder="Optional note"
        />
        <Input aria-label="Capture tags" value={tags} onChange={(event) => setTags(event.target.value)} placeholder="Optional tags, comma separated" />
        <Input
          data-testid={selectors.capture.image}
          aria-label="Image to capture"
          type="file"
          accept="image/*"
          onChange={(event) => setImage(event.target.files?.[0] ?? null)}
        />
        <Button data-testid={selectors.capture.submit} onClick={() => capture.mutate()} disabled={(!source.trim() && !image) || capture.isPending}>
          {capture.isPending ? "Capturing…" : "Capture"}
        </Button>
        {capture.error && <p data-testid={selectors.capture.error} className="text-sm text-app-danger">Capture failed. Please try again.</p>}
        {list.error && <p data-testid={selectors.capture.error} className="text-sm text-app-danger">Could not load the journal.</p>}
        {capture.data?.duplicate && <p data-testid={selectors.capture.duplicate} className="text-sm text-app-muted-foreground">Already captured; reused the existing signal.</p>}
        {list.isLoading && <p data-testid={selectors.capture.loading}>Loading journal…</p>}
        {!list.isLoading && !list.error && list.data && list.data.signals.length === 0 && <div data-testid={selectors.capture.empty}><EmptyState title="No signals captured yet." className="border-0 bg-transparent p-0" /></div>}
        {list.data && list.data.signals.length > 0 && (
          <ul data-testid={selectors.capture.list} aria-label="Captured signals" className="space-y-2">
            {list.data.signals.map((signal) => <li key={signal.id} className="rounded border border-app-border p-2 text-sm">{signal.sourceKind}: {signal.sourceUrl || signal.extractedContent || signal.rawPayloadRef}{signal.needsAttention && <span data-testid={selectors.capture.needsAttention} className="ml-2 text-app-danger">Needs attention</span>}<SignalClassificationControl signalID={signal.id} /></li>)}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
