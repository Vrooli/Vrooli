# Configuration

The API reads no database-path environment variable. The storage resolver chooses the scenario data class and variant-aware namespace from the scenario id, so no inherited value can redirect it at another scenario's database. Lifecycle-provided API/UI ports must be used; hard-coding ports bypasses scenario routing.

Adapters are registered data, not environment toggles. A manual adapter and a file adapter require no secret. Future upstream credentials must enter through the platform secret boundary and must not be written to the journal or committed to this repository.
