/// <reference types="vite/client" />

export {};

declare global {
  interface Window {
    desktop?: {
      auth?: {
        signIn: () => Promise<{ state: string }>;
        signOut: () => Promise<void>;
        getAccessToken: () => Promise<string | null>;
        onAuthChanged?: (callback: (event: { event: string }) => void) => void;
        offAuthChanged?: (callback: (event: { event: string }) => void) => void;
      };
    };
  }
}
