# Task Planner

AI-powered task management system with progressive refinement from unstructured text to implementation.

## Purpose

Task Planner transforms chaotic thoughts, notes, and TODO lists into organized, actionable tasks with AI assistance. It provides a complete pipeline from raw text parsing through research and planning to actual implementation.

## Key Features

- **Text-to-Task Parsing**: Convert unstructured text (emails, notes, voice transcripts) into organized tasks
- **AI Research & Planning**: Automatically research tasks and create implementation plans
- **Progressive Refinement**: Tasks move through stages: Backlog → Staged → In Progress → Completed
- **Multi-Agent Implementation**: Leverages Claude Code and Agent-S2 for actual task implementation
- **Semantic Search**: Vector database for finding related tasks and research
- **Cross-Scenario Integration**: Can be called by other scenarios needing task management

## UI/UX Style

Professional and clean interface with a purple gradient theme. The UI focuses on clarity and efficiency, making task management feel effortless rather than burdensome.

## Dependencies

### Required Resources
- **PostgreSQL**: Task storage and relational data
- **Redis**: Caching and real-time updates
- **Qdrant**: Vector search for semantic task matching
- **n8n**: Workflow automation for research and implementation
- **Windmill**: Alternative dashboard (optional)
- **Ollama**: AI model for text parsing and task refinement

### Optional Resources
- **SearxNG**: Web search for task research
- **Claude Code**: Advanced implementation agent
- **Agent-S2**: Browser automation for complex tasks

## API Endpoints

- `POST /api/parse-text` - Parse unstructured text into tasks
- `GET /api/tasks` - List all tasks with filtering
- `POST /api/tasks/:id/research` - Research and plan a task
- `POST /api/tasks/:id/implement` - Start task implementation
- `GET /api/apps` - List integrated apps/scenarios

## CLI Commands

```bash
task-planner parse "your text here"  # Parse text into tasks
task-planner list                    # List all tasks
task-planner research <task-id>      # Research a specific task
task-planner implement <task-id>     # Implement a task
```

## Integration Points

Other scenarios can leverage task-planner for:
- Breaking down complex projects into manageable tasks
- Processing voice transcripts from morning-vision-walk
- Managing implementation tasks from product-manager-agent
- Organizing research tasks from research-assistant

## Technical Architecture

- **Go API**: High-performance coordination layer
- **Express UI**: Lightweight, responsive web interface
- **n8n Workflows**: Automated task processing pipelines
- **Vector Database**: Semantic similarity for task matching

## Future Enhancements

- Integration with calendar systems for deadline management
- Team collaboration features
- Task templates and recurring tasks
- Mobile app interface
- Voice input integration