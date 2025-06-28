# Routine Creation Backlog

This file contains ideas and requirements for routines to be created. Items here will be processed by the AI creation system to generate complete routine definitions.

## Format

Each backlog item should include:
- **Name**: Descriptive name for the routine
- **Category**: Type of routine (see categories below)
- **Description**: What the routine does and why it's valuable
- **Inputs**: What data the routine needs
- **Outputs**: What the routine produces
- **Strategy**: Recommended execution strategy (conversational, reasoning, deterministic)
- **Priority**: High, Medium, Low
- **Notes**: Additional context or requirements

## Categories

- **Metareasoning**: Strategy selection, process optimization, self-reflection
- **Productivity**: Task management, planning, scheduling
- **Knowledge**: Summarization, research, information processing
- **Content**: Writing, communication, creative tasks
- **Analysis**: Data processing, evaluation, insights
- **Integration**: System connections, automation, workflows
- **Learning**: Skill development, feedback, adaptation
- **Personal Development**: Lifestyle, wellness, habits
- **Business**: Project management, strategic planning
- **Financial**: Budget, expense, investment assistance
- **Data**: Analysis, visualization, processing
- **Technical**: Developer support, code assistance
- **Student**: Academic tools, study aids

## Backlog Items

### METAREASONING (High Priority)

### 1. Yes-Man Avoidance
- **Category**: Metareasoning
- **Description**: Detects when users are applying social pressure for agreement or compliance. Helps AI systems avoid "yes-man" behavior by identifying pressure tactics and generating balanced responses. Based on https://x.com/rubenhssd/status/1924517497853903090
- **Inputs**: User message, conversation context, historical interaction patterns
- **Outputs**: Pressure detection score, manipulation tactics identified, balanced response suggestions
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Essential for maintaining AI trustworthiness and avoiding manipulation. Should detect various pressure types: emotional manipulation, false urgency, leading questions, etc.

### 2. Introspective Self-Review
- **Category**: Metareasoning
- **Description**: A periodic routine where an agent reflects on its recent decisions, successes, and mistakes. Orchestrates multi-step analysis gathering performance data, evaluating outcomes, and storing insights for future improvement.
- **Inputs**: Performance logs, task outcomes, feedback history, time period for review
- **Outputs**: Self-assessment report, improvement recommendations, updated performance metrics
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses embedding search to compare current performance with past experiences. Results stored in swarm's knowledge base for continuous learning.

### 3. Goal Alignment & Progress Checkpoint
- **Category**: Metareasoning
- **Description**: Regularly reviews team progress against goals and ensures current tasks align with overall mission. Can be scheduled for end-of-day or milestone checkpoints.
- **Inputs**: Current goals/OKRs, task status reports, project timeline, team member updates
- **Outputs**: Progress report, alignment analysis, course correction recommendations, updated priorities
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: If misalignment detected, suggests adjustments like task reprioritization or resource reallocation. Can trigger coordination routines automatically.

### 4. Capability Gap Analysis
- **Category**: Metareasoning
- **Description**: Identifies knowledge or skill gaps that could hinder task performance. Scans upcoming tasks and checks swarm's knowledge base for necessary expertise.
- **Inputs**: Upcoming task requirements, current knowledge base inventory, team skill profiles
- **Outputs**: Gap assessment report, missing capabilities list, remediation plan
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses embedding similarity search. Can recommend acquiring data, training new subroutines, or alerting human operators.

### 5. Memory Maintenance
- **Category**: Metareasoning
- **Description**: Performs "garbage collection" and optimization of swarm's shared memory. Prunes outdated entries, consolidates duplicates, and maintains efficient knowledge retrieval.
- **Inputs**: Vector embedding store, access frequency logs, retention policies
- **Outputs**: Cleaned memory store, consolidation report, archived data
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Uses clustering to find redundant entries. Improves lookup speed and relevance of similarity searches.

### 6. Dynamic Task Allocation
- **Category**: Metareasoning
- **Description**: Maps incoming tasks to best-suited agents in the swarm. Uses agent skills, workload, and task requirements for optimal distribution.
- **Inputs**: New task description, agent skill profiles, current workload data
- **Outputs**: Task assignments, delegation plan, workload balance report
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Can be modeled as BPMN flow. Flags tasks for human intervention if no suitable agent available.

### 7. Role & Team Optimizer
- **Category**: Metareasoning
- **Description**: Evaluates and updates swarm's team structure to improve collaboration. Analyzes task distribution and performance to recommend organizational changes.
- **Inputs**: Team performance metrics, task distribution logs, upcoming project requirements
- **Outputs**: Team structure recommendations, role adjustments, new agent proposals
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Can automatically update agent roles or suggest creating/retiring agents based on workload patterns.

### PRODUCTIVITY & TASK MANAGEMENT (High Priority)

### 8. Daily Agenda Planner
- **Category**: Productivity
- **Description**: Compiles a prioritized daily schedule by analyzing tasks, appointments, and deadlines. Uses time-blocking to arrange tasks optimally while avoiding conflicts.
- **Inputs**: Task list, calendar events, deadlines, energy levels, priorities
- **Outputs**: Time-blocked schedule, prioritized task list, buffer time recommendations
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Incorporates Task Prioritizer and Time Block Scheduler as subroutines. Accounts for breaks and focus time.

### 9. Task Prioritizer
- **Category**: Productivity
- **Description**: Sorts tasks by importance, urgency, and deadlines to focus on high-impact items. Often used as a building block in larger planning routines.
- **Inputs**: Task list with metadata (deadlines, importance, effort estimates)
- **Outputs**: Prioritized task list, scoring rationale, critical path identification
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Reusable component. Uses scoring algorithm combined with AI reasoning for edge cases.

### 10. Weekly Review Assistant
- **Category**: Productivity
- **Description**: Guides weekly reflection and planning session. Aggregates completed tasks, summarizes accomplishments, and helps plan upcoming week.
- **Inputs**: Past week's task data, completed items, pending tasks, reflections
- **Outputs**: Week summary, accomplishment report, next week's priorities, action plan
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses Task Prioritizer for upcoming week planning. Encourages reflection on what worked/didn't.

### 11. Time Block Scheduler
- **Category**: Productivity
- **Description**: Allocates time slots for tasks in calendar format. Respects constraints like meetings and creates realistic schedules.
- **Inputs**: Task list with durations, available time slots, constraints, preferences
- **Outputs**: Time-blocked calendar, schedule visualization, conflict warnings
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Often used within larger planning routines. Handles fixed events and flexible tasks.

### 12. Pomodoro Session Manager
- **Category**: Productivity
- **Description**: Facilitates focused work using Pomodoro technique. Manages 25-minute focus intervals with 5-minute breaks.
- **Inputs**: Task to focus on, session preferences, break activities
- **Outputs**: Timer management, progress tracking, motivational messages, session log
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Can suggest task switches based on progress. Logs completed cycles for productivity tracking.

