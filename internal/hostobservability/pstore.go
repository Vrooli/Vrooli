package hostobservability

const (
	GroupName        = "vrooli-observability"
	ExportRoot       = "/var/lib/vrooli/host-observability"
	PstoreExportDir  = ExportRoot + "/pstore"
	PstoreSourceDir  = "/sys/fs/pstore"
	ManifestFilename = "manifest.json"
)
