/**
 * @libraryId react-component-library:IngestProbe
 * @displayName IngestProbe
 * @description Ingested from react-component-library:ui/src/components/ingest-probe/IngestProbe.tsx
 * @version 1.0.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** A throwaway origin used to verify that ingest creates a new asset as a draft only. */
export function IngestProbe({ label = "Probe" }: { label?: string }) {
  return <span data-testid="ingest-probe">{label}</span>;
}