### 13. Deadline Monitor
- **Category**: Productivity
- **Description**: Tracks approaching deadlines and alerts to upcoming due dates. Can run periodically to check task deadlines.
- **Inputs**: Task list with deadlines, notification preferences, risk thresholds
- **Outputs**: Deadline alerts, risk assessment, escalation recommendations
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Calculates time remaining, uses AI for reminder messages. Future: integrate with calendar/email.

### 14. Note-to-Task Converter
- **Category**: Productivity
- **Description**: Parses unstructured notes or meeting minutes to extract actionable tasks. Identifies to-do items and follow-ups automatically.
- **Inputs**: Raw notes, meeting minutes, conversation transcripts
- **Outputs**: Extracted task list, assigned owners, suggested due dates
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses embeddings and keyword search. May use Key Action Item Extractor subroutine.

### 15. Project Task Organizer
- **Category**: Productivity
- **Description**: Breaks down projects into structured task lists with subtasks and milestones. Creates hierarchy and dependencies.
- **Inputs**: Project description, goals, constraints, timeline
- **Outputs**: Task breakdown structure, milestone plan, dependency graph
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: May call Task Prioritizer and Deadline Monitor as subroutines.

### KNOWLEDGE & SUMMARIZATION (High Priority)

### 16. Document Summarizer
- **Category**: Knowledge
- **Description**: Condenses long documents into concise summaries of key points. Handles very large texts by chunking and merging summaries.
- **Inputs**: Document text, summary length preference, focus areas
- **Outputs**: Structured summary, key points list, important quotes
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses embeddings to ensure all major points covered. Maintains document structure in summary.

### 17. Multi-Source Synthesizer
- **Category**: Knowledge
- **Description**: Gathers information from multiple sources and produces unified summary. Performs searches, summarizes each source, then combines findings.
- **Inputs**: Topic, source list or search queries, synthesis goals
- **Outputs**: Unified report, source citations, insight highlights, discrepancy notes
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: May reuse Document Summarizer. Includes Source Comparison subroutine.

### 18. Research Synthesis Workflow
- **Category**: Knowledge
- **Description**: Comprehensive research routine that gathers information from multiple sources, evaluates credibility, and produces structured reports.
- **Inputs**: Research topic, scope parameters, quality requirements, source preferences
- **Outputs**: Research report, source citations, credibility ratings, key findings
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Includes fact-checking and source verification. Can handle academic or business research.

### 19. Meeting Minutes Summarizer
- **Category**: Knowledge
- **Description**: Transforms meeting transcripts into structured minutes with key points, decisions, and action items.
- **Inputs**: Meeting transcript or notes, attendee list, agenda
- **Outputs**: Structured minutes, decision log, action items with owners
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses Action Item Extractor and Decision Highlighter subroutines.

### 20. Contextual Search Assistant
- **Category**: Knowledge
- **Description**: Enhances queries by searching within specific context or dataset. Like AI-powered search for personal/team knowledge.
- **Inputs**: Natural language query, context dataset, search preferences
- **Outputs**: Relevant excerpts, synthesized answer, source references
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses embeddings search. Translates queries to find best matches in knowledge base.

### CONTENT CREATION (Medium Priority)

### 21. Content Outline Generator
- **Category**: Content
- **Description**: Creates structured outlines for long-form content like articles, reports, or presentations.
- **Inputs**: Topic, purpose, target audience, content type
- **Outputs**: Hierarchical outline, section descriptions, logical flow
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Ensures logical progression and comprehensive coverage. Can be handed off to writing routines.

### 22. Social Media Post Composer
- **Category**: Content
- **Description**: Drafts engaging social media posts from topics or source content. Adapts tone and format per platform.
- **Inputs**: Topic/source content, platform(s), tone preferences, hashtags
- **Outputs**: Platform-specific posts, multiple variations, optimal hashtags
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Future: direct scheduling/publishing via APIs.

### 23. Blog Post Auto-Writer
- **Category**: Content
- **Description**: Assists in creating blog content quickly. Generates outline then expands to full draft with cohesive flow.
- **Inputs**: Topic or title, target audience, style preferences, word count
- **Outputs**: Complete blog draft, SEO suggestions, image recommendations
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses OutlineCreator and DraftWriter subroutines. Can produce 1000+ word articles.

### 24. Email Draft Assistant
- **Category**: Content
- **Description**: Composes draft emails based on key points. Adjusts tone (formal/casual) per requirements.
- **Inputs**: Email purpose, key points, recipient context, tone preference
- **Outputs**: Complete email draft, subject line suggestions
- **Strategy**: Conversational
- **Priority**: High
- **Notes**: Ensures clarity and appropriate tone. Future: auto-send with approval.

### 25. Marketing Slogan Generator
- **Category**: Content
- **Description**: Produces creative taglines, slogans, or value propositions for brands/products/campaigns.
- **Inputs**: Product/brand info, key features, target audience, tone
- **Outputs**: Multiple slogan options, variety of styles, rationale for each
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Includes NameGenerator and SloganCreator subroutines.

### ANALYSIS & DECISION SUPPORT (High Priority)

### 26. Decision Support Framework
- **Category**: Analysis
- **Description**: Structured decision-making that evaluates options, weighs criteria, and provides recommendations with reasoning.
- **Inputs**: Decision context, options list, evaluation criteria, constraints
- **Outputs**: Option analysis, scoring matrix, recommendation, risk assessment
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Includes bias detection and alternative perspective consideration.

### 27. Pros & Cons Evaluator
- **Category**: Analysis
- **Description**: Lists advantages and disadvantages of each option to aid decision-making.
- **Inputs**: Options list, decision context, evaluation perspective
- **Outputs**: Structured pros/cons for each option, impact highlights, missing factors
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Highlights most impactful factors. Suggests additional considerations.

### 28. SWOT Analysis Assistant
- **Category**: Analysis
- **Description**: Performs SWOT analysis for projects, business ideas, or decisions.
- **Inputs**: Entity/situation description, context, time horizon
- **Outputs**: Four-quadrant SWOT matrix, strategic insights, action recommendations
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Provides strategic overview for planning and decision-making.

### 29. Risk Assessment Agent
- **Category**: Analysis
- **Description**: Analyzes risks associated with plans/decisions and suggests mitigation strategies.
- **Inputs**: Plan/decision description, context, risk tolerance
- **Outputs**: Risk inventory, impact assessments, mitigation strategies, contingency plans
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Helps foresee challenges and prepare responses.

### 30. Data Insight Extraction
- **Category**: Analysis
- **Description**: Analyzes datasets to identify patterns, trends, anomalies, and actionable insights.
- **Inputs**: Dataset, analysis objectives, business context
- **Outputs**: Key insights, statistical summary, visualization recommendations, confidence levels
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Handles structured and unstructured data. Suggests appropriate visualizations.

### RESEARCH & INFORMATION GATHERING (High Priority)

