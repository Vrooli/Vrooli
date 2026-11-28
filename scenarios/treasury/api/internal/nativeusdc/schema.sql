-- One unresolved outbound transaction per custody wallet prevents competing
-- processes from assigning the same nonce. Terminal settlements release the
-- slot on the next admission; ambiguous outcomes keep it reserved.
CREATE TABLE IF NOT EXISTS native_usdc_send_locks (
 wallet_id TEXT PRIMARY KEY,
 settlement_id TEXT NOT NULL REFERENCES settlements(id)
);
