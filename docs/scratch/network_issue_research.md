## Notes on 'Host internet access to google.com: FAILED' issue

- `internet.sh` checks connectivity via `ping` to `google.com`.
- In the Codex environment, `ping` fails with 'Network is unreachable' while HTTP requests succeed.
- Web search results show others have had similar issues:
  - StackOverflow: "Not able to connect to network inside docker container".
  - Docker Community Forums: "Docker error network is unreachable".
  - AskUbuntu: "No network access from within docker container".
- These indicate that ICMP (ping) may be blocked in some container environments, though HTTP works.
- The setup script was updated to log the ping failure and continue instead of exiting.

