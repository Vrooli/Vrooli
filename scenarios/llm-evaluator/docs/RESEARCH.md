# Research Notes

This file documents research conducted during scenario generation and provides references for future development.

## Uniqueness Check

**Query**: `rg -l 'llm-evaluator' /home/matthalloran8/Vrooli/scenarios/`

**Result**: No existing scenario with this name or similar evaluation capabilities.

### Related Scenarios in Repo

1. **prompt-manager** - Manages prompts/skills but doesn't evaluate output quality
2. **ai-model-orchestra-controller** - Routes models but doesn't measure quality
3. **agent-manager** - Executes agents but doesn't evaluate outputs
4. **knowledge-observatory** - Manages knowledge but not evaluation metrics
5. **test-genie** - Runs tests but focuses on code testing, not LLM evaluation

**Conclusion**: LLM Evaluator fills a distinct gap - no existing scenario provides systematic LLM output evaluation.

## External Research

### LLM Evaluation Frameworks

1. **DeepEval** (https://deepeval.com/)
   - Open-source LLM evaluation framework
   - 50+ LLM-evaluated metrics with research backing
   - Supports G-Eval, hallucination detection, answer relevancy
   - CI/CD integration and synthetic dataset generation
   - Key inspiration for metric implementation

2. **RAGAs** (https://docs.ragas.io/en/stable/)
   - Framework for RAG pipeline evaluation
   - Reference-free evaluation using LLMs
   - Key metrics: context_relevancy, context_recall, faithfulness, answer_relevancy
   - Published at EACL 2024 (https://arxiv.org/abs/2309.15217)

3. **Evidently AI** (https://www.evidentlyai.com/llm-guide/llm-evaluation-metrics)
   - Comprehensive guide on LLM evaluation metrics
   - Covers traditional metrics, semantic similarity, and LLM-as-judge

### Core Metrics Research

#### BLEU (Bilingual Evaluation Understudy)
- Measures n-gram precision between generated and reference text
- Applies brevity penalty for overly short outputs
- Scores range from 0 to 1 (higher is better)
- Best for translation and structured generation tasks
- Limitation: Doesn't capture semantic meaning

#### ROUGE (Recall-Oriented Understudy for Gisting Evaluation)
- Measures n-gram recall between generated and reference text
- Variants: ROUGE-1 (unigrams), ROUGE-2 (bigrams), ROUGE-L (longest common subsequence)
- Best for summarization evaluation
- Limitation: Like BLEU, misses semantic nuance

#### BERTScore
- Uses contextual embeddings from BERT models
- Computes cosine similarity between token embeddings
- Better semantic correlation than BLEU/ROUGE
- Limitation: Computationally expensive

#### LLM-as-Judge
- Uses LLM to evaluate outputs with natural language rubrics
- G-Eval methodology: rubric-based with chain-of-thought reasoning
- Most flexible but slowest approach
- Best practices: Clear rubrics, reasoning before scoring, multi-aspect evaluation

### Best Practices

1. **Multi-dimensional evaluation**: Combine traditional metrics with semantic and LLM-based
2. **Reference-free when possible**: RAGAs approach saves annotation costs
3. **Calibrate judges**: Validate LLM judges against human evaluations
4. **Track over time**: Quality metrics should trend upward as system improves
5. **Domain-specific rubrics**: Generic evaluation misses domain-specific quality signals

## Technical References

- [LLM Evaluation Metrics - Confident AI](https://www.confident-ai.com/blog/llm-evaluation-metrics-everything-you-need-for-llm-evaluation)
- [Large Language Model Evaluation in '26](https://research.aimultiple.com/large-language-model-evaluation/)
- [BLEU & ROUGE Explained](https://medium.com/data-science-in-your-pocket/llm-evaluation-metrics-explained-af14f26536d2)
- [DeepEval GitHub](https://github.com/confident-ai/deepeval)
- [RAGAs Paper](https://arxiv.org/abs/2309.15217)

## Implementation Notes

1. **Start with traditional metrics** - BLEU/ROUGE are well-understood and fast
2. **Add semantic similarity via Qdrant** - Leverage existing knowledge-observatory
3. **Use Ollama for LLM judge** - Avoid external API costs during development
4. **Store raw outputs** - Enable re-evaluation with new metrics later
5. **WebSocket for progress** - Evaluation jobs can be long-running

## Contributor Guidelines

- Update this file when discovering new evaluation techniques
- Link research papers and blog posts with dates
- Note which techniques are implemented vs. planned
