CREATE TABLE IF NOT EXISTS post_types (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  paired_skill TEXT NOT NULL DEFAULT '',
  skill_exists INTEGER NOT NULL DEFAULT 0,
  doc_v1 INTEGER NOT NULL DEFAULT 0,
  responsibilities_declared INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS post_type_failure_modes (
  post_type_id TEXT NOT NULL REFERENCES post_types(id),
  failure_mode TEXT NOT NULL,
  PRIMARY KEY (post_type_id, failure_mode)
);

-- The post-type canon is currently two executable v1 types and ten v0 stubs.
-- These rows are idempotent seed data: operator registration may later enrich a
-- type, while a fresh scenario always reflects the documented starting state.
INSERT OR IGNORE INTO post_types (id, status, paired_skill, skill_exists, doc_v1, responsibilities_declared) VALUES
  ('dev-log', 'active', 'x-dev-log', 1, 1, 1),
  ('scenario-spotlight', 'active', 'x-scenario-spotlight', 1, 1, 1),
  ('single-image-ad', 'v0', 'x-single-image-ad', 0, 0, 0),
  ('slideshow-listicle', 'v0', 'x-slideshow-listicle', 0, 0, 0),
  ('slideshow-tips-then-plug', 'v0', 'x-slideshow-tips-then-plug', 0, 0, 0),
  ('infographic', 'v0', 'x-infographic', 0, 0, 0),
  ('narrative-talking-head', 'v0', 'x-narrative-talking-head', 0, 0, 0),
  ('day-in-life-ugc', 'v0', 'x-day-in-life-ugc', 0, 0, 0),
  ('problem-agitate-solve', 'v0', 'x-problem-agitate-solve', 0, 0, 0),
  ('demo-recording', 'v0', 'x-demo-recording', 0, 0, 0),
  ('comparison-reel', 'v0', 'x-comparison-reel', 0, 0, 0),
  ('slideshow-voiceover', 'v0', 'x-slideshow-voiceover', 0, 0, 0);

INSERT OR IGNORE INTO post_type_failure_modes (post_type_id, failure_mode) VALUES
  ('dev-log', 'what_without_why'),
  ('dev-log', 'internal_vocabulary_leakage'),
  ('scenario-spotlight', 'capability_inflation'),
  ('scenario-spotlight', 'demo_theater'),
  ('single-image-ad', 'misleading_visual_claim'),
  ('slideshow-listicle', 'padded_listicle'),
  ('slideshow-tips-then-plug', 'padded_tips_pre_plug'),
  ('infographic', 'misleading_data_presentation'),
  ('narrative-talking-head', 'undisclosed_persona'),
  ('day-in-life-ugc', 'undisclosed_persona'),
  ('problem-agitate-solve', 'demo_theater'),
  ('demo-recording', 'demo_theater'),
  ('comparison-reel', 'unfair_comparison'),
  ('slideshow-voiceover', 'inherited_slideshow_failure_modes');
