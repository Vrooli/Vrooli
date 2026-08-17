# Inference and discovery

Inference and discovery are governed runtime surfaces, not arbitrary network
access:

```python
verdict = discover("the capability that lists live sessions")
if verdict.head(1)[0].get("null_verdict"):
    print("stop: no governed capability matched")
else:
    print(verdict.head(1))
classification = ai.classify("short text", schema={"type": "object"})
```

Treat a null discovery verdict as a stop. Do not guess a binding path from
natural language.
