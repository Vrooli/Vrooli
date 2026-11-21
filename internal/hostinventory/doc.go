// Package hostinventory owns observed host inventory facts such as devices,
// accelerators, memory, and operating-system identity. It does not mutate the
// host or decide remediation; callers consume its evidence through typed facts.
//
// Linux graphics identity comes from sysfs. Windows and macOS normal inventory
// and integrity collection share the native SetupAPI/IORegistry device adapter.
// Vendor tools enrich those identities; absence of a vendor tool does not hide
// a device. Display metadata alone cannot create native device identities.
//
// Linux Bluetooth reports USB function binding, sysfs HCI registration and
// rfkill state independently from optional passive BlueZ Adapter1 availability.
// Its bounded no-auto-start observation never powers or scans a radio. A missing
// or failed BlueZ probe is unknown; HCI presence alone is not a usable controller
// or peer-function proof. Governed wireless verification shares this decoder.
//
// System RAM is never a dedicated GPU-memory observation. Historical derived
// Apple unified-memory estimates are excluded by GPU.DedicatedVRAMBytes and
// accelerator projections, and collector cache version 2 forces fresh probes.
// Metal selection requires explicit supported-Metal metadata, independently of
// architecture, GPU name, and whether dedicated memory was measured.
package hostinventory
