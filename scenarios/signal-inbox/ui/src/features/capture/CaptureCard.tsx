import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { signalsClient, uploadSignalImage } from "../../api/signals";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";

const signalsKey = ["signals"] as const;

// Capture is always reachable: source kind is inferred from this one field,
// never selected by the operator. Image upload joins this feature through the
// BlobStore transport seam rather than adding bytes to this RPC.
export function CaptureCard() {
  const client = useQueryClient();
  const [source, setSource] = useState("");
  const [note, setNote] = useState("");
  const [image, setImage] = useState<File | null>(null);
  const list = useQuery({ queryKey: signalsKey, queryFn: () => signalsClient.listSignals({}) });
  const capture = useMutation({
    mutationFn: async () => {
      const value = source.trim();
      const request = image
        ? { source: { case: "imagePayloadRef" as const, value: await uploadSignalImage(image) }, captureNote: note }
        : /^https?:\/\//i.test(value)
        ? { source: { case: "url" as const, value }, captureNote: note }
        : { source: { case: "text" as const, value }, captureNote: note };
      return signalsClient.captureSignal(request);
    },
    onSuccess: () => {
      setSource("");
      setNote("");
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
        {!list.isLoading && !list.error && list.data && list.data.signals.length === 0 && <p data-testid={selectors.capture.empty}>No signals captured yet.</p>}
        {list.data && list.data.signals.length > 0 && (
          <ul data-testid={selectors.capture.list} aria-label="Captured signals" className="space-y-2">
            {list.data.signals.map((signal) => <li key={signal.id} className="rounded border border-app-border p-2 text-sm">{signal.sourceKind}: {signal.sourceUrl || signal.extractedContent || signal.rawPayloadRef}{signal.needsAttention && <span data-testid={selectors.capture.needsAttention} className="ml-2 text-app-danger">Needs attention</span>}</li>)}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
