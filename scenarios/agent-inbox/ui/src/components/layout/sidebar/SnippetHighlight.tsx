/** Renders a snippet with the match portion highlighted via <mark> tag. Safe alternative to dangerouslySetInnerHTML. */
export function SnippetHighlight({
  snippet,
  matchStart,
  matchEnd,
}: {
  snippet: string;
  matchStart?: number;
  matchEnd?: number;
}) {
  if (matchStart == null || matchEnd == null || matchStart >= matchEnd || matchStart >= snippet.length) {
    return <>{snippet}</>;
  }
  const before = snippet.slice(0, matchStart);
  const match = snippet.slice(matchStart, matchEnd);
  const after = snippet.slice(matchEnd);
  return (
    <>
      {before}
      <mark className="bg-yellow-500/30 text-yellow-200 px-0.5 rounded">{match}</mark>
      {after}
    </>
  );
}
