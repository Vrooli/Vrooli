// useListState owns the URL-synced filter state machine that both
// /flows and /scenarios use. Callers supply two pure codecs (URL ↔
// state) and an initial default; the hook returns the current state,
// a setter, and the side effect that mirrors changes back into the
// query string.
//
// Keeping the codec functions pure lets the same logic be reused in
// unit tests without mounting React Router.
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";

export interface ListStateCodec<TState> {
  fromUrl(params: URLSearchParams): TState;
  toUrl(state: TState): URLSearchParams;
}

export function useListState<TState>(codec: ListStateCodec<TState>): {
  state: TState;
  setState: (next: TState) => void;
} {
  const [params, setParams] = useSearchParams();
  const [state, setStateRaw] = useState<TState>(() => codec.fromUrl(params));

  useEffect(() => {
    setParams(codec.toUrl(state), { replace: true });
    // setParams identity is stable enough; intentionally only react to
    // state changes so we don't loop on URL → state → URL.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state]);

  return { state, setState: setStateRaw };
}
