-- Reference query for the retained-tail timing report. The runtime store
-- exposes this read model through StartTimingSummaries so CLI callers do not
-- hand-roll SQL or depend on SQLite JSON extensions.
WITH terminal_operations AS (
  SELECT scenario, operation, steps_json
  FROM runtime_start_operations
  WHERE status != 'running'
), step_durations AS (
  SELECT
    scenario,
    operation,
    json_extract(step.value, '$.name') AS step,
    (julianday(json_extract(step.value, '$.ended_at')) -
     julianday(json_extract(step.value, '$.started_at'))) * 86400000.0 AS duration_ms
  FROM terminal_operations, json_each(terminal_operations.steps_json) AS step
  WHERE json_extract(step.value, '$.ended_at') IS NOT NULL
), totals AS (
  SELECT SUM(duration_ms) AS total_ms FROM step_durations
)
SELECT
  scenario,
  operation,
  step,
  COUNT(*) AS count,
  AVG(duration_ms) AS mean_ms,
  SUM(duration_ms) AS total_ms,
  SUM(duration_ms) / NULLIF((SELECT total_ms FROM totals), 0) AS share
FROM step_durations
GROUP BY scenario, operation, step
ORDER BY scenario, operation, step;
