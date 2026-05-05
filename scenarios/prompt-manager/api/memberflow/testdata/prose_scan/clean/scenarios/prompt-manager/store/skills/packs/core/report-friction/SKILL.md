# report-friction

Writes one friction entry per invocation:

`prompt-manager team knowledge-add meta-optimization --topic="friction-inbox/<scope>/<slug>" --content="..."`

`friction-inbox/*` is declared in this skill's `skill.json::writes_to[]`,
so the prose scanner stays clean for this writer skill.
