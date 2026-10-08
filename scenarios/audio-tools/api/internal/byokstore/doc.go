// Package byokstore wraps the encrypted credential store. It owns the
// AES-GCM Encryptor, the redacted Fingerprint helper, and the Store
// surface that handlers and provider chains depend on.
//
// The encryption key is sourced from AUDIO_TOOLS_DB_KEY (32-byte hex)
// or persisted to scenario state on first boot. Key rotation is a
// future helper (RotateKey); for P0 single-tenant the persisted key is
// the only key.
package byokstore
