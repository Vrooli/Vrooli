# Problems and open work

- Engine resource lifecycle and workflow network isolation are not proven. Keep managed n8n stopped and do not admit a container pull until host capacity and the owner policy are evidenced.
- Native n8n control is a release gate: the owner must prove supported unattended API-key provisioning, short expiration/revocation, task scoping, and redaction. Official docs describe key creation in Settings → n8n API, but no public REST key-minting operation; do not depend on undocumented `/rest` routes. Trigger coverage also needs actual n8n evidence for manual, webhook, scheduled, subworkflow, and error paths.
- Workspace Sandbox companion-service provider contract is being implemented by its owner; bind only when the typed service declaration and fail-closed lifecycle are available.
- Phase 3 synthetic engine does not prove native n8n execution. Phase 4 still needs description-driven or recorded-exchange simulator generation and thirty unlike public workflows.
- Current JSON import is capped at 512 KiB and the simulator does not stream payloads. Large-payload workflow coverage is open until a bounded storage/streaming policy is implemented and measured.
- F7 remains full scale: ten seeded repairs, complete trigger-family coverage, Make A1 redo, requested-tool A2/A3, and a synthetic Gmail/Sheets/CRM delivery handoff are outstanding.
