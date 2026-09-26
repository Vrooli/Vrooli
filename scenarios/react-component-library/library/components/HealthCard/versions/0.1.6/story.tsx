import { HealthCard } from "./HealthCard";
import { createElement } from "react";

export function Default({ args }: { args: Record<string, unknown> }) {
  return createElement(HealthCard, args as never);
}

export function ErrorState({ args }: { args: Record<string, unknown> }) {
  return createElement(HealthCard, args as never);
}
