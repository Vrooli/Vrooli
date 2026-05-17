import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render } from "@testing-library/react";
import { fireEvent } from "@testing-library/react";

import { AudioPlayerBar } from "./AudioPlayerBar";

// jsdom doesn't implement the object-URL surface at all (the properties
// are missing, not just stubs), so assign + restore instead of spyOn.
let createObjectURLCalls = 0;
beforeEach(() => {
  createObjectURLCalls = 0;
  (URL as unknown as { createObjectURL: (b: Blob) => string }).createObjectURL = () => {
    createObjectURLCalls += 1;
    return "blob:mock-url";
  };
  (URL as unknown as { revokeObjectURL: (s: string) => void }).revokeObjectURL = () => {};
});

// Leave URL.createObjectURL/revokeObjectURL installed across tests in
// this file — React's commit-phase cleanup runs the AudioPlayerBar
// effect's revokeObjectURL after our `afterEach`, so removing the
// stub here surfaces as an "URL.revokeObjectURL is not a function"
// crash inside React. The next test's beforeEach overwrites it with
// a fresh counter, which is the actual reset point.
afterEach(() => {});

describe("AudioPlayerBar", () => {
  it("renders an <audio> element with controls", () => {
    const { container } = render(<AudioPlayerBar audioUrl="https://example.com/sample.mp3" />);
    const audio = container.querySelector("audio");
    expect(audio).not.toBeNull();
    expect(audio).toHaveAttribute("controls");
    expect(audio).toHaveAttribute("src", "https://example.com/sample.mp3");
  });

  it("wraps audioBytes into a blob URL when no audioUrl is supplied", () => {
    const bytes = new Uint8Array([1, 2, 3, 4]);
    const { container } = render(<AudioPlayerBar audioBytes={bytes} contentType="audio/wav" />);
    const audio = container.querySelector("audio");
    expect(audio).toHaveAttribute("src", "blob:mock-url");
    expect(createObjectURLCalls).toBe(1);
  });

  it("fires onPlayStateChange exactly once on a play event", () => {
    const onPlayStateChange = vi.fn();
    const { container } = render(
      <AudioPlayerBar audioUrl="https://example.com/sample.mp3" onPlayStateChange={onPlayStateChange} />,
    );
    const audio = container.querySelector("audio")!;
    fireEvent.play(audio);
    expect(onPlayStateChange).toHaveBeenCalledTimes(1);
    expect(onPlayStateChange).toHaveBeenCalledWith("playing");
  });

  it("emits paused and ended events on the documented gestures", () => {
    const onPlayStateChange = vi.fn();
    const { container } = render(
      <AudioPlayerBar audioUrl="https://example.com/sample.mp3" onPlayStateChange={onPlayStateChange} />,
    );
    const audio = container.querySelector("audio")!;
    fireEvent.pause(audio);
    fireEvent.ended(audio);
    expect(onPlayStateChange).toHaveBeenNthCalledWith(1, "paused");
    expect(onPlayStateChange).toHaveBeenNthCalledWith(2, "ended");
  });
});
