// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
/* eslint-disable @typescript-eslint/require-await, @typescript-eslint/unbound-method, @typescript-eslint/no-this-alias, no-restricted-syntax */
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MessageSpeechButton } from "./MessageSpeechButton";

const speech = vi.hoisted(() => ({ enabled: false, capabilityCheck: vi.fn(async () => false), synthesize: vi.fn() }));
vi.mock("./portalVoice", () => ({ portalSpeech: speech }));

describe("MessageSpeechButton", () => {
  beforeEach(() => {
    speech.enabled = false;
    speech.capabilityCheck.mockReset().mockResolvedValue(false);
    speech.synthesize.mockReset();
    Object.defineProperty(window, "speechSynthesis", { configurable: true, value: { speak: vi.fn(), cancel: vi.fn() } });
    vi.stubGlobal("SpeechSynthesisUtterance", class { onend: (() => void) | null = null; onerror: ((event: { error: string }) => void) | null = null; constructor(public readonly text: string) {} });
  });

  it("uses browser speech when the optional provider is missing", async () => {
    render(<MessageSpeechButton text="Hello Portal" />);
    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(window.speechSynthesis.speak).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("button", { name: "chat.message.stopSpeaking" })).toBeInTheDocument();
  });

  it("cancels in-flight speech without affecting the message", async () => {
    render(<><MessageSpeechButton text="Keep this text" /><p>Keep this text</p></>);
    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.stopSpeaking" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "chat.message.stopSpeaking" }));
    expect(window.speechSynthesis.cancel).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Keep this text")).toBeInTheDocument();
  });

  it("uses Audio Tools when healthy and aborts synthesis on stop", async () => {
    speech.enabled = true;
    speech.capabilityCheck.mockResolvedValue(true);
    let resolve!: (blob: Blob) => void;
    speech.synthesize.mockImplementation(() => new Promise<Blob>(r => { resolve = r; }));
    render(<MessageSpeechButton text="Server speech" />);
    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(speech.capabilityCheck).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "chat.message.stopSpeaking" }));
    expect(speech.synthesize).toHaveBeenCalledWith("Server speech", expect.any(AbortSignal));
    resolve(new Blob(["late"]));
  });

  it("falls back to browser speech when the optional provider is unhealthy", async () => {
    speech.enabled = true;
    speech.capabilityCheck.mockResolvedValue(false);
    render(<MessageSpeechButton text="Browser fallback" />);

    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(window.speechSynthesis.speak).toHaveBeenCalledTimes(1));
  });

  it("reports unavailable when browser speech is not present", async () => {
    Object.defineProperty(window, "speechSynthesis", { configurable: true, value: undefined });
    render(<MessageSpeechButton text="No speech" />);

    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.speechUnavailable" })).toBeDisabled());
  });

  it("reports a provider synthesis failure", async () => {
    speech.enabled = true;
    speech.capabilityCheck.mockResolvedValue(true);
    speech.synthesize.mockRejectedValue(new Error("server failed"));
    render(<MessageSpeechButton text="Failed speech" />);

    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.speechFailed" })).toBeInTheDocument());
  });

  it("returns browser speech to idle after completion and marks real errors failed", async () => {
    let utterance: { onend: (() => void) | null; onerror: ((event: { error: string }) => void) | null } | undefined;
    vi.stubGlobal("SpeechSynthesisUtterance", class {
      onend: (() => void) | null = null;
      onerror: ((event: { error: string }) => void) | null = null;
      constructor(public readonly text: string) { utterance = this; }
    });
    render(<MessageSpeechButton text="Events" />);

    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.stopSpeaking" })).toBeInTheDocument());
    await act(async () => { utterance?.onend?.(); });
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.speak" })).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: "chat.message.speak" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.stopSpeaking" })).toBeInTheDocument());
    await act(async () => { utterance?.onerror?.({ error: "network" }); });
    await waitFor(() => expect(screen.getByRole("button", { name: "chat.message.speechFailed" })).toBeInTheDocument());
  });
});
