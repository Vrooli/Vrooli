# Asset target and adopter ownership evidence

Captured 2026-08-26 after the catalog ownership split.

| Contract | Evidence |
|---|---|
| `asset` is a valid phase target kind | `scenarios/test-genie/schemas/test-genie-phase-descriptor.schema.json` and `providerdescriptor` accept `asset`; the focused descriptor test passes. |
| Subject-list inventory is bounded and keyed | `InventoryService.Scan` returns one `SubjectFindings` row per supplied subject and rejects a subject without an id; `scan_subjects_test.go` passes. |
| Catalog gate ownership | `catalog/config.json` contains 31 gates: 27 blocking, 4 advisory, 21 attributable, and 10 corpus. None is `i18n-adopted`, `selectors-adopted`, or `adopter-hygiene`. |
| Adopter checks have a health owner | `ui-health/api/internal/uiinterop/checks/standard_component_adoption_contracts.go` owns provider, selector-composition, selector-semantics, and adopter-test ownership checks. The ui-health interop and integration tests pass. |
| Generated contract outputs | `make generate SCENARIO=ui-health` and `make generate SCENARIO=react-component-library` completed successfully. |

The React Component Library inventory handler does not add a scenario-specific
branch to ui-health's generic scan orchestration. The framework-specific
adapter remains behind the declared InventoryService boundary.
