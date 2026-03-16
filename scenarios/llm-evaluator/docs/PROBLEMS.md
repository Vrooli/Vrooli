# Known Problems & Deferred Ideas

This file tracks open issues and deferred ideas for the LLM Evaluator scenario.

## Open Issues

### Technical Risks

1. **LLM-as-Judge Performance**
   - LLM-based evaluation can be slow for large batches
   - Mitigation: Implement batching, caching, and async processing
   - Priority: P0 blocker

2. **Embedding Generation at Scale**
   - BERTScore requires embedding generation which needs GPU resources
   - Mitigation: Use Qdrant (via knowledge-observatory) for pre-computed embeddings
   - Priority: P0 consideration

3. **External API Rate Limits**
   - If using OpenAI/Anthropic for judge models, need graceful rate limit handling
   - Mitigation: Prefer Ollama for local evaluation, add retry logic with backoff
   - Priority: P1

### Integration Challenges

1. **Knowledge Observatory Dependency**
   - Semantic similarity relies on Qdrant integration
   - Need to handle case where knowledge-observatory is not running
   - Priority: P0

2. **Agent Manager Integration**
   - Direct integration for evaluating agent outputs needs careful API design
   - Priority: P1

### UX Concerns

1. **Dashboard Performance with Large Datasets**
   - Need pagination/virtualization for historical data visualization
   - Priority: P1

## Deferred Ideas

### Future Enhancements (P2+)

1. **Model Fine-Tuning Feedback Loop**
   - Use evaluation data to generate fine-tuning datasets
   - Requires separate model training infrastructure

2. **Automated Prompt Optimization**
   - Use evaluation results to suggest prompt improvements
   - Could integrate with prompt-manager

3. **Multi-Language Evaluation**
   - Extend BLEU/ROUGE to support non-English languages
   - Need language-specific tokenization

4. **Crowd-Sourced Evaluation**
   - Allow human annotators to supplement automated evaluation
   - Requires authentication and annotation UI

5. **Evaluation Templates Marketplace**
   - Share evaluation rubrics across teams/scenarios
   - Could be a separate scenario

## Notes for Contributors

- Add new issues with date and brief description
- Move resolved issues to PROGRESS.md
- Keep this file under 200 lines
