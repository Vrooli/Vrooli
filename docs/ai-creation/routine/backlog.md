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

### 1. Contextual Search Assistant
- **Category**: Knowledge
- **Description**: Enhances queries by searching within specific context or dataset. Like AI-powered search for personal/team knowledge.
- **Inputs**: Natural language query, context dataset, search preferences
- **Outputs**: Relevant excerpts, synthesized answer, source references
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses embeddings search. Translates queries to find best matches in knowledge base.

### 2. Content Outline Generator
- **Category**: Content
- **Description**: Creates structured outlines for long-form content like articles, reports, or presentations.
- **Inputs**: Topic, purpose, target audience, content type
- **Outputs**: Hierarchical outline, section descriptions, logical flow
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Ensures logical progression and comprehensive coverage. Can be handed off to writing routines.

### 3. Blog Post Auto-Writer
- **Category**: Content
- **Description**: Assists in creating blog content quickly. Generates outline then expands to full draft with cohesive flow.
- **Inputs**: Topic or title, target audience, style preferences, word count
- **Outputs**: Complete blog draft, SEO suggestions, image recommendations
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses OutlineCreator and DraftWriter subroutines. Can produce 1000+ word articles.

### 4. Marketing Slogan Generator
- **Category**: Content
- **Description**: Produces creative taglines, slogans, or value propositions for brands/products/campaigns.
- **Inputs**: Product/brand info, key features, target audience, tone
- **Outputs**: Multiple slogan options, variety of styles, rationale for each
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Includes NameGenerator and SloganCreator subroutines.

### 5. SWOT Analysis Assistant
- **Category**: Analysis
- **Description**: Performs SWOT analysis for projects, business ideas, or decisions.
- **Inputs**: Entity/situation description, context, time horizon
- **Outputs**: Four-quadrant SWOT matrix, strategic insights, action recommendations
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Provides strategic overview for planning and decision-making.

### 6. Data Insight Extraction
- **Category**: Analysis
- **Description**: Analyzes datasets to identify patterns, trends, anomalies, and actionable insights.
- **Inputs**: Dataset, analysis objectives, business context
- **Outputs**: Key insights, statistical summary, visualization recommendations, confidence levels
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Handles structured and unstructured data. Suggests appropriate visualizations.

### 7. Web Research Agent
- **Category**: Research
- **Description**: Conducts thorough web searches and compiles findings into research briefs.
- **Inputs**: Research query, depth preference, source requirements
- **Outputs**: Research brief, key findings, source links, credibility notes
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses multiple search angles. Summarizes and cites sources properly.

### 8. Competitor Analysis Agent
- **Category**: Research
- **Description**: Gathers intelligence on companies, products, or individuals for competitive research.
- **Inputs**: Competitor list, analysis dimensions, time period
- **Outputs**: Comparative report, strengths/weaknesses, recent developments, feature comparison
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Reuses Web Research Agent. Includes Comparison Compiler subroutine.

### 9. Market Trends Monitor
- **Category**: Research
- **Description**: Analyzes current trends in given industry by aggregating recent news and data.
- **Inputs**: Industry/topic, time window, trend indicators
- **Outputs**: Trend report, emerging themes, opportunities/threats, supporting evidence
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Future: scheduled monitoring with alerts for new trends.

### 10. Topic Deep-Dive Researcher
- **Category**: Research
- **Description**: Performs in-depth literature review on specific subjects. Searches for high-quality sources and synthesizes findings.
- **Inputs**: Research topic, depth level, source preferences (academic, expert, etc.)
- **Outputs**: Comprehensive report, multiple perspectives, references, knowledge gaps
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Integrates with Multi-Source Synthesizer. Uses Citation Manager subroutine.

### 11. Idea Refinement Assistant
- **Category**: Creative
- **Description**: Takes raw ideas and helps elaborate/improve them. Identifies strengths, weaknesses, and enhancement opportunities.
- **Inputs**: Raw idea description, goals, constraints
- **Outputs**: Refined concept, improvement suggestions, potential issues, differentiation strategies
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses WeaknessFinder and EnhancementSuggester subroutines.

### 12. Story & Metaphor Machine
- **Category**: Creative
- **Description**: Creates analogies, metaphors, or stories to explain concepts or entertain.
- **Inputs**: Concept to explain, target audience, style preference
- **Outputs**: Creative metaphors, short stories, relatable analogies
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Useful for presentations and teaching. Makes complex ideas accessible.

