/// <reference types="vite/client" />

export {};

interface VrooliRuntimeConfig { apiUrl?: string }
declare global { interface Window { __VROOLI_CONFIG__?: VrooliRuntimeConfig } }
