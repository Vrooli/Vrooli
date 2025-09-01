# 📚 Study Buddy

A cozy, AI-powered study assistant with flashcards, quizzes, and spaced repetition learning. Features a beautiful Lofi Girl-inspired aesthetic to make studying feel relaxing and productive.

## Features

### 🎴 Smart Flashcards
- AI-generated flashcards from any content
- Spaced repetition algorithm for optimal retention
- Difficulty tracking (Easy/Medium/Hard)
- Semantic search for finding related cards
- Hints and contextual information

### 📝 Interactive Quizzes
- Multiple choice questions generated from study material
- Immediate feedback with explanations
- Progress tracking and scoring
- Adaptive difficulty based on performance

### 📔 Study Notes
- Markdown-enabled note editor
- Automatic organization by subject
- Link notes to flashcards and quizzes
- Export notes in various formats

### 📊 Progress Tracking
- Daily study streaks and XP system
- Detailed analytics on study patterns
- Personalized study recommendations
- Session timer with Pomodoro support

### 🎨 Cozy UI Experience
- Lofi Girl-inspired aesthetic
- Animated coffee steam and twinkling stars
- Purple and pink gradient theme
- Smooth animations and transitions
- Decorative plants and cat companion

## Architecture

### Resources Used
- **Ollama**: AI model for content generation
- **N8n**: Workflow automation for flashcard/quiz generation
- **PostgreSQL**: Store users, subjects, flashcards, sessions
- **Qdrant**: Vector database for semantic search
- **Redis**: Cache for session data and spaced repetition intervals

### N8n Workflows
- `flashcard-generator.json`: Generate flashcards from content
- `quiz-generator.json`: Create quizzes from study material  
- `study-plan-generator.json`: AI-powered study schedule
- `progress-analyzer.json`: Track and analyze learning patterns
- `content-generator.json`: Generate explanations and summaries
- `study-session-tracker.json`: Monitor study sessions

## Usage

### Starting the Application
```bash
# Run the study buddy scenario
vrooli scenario run study-buddy
```

### API Endpoints
- `POST /api/flashcards/generate` - Generate flashcards from content
- `POST /api/quizzes/create` - Create a quiz
- `GET /api/subjects` - List all subjects
- `POST /api/sessions/start` - Start study session
- `GET /api/progress/stats` - Get progress statistics

### CLI Commands
```bash
# Generate flashcards from a file
study-buddy generate-flashcards --file notes.txt --count 10

# Create a quiz
study-buddy create-quiz --subject math --questions 20

# View progress
study-buddy view-progress --period week
```

## Configuration

The scenario is configured via `.vrooli/service.json`:
- API runs on port 8090 (configurable)
- UI runs on port 8091 (configurable)
- Requires Ollama, N8n, PostgreSQL, Qdrant, and Redis

## Development

### File Structure
```
study-buddy/
├── .vrooli/           # Service configuration
│   └── service.json
├── api/               # Go API server
│   └── main.go
├── cli/               # CLI scripts
├── initialization/    # Resource initialization
│   ├── n8n/          # Workflow definitions
│   └── postgres/     # Database schema
├── ui/               # Frontend application
│   ├── index.html    # Lofi Girl themed UI
│   ├── script.js     # Interactive functionality
│   └── server.js     # Express server
└── test/             # Test files
```

## Testing

Run the scenario tests:
```bash
vrooli test scenarios --name study-buddy
```

## Future Enhancements

- Voice input for creating flashcards
- Collaborative study rooms
- Mobile app with offline sync
- Integration with popular textbooks
- AR flashcard visualization
- Study buddy AI chat assistant
- Gamification with achievements
- Export to Anki/Quizlet formats