### 31. Web Research Agent
- **Category**: Research
- **Description**: Conducts thorough web searches and compiles findings into research briefs.
- **Inputs**: Research query, depth preference, source requirements
- **Outputs**: Research brief, key findings, source links, credibility notes
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses multiple search angles. Summarizes and cites sources properly.

### 32. Competitor Analysis Agent
- **Category**: Research
- **Description**: Gathers intelligence on companies, products, or individuals for competitive research.
- **Inputs**: Competitor list, analysis dimensions, time period
- **Outputs**: Comparative report, strengths/weaknesses, recent developments, feature comparison
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Reuses Web Research Agent. Includes Comparison Compiler subroutine.

### 33. Market Trends Monitor
- **Category**: Research
- **Description**: Analyzes current trends in given industry by aggregating recent news and data.
- **Inputs**: Industry/topic, time window, trend indicators
- **Outputs**: Trend report, emerging themes, opportunities/threats, supporting evidence
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Future: scheduled monitoring with alerts for new trends.

### 34. Fact-Checking Agent
- **Category**: Research
- **Description**: Verifies truth of claims by finding credible sources. Breaks claims into checkable facts.
- **Inputs**: Claim/statement, context, credibility standards
- **Outputs**: Fact check report, evidence for/against, source citations, confidence rating
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Helps validate information and debunk misinformation with proper references.

### 35. Topic Deep-Dive Researcher
- **Category**: Research
- **Description**: Performs in-depth literature review on specific subjects. Searches for high-quality sources and synthesizes findings.
- **Inputs**: Research topic, depth level, source preferences (academic, expert, etc.)
- **Outputs**: Comprehensive report, multiple perspectives, references, knowledge gaps
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Integrates with Multi-Source Synthesizer. Uses Citation Manager subroutine.

### CREATIVE IDEATION (Medium Priority)

### 36. Brainstorm Buddy
- **Category**: Creative
- **Description**: Facilitates brainstorming on any topic. Generates ideas and organizes them into themes.
- **Inputs**: Brainstorm prompt, constraints, idea quantity preference
- **Outputs**: Categorized idea list, theme clusters, expansion suggestions
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Can iterate based on feedback. Uses clustering to group similar ideas.

### 37. Idea Refinement Assistant
- **Category**: Creative
- **Description**: Takes raw ideas and helps elaborate/improve them. Identifies strengths, weaknesses, and enhancement opportunities.
- **Inputs**: Raw idea description, goals, constraints
- **Outputs**: Refined concept, improvement suggestions, potential issues, differentiation strategies
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses WeaknessFinder and EnhancementSuggester subroutines.

### 38. Story & Metaphor Machine
- **Category**: Creative
- **Description**: Creates analogies, metaphors, or stories to explain concepts or entertain.
- **Inputs**: Concept to explain, target audience, style preference
- **Outputs**: Creative metaphors, short stories, relatable analogies
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Useful for presentations and teaching. Makes complex ideas accessible.

### 39. Problem Reframer
- **Category**: Creative
- **Description**: Helps look at problems from new angles through multiple reframings.
- **Inputs**: Problem description, current perspective, constraints
- **Outputs**: Multiple reframings, new questions, metaphorical views, hidden assumptions
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses PerspectiveShifter and AnalogyMaker subroutines. Triggers creative solutions.

### 40. Auto-Marketing MCP
- **Category**: Creative
- **Description**: Automated marketing content creation inspired by Layers approach. Generates comprehensive marketing materials.
- **Inputs**: Product/service info, target audience, campaign goals
- **Outputs**: Ad copy, social posts, email campaigns, landing page content
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Based on https://x.com/hasantoxr/status/1924651551152603266. Creates cohesive multi-channel content.

### COMMUNICATION & WRITING (Medium Priority)

### 41. Report Writer
- **Category**: Communication
- **Description**: Turns structured information into well-written reports with logical flow.
- **Inputs**: Outline/bullet points, data findings, target audience, tone
- **Outputs**: Complete report, executive summary, supporting sections
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: May use Content Outline Generator and Proofreading Helper subroutines.

### 42. Editing & Proofreading Assistant
- **Category**: Communication
- **Description**: Reviews and improves draft texts. Checks grammar, structure, and clarity.
- **Inputs**: Draft text, editing level (light/heavy), style preferences
- **Outputs**: Edited text, change explanations, improvement suggestions
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Can do simple corrections or substantive rewriting.

### 43. Tone & Style Transformer
- **Category**: Communication
- **Description**: Converts text between different tones/styles while preserving meaning.
- **Inputs**: Original text, target tone/style, audience
- **Outputs**: Transformed text, style adjustments applied
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: E.g., formal to friendly, technical to simple, casual to professional.

### 44. Resume and Cover Letter Assistant
- **Category**: Communication
- **Description**: Crafts resume bullets and cover letter paragraphs from raw experience data.
- **Inputs**: Job experience, skills, target role, job description
- **Outputs**: Polished resume statements, cover letter sections, achievement highlights
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses action verbs, quantifies results, ties experience to requirements.

### 45. Video/Podcast Script Writer
- **Category**: Communication
- **Description**: Creates detailed scripts for videos or podcast episodes with proper flow.
- **Inputs**: Topic, target length, format preference, audience
- **Outputs**: Complete script with segments, narration text, cues for pauses/music
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Uses ScriptOutliner and NarrativeExpander subroutines.

### PERSONAL DEVELOPMENT & LIFESTYLE (Medium Priority)

### 46. Habit Tracker & Coach
- **Category**: Personal Development
- **Description**: Tracks daily habits and provides personalized feedback and encouragement.
- **Inputs**: Habit list, daily check-ins, goals, preferences
- **Outputs**: Streak tracking, success statistics, coaching messages, improvement tips
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Adjusts feedback style based on progress patterns.

### 47. Daily Journal Insight
- **Category**: Personal Development
- **Description**: Analyzes journal entries to detect themes, emotions, and provide reflective insights.
- **Inputs**: Journal entry text, mood indicators
- **Outputs**: Theme summary, emotion analysis, reflection questions, gratitude suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Acts as journaling coach for deeper self-reflection.

### 48. Goal Planner & Tracker
- **Category**: Personal Development
- **Description**: Helps set specific goals, break into steps, and track progress with adjustments.
- **Inputs**: Goal description, timeline, constraints, current status
- **Outputs**: Goal breakdown, milestone plan, progress tracking, encouragement
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Time Block Scheduler and Deadline Monitor subroutines.

### 49. Wellness Check-in Bot
- **Category**: Personal Development
- **Description**: Performs guided well-being check-ins with reflective questions and support.
- **Inputs**: Current mood, stressors, recent events
- **Outputs**: Reflection summary, coping strategies, affirmations, resources
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Not a substitute for professional help. Daily wellness companion.

### 50. Integrated Morning Briefing & Planner
- **Category**: Personal Development
- **Description**: Comprehensive morning routine combining wellness check, task planning, and daily briefing.
- **Inputs**: To-do list, calendar, sleep data, priorities
- **Outputs**: Morning briefing report, prioritized schedule, wellness tips, motivational quote
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses BPMN workflow. Combines WellnessCheck, TaskPlanner, and BriefingComposer.

