"""Classify a corpus through one ordered governed inference call."""

corpus = [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
    "The deployment lost its database connection.",
]

result = vrooli.ai.batch(
    corpus,
    {"type": "string", "enum": ["infra", "user"]},
    "Choose the primary failure class.",
    role="classify.fast",
)
print({"documents": len(corpus), "results": result.head(3)})

# Live validation returned three ordered, schema-validated results from Ollama.
