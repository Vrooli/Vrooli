/**
 * Docs / Integration Viewer — in-product browser for audio-tools docs and
 * embed examples. P0 skeleton lists the canonical docs by name; full
 * rendered Markdown + copy-paste snippets land in Phase G.
 */
export function DocsViewer(): JSX.Element {
  const sections: Array<{ title: string; path: string; description: string }> = [
    { title: "PRD", path: "PRD.md", description: "Operational targets and scope" },
    { title: "Architecture", path: "docs/concepts/ARCHITECTURE.md", description: "Surface map and load-bearing principles" },
    { title: "Domains", path: "docs/concepts/DOMAINS.md", description: "Capability domain inventory" },
    { title: "Flows", path: "docs/concepts/FLOWS.md", description: "End-to-end request flows" },
    { title: "Integrations", path: "docs/concepts/INTEGRATIONS.md", description: "Resource and scenario dependencies" },
    { title: "API endpoints", path: "docs/reference/api-endpoints.md", description: "Connect-RPC + REST exception inventory" },
    { title: "CLI commands", path: "docs/reference/cli-commands.md", description: "audio-tools subcommand reference" },
    { title: "Configuration", path: "docs/reference/configuration.md", description: "Env vars and runtime knobs" },
    { title: "Adoption snippets", path: "docs/reference/adoption.md", description: "Copy-paste snippets for consumer scenarios" },
    { title: "Invariants", path: "docs/internal/INVARIANTS.md", description: "Non-negotiable contracts" },
    { title: "Seams", path: "docs/internal/SEAMS.md", description: "Test seams + cross-scenario boundaries" },
    { title: "Extraction sources", path: "docs/internal/EXTRACTION-SOURCES.md", description: "Source-by-source migration provenance" },
  ];
  return (
    <section aria-label="Docs Viewer" className="audio-tools-feature-docs">
      <header>
        <h2>Docs &amp; Integration Viewer</h2>
        <p>Browse PRD, architecture, reference, and adoption snippets without leaving the console.</p>
      </header>
      <ul>
        {sections.map((s) => (
          <li key={s.path}>
            <strong>{s.title}</strong> — {s.description} <code>{s.path}</code>
          </li>
        ))}
      </ul>
      <EmbedExamples />
    </section>
  );
}

function EmbedExamples(): JSX.Element {
  const tsx = `import { VoiceInputButton, AudioPlayerBar } from "@audio-tools/embed";

export function MyVoiceFeature() {
  return (
    <>
      <VoiceInputButton onTranscript={(t) => console.log(t)} />
      <AudioPlayerBar audioUrl="/cache/example.mp3" />
    </>
  );
}`;
  const curl = `curl -X POST http://localhost:\${API_PORT}/vrooli.audio_tools.v1.tts.TTSService/Synthesize \\
  -H 'Content-Type: application/json' \\
  -d '{"text":"hello","voice":"voice.feminine.warm","response_format":"mp3"}' \\
  --output out.mp3`;
  return (
    <article aria-label="Embed examples">
      <h3>Embedding snippets</h3>
      <pre><code>{tsx}</code></pre>
      <pre><code>{curl}</code></pre>
    </article>
  );
}
