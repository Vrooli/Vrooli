import { useState } from "react";

/**
 * Diagnostics Workbench — operator try-it surface.
 *
 * Four try-it panels (Transcribe, Synthesize, Summarize, Transcode) +
 * a per-call provider trace card. Lets operators verify provider routing
 * end-to-end and surface per-tier latency.
 *
 * Status: P0 skeleton. Wired to the audio-tools Connect clients in a
 * follow-up; today renders the form shape + a placeholder trace card.
 */
export function DiagnosticsWorkbench(): JSX.Element {
  return (
    <section aria-label="Diagnostics Workbench" className="audio-tools-feature-diagnostics">
      <header>
        <h2>Diagnostics Workbench</h2>
        <p>Send a manual STT / TTS / summarize / transcode call and see the provider trace.</p>
      </header>
      <div className="audio-tools-diagnostics-grid">
        <TranscribeTryIt />
        <SynthesizeTryIt />
        <SummarizeTryIt />
        <TranscodeTryIt />
      </div>
      <ProviderTraceCard />
    </section>
  );
}

function TranscribeTryIt(): JSX.Element {
  return (
    <article aria-label="Transcribe try-it">
      <h3>Transcribe</h3>
      <input type="file" accept="audio/*" />
      <button type="button">Transcribe</button>
    </article>
  );
}

function SynthesizeTryIt(): JSX.Element {
  const [text, setText] = useState("");
  return (
    <article aria-label="Synthesize try-it">
      <h3>Synthesize</h3>
      <textarea value={text} onChange={(e) => setText(e.target.value)} aria-label="Synthesize input" />
      <select aria-label="Voice">
        <option value="voice.feminine.warm">voice.feminine.warm</option>
        <option value="voice.feminine.neutral">voice.feminine.neutral</option>
        <option value="voice.masculine.warm">voice.masculine.warm</option>
        <option value="voice.masculine.neutral">voice.masculine.neutral</option>
        <option value="voice.neutral.default">voice.neutral.default</option>
      </select>
      <button type="button">Synthesize</button>
    </article>
  );
}

function SummarizeTryIt(): JSX.Element {
  return (
    <article aria-label="Summarize try-it">
      <h3>Summarize</h3>
      <textarea aria-label="Summarize input" />
      <select aria-label="Level">
        <option value="light">light</option>
        <option value="moderate">moderate</option>
        <option value="heavy">heavy</option>
      </select>
      <button type="button">Summarize</button>
    </article>
  );
}

function TranscodeTryIt(): JSX.Element {
  return (
    <article aria-label="Transcode try-it">
      <h3>Transcode</h3>
      <input type="file" accept="audio/*" />
      <select aria-label="Target format">
        <option value="wav">wav</option>
        <option value="mp3">mp3</option>
        <option value="flac">flac</option>
      </select>
      <button type="button">Transcode</button>
    </article>
  );
}

function ProviderTraceCard(): JSX.Element {
  return (
    <article aria-label="Provider trace" className="audio-tools-provider-trace">
      <h3>Provider Trace</h3>
      <dl>
        <dt>Tier</dt><dd>—</dd>
        <dt>Provider</dt><dd>—</dd>
        <dt>Model</dt><dd>—</dd>
        <dt>Latency</dt><dd>—</dd>
      </dl>
    </article>
  );
}