### BUSINESS & PROJECT MANAGEMENT (Medium Priority)

### 51. Project Planner & Task Breakdown
- **Category**: Business
- **Description**: Scopes projects by breaking into phases, milestones, and tasks with timelines.
- **Inputs**: Project goal, constraints, resources, timeline
- **Outputs**: Project plan, phase breakdown, task list, timeline estimates
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses Project Task Organizer and Time Block Scheduler subroutines.

### 52. Meeting Agenda & Minutes Orchestrator
- **Category**: Business
- **Description**: Streamlines meeting prep and follow-up. Generates agendas before, minutes after.
- **Inputs**: Meeting purpose, topics, participants, notes/transcript
- **Outputs**: Structured agenda with time slots, minutes with action items
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Future: email integration for distribution.

### 53. Team Stand-up Synthesizer
- **Category**: Business
- **Description**: Collects team status updates and synthesizes into coherent team report.
- **Inputs**: Individual updates, blockers, plans
- **Outputs**: Team status summary, common themes, shared blockers, progress overview
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Saves managers from manual compilation. Provides unified view.

### 54. Client Proposal Assistant
- **Category**: Business
- **Description**: Drafts business proposals with standard sections and persuasive language.
- **Inputs**: Client needs, solution details, pricing, timeline
- **Outputs**: Polished proposal document, executive summary, value proposition
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses Content Outline Generator and Editing Assistant subroutines.

### 55. Negotiation Strategist
- **Category**: Business
- **Description**: Prepares negotiation strategies including BATNA, talking points, and counter-arguments.
- **Inputs**: Negotiation scenario, objectives, other party info
- **Outputs**: Strategy briefing, BATNA analysis, concession options, response scripts
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses OtherPartySimulator and StrategyFormulator subroutines.

### STUDENT & LEARNING TOOLS (Medium Priority)

### 56. Smart Study Planner
- **Category**: Student
- **Description**: Plans study schedules for exams, allocating time per subject based on difficulty.
- **Inputs**: Exam dates, subjects, difficulty ratings, available study time
- **Outputs**: Study timetable, daily targets, review schedule, tips
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses ScheduleCalculator and PlanGenerator subroutines.

### 57. Lecture/Chapter Summarizer
- **Category**: Student
- **Description**: Summarizes academic texts focusing on key terms, concepts, and examples.
- **Inputs**: Lecture notes or textbook chapter, course context
- **Outputs**: Condensed summary, key terms glossary, main concepts
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses AcademicSummarizer and TermHighlighter subroutines.

### 58. Flashcard & Quiz Generator
- **Category**: Student
- **Description**: Creates study questions and flashcards from source material for active recall.
- **Inputs**: Study material, topic list, question preferences
- **Outputs**: Flashcard Q&A pairs, quiz questions, multiple choice options
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Uses QuestionGenerator and AnswerGenerator subroutines.

### 59. Essay Outline Assistant
- **Category**: Student
- **Description**: Helps organize thoughts into structured essay outlines with logical flow.
- **Inputs**: Essay prompt, thesis idea, key points
- **Outputs**: Essay outline with sections, topic sentences, evidence placeholders
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses ThesisGenerator and OutlineBuilder subroutines.

### 60. Writing Feedback Assistant
- **Category**: Student
- **Description**: Reviews student writing and provides multi-level feedback for improvement.
- **Inputs**: Draft text, assignment requirements, rubric
- **Outputs**: Feedback on structure, clarity, grammar, suggestions, revised sentences
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses GrammarCorrector and ContentCritique subroutines.

### FINANCIAL & SHOPPING ASSISTANCE (Lower Priority - Can Wait)

### 61. Expense Tracker & Analyzer
- **Category**: Financial
- **Description**: Helps users log and analyze expenses. Categorizes transactions and provides spending insights.
- **Inputs**: Expense list or statement, budget goals, time period
- **Outputs**: Categorized expenses, spending summary, budget analysis, alerts
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses code to categorize by merchant/keywords. AI provides spending advice.

### 62. Budget Planner
- **Category**: Financial
- **Description**: Creates and adjusts personal/household budgets based on income and goals.
- **Inputs**: Income sources, expenses, savings goals, priorities
- **Outputs**: Budget allocations, balance sheet, scenario simulations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses frameworks like 50/30/20 rule. Simulates what-if scenarios.

### 63. Investment Research Assistant
- **Category**: Financial
- **Description**: Gathers and summarizes investment information from public sources.
- **Inputs**: Asset names/tickers, portfolio, research focus
- **Outputs**: Market summary, news highlights, trend analysis, general sentiment
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Future: integrate with financial APIs for real-time data.

### 64. Price Comparison Bot
- **Category**: Financial
- **Description**: Compares prices for products across multiple online sources.
- **Inputs**: Product name/specifications, preferences
- **Outputs**: Price list by vendor, best deals, shipping/return notes
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses web search. Future: shopping API integration.

### 65. Subscription Manager
- **Category**: Financial
- **Description**: Tracks subscriptions and recurring bills, identifies savings opportunities.
- **Inputs**: Subscription list, usage data, budget constraints
- **Outputs**: Renewal calendar, cost analysis, cancellation recommendations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Highlights low-usage subscriptions. Schedules renewal reminders.

### DATA ANALYSIS & VISUALIZATION (Lower Priority - Can Wait)

### 66. Data Summarizer & Visualizer
- **Category**: Data
- **Description**: Provides quick analysis and visualization of datasets.
- **Inputs**: Dataset (CSV/table), analysis goals, visualization preferences
- **Outputs**: Statistical summary, charts/graphs, written insights
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses pandas/matplotlib in sandbox. AI explains key findings.

### 67. Correlation Finder
- **Category**: Data
- **Description**: Analyzes relationships between variables in datasets.
- **Inputs**: Dataset, variable pairs, analysis type
- **Outputs**: Correlation coefficients, regression results, plain language explanation
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Warns about spurious correlations and causation vs correlation.

### 68. Anomaly Detector
- **Category**: Data
- **Description**: Scans data to find outliers and unusual patterns.
- **Inputs**: Time-series or categorical data, sensitivity thresholds
- **Outputs**: Anomaly list, potential explanations, visualization
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses statistical methods. AI suggests possible causes.

### 69. Survey Results Analyzer
- **Category**: Data
- **Description**: Extracts insights from survey data including quantitative and text responses.
- **Inputs**: Survey response data, question types
- **Outputs**: Statistical summary, theme analysis, key takeaways report
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Handles mixed data types. Clusters open-ended responses.

### 70. Content Categorizer
- **Category**: Data
- **Description**: Automatically classifies documents/content into meaningful categories.
- **Inputs**: Document collection, category preferences
- **Outputs**: Category assignments, theme descriptions, similarity clusters
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses embeddings for similarity. AI names categories descriptively.

