"""Classify a corpus through one ordered governed inference call."""

corpus = [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
    "The deployment lost its database connection.",
]

result = ai.batch(
    corpus,
    {"type": "string", "enum": ["infra", "user"]},
    "Choose the primary failure class.",
    role="classify.fast",
)
print({"documents": len(corpus), "results": result.head(3)})

# Live output (2026-08-12):
# {'documents': 3, 'results': [{'model': 'qwen3.5:4b', 'provider': 'ollama',
#   'usage': {'inputTokens': '85', 'outputTokens': '4'}, 'validated': True,
#   'valueJson': '"infra"'}, {'model': 'qwen3.5:4b', 'provider': 'ollama',
#   'usage': {'inputTokens': '86', 'outputTokens': '3'}, 'validated': True,
#   'valueJson': '"user"'}, {'model': 'qwen3.5:4b', 'provider': 'ollama',
#   'usage': {'inputTokens': '87', 'outputTokens': '4'}, 'validated': True,
#   'valueJson': '"infra"'}]}