### 13. Problem Reframer
- **Category**: Creative
- **Description**: Helps look at problems from new angles through multiple reframings.
- **Inputs**: Problem description, current perspective, constraints
- **Outputs**: Multiple reframings, new questions, metaphorical views, hidden assumptions
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses PerspectiveShifter and AnalogyMaker subroutines. Triggers creative solutions.

### 14. Auto-Marketing MCP
- **Category**: Creative
- **Description**: Automated marketing content creation inspired by Layers approach. Generates comprehensive marketing materials.
- **Inputs**: Product/service info, target audience, campaign goals
- **Outputs**: Ad copy, social posts, email campaigns, landing page content
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Based on https://x.com/hasantoxr/status/1924651551152603266. Creates cohesive multi-channel content.

### 15. Report Writer
- **Category**: Communication
- **Description**: Turns structured information into well-written reports with logical flow.
- **Inputs**: Outline/bullet points, data findings, target audience, tone
- **Outputs**: Complete report, executive summary, supporting sections
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: May use Content Outline Generator and Proofreading Helper subroutines.

### 16. Editing & Proofreading Assistant
- **Category**: Communication
- **Description**: Reviews and improves draft texts. Checks grammar, structure, and clarity.
- **Inputs**: Draft text, editing level (light/heavy), style preferences
- **Outputs**: Edited text, change explanations, improvement suggestions
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Can do simple corrections or substantive rewriting.

### 17. Tone & Style Transformer
- **Category**: Communication
- **Description**: Converts text between different tones/styles while preserving meaning.
- **Inputs**: Original text, target tone/style, audience
- **Outputs**: Transformed text, style adjustments applied
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: E.g., formal to friendly, technical to simple, casual to professional.

### 18. Resume and Cover Letter Assistant
- **Category**: Communication
- **Description**: Crafts resume bullets and cover letter paragraphs from raw experience data.
- **Inputs**: Job experience, skills, target role, job description
- **Outputs**: Polished resume statements, cover letter sections, achievement highlights
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses action verbs, quantifies results, ties experience to requirements.

### 19. Video/Podcast Script Writer
- **Category**: Communication
- **Description**: Creates detailed scripts for videos or podcast episodes with proper flow.
- **Inputs**: Topic, target length, format preference, audience
- **Outputs**: Complete script with segments, narration text, cues for pauses/music
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Uses ScriptOutliner and NarrativeExpander subroutines.

### 20. Habit Tracker & Coach
- **Category**: Personal Development
- **Description**: Tracks daily habits and provides personalized feedback and encouragement.
- **Inputs**: Habit list, daily check-ins, goals, preferences
- **Outputs**: Streak tracking, success statistics, coaching messages, improvement tips
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Adjusts feedback style based on progress patterns.

### 21. Daily Journal Insight
- **Category**: Personal Development
- **Description**: Analyzes journal entries to detect themes, emotions, and provide reflective insights.
- **Inputs**: Journal entry text, mood indicators
- **Outputs**: Theme summary, emotion analysis, reflection questions, gratitude suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Acts as journaling coach for deeper self-reflection.

### 22. Goal Planner & Tracker
- **Category**: Personal Development
- **Description**: Helps set specific goals, break into steps, and track progress with adjustments.
- **Inputs**: Goal description, timeline, constraints, current status
- **Outputs**: Goal breakdown, milestone plan, progress tracking, encouragement
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Time Block Scheduler and Deadline Monitor subroutines.

### 23. Wellness Check-in Bot
- **Category**: Personal Development
- **Description**: Performs guided well-being check-ins with reflective questions and support.
- **Inputs**: Current mood, stressors, recent events
- **Outputs**: Reflection summary, coping strategies, affirmations, resources
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Not a substitute for professional help. Daily wellness companion.

### 24. Integrated Morning Briefing & Planner
- **Category**: Personal Development
- **Description**: Comprehensive morning routine combining wellness check, task planning, and daily briefing.
- **Inputs**: To-do list, calendar, sleep data, priorities
- **Outputs**: Morning briefing report, prioritized schedule, wellness tips, motivational quote
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses BPMN workflow. Combines WellnessCheck, TaskPlanner, and BriefingComposer.

### 25. Project Planner & Task Breakdown
- **Category**: Business
- **Description**: Scopes projects by breaking into phases, milestones, and tasks with timelines.
- **Inputs**: Project goal, constraints, resources, timeline
- **Outputs**: Project plan, phase breakdown, task list, timeline estimates
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Uses Project Task Organizer and Time Block Scheduler subroutines.