### TECHNICAL & DEVELOPER SUPPORT (Lower Priority - Can Wait)

### 71. Code Generator & Tester
- **Category**: Technical
- **Description**: Generates code from specifications and validates with tests.
- **Inputs**: Function specification, language, test cases
- **Outputs**: Generated code, test results, self-debugging attempts
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Runs code in sandbox. Attempts fixes if tests fail.

### 72. Bug Finder & Fixer
- **Category**: Technical
- **Description**: Helps locate and fix bugs in source code.
- **Inputs**: Code with bug, error messages, expected behavior
- **Outputs**: Bug diagnosis, fix suggestions, corrected code
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses static analysis and AI reasoning. Can apply and test fixes.

### 73. Documentation Summarizer
- **Category**: Technical
- **Description**: Summarizes API docs, technical manuals, or libraries for quick understanding.
- **Inputs**: Documentation text/link, focus areas
- **Outputs**: Key functions/classes, usage examples, quick reference
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Extracts most important parts. Creates concise developer guide.

### 74. Regex Generator & Tester
- **Category**: Technical
- **Description**: Creates regular expressions from requirements and tests them.
- **Inputs**: Pattern description, example strings, edge cases
- **Outputs**: Regex pattern, test results, pattern explanation
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Tests against provided cases. Explains regex components.

### 75. Tech Stack Recommender
- **Category**: Technical
- **Description**: Suggests technology choices for projects based on requirements.
- **Inputs**: Project type, constraints, preferences, scale
- **Outputs**: Stack recommendations with rationale, alternatives, trade-offs
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Covers all layers: language, framework, database, hosting, tools.

### ADDITIONAL HIGH-VALUE ROUTINES

### 76. Recurring Task Scheduler
- **Category**: Productivity
- **Description**: Manages repetitive tasks by generating future occurrences and reminders.
- **Inputs**: Task definition, recurrence pattern, start date, end conditions
- **Outputs**: Scheduled occurrences, reminder calendar, contextual tips
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Handles daily/weekly/monthly patterns. AI adds context per occurrence.

### 77. Priority Matrix (Eisenhower Assistant)
- **Category**: Productivity
- **Description**: Sorts tasks into urgent/important matrix for better prioritization.
- **Inputs**: Task list with urgency/importance data, deadlines
- **Outputs**: 2x2 matrix categorization, action recommendations per quadrant
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Do First, Schedule, Delegate, Eliminate quadrants. Helps focus on what matters.

### 78. What-If Scenario Planner
- **Category**: Analysis
- **Description**: Explores hypothetical scenarios and their potential outcomes.
- **Inputs**: Scenario description, variables, constraints
- **Outputs**: Outcome predictions, benefits/drawbacks, similar case examples
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helps evaluate decisions by visualizing possible futures.

### 79. Language Translator & Learning Aid
- **Category**: Student
- **Description**: Translates text while teaching grammar and vocabulary.
- **Inputs**: Source text, source/target languages, learning level
- **Outputs**: Translation, grammar explanations, vocabulary notes, usage examples
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Goes beyond translation to teach. Uses Translator and LinguisticAnalyzer subroutines.

### 80. Learning Plan Generator
- **Category**: Learning
- **Description**: Creates personalized learning curriculum for new skills or subjects.
- **Inputs**: Learning goal, time availability, learning style, current knowledge
- **Outputs**: Structured curriculum, resource list, milestones, practice exercises
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Mixes different resource types. Adapts based on progress.

### 81. Background Briefing Assistant
- **Category**: Research
- **Description**: Prepares quick background briefs on people or organizations.
- **Inputs**: Name/entity, context (meeting, interview, etc.), focus areas
- **Outputs**: Background summary, recent news, key points, conversation starters
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses only publicly available information. Helps with meeting prep.

### 82. Expert Advice Aggregator
- **Category**: Research
- **Description**: Collects and synthesizes expert advice on specific questions.
- **Inputs**: Question/topic, expertise level needed, source preferences
- **Outputs**: Curated advice summary, common recommendations, unique insights
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Searches expert articles, interviews, forums. Highlights consensus vs unique views.

### 83. SEO Content Optimizer
- **Category**: Content
- **Description**: Refines content for search engine optimization without losing quality.
- **Inputs**: Content text, target keywords, SEO goals
- **Outputs**: Optimized content, keyword density analysis, meta suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Uses KeywordAnalyzer and SEORewriter subroutines. Maintains readability.

### 84. Content Repurposer
- **Category**: Content
- **Description**: Transforms single content piece into multiple formats for different platforms.
- **Inputs**: Source content, target formats, platform requirements
- **Outputs**: Content kit with social posts, email blurbs, summaries, etc.
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Maximizes content ROI. Uses SummaryGenerator and FormatAdapter subroutines.

### 85. Meal Planner
- **Category**: Personal Development
- **Description**: Suggests meal plans based on dietary preferences and goals.
- **Inputs**: Dietary preferences, ingredients on hand, calorie targets, schedule
- **Outputs**: Weekly meal plan, recipes, grocery list, nutrition estimates
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses Recipe Finder and Nutrition Estimator subroutines.

### 86. Workout Routine Builder
- **Category**: Personal Development
- **Description**: Creates personalized workout plans based on fitness goals and equipment.
- **Inputs**: Fitness goals, equipment available, schedule, experience level
- **Outputs**: Weekly workout schedule, exercise descriptions, progression plan
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Balances muscle groups, includes rest days, provides progression guidance.

### 87. Travel Itinerary Planner
- **Category**: Personal Development
- **Description**: Generates travel itineraries with daily activities and logistics.
- **Inputs**: Destination, trip length, interests, budget
- **Outputs**: Day-by-day itinerary, attraction list, travel tips, time estimates
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Balances activities with downtime. Clusters by location for efficiency.

### 88. Customer Feedback Summarizer
- **Category**: Business
- **Description**: Analyzes customer feedback to extract key insights and trends.
- **Inputs**: Feedback data (reviews, surveys, support tickets)
- **Outputs**: Theme summary, sentiment analysis, improvement priorities
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Identifies praise points and pain points. Quantifies sentiment percentages.

### 89. Performance Metrics Reporter
- **Category**: Business
- **Description**: Turns raw performance data into narrative reports.
- **Inputs**: Performance data, targets, time period, audience
- **Outputs**: Narrative report, key metrics, trend analysis, recommendations
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Uses Data Analyzer and Narrative Generator subroutines.

### 90. Opportunity Evaluator (ROI & Fit Analysis)
- **Category**: Business
- **Description**: Evaluates opportunities with both financial and strategic analysis.
- **Inputs**: Opportunity details, costs/benefits, strategic goals
- **Outputs**: ROI calculations, strategic fit analysis, pros/cons, recommendations
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses FinancialCalculator and FeasibilityAdvisor subroutines.

### MISSING ROUTINES FROM ORIGINAL LIST

