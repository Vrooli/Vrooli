import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function prettyNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1
  }).format(value);
}

export function getApiBaseUrl(): string {
  const fromEnv = import.meta.env.VITE_API_BASE_URL;
  if (fromEnv) {
    return fromEnv.replace(/\/$/, "");
  }

  const { protocol, hostname } = window.location;
  const defaultPort = hostname === "localhost" ? "20400" : window.location.port;
  const port = import.meta.env.VITE_API_PORT ?? defaultPort;
  return `${protocol}//${hostname}${port ? `:${port}` : ""}`;
}
