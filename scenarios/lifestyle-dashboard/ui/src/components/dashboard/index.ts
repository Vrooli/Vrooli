/**
 * Dashboard components barrel export.
 * Provides centralized access to all dashboard-specific components.
 *
 * Architecture note: Components are organized by domain concept (dashboard)
 * rather than by technical category (e.g., cards, charts) to make the
 * codebase "scream" its purpose.
 */

export { StatusBadge } from "./StatusBadge";
export { DomainCard } from "./DomainCard";
export { EventRow } from "./EventRow";
export { TimelineChart } from "./TimelineChart";
export { StatCard } from "./StatCard";
export { DomainBreakdown } from "./DomainBreakdown";
export { Header } from "./Header";
export { LifestyleScoreCard } from "./LifestyleScoreCard";
export { default as BriefCard } from "./BriefCard";
export { default as BriefPreview } from "./BriefPreview";