### Knowledge & Summarization (Additional)

### 91. Email Thread Condenser
- **Category**: Knowledge
- **Description**: Summarizes long email threads or chat conversations to bring someone up to speed quickly.
- **Inputs**: Email thread or chat history, key participants
- **Outputs**: Timeline summary, main points, decisions made, next steps
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Preserves chronological flow and key decisions.

### 92. Note Summarizer
- **Category**: Knowledge
- **Description**: Cleans up and summarizes informal notes from brainstorming sessions or lectures.
- **Inputs**: Raw notes, context (meeting type, subject)
- **Outputs**: Organized summary, key themes, suggested follow-ups
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Can integrate with Note-to-Task Converter for action items.

### 93. Glossary Generator
- **Category**: Knowledge
- **Description**: Extracts important terms or jargon from documents and provides definitions.
- **Inputs**: Document text, domain/field context
- **Outputs**: Term list with definitions, usage examples
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Scans for capitalized terms, frequent jargon. Can retrieve definitions via search.

### 94. Highlight Extractor
- **Category**: Knowledge
- **Description**: Identifies the most insightful or relevant sentences from documents.
- **Inputs**: Document(s), relevance criteria, keyword preferences
- **Outputs**: Top highlights list, context for each, importance scores
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses heuristic scoring and AI to find key insights.

### Creative Ideation (Additional)

### 95. Design Brainstormer
- **Category**: Creative
- **Description**: Generates design or feature ideas for products and projects.
- **Inputs**: Design problem, product concept, constraints
- **Outputs**: Feature suggestions, style ideas, innovation concepts
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Outputs text descriptions of design concepts for prototyping.

### 96. Title and Headline Generator
- **Category**: Creative
- **Description**: Suggests attention-grabbing titles for various content types.
- **Inputs**: Content summary, keywords, target audience, medium
- **Outputs**: Multiple title options, engagement predictions, rationale
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Leverages engagement principles for different platforms.

### 97. Question Stormer
- **Category**: Creative
- **Description**: Generates insightful or probing questions on a given topic.
- **Inputs**: Topic, research goals, audience level
- **Outputs**: Question list by category, exploration angles, discussion prompts
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps with FAQ creation, interview prep, deeper thinking.

### 98. Campaign Idea Factory
- **Category**: Creative
- **Description**: Generates creative marketing campaign concepts.
- **Inputs**: Product/service, campaign goals, target audience, budget level
- **Outputs**: Campaign themes, channel strategies, taglines, tactics
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Provides multiple concepts for further development.

### 99. Character & Dialogue Generator
- **Category**: Creative
- **Description**: Creates fictional characters and dialogue for creative projects.
- **Inputs**: Character requirements, story context, personality traits
- **Outputs**: Character profiles, dialogue samples, interaction scenarios
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Useful for writing, role-play training, simulations.

### Communication & Writing (Additional)

### 100. Follow-up Email Generator
- **Category**: Communication
- **Description**: Prepares professional follow-up messages after meetings or interactions.
- **Inputs**: Meeting notes, key outcomes, commitments, recipient context
- **Outputs**: Email draft, subject line, next steps summary
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Ensures consistency and professionalism in follow-ups.

### 101. Language Translator & Summarizer
- **Category**: Communication
- **Description**: Translates text between languages and optionally summarizes.
- **Inputs**: Source text, source/target languages, summary preference
- **Outputs**: Translation, optional summary, key points extraction
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Future: integrate specialized translation APIs for accuracy.

### 102. Presentation Slide Condenser
- **Category**: Communication
- **Description**: Converts lengthy text into concise presentation bullets.
- **Inputs**: Source text, slide count limit, presentation style
- **Outputs**: Bullet points per slide, suggested titles, speaker notes
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helps quickly create presentations from written material.

### 103. Meeting Agenda Emailer
- **Category**: Communication
- **Description**: Generates agenda emails with discussion points and time allocations.
- **Inputs**: Meeting topics, goals, participants, duration
- **Outputs**: Formatted agenda email, time blocks, pre-read suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Standardizes meeting communication.

### 104. Public Relations Statement Assistant
- **Category**: Communication
- **Description**: Drafts public-facing statements or press releases.
- **Inputs**: Key facts, desired tone, audience, message goals
- **Outputs**: Statement draft, key messages, Q&A prep
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Focuses on clarity, tone, and comprehensive coverage.

### 105. Social Media Content Batch
- **Category**: Communication
- **Description**: Generates content for multiple social platforms from one theme.
- **Inputs**: Theme/campaign, platform list, tone, hashtag preferences
- **Outputs**: Platform-specific posts, hashtags, posting schedule
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses PostIdeasGenerator and MultiPlatformFormatter subroutines.

### Personal Development (Additional)

### 106. Gift Recommendation Assistant
- **Category**: Personal Development
- **Description**: Suggests thoughtful gift ideas based on recipient profile.
- **Inputs**: Recipient info (age, interests), occasion, budget
- **Outputs**: Gift ideas list, rationale for each, where to buy
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses AI and web search for current gift guides.

### 107. Home Maintenance Scheduler
- **Category**: Personal Development
- **Description**: Tracks and reminds about routine home maintenance tasks.
- **Inputs**: Home type, custom tasks, maintenance history
- **Outputs**: Annual schedule, task reminders, importance explanations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Future: notification integration for reminders.

### 108. Sleep & Recovery Analyzer
- **Category**: Personal Development
- **Description**: Analyzes sleep patterns and provides optimization advice.
- **Inputs**: Sleep/wake times, quality ratings, lifestyle factors
- **Outputs**: Sleep trends, deficit analysis, improvement suggestions
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Tracks patterns over time for personalized recommendations.

### 109. Stress Relief Break Coach
- **Category**: Personal Development
- **Description**: Suggests stress-relief activities during work breaks.
- **Inputs**: Stress level, time available, preferences
- **Outputs**: Activity suggestions, guided exercises, timing
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Includes breathing exercises, stretches, mindfulness prompts.

### Business & Project Management (Additional)

### 110. Invoice & Expense Processor
- **Category**: Business
- **Description**: Extracts and records financial data from invoices/receipts.
- **Inputs**: Invoice text/image, expense categories
- **Outputs**: Structured data, accounting entries, summaries
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Future: accounting system integration.

### 111. Policy/Document Summarizer
- **Category**: Business
- **Description**: Creates executive summaries of long policy or technical documents.
- **Inputs**: Full document, target audience, focus areas
- **Outputs**: Executive summary, key points by section, action items
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Document Summarizer with Context Filter subroutine.

### 112. Resource Allocation Advisor
- **Category**: Business
- **Description**: Helps distribute resources across projects optimally.
- **Inputs**: Project list, resource constraints, priorities, deadlines
- **Outputs**: Allocation plan, scenario comparisons, rationale
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Models different scenarios with trade-offs.

