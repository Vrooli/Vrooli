package sources

import "music-tools/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "sources-import", Path: "/api/v1/sources", Method: "POST",
		Summary: "Import a private local audio source into managed Music Tools storage.", Category: "sources",
		Request:       &module.Schema{Type: "MultipartSource", Properties: map[string]string{"file": "audio file (required; max 128 MiB)", "title": "string (required)", "artist": "string (optional)", "uploader": "string (optional)", "source_url": "HTTPS YouTube URL (optional provenance only; never fetched)", "acquired_at": "RFC3339 timestamp (optional)"}},
		Response:      &module.Schema{Type: "SourceRecord", Properties: map[string]string{"sha256": "string", "source_blob_key": "string", "title": "string", "media_type": "string", "size_bytes": "int64"}},
		Examples:      []module.Example{{Name: "Import a privately acquired source", Curl: `curl -F 'file=@reference.m4a' -F 'title=Reference' -F 'artist=Artist' -F 'source_url=https://www.youtube.com/watch?v=example' "$API/api/v1/sources"`}},
		RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "Imports opaque audio bytes and provenance via multipart; source remains private and content-addressed."},
	},
	{ID: "sources-get", Path: "/api/v1/sources/{sha256}", Method: "GET", Summary: "Read local source provenance by content hash.", Category: "sources", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Returns metadata for a locally managed source without reading or fetching its external URL."}},
	{ID: "sources-audio", Path: "/api/v1/sources/{sha256}/audio", Method: "GET", Summary: "Stream a private managed source audio file with byte-range support.", Category: "sources", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Streams local source bytes from managed storage for private review/playback; it never fetches an external source."}},
}
