/// <reference types="vite/client" />

export {};

declare global {
  interface HTMLVideoElement {
    /** Native inline-video fullscreen on iPhone Safari. */
    webkitEnterFullscreen?: () => void;
  }
  interface Window {
    desktop?: {
      components?: {
        list: () => Promise<unknown>;
        install: (id: string) => Promise<unknown>;
        remove: (id: string, keepData: boolean) => Promise<unknown>;
      };
      auth?: {
        signIn: () => Promise<{ state: string }>;
        signOut: () => Promise<void>;
        getAccessToken: () => Promise<string | null>;
        getLocalSessionToken?: () => Promise<string | null>;
        onAuthChanged?: (callback: (event: { event: string }) => void) => void;
        offAuthChanged?: (callback: (event: { event: string }) => void) => void;
      };
    };
  }
}
