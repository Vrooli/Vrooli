CREATE TABLE IF NOT EXISTS readiness_reviews (
    review_key TEXT PRIMARY KEY,
    scenario TEXT NOT NULL,
    profile_id TEXT NOT NULL,
    candidate_commit TEXT NOT NULL,
    artifact_digest TEXT NOT NULL,
    targets_json TEXT NOT NULL,
    channel TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    status TEXT NOT NULL,
    comparison_mode TEXT NOT NULL,
    predecessor_release_id TEXT,
    predecessor_commit TEXT,
    predecessor_artifact_digest TEXT,
    goal_ref TEXT,
    goal_closed_at TIMESTAMP,
    approved_at TIMESTAMP,
    approved_by TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (scenario, profile_id, candidate_commit, artifact_digest, targets_json, channel, policy_version)
);

CREATE TABLE IF NOT EXISTS readiness_evidence (
    review_key TEXT NOT NULL REFERENCES readiness_reviews(review_key) ON DELETE CASCADE,
    criterion_id TEXT NOT NULL,
    status TEXT NOT NULL,
    applicability TEXT NOT NULL,
    applicability_reason TEXT,
    producer TEXT NOT NULL,
    producer_version TEXT,
    candidate_commit TEXT NOT NULL,
    artifact_digest TEXT NOT NULL,
    target TEXT NOT NULL,
    environment TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    observed_at TIMESTAMP NOT NULL,
    evidence_reference TEXT NOT NULL,
    detail TEXT,
    PRIMARY KEY (review_key, criterion_id, target)
);

CREATE TABLE IF NOT EXISTS readiness_findings (
    review_key TEXT NOT NULL REFERENCES readiness_reviews(review_key) ON DELETE CASCADE,
    criterion_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    PRIMARY KEY (review_key, criterion_id)
);

CREATE TABLE IF NOT EXISTS readiness_human_checks (
    review_key TEXT NOT NULL REFERENCES readiness_reviews(review_key) ON DELETE CASCADE,
    criterion_id TEXT NOT NULL,
    verdict TEXT NOT NULL,
    actor TEXT NOT NULL,
    evidence_reference TEXT NOT NULL,
    reviewed_at TIMESTAMP NOT NULL,
    PRIMARY KEY (review_key, criterion_id)
);

CREATE TABLE IF NOT EXISTS readiness_review_waivers (
    review_key TEXT NOT NULL REFERENCES readiness_reviews(review_key) ON DELETE CASCADE,
    criterion_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    reason TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    invalidation_trigger TEXT,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (review_key, criterion_id)
);

CREATE TABLE IF NOT EXISTS readiness_observations (
    identity_key TEXT NOT NULL,
    criterion_id TEXT NOT NULL,
    producer_binding TEXT NOT NULL,
    producer TEXT NOT NULL,
    producer_version TEXT,
    candidate_commit TEXT NOT NULL,
    artifact_digest TEXT NOT NULL,
    target TEXT NOT NULL,
    environment TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    status TEXT NOT NULL,
    observed_at TIMESTAMP NOT NULL,
    evidence_reference TEXT NOT NULL,
    detail TEXT,
    PRIMARY KEY (identity_key, criterion_id, producer_binding)
);

CREATE INDEX IF NOT EXISTS idx_readiness_reviews_lineage
ON readiness_reviews (scenario, profile_id, channel, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_readiness_evidence_producer
ON readiness_evidence (producer, observed_at DESC);