### 26. Meeting Agenda & Minutes Orchestrator
- **Category**: Business
- **Description**: Streamlines meeting prep and follow-up. Generates agendas before, minutes after.
- **Inputs**: Meeting purpose, topics, participants, notes/transcript
- **Outputs**: Structured agenda with time slots, minutes with action items
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Future: email integration for distribution.

### 27. Team Stand-up Synthesizer
- **Category**: Business
- **Description**: Collects team status updates and synthesizes into coherent team report.
- **Inputs**: Individual updates, blockers, plans
- **Outputs**: Team status summary, common themes, shared blockers, progress overview
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Saves managers from manual compilation. Provides unified view.

### 28. Client Proposal Assistant
- **Category**: Business
- **Description**: Drafts business proposals with standard sections and persuasive language.
- **Inputs**: Client needs, solution details, pricing, timeline
- **Outputs**: Polished proposal document, executive summary, value proposition
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses Content Outline Generator and Editing Assistant subroutines.

### 29. Negotiation Strategist
- **Category**: Business
- **Description**: Prepares negotiation strategies including BATNA, talking points, and counter-arguments.
- **Inputs**: Negotiation scenario, objectives, other party info
- **Outputs**: Strategy briefing, BATNA analysis, concession options, response scripts
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses OtherPartySimulator and StrategyFormulator subroutines.

### 30. Smart Study Planner
- **Category**: Student
- **Description**: Plans study schedules for exams, allocating time per subject based on difficulty.
- **Inputs**: Exam dates, subjects, difficulty ratings, available study time
- **Outputs**: Study timetable, daily targets, review schedule, tips
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses ScheduleCalculator and PlanGenerator subroutines.

### 31. Lecture/Chapter Summarizer
- **Category**: Student
- **Description**: Summarizes academic texts focusing on key terms, concepts, and examples.
- **Inputs**: Lecture notes or textbook chapter, course context
- **Outputs**: Condensed summary, key terms glossary, main concepts
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses AcademicSummarizer and TermHighlighter subroutines.

### 32. Flashcard & Quiz Generator
- **Category**: Student
- **Description**: Creates study questions and flashcards from source material for active recall.
- **Inputs**: Study material, topic list, question preferences
- **Outputs**: Flashcard Q&A pairs, quiz questions, multiple choice options
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Uses QuestionGenerator and AnswerGenerator subroutines.

### 33. Essay Outline Assistant
- **Category**: Student
- **Description**: Helps organize thoughts into structured essay outlines with logical flow.
- **Inputs**: Essay prompt, thesis idea, key points
- **Outputs**: Essay outline with sections, topic sentences, evidence placeholders
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses ThesisGenerator and OutlineBuilder subroutines.

### 34. Writing Feedback Assistant
- **Category**: Student
- **Description**: Reviews student writing and provides multi-level feedback for improvement.
- **Inputs**: Draft text, assignment requirements, rubric
- **Outputs**: Feedback on structure, clarity, grammar, suggestions, revised sentences
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses GrammarCorrector and ContentCritique subroutines.

### 35. Expense Tracker & Analyzer
- **Category**: Financial
- **Description**: Helps users log and analyze expenses. Categorizes transactions and provides spending insights.
- **Inputs**: Expense list or statement, budget goals, time period
- **Outputs**: Categorized expenses, spending summary, budget analysis, alerts
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses code to categorize by merchant/keywords. AI provides spending advice.

### 36. Budget Planner
- **Category**: Financial
- **Description**: Creates and adjusts personal/household budgets based on income and goals.
- **Inputs**: Income sources, expenses, savings goals, priorities
- **Outputs**: Budget allocations, balance sheet, scenario simulations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses frameworks like 50/30/20 rule. Simulates what-if scenarios.

### 37. Investment Research Assistant
- **Category**: Financial
- **Description**: Gathers and summarizes investment information from public sources.
- **Inputs**: Asset names/tickers, portfolio, research focus
- **Outputs**: Market summary, news highlights, trend analysis, general sentiment
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Future: integrate with financial APIs for real-time data.

### 38. Price Comparison Bot
- **Category**: Financial
- **Description**: Compares prices for products across multiple online sources.
- **Inputs**: Product name/specifications, preferences
- **Outputs**: Price list by vendor, best deals, shipping/return notes
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses web search. Future: shopping API integration.

### 39. Subscription Manager
- **Category**: Financial
- **Description**: Tracks subscriptions and recurring bills, identifies savings opportunities.
- **Inputs**: Subscription list, usage data, budget constraints
- **Outputs**: Renewal calendar, cost analysis, cancellation recommendations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Highlights low-usage subscriptions. Schedules renewal reminders.

