import { describe, expect, it, vi } from "vitest";
import { readAudio } from "./audition-audio";

describe("audition streamed byte bounds", () => {
  it("joins chunked responses without Content-Length", async () => {
    const stream = new ReadableStream({ start(controller) { controller.enqueue(Uint8Array.from([1, 2])); controller.enqueue(Uint8Array.from([3])); controller.close(); } });
    expect([...new Uint8Array(await readAudio(new Response(stream), 3))]).toEqual([1, 2, 3]);
    expect(stream.locked).toBe(false);
  });
  it("cancels an oversized stream and releases its lock", async () => {
    const cancel = vi.fn();
    const stream = new ReadableStream({ start(controller) { controller.enqueue(Uint8Array.from([1, 2, 3])); }, cancel });
    await expect(readAudio(new Response(stream), 2)).rejects.toThrow("browser budget");
    expect(cancel).toHaveBeenCalledOnce();
    expect(stream.locked).toBe(false);
  });
  it("cancels a declared oversized response before acquiring or reading its body", async () => {
    const cancel = vi.fn();
    const stream = new ReadableStream({ cancel });
    const response = new Response(stream, { headers: { "Content-Length": "9" } });
    await expect(readAudio(response, 8)).rejects.toThrow("browser budget");
    expect(cancel).toHaveBeenCalledOnce();
    expect(stream.locked).toBe(false);
  });
  it("refuses oversized fallback bytes when no stream reader is supplied", async () => {
    const arrayBuffer = vi.fn().mockResolvedValue(Uint8Array.from([1, 2, 3]).buffer);
    const response = { headers: new Headers(), arrayBuffer } as unknown as Response;
    await expect(readAudio(response, 2)).rejects.toThrow("browser budget");
    expect(arrayBuffer).toHaveBeenCalledOnce();
  });

});
