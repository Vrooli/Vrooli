-- Template-scaffold residue. `notes` was replaced end-to-end by the jobs +
-- models domains (DECISIONS.md 2026-06-17); `attachments` was never used by this
-- product. Both had zero references in code. Dropping them is a migration rather
-- than a schema edit because a declarative CREATE TABLE cannot express a drop.
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS notes;