### 40. Data Summarizer & Visualizer
- **Category**: Data
- **Description**: Provides quick analysis and visualization of datasets.
- **Inputs**: Dataset (CSV/table), analysis goals, visualization preferences
- **Outputs**: Statistical summary, charts/graphs, written insights
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses pandas/matplotlib in sandbox. AI explains key findings.

### 41. Correlation Finder
- **Category**: Data
- **Description**: Analyzes relationships between variables in datasets.
- **Inputs**: Dataset, variable pairs, analysis type
- **Outputs**: Correlation coefficients, regression results, plain language explanation
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Warns about spurious correlations and causation vs correlation.

### 42. Anomaly Detector
- **Category**: Data
- **Description**: Scans data to find outliers and unusual patterns.
- **Inputs**: Time-series or categorical data, sensitivity thresholds
- **Outputs**: Anomaly list, potential explanations, visualization
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses statistical methods. AI suggests possible causes.

### 43. Survey Results Analyzer
- **Category**: Data
- **Description**: Extracts insights from survey data including quantitative and text responses.
- **Inputs**: Survey response data, question types
- **Outputs**: Statistical summary, theme analysis, key takeaways report
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Handles mixed data types. Clusters open-ended responses.

### 44. Content Categorizer
- **Category**: Data
- **Description**: Automatically classifies documents/content into meaningful categories.
- **Inputs**: Document collection, category preferences
- **Outputs**: Category assignments, theme descriptions, similarity clusters
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses embeddings for similarity. AI names categories descriptively.

### 45. Code Generator & Tester
- **Category**: Technical
- **Description**: Generates code from specifications and validates with tests.
- **Inputs**: Function specification, language, test cases
- **Outputs**: Generated code, test results, self-debugging attempts
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Runs code in sandbox. Attempts fixes if tests fail.

### 46. Bug Finder & Fixer
- **Category**: Technical
- **Description**: Helps locate and fix bugs in source code.
- **Inputs**: Code with bug, error messages, expected behavior
- **Outputs**: Bug diagnosis, fix suggestions, corrected code
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses static analysis and AI reasoning. Can apply and test fixes.

### 47. Documentation Summarizer
- **Category**: Technical
- **Description**: Summarizes API docs, technical manuals, or libraries for quick understanding.
- **Inputs**: Documentation text/link, focus areas
- **Outputs**: Key functions/classes, usage examples, quick reference
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Extracts most important parts. Creates concise developer guide.

### 48. Regex Generator & Tester
- **Category**: Technical
- **Description**: Creates regular expressions from requirements and tests them.
- **Inputs**: Pattern description, example strings, edge cases
- **Outputs**: Regex pattern, test results, pattern explanation
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Tests against provided cases. Explains regex components.

### 49. Tech Stack Recommender
- **Category**: Technical
- **Description**: Suggests technology choices for projects based on requirements.
- **Inputs**: Project type, constraints, preferences, scale
- **Outputs**: Stack recommendations with rationale, alternatives, trade-offs
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Covers all layers: language, framework, database, hosting, tools.

### 50. Recurring Task Scheduler
- **Category**: Productivity
- **Description**: Manages repetitive tasks by generating future occurrences and reminders.
- **Inputs**: Task definition, recurrence pattern, start date, end conditions
- **Outputs**: Scheduled occurrences, reminder calendar, contextual tips
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Handles daily/weekly/monthly patterns. AI adds context per occurrence.

### 51. Priority Matrix (Eisenhower Assistant)
- **Category**: Productivity
- **Description**: Sorts tasks into urgent/important matrix for better prioritization.
- **Inputs**: Task list with urgency/importance data, deadlines
- **Outputs**: 2x2 matrix categorization, action recommendations per quadrant
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Do First, Schedule, Delegate, Eliminate quadrants. Helps focus on what matters.

### 52. What-If Scenario Planner
- **Category**: Analysis
- **Description**: Explores hypothetical scenarios and their potential outcomes.
- **Inputs**: Scenario description, variables, constraints
- **Outputs**: Outcome predictions, benefits/drawbacks, similar case examples
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helps evaluate decisions by visualizing possible futures.

### 53. Language Translator & Learning Aid
- **Category**: Student
- **Description**: Translates text while teaching grammar and vocabulary.
- **Inputs**: Source text, source/target languages, learning level
- **Outputs**: Translation, grammar explanations, vocabulary notes, usage examples
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Goes beyond translation to teach. Uses Translator and LinguisticAnalyzer subroutines.

