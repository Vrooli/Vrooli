"""Classify a small corpus concurrently through the governed ai facade."""

corpus = [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
]


results = vrooli.gather(*[
    lambda text=text: ai.classify(
        text,
        {"type": "string", "enum": ["infra", "user"]},
        "Choose the primary failure class.",
    )
    for text in corpus
])
print({"documents": len(results), "projections": [result.head(1) for result in results]})

# Live output (2026-08-12):
# {'documents': 2, 'projections': [[{'model': 'qwen3.5:4b', 'provider': 'ollama',
#   'usage': {'inputTokens': '85', 'outputTokens': '4'}, 'validated': True,
#   'valueJson': '"infra"'}], [{'model': 'qwen3.5:4b', 'provider': 'ollama',
#   'usage': {'inputTokens': '86', 'outputTokens': '3'}, 'validated': True,
#   'valueJson': '"user"'}]]}
