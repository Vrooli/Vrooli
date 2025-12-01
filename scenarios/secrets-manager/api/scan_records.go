package main

// storeScanRecord persists a scan record when a database connection is available.
// It remains no-op safe when the DB is intentionally skipped.
func storeScanRecord(scan SecretScan) {
	if db == nil {
		logger.Info("Scan completed: %d secrets discovered in %dms", scan.SecretsDiscovered, scan.ScanDurationMs)
		return
	}

	scanner := NewSecretScanner(db)
	if err := scanner.storeScanRecord(scan); err != nil {
		logger.Info("failed to store scan record: %v", err)
	}
}
