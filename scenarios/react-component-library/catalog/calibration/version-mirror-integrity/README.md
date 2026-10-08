# Version mirror integrity calibration

This gate is calibrated against the live version ledger. Its production
runner requires the scenario database and therefore uses the ledger-backed
calibration harness rather than a copied filesystem fixture.
