const configured = (import.meta.env as { VITE_LANDING_PAGE_URL?: unknown }).VITE_LANDING_PAGE_URL;

export const LANDING_PAGE_URL = typeof configured === "string" && configured.length > 0
  ? configured
  : "https://vrooli.com";
