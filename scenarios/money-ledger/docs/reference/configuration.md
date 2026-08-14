# Configuration

The API accepts `SQLITE_PATH` as the explicit database path and `SQLITE_DB` as a compatibility alias. When neither is set, the storage resolver chooses the scenario data class and variant-aware namespace. Lifecycle-provided API/UI ports must be used; hard-coding ports bypasses scenario routing.

Adapters are registered data, not environment toggles. A manual adapter and a file adapter require no secret. Future upstream credentials must enter through the platform secret boundary and must not be written to the journal or committed to this repository.
