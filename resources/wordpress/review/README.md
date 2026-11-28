# F11 WordPress category review

**State: building · Active work: about 23.1 hours · Commission target: 60–150 hours per category**

F11 is the approved WordPress resource direction. The K3 category work covers a managed local WordPress resource, reusable recipes and hard ground, client-shaped forms/shop/content jobs, and review evidence. It is not ready to mark `proven`: free third-party plugins have not been staged through Scenario Dependency Analyzer, the A3 plugin hooks and WooCommerce webhooks have not run, n8n import/execution is unverified, the F7 edge remains unavailable, and operator go is still pending.

## Deliverable status

| Area | Current evidence | State and limit |
|---|---|---|
| Managed WordPress resource | Digest-pinned WordPress/MariaDB/WP-CLI; owner start, stop, reset and snapshot; local isolated site; owner-label and restart-recovery tests | Proven for the managed local resource. No host port or network egress is published. |
| Form jobs | Contact Form 7, WPForms Lite and Forminator recipes; event-ID helper; exported inactive n8n normalizer; 18/18 n8n-code cases (11/11 destination bridge); 9/9 offline PHP adapter/destination cases; live first-party destination status/idempotency checks | Job mapping and first-party destination are proven locally. The PHP harness stubs WordPress and network functions; third-party plugin hooks and imported n8n execution are not proven. |
| Shop job | Free WooCommerce recipe; distinct HMAC/order-sync export; 7/7 fixture cases; first-party order destination and revision behavior | Job contract and synthetic destination are proven. WooCommerce runtime/webhook behavior is not. |
| Content jobs | Core blog/custom-field recipe; read-only content audit; bounded taxonomy editor; synthetic cross-site content migration with a directly exportable versioned archive, plan, collision refusal, category/tag preservation and replay | Proven on local WordPress for synthetic core posts. Pods content types and arbitrary/client content are not in this proof. |
| Hard ground | 51/51: forms 16/16, shop 7/7, content 28/28 | Ground executes resource/workflow logic. It does not substitute for the unrun third-party plugin and n8n paths. |

## Source map

| Capability | Source |
|---|---|
| WordPress resource and owner lifecycle | [`resource.json`](../resource.json), [`owner.go`](../../../internal/resources/wordpress/owner.go), [`images.go`](../../../internal/resources/wordpress/images.go) |
| CLI and command handlers | [`wordpress.go`](../../../internal/cli/resourcecli/wordpress.go), [`handlers.go`](../../../internal/cli/resourcehandlers/handlers.go) |
| Free recipe definitions and package pins | [`recipes.json`](../recipes/recipes.json), [`plugins.lock.json`](../recipes/plugins.lock.json) |
| A3 forms job, adapters and event IDs | [`a3-n8n-workflow.json`](../jobs/forms/a3-n8n-workflow.json), [`shared adapters and harness`](../jobs/shared/) |
| Shop order-sync job | [`order-n8n-workflow.json`](../jobs/shop-order-sync/order-n8n-workflow.json) |
| Core content migration | [`content_migration.go`](../../../internal/resources/wordpress/content_migration.go), [`migration job`](../jobs/content-migration/README.md), [`migration contract`](../jobs/content-migration/contract.schema.json) |
| Content audit | [`content_audit.go`](../../../internal/resources/wordpress/content_audit.go), [`audit job`](../jobs/content-audit/README.md), [`audit contract`](../jobs/content-audit/contract.schema.json), [`live ground receipt schema`](../jobs/content-audit/ground-receipt.schema.json) |
| Fixture grader and regression tests | [`ground.go`](../../../internal/resources/wordpress/ground.go), [`wordpress_test.go`](../../../internal/resources/wordpress/wordpress_test.go), [`ground levels`](../ground/levels.json), [`fixtures`](../ground/fixtures/) |

## Run evidence

The run receipts and active-hours checkpoint are kept in the commission workspace:

- [K3 checkpoint](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/CHECKPOINT.md>)
- [Hard-ground results, 51/51](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/GROUND-RESULTS.json>)
- [Unicode boundary increment, 51/51 ground and source hashes](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/unicode-boundary/result.json>)
- [Managed resource live proof](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/LIVE-PROOF.json>)
- [Core content recipe proof](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/CORE-RECIPE-RESULT.json>)
- [Content migration, taxonomy and reset-isolation proof](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/CONTENT-MIGRATION-RESULT.json>)
- [Versioned synthetic archive export](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/CORE-CONTENT-EXPORT-RESULT.json>)
- [Content audit proof](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/CONTENT-AUDIT-RESULT.json>)
- [Offline PHP adapter results, 6/6](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/FORM-ADAPTER-RESULTS.json>)
- [n8n node fixture results](</home/matthalloran8/.vrooli/research/topics/freelance-workflow-readiness/runs/steward-channel-agent-commerce-20261114/k3/N8N-NORMALIZER-RESULTS.json>)

At the latest footprint check, `k3-ground`, `k3-migration`, and `k3-conflict` all reported stopped with database and WordPress containers exited. Docker had no WordPress-labelled containers.

## Reproduction and pending gates

Run `go test ./internal/resources/wordpress ./internal/cli/resourcecli`, `go test ./internal/cli/resourcehandlers -run '^$'`, and `go run ./cmd/vrooli resource wordpress grade-ground --json`. The resource manifest check is `go run ./cmd/vrooli resource validate wordpress`.

The exact operator actions still pending are: authorize WordPress package support through Scenario Dependency Analyzer; decide whether to permit a local synthetic WordPress Application Password for the destination's authenticated route proof; review F7's persistent-state root activation and run its required `make rebaseline` before attempting its host runtime; then review this category and give go. Until those gates clear, do not claim third-party plugin hooks, WooCommerce webhooks, n8n runtime, or the F7 network edge as proven. No paid plugin, credential, client data, external contact, or external publication was used.
