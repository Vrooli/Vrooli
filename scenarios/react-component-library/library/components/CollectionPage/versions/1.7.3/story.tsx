import { useState } from "react";
import { CollectionPage } from "./CollectionPage";

const services = [
  { id: "billing", name: "Billing API", owner: "Commerce", status: "Healthy" },
  { id: "catalog", name: "Catalog worker", owner: "Data Platform", status: "Deploying" },
  { id: "identity", name: "Identity gateway", owner: "Platform", status: "Healthy" },
];

function ServiceCatalogView({ initialPane }: { initialPane: "collection" | "inspector" }) {
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState("billing");
  const [mobilePane, setMobilePane] = useState<"collection" | "inspector">(initialPane);
  const visibleServices = services.filter((service) =>
    `${service.name} ${service.owner} ${service.status}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  );
  const selected = services.find((service) => service.id === selectedId);

  return (
    <CollectionPage
      mode="controlled"
      query={query}
      onQueryChange={setQuery}
      onFilterSubmit={setQuery}
      mobilePane={mobilePane}
      onMobilePaneChange={setMobilePane}
      backLabel="Back to services"
      detailLabel="Service details"
      detailTitle={selected?.name ?? "No service selected"}
      gutter="page"
      regions={{
        header: (
          <header>
            <h1 style={{ margin: 0 }}>Service catalog</h1>
            <p>Browse services, filter the list, and inspect the selected record.</p>
          </header>
        ),
        collection: (
          <section aria-label="Services">
            <p aria-live="polite">{visibleServices.length} services</p>
            {visibleServices.length === 0 ? (
              <p>No services match this filter.</p>
            ) : (
              <ul style={{ display: "grid", gap: 8, listStyle: "none", margin: 0, padding: 0 }}>
                {visibleServices.map((service) => (
                  <li key={service.id}>
                    <button
                      type="button"
                      data-collection-page-story-row
                      aria-pressed={selectedId === service.id}
                      onClick={() => {
                        setSelectedId(service.id);
                        setMobilePane("inspector");
                      }}
                      style={{
                        display: "grid",
                        gap: 4,
                        minHeight: 52,
                        width: "100%",
                        padding: 12,
                        textAlign: "left",
                      }}
                    >
                      <strong>{service.name}</strong>
                      <span>
                        {service.owner} · {service.status}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        ),
        inspector: selected ? (
          <section>
            <p>
              <strong>Owner:</strong> {selected.owner}
            </p>
            <p>
              <strong>Status:</strong> {selected.status}
            </p>
            <p>Last health check: 2 minutes ago</p>
          </section>
        ) : (
          <p>Select a service to inspect its details.</p>
        ),
        inspectorNotice: <p>Health checks are current.</p>,
      }}
    />
  );
}

export function ServiceCatalog() {
  return <ServiceCatalogView initialPane="collection" />;
}

export function MobileCollection() {
  return <ServiceCatalogView initialPane="collection" />;
}

export function MobileInspector() {
  return <ServiceCatalogView initialPane="inspector" />;
}

export function EmptyCollection() {
  return <CollectionPage state="empty" />;
}
