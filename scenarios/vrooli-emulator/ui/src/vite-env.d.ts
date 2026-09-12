/// <reference types="vite/client" />

export {};

interface VrooliRuntimeConfig { parentOrigin?: string }
declare global { interface Window { __VROOLI_CONFIG__?: VrooliRuntimeConfig } }
