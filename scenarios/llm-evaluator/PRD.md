# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Comprehensive framework for evaluating LLM outputs across quality, accuracy, faithfulness, and cost-effectiveness dimensions, enabling systematic measurement and continuous improvement of all LLM-powered scenarios within the Vrooli ecosystem.
- **Primary users/verticals**: Internal Vrooli teams, AI/ML Engineers benchmarking models, QA teams establishing quality gates, Product Managers tracking quality metrics.
- **Deployment surfaces**: REST API for integrations, React Dashboard for visualization, CLI for automation/scripting, WebSocket for real-time evaluation updates.
- **Value promise**: Unified quality measurement across all LLM interactions, enabling feedback loops that improve prompts, model selection, and output quality while optimizing cost-per-quality ratios.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Evaluation Job Management | Create, queue, execute, and track evaluation jobs against LLM outputs with status monitoring and progress tracking
- [ ] OT-P0-002 | BLEU Score Calculation | Calculate BLEU scores for measuring n-gram overlap between generated and reference text
- [ ] OT-P0-003 | ROUGE Score Calculation | Calculate ROUGE-1, ROUGE-2, and ROUGE-L scores for summarization and generation evaluation
- [ ] OT-P0-004 | Semantic Similarity Metrics | Calculate embedding-based similarity scores (BERTScore) using vector comparisons
- [ ] OT-P0-005 | LLM-as-Judge Evaluation | Implement rubric-based evaluation using LLM reasoning to score outputs against criteria
- [ ] OT-P0-006 | Results Storage & Retrieval | PostgreSQL storage for evaluation jobs, results, and metrics with query capabilities
- [ ] OT-P0-007 | Dashboard Visualization | Real-time metric displays with charts, comparison views, and trend monitoring
- [ ] OT-P0-008 | REST API Integration | API endpoints for submitting evaluations, retrieving results, and webhook delivery

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | RAG Context Relevancy | Evaluate retrieval quality using context relevancy scoring (RAGAs framework)
- [ ] OT-P1-002 | RAG Faithfulness Scoring | Measure if generated answers are grounded in provided context
- [ ] OT-P1-003 | RAG Answer Relevancy | Score how well generated answers address the original question
- [ ] OT-P1-004 | G-Eval Implementation | Multi-step rubric-based evaluation with chain-of-thought reasoning
- [ ] OT-P1-005 | Model Comparison Suite | Side-by-side output comparison with A/B testing framework
- [ ] OT-P1-006 | Quality Gates & Thresholds | Define minimum quality thresholds with automatic flagging and CI/CD integration
- [ ] OT-P1-007 | Export & Reporting | Generate detailed evaluation reports in PDF, JSON, and CSV formats
- [ ] OT-P1-008 | Multi-Scenario Integration | Direct integration with agent-manager, prompt-manager, and knowledge-observatory

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Hallucination Detection | Fact-checking against context with confidence scoring for generated claims
- [ ] OT-P2-002 | Custom Metric Definition | User-defined evaluation criteria with domain-specific rubric creation
- [ ] OT-P2-003 | Historical Trend Analysis | Long-term quality dashboards with model degradation alerts and pattern detection
- [ ] OT-P2-004 | Cost-Quality Optimization | Quality-per-dollar metrics with model recommendation engine and budget forecasting
- [ ] OT-P2-005 | Adversarial Testing | Edge case evaluation datasets with robustness scoring and stress testing
- [ ] OT-P2-006 | Team Collaboration | Shared evaluation templates, annotation tools, and inter-rater reliability scoring

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (standard library HTTP), React + Vite + TypeScript UI, Go CLI following Vrooli patterns
- Data + storage expectations: PostgreSQL for evaluation jobs, results, metrics, and configurations; Qdrant integration (via knowledge-observatory) for semantic similarity; file-based storage for evaluation templates and rubrics
- Integration strategy: REST API for external/internal integrations, WebSocket for real-time updates, direct Ollama integration for local LLM-as-judge, optional external API support (OpenAI, Anthropic) for judge models
- Non-goals / guardrails: Not a model training/fine-tuning platform, not a prompt management system (that's prompt-manager), not responsible for running actual LLM inference (only evaluating outputs)

## 🤝 Dependencies & Launch Plan
- Required resources: postgres (enabled, primary data store), ollama (optional but recommended for LLM-as-judge), qdrant (optional, for semantic metrics)
- Scenario dependencies: Can operate standalone; enhanced with knowledge-observatory for semantic search; feeds data to prompt-manager for prompt optimization
- Operational risks: LLM-as-judge can be slow for large batches; embedding generation requires GPU resources; need graceful handling of external API rate limits
- Launch sequencing: Phase 1 (P0) - Core evaluation framework and dashboard; Phase 2 (P1) - RAG evaluation and model comparison; Phase 3 (P2) - Advanced analytics and optimization

## 🎨 UX & Branding
- Look & feel: Clean, data-focused dashboard similar to monitoring tools; dark mode support following Vrooli patterns; clear visualization of quality metrics and trends; minimal cognitive load for routine tasks
- Accessibility: WCAG 2.1 AA compliance; keyboard navigation for all features; screen reader compatible data visualizations
- Voice & messaging: Professional and analytical; confidence-inspiring through clear data presentation; technical but approachable
- Branding hooks: Standard Vrooli branding with emphasis on quality metrics and trustworthy evaluation

## 📎 Appendix

### External References
- [LLM Evaluation Metrics Guide - Confident AI](https://www.confident-ai.com/blog/llm-evaluation-metrics-everything-you-need-for-llm-evaluation)
- [DeepEval - LLM Evaluation Framework](https://deepeval.com/)
- [RAGAs - Automated Evaluation of RAG](https://docs.ragas.io/en/stable/)
- [BLEU & ROUGE Metrics Explained](https://medium.com/data-science-in-your-pocket/llm-evaluation-metrics-explained-af14f26536d2)

### Recursive Value
This scenario fills the most critical gap in Vrooli's AI ecosystem by enabling:
- Unified quality measurement across all LLM-powered scenarios
- Feedback loops that improve prompts, model selection, and output quality
- Cost-quality optimization to maximize AI ROI
- Quality gates that prevent poor outputs from reaching downstream systems
