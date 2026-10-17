// Package presentationqualification performs read-only publication
// qualification for capabilities marked available in an LPBS document.
//
// Configured owner claims are not authority. The verifier reads a terminal
// successful Test Genie validation receipt using
// test-genie:validation:<id>, checks exact admitted/observed source identity
// and server-owned behavioral evidence selectors, then reads the matching
// Deployment Manager release using deployment-manager:release:<id>. The
// release candidate must bind its source revision and support owner. Neither
// adapter writes receipts, imports keys, persists state, or requires the whole
// LPBS delivery to be published. Preview and coming-soon capabilities are
// intentionally exempt; available claims fail closed without all authorities.
package presentationqualification
