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

## Backlog Items

### 1. Critical Thinking Pressure Detection
- **Category**: Metareasoning
- **Description**: Detects when users are applying social pressure for agreement or compliance. Helps AI systems avoid "yes-man" behavior by identifying pressure tactics and generating balanced responses.
- **Inputs**: User message, conversation context
- **Outputs**: Pressure analysis, balanced response recommendations
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Essential for maintaining AI trustworthiness and avoiding manipulation

### 2. Research Synthesis Workflow
- **Category**: Knowledge
- **Description**: Comprehensive research routine that gathers information from multiple sources, evaluates credibility, synthesizes insights, and produces structured reports.
- **Inputs**: Research topic, scope parameters, quality requirements
- **Outputs**: Structured research report, source citations, confidence ratings
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Should include fact-checking and source verification steps

### 3. Project Planning Decomposition
- **Category**: Productivity
- **Description**: Breaks down complex projects into manageable tasks, estimates effort, identifies dependencies, and creates actionable timelines.
- **Inputs**: Project description, constraints, resources, timeline
- **Outputs**: Task breakdown, timeline, dependency map, risk assessment
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Should handle both technical and non-technical projects

### 4. Content Quality Enhancement
- **Category**: Content
- **Description**: Improves written content for clarity, engagement, and effectiveness. Analyzes tone, structure, and messaging to provide enhancement suggestions.
- **Inputs**: Original content, target audience, purpose/goals
- **Outputs**: Enhanced content, improvement suggestions, quality metrics
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Should preserve author's voice while improving quality

### 5. Decision Support Framework
- **Category**: Analysis
- **Description**: Structured decision-making process that evaluates options, weighs criteria, considers consequences, and provides recommendation with reasoning.
- **Inputs**: Decision context, available options, evaluation criteria, constraints
- **Outputs**: Option analysis, weighted evaluation, recommendation, risk assessment
- **Strategy**: Reasoning
- **Priority**: High
- **Notes**: Should include bias detection and alternative perspective consideration

### 6. Learning Path Optimization
- **Category**: Learning
- **Description**: Creates personalized learning paths based on current knowledge, goals, learning style, and available resources. Adapts based on progress.
- **Inputs**: Learning goals, current knowledge level, time constraints, preferred resources
- **Outputs**: Structured learning path, milestones, resource recommendations, progress tracking
- **Strategy**: Reasoning
- **Priority**: Medium
- **Notes**: Should include different learning modalities and assessment points

### 7. Data Insight Extraction
- **Category**: Analysis
- **Description**: Analyzes datasets to identify patterns, trends, anomalies, and actionable insights. Handles various data formats and provides visualization recommendations.
- **Inputs**: Dataset, analysis objectives, context information
- **Outputs**: Key insights, statistical analysis, visualization suggestions, confidence levels
- **Strategy**: Deterministic
- **Priority**: Medium
- **Notes**: Should handle both structured and unstructured data

### 8. Workflow Automation Designer
- **Category**: Integration
- **Description**: Designs and implements automated workflows that connect different systems and processes. Identifies automation opportunities and creates implementation plans.
- **Inputs**: Current process description, system constraints, automation goals
- **Outputs**: Workflow design, implementation plan, ROI analysis, maintenance requirements
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Should consider error handling and monitoring requirements

### 9. Meeting Preparation Assistant
- **Category**: Productivity
- **Description**: Prepares comprehensive meeting materials including agendas, background research, talking points, and follow-up templates based on meeting objectives.
- **Inputs**: Meeting purpose, participants, topics, time constraints
- **Outputs**: Detailed agenda, background materials, talking points, action item templates
- **Strategy**: Conversational
- **Priority**: Medium
- **Notes**: Should adapt to different meeting types (planning, review, decision-making)

### 10. Code Review Quality Checker
- **Category**: Analysis
- **Description**: Systematically reviews code for quality, security, performance, and maintainability issues. Provides specific improvement recommendations with examples.
- **Inputs**: Code files, project context, quality standards, technology stack
- **Outputs**: Quality assessment, issue identification, improvement recommendations, priority ratings
- **Strategy**: Deterministic
- **Priority**: Low
- **Notes**: Should support multiple programming languages and frameworks

## Processing Instructions

When processing backlog items:
1. Review the item description and requirements
2. Determine if it should be Sequential or BPMN based on complexity
3. Plan the step-by-step workflow
4. Design appropriate input/output forms
5. Select placeholder subroutine IDs that match the required functionality
6. Create complete routine JSON with proper data flow
7. Save to `staged/` directory with descriptive filename
8. Mark item as processed and move to completed section

## Completed Items

Items that have been processed and moved to staged/ will be listed here with their filename references.

<!-- Processed items will be moved here -->