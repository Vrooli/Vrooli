# ACE-Step Usage Examples

Install and start through the governed resource lifecycle:

```bash
vrooli resource install ace-step
vrooli resource start ace-step
curl -s http://localhost:18895/health | jq .
curl -s http://localhost:18895/v1/variants | jq .
```

Generate one take:

```bash
curl -fS -X POST http://localhost:18895/v1/compose \
  -H 'content-type: application/json' \
  -d '{"caption":"cold detuned metallic lead, menacing and restless","lyrics":"[instrumental]","duration":45,"bpm":144,"seed":42}' \
  -o take.wav
```

The common install traps are: exclude `flash-attn`, increase the governed wheel
download timeout, acquire the separate 0.6B planner, and pin the `ACE-Step-1.5`
repository rather than the similarly named v1 repository.
