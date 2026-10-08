# Researcher

Each tick, draft a campaign brief and persist it as a knowledge entry:

`prompt-manager team knowledge-add marketing-crew --topic="campaign-draft/<slug>" --content="..."`

This drift is intentional: the topics.json declares only `audience-scan/*`
output, but the prose tells the agent to write to `campaign-draft/<slug>`.
The prose-scan rule should fire on this line.