### 113. Strategic Plan Generator
- **Category**: Business
- **Description**: Assists in drafting strategic plans and OKRs.
- **Inputs**: High-level goals, challenges, timeframe
- **Outputs**: Strategic objectives, key results, initiatives, SMART goals
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Ensures objectives are measurable and aligned.

### Data Analysis (Additional)

### 114. Data Cleaner
- **Category**: Data
- **Description**: Cleans and preprocesses raw data for analysis.
- **Inputs**: Dataset with issues, cleaning preferences
- **Outputs**: Cleaned dataset, issue report, transformation log
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Handles missing values, duplicates, format standardization.

### 115. A/B Test Evaluator
- **Category**: Data
- **Description**: Analyzes A/B test results for statistical significance.
- **Inputs**: Variant data, success metrics, sample sizes
- **Outputs**: Statistical analysis, winner recommendation, confidence levels
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses appropriate statistical tests, explains results clearly.

### 116. Excel-to-Database Importer
- **Category**: Data
- **Description**: Converts spreadsheet data into queryable database format.
- **Inputs**: Excel/CSV file, schema preferences
- **Outputs**: Database-ready data, SQL access, query interface
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Enables SQL queries on spreadsheet data.

### 117. Chart Generator
- **Category**: Data
- **Description**: Creates visualizations from data on demand.
- **Inputs**: Data, chart type preference, styling options
- **Outputs**: Chart image, caption, data insights
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Reusable subroutine for other analysis routines.

### 118. Content Clusterer
- **Category**: Data
- **Description**: Organizes documents into thematic clusters.
- **Inputs**: Document collection, clustering preferences
- **Outputs**: Cluster groups, theme labels, similarity scores
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses embeddings and clustering algorithms.

### Technical & Developer (Additional)

### 119. StackOverflow Q&A Assistant
- **Category**: Technical
- **Description**: Finds programming solutions from Stack Overflow and forums.
- **Inputs**: Coding issue description, language/framework
- **Outputs**: Solution summary, code snippets, source links
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Summarizes consensus answers with citations.

### 120. Dev Environment Troubleshooter
- **Category**: Technical
- **Description**: Diagnoses build errors and configuration issues.
- **Inputs**: Error messages, environment details, symptoms
- **Outputs**: Likely causes, step-by-step solutions, diagnostic commands
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Guides through common setup and dependency issues.

### 121. Performance Optimizer
- **Category**: Technical
- **Description**: Analyzes code for performance improvements.
- **Inputs**: Code, performance metrics, constraints
- **Outputs**: Bottleneck analysis, optimization suggestions, benchmarks
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Identifies algorithmic improvements and efficiency gains.

### 122. API Integration Planner
- **Category**: Technical
- **Description**: Outlines steps for third-party API integration.
- **Inputs**: API documentation, integration goals
- **Outputs**: Integration plan, auth setup, code examples, error handling
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Provides implementation roadmap without actual API calls.

### 123. Git Workflow Assistant
- **Category**: Technical
- **Description**: Guides through Git version control tasks and issues.
- **Inputs**: Git scenario, error messages, repository state
- **Outputs**: Command sequences, best practices, issue resolution
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps with merges, conflicts, history management.

### Final Missing Routines

### 124. Email Triage Assistant
- **Category**: Productivity
- **Description**: Assists in quickly processing email inboxes by classifying and summarizing messages.
- **Inputs**: Email list or text, classification preferences
- **Outputs**: Categorized emails (urgent, FYI, tasks), summaries, extracted tasks
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Future: email API integration. Currently requires manual email text input.

### 125. Local Information Scout
- **Category**: Research
- **Description**: Finds location-specific information like restaurants, services, or events.
- **Inputs**: Location, search criteria (restaurants, services, etc.), preferences
- **Outputs**: Recommendations list, ratings/reviews, contact details, tips
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses web search until local APIs become available. Parses review sites and maps results.

### 126. Trade-off Analyzer
- **Category**: Analysis
- **Description**: Analyzes trade-offs between choices that have both upsides and downsides.
- **Inputs**: Options list, key factors, priorities
- **Outputs**: Trade-off matrix, gain vs. loss analysis, recommendation
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helpful for nuanced decisions where there's no clear winner.

### 127. Delegate-or-Automate Recommender
- **Category**: Productivity
- **Description**: Reviews tasks and recommends which to delegate or automate.
- **Inputs**: Task list, team capabilities, automation options
- **Outputs**: Delegation recommendations, automation opportunities, reasoning
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Considers task complexity, frequency, and expertise requirements.

### 128. Product Comparison Researcher
- **Category**: Research
- **Description**: Compares multiple products or options before decision-making.
- **Inputs**: Product list, comparison criteria, budget constraints
- **Outputs**: Comparison chart, pros/cons analysis, recommendation
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Web Research Agent and Tabular Formatter subroutines.

### 129. News Digest Summarizer
- **Category**: Research
- **Description**: Compiles daily/weekly news digests on topics of interest.
- **Inputs**: Topics list, time period, source preferences
- **Outputs**: Categorized news summary, key developments, trend highlights
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps stay informed without reading multiple articles.

### 130. Coupon & Deal Finder
- **Category**: Financial
- **Description**: Searches for discount codes and ongoing sales for products/stores.
- **Inputs**: Store/product name, purchase timeline
- **Outputs**: Valid coupon codes, sale notifications, savings estimates
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Filters expired deals. Future: direct coupon database integration.

### 131. Financial Goal Tracker
- **Category**: Financial
- **Description**: Guides users in setting and monitoring financial goals.
- **Inputs**: Goal details, current status, contribution schedule
- **Outputs**: Progress tracking, timeline projections, tips for acceleration
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Provides encouragement and calculates impact of contribution changes.

### 132. Loan Calculator & Advisor
- **Category**: Financial
- **Description**: Calculates loan payments and provides guidance on loan decisions.
- **Inputs**: Loan parameters, scenarios to compare
- **Outputs**: Payment calculations, total interest, advice on terms
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Explains loan concepts and trade-offs in plain language.

### 133. Tax Document Organizer
- **Category**: Financial
- **Description**: Assists in organizing information for tax filing.
- **Inputs**: Income/expense data, deduction items
- **Outputs**: Tax-ready categorization, document checklist, preparation tips
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: For preparation only, not official tax advice.

### 134. Financial Report Summarizer
- **Category**: Financial
- **Description**: Summarizes financial statements and reports for easier understanding.
- **Inputs**: Financial report, focus areas
- **Outputs**: Key figures summary, trend analysis, plain-language insights
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Useful for both personal statements and company reports.

### THREE FINAL MISSING ROUTINES

### 135. Decision Matrix Maker
- **Category**: Analysis
- **Description**: Creates a weighted decision matrix to quantitatively compare options against multiple criteria.
- **Inputs**: Options list, criteria with weights, scoring preferences
- **Outputs**: Decision matrix, weighted scores, top recommendation, rationale
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Uses Scoring Engine and Matrix Formatter subroutines. Provides quantitative decision support.

