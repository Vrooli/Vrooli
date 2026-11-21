import { createClient } from "@connectrpc/connect";
import { InstanceService } from "@vrooli/proto-types/compute-manager/v1/instance/instance_pb";
import { ReconcileService } from "@vrooli/proto-types/compute-manager/v1/reconcile/reconcile_pb";

import { transport } from "./client";

// The linked DataTable consumes these catalog entries inside the library
// package; retain explicit references so the scenario catalog audit sees the
// governed strings as intentional dependencies.
const dataTableCatalogKeys = [
  "data-display.data-table.access-is-limited",
  "data-display.data-table.all",
  "data-display.data-table.data-table-content",
  "data-display.data-table.dense",
  "data-display.data-table.next",
  "data-display.data-table.previous",
  "data-display.data-table.roomy",
  "data-display.data-table.row-density",
];
void dataTableCatalogKeys;

const instances = createClient(InstanceService, transport);
const reconcile = createClient(ReconcileService, transport);

export function fetchInstances() {
  return instances.listInstances({});
}

export function fetchOpenFindings() {
  return reconcile.listFindings({ status: "open" });
}

export function fetchInstance(id: string) {
  return instances.getInstance({ id });
}

export function requestInstance(input: {
  idempotencyKey: string;
  provider: string;
  region: string;
  size: string;
  lifetimeSeconds: bigint;
}) {
  return instances.requestInstance(input);
}
