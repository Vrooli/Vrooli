/// <reference types="vite/client" />

export {};

interface VrooliRuntimeConfig {
  apiUrl?: string;
  apiPort?: string;
  localApiEndpoint?: string;
  parentOrigin?: string;
}

declare global { interface Window { __VROOLI_CONFIG__?: VrooliRuntimeConfig } }