### 136. Goal Alignment Checker
- **Category**: Analysis
- **Description**: Reviews tasks or initiatives against stated high-level goals or OKRs.
- **Inputs**: Goal/OKR description, current activities list, priority weights
- **Outputs**: Alignment analysis, misaligned items, prioritization recommendations
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Ensures effort aligns with strategic objectives. Flags activities to prioritize or drop.

### 137. Keyword Extractor for Text
- **Category**: Data
- **Description**: Identifies the most important keywords or phrases in text datasets.
- **Inputs**: Text documents, extraction method preferences, topic focus
- **Outputs**: Keyword list with importance scores, topic groupings, context suggestions
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses TF-IDF or similar algorithms. Useful for content analysis and search indexing.

## Processing Instructions

When processing backlog items:
1. Review the item description and requirements
2. Determine if it should be Sequential or BPMN based on complexity
3. Plan the step-by-step workflow
4. Design appropriate input/output forms
5. Select actual subroutine IDs from available-subroutines.txt if available
6. Create complete routine JSON with proper data flow
7. Save to `staged/` directory with descriptive filename
8. Mark item as processed and move to completed section

## Completed Items

Items that have been processed and moved to staged/ will be listed here with their filename references.

### Completed on 2024-01-27

#### 1. Yes-Man Avoidance (COMPLETED)
- **Generated File**: `staged/yes-man-avoidance.json`
- **Type**: Multi-step routine
- **Strategy**: Reasoning
- **Notes**: Detects social pressure tactics and generates balanced responses

#### 8. Daily Agenda Planner (COMPLETED)
- **Generated File**: `staged/daily-agenda-planner.json`
- **Type**: Multi-step routine with 4 steps
- **Strategy**: Mixed (deterministic for scheduling, conversational for recommendations)
- **Notes**: Creates time-blocked schedules accounting for priorities, energy, and constraints

#### 16. Document Summarizer (COMPLETED)
- **Generated File**: `staged/subroutines/document-summarizer.json`
- **Type**: Single-step generate routine (subroutine)
- **Strategy**: Reasoning
- **Notes**: Reusable subroutine for condensing documents while preserving key information

#### 26. Decision Support Framework (COMPLETED)
- **Generated File**: `staged/decision-support-framework.json`
- **Type**: Multi-step routine with 4 steps
- **Strategy**: Reasoning
- **Notes**: Systematic decision analysis with bias detection and risk assessment

#### 9. Task Prioritizer (COMPLETED)
- **Generated File**: `staged/subroutines/task-prioritizer.json`
- **Type**: Single-step code routine (subroutine)
- **Strategy**: Deterministic
- **Notes**: Reusable component for sorting tasks by multiple factors using scoring algorithm

#### 2. Introspective Self-Review (COMPLETED)
- **Generated File**: `staged/introspective-self-review.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Mixed (reasoning and deterministic)
- **Notes**: AI agent self-reflection with performance analysis and knowledge base updates

#### 3. Goal Alignment & Progress Checkpoint (COMPLETED)
- **Generated File**: `staged/goal-alignment-checkpoint.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Mixed (reasoning and deterministic)
- **Notes**: Reviews progress against OKRs and triggers course corrections

#### 31. Web Research Agent (COMPLETED)
- **Generated File**: `staged/subroutines/web-research-agent.json`
- **Type**: Single-step web routine (subroutine)
- **Strategy**: Reasoning
- **Notes**: Comprehensive web search with multiple angles and source credibility

#### 34. Fact-Checking Agent (COMPLETED)
- **Generated File**: `staged/fact-checking-agent.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Reasoning
- **Notes**: Decomposes claims, searches evidence, evaluates credibility, and generates reports

#### 4. Capability Gap Analysis (COMPLETED)
- **Generated File**: `staged/capability-gap-analysis.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Mixed (reasoning and deterministic)
- **Notes**: Identifies knowledge gaps and creates remediation plans

#### 6. Dynamic Task Allocation (COMPLETED)
- **Generated File**: `staged/dynamic-task-allocation.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Mixed (reasoning, deterministic, conversational)
- **Notes**: Optimally assigns tasks to agents based on skills and workload

#### 13. Deadline Monitor (COMPLETED)
- **Generated File**: `staged/subroutines/deadline-monitor.json`
- **Type**: Single-step code routine (subroutine)
- **Strategy**: Deterministic
- **Notes**: Tracks deadlines and generates risk-based alerts

#### 18. Research Synthesis Workflow (COMPLETED)
- **Generated File**: `staged/research-synthesis-workflow.json`
- **Type**: Multi-step routine with 6 steps
- **Strategy**: Reasoning
- **Notes**: Comprehensive research with credibility assessment and synthesis

#### 29. Risk Assessment Agent (COMPLETED)
- **Generated File**: `staged/risk-assessment-agent.json`
- **Type**: Multi-step routine with 6 steps
- **Strategy**: Mixed (reasoning and deterministic)
- **Notes**: Full risk analysis with mitigation and contingency planning

#### 15. Project Task Organizer (COMPLETED)
- **Generated File**: `staged/project-task-organizer.json`
- **Type**: Multi-step routine with 6 steps
- **Strategy**: Mixed (reasoning, deterministic, conversational)
- **Notes**: Creates work breakdown structures with dependencies and milestones

#### 10. Weekly Review Assistant (COMPLETED)
- **Generated File**: `staged/weekly-review-assistant.json`
- **Type**: Multi-step routine with 6 steps
- **Strategy**: Conversational with mixed subroutines
- **Notes**: Aggregates week data, analyzes accomplishments, extracts lessons, plans next week

#### 17. Multi-Source Synthesizer (COMPLETED)
- **Generated File**: `staged/multi-source-synthesizer.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Reasoning
- **Notes**: Gathers from multiple sources, compares findings, synthesizes unified report

#### 24. Email Draft Assistant (COMPLETED)
- **Generated File**: `staged/subroutines/email-draft-assistant.json`
- **Type**: Single-step generate routine (subroutine)
- **Strategy**: Conversational
- **Notes**: Composes professional emails with appropriate tone and structure

#### 27. Pros & Cons Evaluator (COMPLETED)
- **Generated File**: `staged/pros-cons-evaluator.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Mixed (reasoning and deterministic)
- **Notes**: Structured analysis with weighted factors and recommendations

#### 19. Meeting Minutes Summarizer (COMPLETED)
- **Generated File**: `staged/meeting-minutes-summarizer.json`
- **Type**: Multi-step routine with 5 steps
- **Strategy**: Conversational with reasoning subroutines
- **Notes**: Extracts key points, decisions, and action items from meeting transcripts

#### 12. Pomodoro Session Manager (COMPLETED)
- **Generated File**: `staged/subroutines/pomodoro-session-manager.json`
- **Type**: Single-step generate routine (subroutine)
- **Strategy**: Conversational
- **Notes**: Manages focus sessions with motivational guidance and break suggestions

<!-- Processed items will be moved here -->