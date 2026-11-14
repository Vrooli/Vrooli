-- Recovery metadata only. Prompts, provider output, consumer credentials, and
-- the HostedDoor device key are never stored in this ledger.
CREATE TABLE IF NOT EXISTS ai_gateway_inference_roots (
    root_operation_id TEXT PRIMARY KEY,
    request_sha256 TEXT NOT NULL,
    device_key_generation TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ai_gateway_inference_attempts (
    root_operation_id TEXT NOT NULL REFERENCES ai_gateway_inference_roots(root_operation_id),
    attempt_ordinal INTEGER NOT NULL,
    validation_attempt INTEGER NOT NULL,
    candidate_index INTEGER NOT NULL,
    attempt_key TEXT NOT NULL UNIQUE,
    provider_id TEXT NOT NULL,
    resource_role TEXT NOT NULL,
    state TEXT NOT NULL,
    accounting_status TEXT NOT NULL DEFAULT '',
    charged_credits INTEGER,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (root_operation_id, attempt_ordinal),
    UNIQUE (root_operation_id, validation_attempt, candidate_index)
);
