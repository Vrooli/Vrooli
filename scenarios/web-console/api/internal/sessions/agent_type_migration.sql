CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			backend TEXT NOT NULL DEFAULT 'standard',
			shell TEXT NOT NULL DEFAULT '/bin/bash',
			cols INTEGER NOT NULL DEFAULT 80,
			rows INTEGER NOT NULL DEFAULT 24,
			policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
			policy_duration TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
			detached INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'live'
				CHECK(status IN ('live','awaiting_recovery','dismissed')),
			agent_type TEXT NOT NULL DEFAULT 'none'
				CHECK(agent_type IN ('none','codex','claude','opencode','grok')),
			launch_command TEXT NOT NULL DEFAULT '',
			agent_session_id TEXT NOT NULL DEFAULT '',
			cwd TEXT NOT NULL DEFAULT '',
			last_rollout_path TEXT NOT NULL DEFAULT '',
			last_activity_at TEXT NOT NULL DEFAULT '',
			orphaned_at TEXT NOT NULL DEFAULT '',
			recovered_into TEXT NOT NULL DEFAULT '',
			origin TEXT NOT NULL DEFAULT 'ui',
			owner TEXT NOT NULL DEFAULT '',
			display_label TEXT NOT NULL DEFAULT ''
		);
