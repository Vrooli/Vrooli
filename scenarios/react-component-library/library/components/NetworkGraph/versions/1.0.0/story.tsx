import { NetworkGraph } from "./NetworkGraph";
export default function NetworkGraphStory() {
  return (
    <NetworkGraph
      nodes={[
        { id: "catalog", label: "Catalog", health: "healthy" },
        { id: "asset", label: "Asset", health: "degraded" },
      ]}
      edges={[{ from: "catalog", to: "asset" }]}
    />
  );
}
