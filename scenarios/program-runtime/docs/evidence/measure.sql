-- Reproducible measurement set for the program-runtime reuse loop.
-- Run: sqlite3 <db> < measure.sql
-- "shape" is the set of DISTINCT binding ids invoked SUCCESSFULLY.
-- binding_invocations.outcome is authoritative; library_programs is not.
.mode list
.headers off

SELECT '== M01 corpus size: programs by status ==';
SELECT status, count(*) FROM programs GROUP BY status;
SELECT '== M02 corpus age: oldest and newest program ==';
SELECT min(created_at), max(created_at) FROM programs;
SELECT '== M03 library rows by tier ==';
SELECT tier, count(*) FROM library_programs GROUP BY tier;
SELECT '== M04 library_current selections (the only callable rows) ==';
SELECT count(*) FROM library_current;
SELECT '== M05 source bytes by tier (kernel catalog payload) ==';
SELECT tier, count(*), sum(length(source)) FROM library_programs GROUP BY tier;
SELECT '== M06 callable source bytes vs total written to every kernel ==';
SELECT (SELECT sum(length(source)) FROM library_programs p JOIN library_current c ON c.name=p.name AND c.version=p.version),
       (SELECT sum(length(source)) FROM library_programs);
SELECT '== M07 stored binding set vs authoritative successful invocations ==';
WITH truth AS (SELECT program_id, count(DISTINCT binding_id) n FROM binding_invocations WHERE outcome='success' GROUP BY 1),
 cand AS (SELECT source_program_id pid, CASE WHEN called_binding_ids IN ('','[]') THEN 0 ELSE (length(called_binding_ids)-length(replace(called_binding_ids,',','')))+1 END n FROM library_programs WHERE tier='candidate')
SELECT CASE WHEN c.n > COALESCE(t.n,0) THEN 'stored_overcounts_includes_failed_or_refused' WHEN c.n < COALESCE(t.n,0) THEN 'stored_undercounts_misses_real_calls' ELSE 'match' END, count(*)
FROM cand c LEFT JOIN truth t ON t.program_id=c.pid GROUP BY 1 ORDER BY 2 DESC;
SELECT '== M08 origin column truth: claimed vs actual provenance ==';
SELECT l.origin, l.tier, count(*) FROM library_programs l GROUP BY 1,2;
SELECT p.provenance, count(*) FROM library_programs l JOIN programs p ON p.id=l.source_program_id WHERE l.tier='candidate' GROUP BY 1 ORDER BY 2 DESC;
SELECT '== M09 retention pin: programs exempt by any row vs by a promoted row ==';
SELECT (SELECT count(*) FROM programs), (SELECT count(*) FROM programs WHERE EXISTS(SELECT 1 FROM library_programs l WHERE l.source_program_id=programs.id)), (SELECT count(*) FROM programs WHERE EXISTS(SELECT 1 FROM library_programs l WHERE l.source_program_id=programs.id AND l.tier='promoted')), (SELECT count(*) FROM programs WHERE created_at < datetime('now','-90 day'));
SELECT '== M10 admission threshold calibration (whole succeeded corpus) ==';
WITH truth AS (SELECT p.id pid, p.session_id, p.provenance prov, (SELECT count(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') nsucc, (SELECT group_concat(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') shape FROM programs p WHERE p.status='succeeded')
SELECT 'every_succeeded_run', count(*), count(DISTINCT shape) FROM truth UNION ALL SELECT 'succ_ge1_binding', count(*), count(DISTINCT shape) FROM truth WHERE nsucc>=1 UNION ALL SELECT 'succ_ge2_bindings', count(*), count(DISTINCT shape) FROM truth WHERE nsucc>=2 UNION ALL SELECT 'agent_prov_only', count(*), count(DISTINCT shape) FROM truth WHERE prov='1' UNION ALL SELECT 'agent_and_ge1', count(*), count(DISTINCT shape) FROM truth WHERE prov='1' AND nsucc>=1 UNION ALL SELECT 'agent_and_ge2_PRIOR_PROPOSAL', count(*), count(DISTINCT shape) FROM truth WHERE prov='1' AND nsucc>=2 UNION ALL SELECT 'nontest_and_ge2', count(*), count(DISTINCT shape) FROM truth WHERE prov<>'3' AND nsucc>=2;
SELECT '== M11 nomination gate calibration: shapes with >=2 successful bindings ==';
WITH truth AS (SELECT p.session_id, p.provenance prov, (SELECT count(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') n, (SELECT group_concat(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') shape FROM programs p WHERE p.status='succeeded')
SELECT count(*), count(DISTINCT session_id), count(DISTINCT CASE WHEN prov<>'3' THEN session_id END), sum(prov='1'), sum(prov='2'), sum(prov='3'), shape FROM truth WHERE n>=2 GROUP BY shape ORDER BY 1 DESC LIMIT 15;
SELECT '== M12 nominations under the chosen gate (occ>=3 AND nontest sessions>=2) ==';
WITH truth AS (SELECT p.session_id, p.provenance prov, (SELECT count(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') n, (SELECT group_concat(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=p.id AND b.outcome='success') shape FROM programs p WHERE p.status='succeeded')
SELECT shape, count(*) occ, count(DISTINCT CASE WHEN prov<>'3' THEN session_id END) nontest_sessions FROM truth WHERE n>=2 GROUP BY shape HAVING occ>=3 AND nontest_sessions>=2 ORDER BY occ DESC;
SELECT '== M13 promoted rows: stored set vs authoritative successful invocations ==';
SELECT l.name, l.version, l.called_binding_ids, COALESCE((SELECT group_concat(DISTINCT b.binding_id) FROM binding_invocations b WHERE b.program_id=l.source_program_id AND b.outcome='success'),'') , length(l.source) FROM library_programs l WHERE l.tier='promoted' ORDER BY l.name, l.version;
SELECT '== M14 candidate binding-count histogram (stored, i.e. attempted) ==';
SELECT n, count(*) FROM (SELECT CASE WHEN called_binding_ids IN ('','[]') THEN 0 ELSE (length(called_binding_ids)-length(replace(called_binding_ids,',','')))+1 END n FROM library_programs WHERE tier='candidate') GROUP BY n ORDER BY n;
SELECT '== M15 candidate accumulation rate by day ==';
SELECT substr(created_at,1,10), count(*) FROM library_programs WHERE tier='candidate' GROUP BY 1 ORDER BY 1 DESC;
