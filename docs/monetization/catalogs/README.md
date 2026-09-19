# Monetization Catalogs

Catalogs are durable inventories of monetizable entities. Add new entries here when a thing should be tracked as part of SKU, revenue-line, channel, or scenario-membership canon.

| Catalog | Purpose |
|---|---|
| [`CATALOG.md`](CATALOG.md) | SKU index: base bundles, add-ons, lifecycle state, revisit triggers, and scenario membership. |
| Offer Desk `belongs_to` graph | Machine-readable scenario-to-SKU mapping and validation findings. |
| [`skus/`](skus/) | One file per base bundle or add-on SKU. |
| [`revenue-lines/`](revenue-lines/README.md) | Revenue-line index plus one file per revenue line. |
| [`channels/`](channels/README.md) | Acquisition/distribution channel index plus one file per channel. |

Add SKU-like entities under `skus/`, money-making mechanisms under `revenue-lines/`, and acquisition/distribution surfaces under `channels/`. Directional strategy belongs in [`../strategy/`](../strategy/README.md); supporting proof belongs in [`../evidence/`](../evidence/README.md).