### 54. Learning Plan Generator
- **Category**: Learning
- **Description**: Creates personalized learning curriculum for new skills or subjects.
- **Inputs**: Learning goal, time availability, learning style, current knowledge
- **Outputs**: Structured curriculum, resource list, milestones, practice exercises
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Mixes different resource types. Adapts based on progress.

### 55. Background Briefing Assistant
- **Category**: Research
- **Description**: Prepares quick background briefs on people or organizations.
- **Inputs**: Name/entity, context (meeting, interview, etc.), focus areas
- **Outputs**: Background summary, recent news, key points, conversation starters
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses only publicly available information. Helps with meeting prep.

### 56. Expert Advice Aggregator
- **Category**: Research
- **Description**: Collects and synthesizes expert advice on specific questions.
- **Inputs**: Question/topic, expertise level needed, source preferences
- **Outputs**: Curated advice summary, common recommendations, unique insights
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Searches expert articles, interviews, forums. Highlights consensus vs unique views.

### 57. SEO Content Optimizer
- **Category**: Content
- **Description**: Refines content for search engine optimization without losing quality.
- **Inputs**: Content text, target keywords, SEO goals
- **Outputs**: Optimized content, keyword density analysis, meta suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Uses KeywordAnalyzer and SEORewriter subroutines. Maintains readability.

### 58. Content Repurposer
- **Category**: Content
- **Description**: Transforms single content piece into multiple formats for different platforms.
- **Inputs**: Source content, target formats, platform requirements
- **Outputs**: Content kit with social posts, email blurbs, summaries, etc.
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Maximizes content ROI. Uses SummaryGenerator and FormatAdapter subroutines.

### 59. Meal Planner
- **Category**: Personal Development
- **Description**: Suggests meal plans based on dietary preferences and goals.
- **Inputs**: Dietary preferences, ingredients on hand, calorie targets, schedule
- **Outputs**: Weekly meal plan, recipes, grocery list, nutrition estimates
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses Recipe Finder and Nutrition Estimator subroutines.

### 60. Workout Routine Builder
- **Category**: Personal Development
- **Description**: Creates personalized workout plans based on fitness goals and equipment.
- **Inputs**: Fitness goals, equipment available, schedule, experience level
- **Outputs**: Weekly workout schedule, exercise descriptions, progression plan
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Balances muscle groups, includes rest days, provides progression guidance.

### 61. Travel Itinerary Planner
- **Category**: Personal Development
- **Description**: Generates travel itineraries with daily activities and logistics.
- **Inputs**: Destination, trip length, interests, budget
- **Outputs**: Day-by-day itinerary, attraction list, travel tips, time estimates
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Balances activities with downtime. Clusters by location for efficiency.

### 62. Customer Feedback Summarizer
- **Category**: Business
- **Description**: Analyzes customer feedback to extract key insights and trends.
- **Inputs**: Feedback data (reviews, surveys, support tickets)
- **Outputs**: Theme summary, sentiment analysis, improvement priorities
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Identifies praise points and pain points. Quantifies sentiment percentages.

### 63. Performance Metrics Reporter
- **Category**: Business
- **Description**: Turns raw performance data into narrative reports.
- **Inputs**: Performance data, targets, time period, audience
- **Outputs**: Narrative report, key metrics, trend analysis, recommendations
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Uses Data Analyzer and Narrative Generator subroutines.

### 64. Opportunity Evaluator (ROI & Fit Analysis)
- **Category**: Business
- **Description**: Evaluates opportunities with both financial and strategic analysis.
- **Inputs**: Opportunity details, costs/benefits, strategic goals
- **Outputs**: ROI calculations, strategic fit analysis, pros/cons, recommendations
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses FinancialCalculator and FeasibilityAdvisor subroutines.

### 65. Email Thread Condenser
- **Category**: Knowledge
- **Description**: Summarizes long email threads or chat conversations to bring someone up to speed quickly.
- **Inputs**: Email thread or chat history, key participants
- **Outputs**: Timeline summary, main points, decisions made, next steps
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Preserves chronological flow and key decisions.

### 66. Note Summarizer
- **Category**: Knowledge
- **Description**: Cleans up and summarizes informal notes from brainstorming sessions or lectures.
- **Inputs**: Raw notes, context (meeting type, subject)
- **Outputs**: Organized summary, key themes, suggested follow-ups
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Can integrate with Note-to-Task Converter for action items.

