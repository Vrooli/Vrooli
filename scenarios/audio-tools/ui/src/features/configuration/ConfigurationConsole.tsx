/**
 * Configuration Console — audio-tools' operator-facing settings surface.
 *
 * Renders five panels:
 *   - Provider Routing: per-capability tier enable flags (BYOK/Vrooli/Local)
 *   - BYOK Credentials: per-provider redacted credential matrix
 *   - Canonical Voice Mapping: voice.X.Y -> adapter voice name
 *   - Local Resource Health: whisper/kokoro/ollama availability
 *   - Env Overrides View: read-only display of effective env vars
 *
 * Status: P0 skeleton. Backed by SettingsService once that handler lands;
 * today the panels render their static defaults so the archetype shape is
 * visible end-to-end.
 */
export function ConfigurationConsole(): JSX.Element {
  return (
    <section aria-label="Configuration Console" className="audio-tools-feature-configuration">
      <header>
        <h2>Configuration Console</h2>
        <p>Provider routing, BYOK credentials, canonical voices, and local-resource health.</p>
      </header>
      <ProviderRoutingPanel />
      <BYOKCredentialsPanel />
      <CanonicalVoiceMatrix />
      <LocalResourceHealth />
      <EnvOverridesView />
    </section>
  );
}

function ProviderRoutingPanel(): JSX.Element {
  return (
    <article aria-label="Provider routing">
      <h3>Provider Routing</h3>
      <table>
        <thead>
          <tr>
            <th>Capability</th>
            <th>BYOK</th>
            <th>Vrooli</th>
            <th>Local</th>
          </tr>
        </thead>
        <tbody>
          {(["STT", "TTS", "Summarize"] as const).map((cap) => (
            <tr key={cap}>
              <td>{cap}</td>
              <td>enabled</td>
              <td>flag-off</td>
              <td>enabled</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p>Precedence is fixed: BYOK → Vrooli → Local. ErrInsufficientCredits short-circuits.</p>
    </article>
  );
}

function BYOKCredentialsPanel(): JSX.Element {
  const providers = [
    { id: "openai-whisper", cap: "STT" },
    { id: "deepgram", cap: "STT" },
    { id: "openai-tts", cap: "TTS" },
    { id: "elevenlabs", cap: "TTS" },
    { id: "openrouter", cap: "Summarize" },
  ];
  return (
    <article aria-label="BYOK credentials">
      <h3>BYOK Credentials</h3>
      <ul>
        {providers.map((p) => (
          <li key={p.id}>
            <strong>{p.id}</strong> ({p.cap}) — <em>not configured</em>
          </li>
        ))}
      </ul>
      <p>Keys entered here are stored locally in audio-tools state and displayed redacted (sk-***abcd).</p>
    </article>
  );
}

function CanonicalVoiceMatrix(): JSX.Element {
  const canonical = [
    "voice.feminine.warm",
    "voice.feminine.neutral",
    "voice.masculine.warm",
    "voice.masculine.neutral",
    "voice.neutral.default",
  ];
  const adapters = ["local:kokoro-local", "byok:openai-tts", "byok:elevenlabs"];
  return (
    <article aria-label="Canonical voice mapping">
      <h3>Canonical Voices</h3>
      <table>
        <thead>
          <tr>
            <th>Canonical</th>
            {adapters.map((a) => <th key={a}>{a}</th>)}
          </tr>
        </thead>
        <tbody>
          {canonical.map((c) => (
            <tr key={c}>
              <td>{c}</td>
              {adapters.map((a) => <td key={a}>—</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </article>
  );
}

function LocalResourceHealth(): JSX.Element {
  return (
    <article aria-label="Local resource health">
      <h3>Local Resources</h3>
      <ul>
        <li>whisper — unknown</li>
        <li>kokoro — unknown</li>
        <li>ollama — unknown</li>
      </ul>
    </article>
  );
}

function EnvOverridesView(): JSX.Element {
  const vars = [
    "AUDIO_AI_ENABLE_BYOK",
    "AUDIO_AI_ENABLE_VROOLI",
    "AUDIO_AI_ENABLE_LOCAL",
    "AUDIO_WHISPER_URL",
    "AUDIO_KOKORO_URL",
    "AUDIO_OLLAMA_URL",
    "AUDIO_LPBS_BASE_URL",
    "AUDIO_AVAIL_TTL_BYOK",
    "AUDIO_AVAIL_TTL_VROOLI",
  ];
  return (
    <article aria-label="Environment overrides">
      <h3>Environment Overrides</h3>
      <ul>
        {vars.map((v) => <li key={v}><code>{v}</code></li>)}
      </ul>
    </article>
  );
}
