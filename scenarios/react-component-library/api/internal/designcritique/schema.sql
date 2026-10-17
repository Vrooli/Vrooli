CREATE TABLE IF NOT EXISTS design_critiques (
 id TEXT PRIMARY KEY,
 review_hash TEXT NOT NULL,
 review_json TEXT NOT NULL,
 assessment_json TEXT NOT NULL,
 recorded_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS design_critiques_target_order ON design_critiques(
 json_extract(review_json,'$.target.scenario'),
 json_extract(review_json,'$.target.designId'),
 json_extract(review_json,'$.target.revision'),
 json_extract(review_json,'$.target.renderHash'),
 recorded_at DESC,id DESC
);