### 67. Glossary Generator
- **Category**: Knowledge
- **Description**: Extracts important terms or jargon from documents and provides definitions.
- **Inputs**: Document text, domain/field context
- **Outputs**: Term list with definitions, usage examples
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Scans for capitalized terms, frequent jargon. Can retrieve definitions via search.

### 68. Highlight Extractor
- **Category**: Knowledge
- **Description**: Identifies the most insightful or relevant sentences from documents.
- **Inputs**: Document(s), relevance criteria, keyword preferences
- **Outputs**: Top highlights list, context for each, importance scores
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses heuristic scoring and AI to find key insights.

### 69. Design Brainstormer
- **Category**: Creative
- **Description**: Generates design or feature ideas for products and projects.
- **Inputs**: Design problem, product concept, constraints
- **Outputs**: Feature suggestions, style ideas, innovation concepts
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Outputs text descriptions of design concepts for prototyping.

### 70. Title and Headline Generator
- **Category**: Creative
- **Description**: Suggests attention-grabbing titles for various content types.
- **Inputs**: Content summary, keywords, target audience, medium
- **Outputs**: Multiple title options, engagement predictions, rationale
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Leverages engagement principles for different platforms.

### 71. Question Stormer
- **Category**: Creative
- **Description**: Generates insightful or probing questions on a given topic.
- **Inputs**: Topic, research goals, audience level
- **Outputs**: Question list by category, exploration angles, discussion prompts
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps with FAQ creation, interview prep, deeper thinking.

### 72. Campaign Idea Factory
- **Category**: Creative
- **Description**: Generates creative marketing campaign concepts.
- **Inputs**: Product/service, campaign goals, target audience, budget level
- **Outputs**: Campaign themes, channel strategies, taglines, tactics
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Provides multiple concepts for further development.

### 73. Character & Dialogue Generator
- **Category**: Creative
- **Description**: Creates fictional characters and dialogue for creative projects.
- **Inputs**: Character requirements, story context, personality traits
- **Outputs**: Character profiles, dialogue samples, interaction scenarios
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Useful for writing, role-play training, simulations.

### 74. Follow-up Email Generator
- **Category**: Communication
- **Description**: Prepares professional follow-up messages after meetings or interactions.
- **Inputs**: Meeting notes, key outcomes, commitments, recipient context
- **Outputs**: Email draft, subject line, next steps summary
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Ensures consistency and professionalism in follow-ups.

### 75. Language Translator & Summarizer
- **Category**: Communication
- **Description**: Translates text between languages and optionally summarizes.
- **Inputs**: Source text, source/target languages, summary preference
- **Outputs**: Translation, optional summary, key points extraction
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Future: integrate specialized translation APIs for accuracy.

### 76. Presentation Slide Condenser
- **Category**: Communication
- **Description**: Converts lengthy text into concise presentation bullets.
- **Inputs**: Source text, slide count limit, presentation style
- **Outputs**: Bullet points per slide, suggested titles, speaker notes
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helps quickly create presentations from written material.

### 77. Meeting Agenda Emailer
- **Category**: Communication
- **Description**: Generates agenda emails with discussion points and time allocations.
- **Inputs**: Meeting topics, goals, participants, duration
- **Outputs**: Formatted agenda email, time blocks, pre-read suggestions
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Standardizes meeting communication.

### 78. Public Relations Statement Assistant
- **Category**: Communication
- **Description**: Drafts public-facing statements or press releases.
- **Inputs**: Key facts, desired tone, audience, message goals
- **Outputs**: Statement draft, key messages, Q&A prep
- **Strategy**: Conversational
- **Priority**: Low
- **Notes**: Focuses on clarity, tone, and comprehensive coverage.

### 79. Social Media Content Batch
- **Category**: Communication
- **Description**: Generates content for multiple social platforms from one theme.
- **Inputs**: Theme/campaign, platform list, tone, hashtag preferences
- **Outputs**: Platform-specific posts, hashtags, posting schedule
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Uses PostIdeasGenerator and MultiPlatformFormatter subroutines.

### 80. Gift Recommendation Assistant
- **Category**: Personal Development
- **Description**: Suggests thoughtful gift ideas based on recipient profile.
- **Inputs**: Recipient info (age, interests), occasion, budget
- **Outputs**: Gift ideas list, rationale for each, where to buy
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses AI and web search for current gift guides.

### 81. Home Maintenance Scheduler
- **Category**: Personal Development
- **Description**: Tracks and reminds about routine home maintenance tasks.
- **Inputs**: Home type, custom tasks, maintenance history
- **Outputs**: Annual schedule, task reminders, importance explanations
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Future: notification integration for reminders.

