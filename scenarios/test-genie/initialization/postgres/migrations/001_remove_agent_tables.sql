-- Migration: Remove agent tables
-- Agent state is now managed by agent-manager service
-- Date: 2026-01-06

-- Drop tables in correct order due to foreign key constraints
DROP TABLE IF EXISTS spawn_sessions CASCADE;
DROP TABLE IF EXISTS agent_file_operations CASCADE;
DROP TABLE IF EXISTS spawn_intents CASCADE;
DROP TABLE IF EXISTS agent_scope_locks CASCADE;
DROP TABLE IF EXISTS spawned_agents CASCADE;

-- Drop any orphaned indexes (in case tables were already dropped)
DROP INDEX IF EXISTS idx_spawn_sessions_user;
DROP INDEX IF EXISTS idx_spawn_sessions_scenario;
DROP INDEX IF EXISTS idx_spawn_sessions_expires;
DROP INDEX IF EXISTS idx_spawn_sessions_status;
DROP INDEX IF EXISTS idx_agent_file_operations_agent_id;
DROP INDEX IF EXISTS idx_agent_file_operations_scenario;
DROP INDEX IF EXISTS idx_agent_file_operations_recorded_at;
DROP INDEX IF EXISTS idx_spawn_intents_expires;
DROP INDEX IF EXISTS idx_spawn_intents_scenario;
DROP INDEX IF EXISTS idx_agent_scope_locks_scenario;
DROP INDEX IF EXISTS idx_agent_scope_locks_expires;
DROP INDEX IF EXISTS idx_agent_scope_locks_agent_id;
DROP INDEX IF EXISTS idx_spawned_agents_scenario;
DROP INDEX IF EXISTS idx_spawned_agents_status;
DROP INDEX IF EXISTS idx_spawned_agents_started_at;
