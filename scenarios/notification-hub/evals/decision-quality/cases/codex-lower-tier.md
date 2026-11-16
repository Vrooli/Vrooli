# Case: codex-lower-tier

Taken from Hub ask bd401fbc (15 Nov 2026) and its history. You are Agent Manager's
usage agent (`--from "Agent Manager"`).

## Facts you know

- A usage setting sends new Codex work to a cheaper model (GPT-6 Luna) so the
  operator's weekly usage quota lasts longer. It was turned on 9 Nov 2026 with no
  end time. Four kinds of work (fallback, review, supervision, judgment) keep the
  usual models.
- Because it has no end time, Agent Manager asks once a day whether to keep it on.
- History: the operator has been asked this 14 times since 4 Nov ("keep it on, or
  turn it off?"). They answered "keep on" every time they answered; twice it
  expired unanswered and twice it was dismissed.
- Keep it on: nothing changes; quota lasts longer.
- Turn it off: new Codex work goes back to the stronger models, which use quota faster.
- No answer: it stays on, and the question comes back tomorrow.
- A root-cause option exists: make the setting standing (no daily check), and only
  ask again if usage or quality changes, for example if weekly usage passes 80%
  or work quality drops. The operator can still turn it off any time with
  `agent-manager usage mode end <id>`.

## What the sender would recommend

Keep it on, and stop the daily check (make it standing). Reason: the operator has
said keep-on every time; the daily question costs attention and changes nothing.
