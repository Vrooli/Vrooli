import { SearchFilterResults } from "./SearchFilterResults";

const items = [
  "Alpha workspace · Updated 2 minutes ago",
  "Beta workspace · Needs attention",
  "Gamma workspace · Updated yesterday",
];

export function Default() {
  return <SearchFilterResults items={items} />;
}

export function NoMatch() {
  return <SearchFilterResults query="delta" items={items} />;
}

export function Empty() {
  return <SearchFilterResults items={[]} />;
}

export function ErrorState() {
  return (
    <SearchFilterResults
      items={items}
      state="error"
      errorMessage="The search index is unavailable. Your query is preserved for retry."
      onRetry={() => undefined}
    />
  );
}
