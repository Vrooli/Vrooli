import { TOKEN_RAMPS, tokens } from "./Tokens";

export function Default() {
  return (
    <section data-testid="foundations.tokens">
      <h2>Design token contract</h2>
      <dl>
        <dt>Spacing steps</dt>
        <dd data-testid="foundations.tokens.space-count">{TOKEN_RAMPS.space.length}</dd>
        <dt>Semantic tokens</dt>
        <dd data-testid="foundations.tokens.semantic-count">
          {Object.keys(tokens.semantic).length}
        </dd>
      </dl>
    </section>
  );
}
