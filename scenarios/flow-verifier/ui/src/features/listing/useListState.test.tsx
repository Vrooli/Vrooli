import { afterEach, describe, expect, it } from "vitest";
import { act, cleanup, render } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";

import { useListState, type ListStateCodec } from "./useListState";

interface S {
  q: string;
  flag: boolean;
}

const codec: ListStateCodec<S> = {
  fromUrl(p) {
    return { q: p.get("q") ?? "", flag: p.get("flag") === "1" };
  },
  toUrl(s) {
    const out = new URLSearchParams();
    if (s.q) out.set("q", s.q);
    if (s.flag) out.set("flag", "1");
    return out;
  },
};

function Probe({ onReady }: { onReady: (api: ReturnType<typeof useListState<S>>, search: string) => void }) {
  const api = useListState(codec);
  const loc = useLocation();
  onReady(api, loc.search);
  return null;
}

describe("useListState", () => {
  afterEach(() => cleanup());

  it("reads initial state from the URL", () => {
    let captured: S | null = null;
    render(
      <MemoryRouter initialEntries={["/x?q=hello&flag=1"]}>
        <Probe onReady={(api) => { captured = api.state; }} />
      </MemoryRouter>,
    );
    expect(captured).toEqual({ q: "hello", flag: true });
  });

  it("writes state back to the URL via the codec", () => {
    let api: ReturnType<typeof useListState<S>> | null = null;
    let search = "";
    render(
      <MemoryRouter initialEntries={["/x"]}>
        <Probe
          onReady={(a, s) => {
            api = a;
            search = s;
          }}
        />
      </MemoryRouter>,
    );
    act(() => {
      api!.setState({ q: "abc", flag: false });
    });
    expect(search).toContain("q=abc");
    expect(search).not.toContain("flag");
  });
});
