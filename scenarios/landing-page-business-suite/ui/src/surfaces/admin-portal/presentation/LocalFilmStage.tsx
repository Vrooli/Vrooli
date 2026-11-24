import { useEffect, useState, type ChangeEvent } from 'react';

/** Browser-local film staging surface. Mount only inside the authenticated admin page. */
export function LocalFilmStage({ variantSlug }: { variantSlug: string }) {
  const [file, setFile] = useState<File>();
  const [source, setSource] = useState('');
  const [hash, setHash] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!file) { setSource(''); return; }
    const url = URL.createObjectURL(file);
    setSource(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  const choose = (event: ChangeEvent<HTMLInputElement>) => {
    const selected = event.currentTarget.files?.[0];
    const accepted = selected && (selected.type === 'video/mp4' || selected.name.toLowerCase().endsWith('.mp4')) ? selected : undefined;
    if (hash && accepted) { setMessage('Remove the existing private copy before selecting another MP4.'); event.currentTarget.value = ''; return; }
    setFile(accepted);
    setMessage(accepted ? '' : selected ? 'Choose a valid MP4 file.' : '');
    // Permit reselecting the same file after clearing it.
    event.currentTarget.value = '';
  };

  const upload = async () => {
    if (!file || busy) return;
    setBusy(true); setMessage('');
    try {
      const response = await fetch('/api/v1/admin/presentation-media', { method: 'POST', credentials: 'include', cache: 'no-store', headers: { 'Content-Type': 'video/mp4' }, body: file });
      if (!response.ok) throw new Error(response.status === 403 ? 'Reauthenticate as an administrator, then retry.' : await response.text());
      const result: unknown = await response.json();
      if (!result || typeof result !== 'object' || !('sha256' in result) || typeof result.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(result.sha256)) throw new Error('The media owner returned an invalid hash receipt.');
      setHash(result.sha256); setMessage('Private MP4 stored and verified. Save this hash in the recorded product-demo draft to preview the protected copy.');
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Private media upload failed.'); }
    finally { setBusy(false); }
  };

  const discard = async () => {
    if (!hash || busy) return;
    setBusy(true); setMessage('');
    try {
      const response = await fetch(`/api/v1/admin/presentation-media/${encodeURIComponent(hash)}`, { method: 'DELETE', credentials: 'include', cache: 'no-store' });
      if (!response.ok) throw new Error(response.status === 409 ? 'This MP4 is referenced by a saved draft or retained published revision and cannot be removed.' : await response.text());
      setHash(''); setMessage('Unreferenced private MP4 removed.');
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Private media cleanup failed.'); }
    finally { setBusy(false); }
  };

  return <section className="space-y-4 rounded-xl border border-cyan-300/30 bg-slate-900/80 p-5" aria-label="Private local film stage" data-testid="local-film-stage">
    <header><p className="text-xs uppercase tracking-widest text-cyan-200">Private staging · {variantSlug}</p><h2 className="mt-1 text-xl font-semibold">Local film preview</h2><p className="mt-1 text-sm text-slate-300">Preview an MP4 in this browser or upload a content-addressed private copy. Upload requires recent admin reauthentication and the shared disk-capacity gate. No public release occurs here.</p></header>
    <label className="block text-sm" htmlFor="local-film-file">Local MP4
      <input id="local-film-file" className="mt-2 block w-full text-sm file:mr-3 file:rounded-md file:border-0 file:bg-cyan-200 file:px-3 file:py-2 file:font-medium file:text-slate-950" type="file" accept="video/mp4,.mp4" onChange={choose} />
    </label>
    {file && source ? <>
      <p className="break-all text-xs text-slate-300">Selected: <span className="font-mono">{file.name}</span> · {(file.size / 1024 / 1024).toFixed(1)} MiB</p>
      <video className="block max-h-[72vh] w-full rounded-lg bg-black" src={source} controls playsInline preload="metadata" aria-label={`Aquila film: ${file.name}`} />
      <div className="flex flex-wrap gap-3"><button type="button" className="rounded-md border border-cyan-200/40 px-3 py-2 text-sm hover:bg-white/10 disabled:opacity-50" disabled={busy || file.size > 256 * 1024 * 1024 || Boolean(hash)} onClick={() => void upload()}>{busy ? 'Working…' : 'Store private MP4 copy'}</button><button type="button" className="rounded-md border border-white/20 px-3 py-2 text-sm hover:bg-white/10" onClick={() => setFile(undefined)}>Clear browser selection</button></div>
      {file.size > 256 * 1024 * 1024 && <p role="alert" className="text-sm text-amber-200">The private media owner accepts files up to 256 MiB.</p>}
    </> : <p className="rounded-lg border border-dashed border-white/20 p-8 text-center text-sm text-slate-400">No local film selected. Choose an MP4 to start playback.</p>}
    {hash && <><p className="break-all text-xs text-slate-200">Draft media hash: <code>{hash}</code></p><pre className="overflow-x-auto rounded bg-black/40 p-3 text-xs">{JSON.stringify({ media_ref: hash, playback: { provider: 'local', external_url: '', layout: 'stacked', play_label: 'Play film', caption: 'Product film', unavailable_label: 'Video unavailable' } }, null, 2)}</pre><button type="button" className="rounded-md border border-red-300/40 px-3 py-2 text-sm hover:bg-red-950/40 disabled:opacity-50" disabled={busy} onClick={() => void discard()}>Remove if unreferenced</button></>}
    {message && <p role="status" className="break-words text-sm text-cyan-100">{message}</p>}
  </section>;
}
