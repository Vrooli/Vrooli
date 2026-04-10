import { useEffect, useState } from 'react';

interface RulesCountResponse {
  data?: {
    rules?: unknown[];
  };
}

/** Total rule count fetched from the API, for use in other components. */
export function useRuleCount(): number {
  const [count, setCount] = useState(0);
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch('/api/v1/rules');
        if (!res.ok) return;
        const json = (await res.json()) as RulesCountResponse;
        setCount(json.data?.rules?.length ?? 0);
      } catch { /* ignore */ }
    })();
  }, []);
  return count;
}
