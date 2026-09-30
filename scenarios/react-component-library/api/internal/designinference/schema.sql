CREATE TABLE IF NOT EXISTS design_inference_operations (
 id TEXT PRIMARY KEY,
 request_hash TEXT NOT NULL,
 request_json TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('prepared','dispatch_unknown','completed','failed')),
 response_json TEXT NOT NULL DEFAULT '',
 detail TEXT NOT NULL DEFAULT ''
);
