# Program Runtime reuse-loop evidence

Run `./measure.sh` to reproduce measurements M01 through M15 against the live
program-runtime database. Pass a database path to measure a read-only snapshot.

M01-M02 measure the program corpus and its age. M03-M06 measure library tiers,
current callable rows, source payload size, and the kernel catalog ratio.
M07-M08 compare stored binding and provenance claims with recorded program data.
M09 measures retention pins. M10-M12 measure admission and nomination gates.
M13 compares promoted rows with their authoritative invocation sets. M14-M15
measure the retired candidate tier's stored binding histogram and growth rate.

The authoritative definition of a program shape is the distinct binding IDs in
`binding_invocations` where `outcome = 'success'`. The invocation outcome is the
authority on what a program did. The `library_programs` binding-set column is
diagnostic only and must not be used as the source of truth.