### 82. Sleep & Recovery Analyzer
- **Category**: Personal Development
- **Description**: Analyzes sleep patterns and provides optimization advice.
- **Inputs**: Sleep/wake times, quality ratings, lifestyle factors
- **Outputs**: Sleep trends, deficit analysis, improvement suggestions
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Tracks patterns over time for personalized recommendations.

### 83. Stress Relief Break Coach
- **Category**: Personal Development
- **Description**: Suggests stress-relief activities during work breaks.
- **Inputs**: Stress level, time available, preferences
- **Outputs**: Activity suggestions, guided exercises, timing
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Includes breathing exercises, stretches, mindfulness prompts.

### 84. Invoice & Expense Processor
- **Category**: Business
- **Description**: Extracts and records financial data from invoices/receipts.
- **Inputs**: Invoice text/image, expense categories
- **Outputs**: Structured data, accounting entries, summaries
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Future: accounting system integration.

### 85. Policy/Document Summarizer
- **Category**: Business
- **Description**: Creates executive summaries of long policy or technical documents.
- **Inputs**: Full document, target audience, focus areas
- **Outputs**: Executive summary, key points by section, action items
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Document Summarizer with Context Filter subroutine.

### 86. Resource Allocation Advisor
- **Category**: Business
- **Description**: Helps distribute resources across projects optimally.
- **Inputs**: Project list, resource constraints, priorities, deadlines
- **Outputs**: Allocation plan, scenario comparisons, rationale
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Models different scenarios with trade-offs.

### 87. Strategic Plan Generator
- **Category**: Business
- **Description**: Assists in drafting strategic plans and OKRs.
- **Inputs**: High-level goals, challenges, timeframe
- **Outputs**: Strategic objectives, key results, initiatives, SMART goals
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Ensures objectives are measurable and aligned.

### 88. Data Cleaner
- **Category**: Data
- **Description**: Cleans and preprocesses raw data for analysis.
- **Inputs**: Dataset with issues, cleaning preferences
- **Outputs**: Cleaned dataset, issue report, transformation log
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Handles missing values, duplicates, format standardization.

### 89. A/B Test Evaluator
- **Category**: Data
- **Description**: Analyzes A/B test results for statistical significance.
- **Inputs**: Variant data, success metrics, sample sizes
- **Outputs**: Statistical analysis, winner recommendation, confidence levels
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses appropriate statistical tests, explains results clearly.

### 90. Excel-to-Database Importer
- **Category**: Data
- **Description**: Converts spreadsheet data into queryable database format.
- **Inputs**: Excel/CSV file, schema preferences
- **Outputs**: Database-ready data, SQL access, query interface
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Enables SQL queries on spreadsheet data.

### 91. Chart Generator
- **Category**: Data
- **Description**: Creates visualizations from data on demand.
- **Inputs**: Data, chart type preference, styling options
- **Outputs**: Chart image, caption, data insights
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Reusable subroutine for other analysis routines.

### 92. Content Clusterer
- **Category**: Data
- **Description**: Organizes documents into thematic clusters.
- **Inputs**: Document collection, clustering preferences
- **Outputs**: Cluster groups, theme labels, similarity scores
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses embeddings and clustering algorithms.

### 93. Dev Environment Troubleshooter
- **Category**: Technical
- **Description**: Diagnoses build errors and configuration issues.
- **Inputs**: Error messages, environment details, symptoms
- **Outputs**: Likely causes, step-by-step solutions, diagnostic commands
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Guides through common setup and dependency issues.

### 94. Performance Optimizer
- **Category**: Technical
- **Description**: Analyzes code for performance improvements.
- **Inputs**: Code, performance metrics, constraints
- **Outputs**: Bottleneck analysis, optimization suggestions, benchmarks
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Identifies algorithmic improvements and efficiency gains.

### 95. API Integration Planner
- **Category**: Technical
- **Description**: Outlines steps for third-party API integration.
- **Inputs**: API documentation, integration goals
- **Outputs**: Integration plan, auth setup, code examples, error handling
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Provides implementation roadmap without actual API calls.

### 96. Git Workflow Assistant
- **Category**: Technical
- **Description**: Guides through Git version control tasks and issues.
- **Inputs**: Git scenario, error messages, repository state
- **Outputs**: Command sequences, best practices, issue resolution
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps with merges, conflicts, history management.

### 97. Email Triage Assistant
- **Category**: Productivity
- **Description**: Assists in quickly processing email inboxes by classifying and summarizing messages.
- **Inputs**: Email list or text, classification preferences
- **Outputs**: Categorized emails (urgent, FYI, tasks), summaries, extracted tasks
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Future: email API integration. Currently requires manual email text input.

