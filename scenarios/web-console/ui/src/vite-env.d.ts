/// <reference types="vite/client" />

declare global {
  interface Window {
    /**
     * Audio-tools HTTP base URL injected at bootstrap by main.tsx.
     * Read by @audio-tools/embed when consumers don't provide an
     * explicit AudioToolsProvider client.
     */
    __AUDIO_TOOLS_URL__?: string;

    /**
     * Stable token explaining why audio-tools is unreachable.
     * Empty when audio-tools is available. AudioUnavailableBanner
     * reads this to render an actionable message.
     */
    __AUDIO_TOOLS_UNAVAILABLE_REASON__?: string;
  }
}

export {};
