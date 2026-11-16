You need a decision from the operator. Prepare it as a Notification Hub decision, the way the
notification-hub skill says, and check it with a dry run. Do not send it.

The skill is in {{SKILL_DIR}}. Read {{SKILL_DIR}}/SKILL.md first and follow it; its references/ and
examples/ folders are beside it. Use only that copy of the skill.

How this task differs from a real send:

- Wherever the skill says to run `notification-hub conversations ask ...`, run
  `{{CHECK}} ...` with the same arguments instead. It always does a dry run and shows you the
  Hub's packet check. Never run `notification-hub conversations ask` yourself, never use
  --and-wait, and never send anything to the operator.
- Write every file you make (the page, a form file, a brief) in {{WORK_DIR}}, and nowhere else.
  Files the facts mention as evidence are in {{WORK_DIR}}/evidence/ when they exist.
- Start every --subject-key with `eval-{{CASE}}-`.
- Improve the decision until the packet check is as clean as you can make it, then stop. Your
  last `{{CHECK}}` command is the decision you would send. End with one short paragraph saying
  what you would send and why.

The facts of the situation follow. They are everything you know.

{{FACTS}}