### 98. Local Information Scout
- **Category**: Research
- **Description**: Finds location-specific information like restaurants, services, or events.
- **Inputs**: Location, search criteria (restaurants, services, etc.), preferences
- **Outputs**: Recommendations list, ratings/reviews, contact details, tips
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Uses web search until local APIs become available. Parses review sites and maps results.

### 99. Trade-off Analyzer
- **Category**: Analysis
- **Description**: Analyzes trade-offs between choices that have both upsides and downsides.
- **Inputs**: Options list, key factors, priorities
- **Outputs**: Trade-off matrix, gain vs. loss analysis, recommendation
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Helpful for nuanced decisions where there's no clear winner.

### 100. Delegate-or-Automate Recommender
- **Category**: Productivity
- **Description**: Reviews tasks and recommends which to delegate or automate.
- **Inputs**: Task list, team capabilities, automation options
- **Outputs**: Delegation recommendations, automation opportunities, reasoning
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Considers task complexity, frequency, and expertise requirements.

### 101. Product Comparison Researcher
- **Category**: Research
- **Description**: Compares multiple products or options before decision-making.
- **Inputs**: Product list, comparison criteria, budget constraints
- **Outputs**: Comparison chart, pros/cons analysis, recommendation
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Uses Web Research Agent and Tabular Formatter subroutines.

### 102. News Digest Summarizer
- **Category**: Research
- **Description**: Compiles daily/weekly news digests on topics of interest.
- **Inputs**: Topics list, time period, source preferences
- **Outputs**: Categorized news summary, key developments, trend highlights
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Helps stay informed without reading multiple articles.

### 103. Coupon & Deal Finder
- **Category**: Financial
- **Description**: Searches for discount codes and ongoing sales for products/stores.
- **Inputs**: Store/product name, purchase timeline
- **Outputs**: Valid coupon codes, sale notifications, savings estimates
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Filters expired deals. Future: direct coupon database integration.

### 104. Financial Goal Tracker
- **Category**: Financial
- **Description**: Guides users in setting and monitoring financial goals.
- **Inputs**: Goal details, current status, contribution schedule
- **Outputs**: Progress tracking, timeline projections, tips for acceleration
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Provides encouragement and calculates impact of contribution changes.

### 105. Loan Calculator & Advisor
- **Category**: Financial
- **Description**: Calculates loan payments and provides guidance on loan decisions.
- **Inputs**: Loan parameters, scenarios to compare
- **Outputs**: Payment calculations, total interest, advice on terms
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Explains loan concepts and trade-offs in plain language.

### 106. Tax Document Organizer
- **Category**: Financial
- **Description**: Assists in organizing information for tax filing.
- **Inputs**: Income/expense data, deduction items
- **Outputs**: Tax-ready categorization, document checklist, preparation tips
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: For preparation only, not official tax advice.

### 107. Financial Report Summarizer
- **Category**: Financial
- **Description**: Summarizes financial statements and reports for easier understanding.
- **Inputs**: Financial report, focus areas
- **Outputs**: Key figures summary, trend analysis, plain-language insights
- **Strategy**: Reasoning
- **Priority**: Low
- **Notes**: Useful for both personal statements and company reports.

### 108. Decision Matrix Maker
- **Category**: Analysis
- **Description**: Creates a weighted decision matrix to quantitatively compare options against multiple criteria.
- **Inputs**: Options list, criteria with weights, scoring preferences
- **Outputs**: Decision matrix, weighted scores, top recommendation, rationale
- **Strategy**: Deterministic
- **Priority**: High
- **Notes**: Uses Scoring Engine and Matrix Formatter subroutines. Provides quantitative decision support.

### 109. Goal Alignment Checker
- **Category**: Analysis
- **Description**: Reviews tasks or initiatives against stated high-level goals or OKRs.
- **Inputs**: Goal/OKR description, current activities list, priority weights
- **Outputs**: Alignment analysis, misaligned items, prioritization recommendations
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Ensures effort aligns with strategic objectives. Flags activities to prioritize or drop.

### 110. Keyword Extractor for Text
- **Category**: Data
- **Description**: Identifies the most important keywords or phrases in text datasets.
- **Inputs**: Text documents, extraction method preferences, topic focus
- **Outputs**: Keyword list with importance scores, topic groupings, context suggestions
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Uses TF-IDF or similar algorithms. Useful for content analysis and search indexing.
