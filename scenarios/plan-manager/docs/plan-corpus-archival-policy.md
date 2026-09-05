# Plan corpus archival policy

Plan Manager indexes every importable Markdown plan under the operator plan
root. A completed plan remains in the active index for 365 days after its last
phase reaches a terminal state. After that period it may move to the archive
index, but the source Markdown and its mirror metadata remain recoverable.

Plans with no terminal execution state, an active execution, or unresolved
validation evidence are never archived by age alone. Reconciliation reports
parse failures and mirror repair needs explicitly; it does not delete source
documents. This policy keeps Recall complete while preventing genuinely old,
finished work from competing with current plans indefinitely.
