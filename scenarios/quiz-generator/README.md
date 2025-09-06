# Quiz Generator 🎓

> AI-powered quiz generation and assessment platform that creates intelligent quizzes from any content

## 🎯 Overview

The Quiz Generator is a foundational educational scenario for Vrooli that provides intelligent quiz creation, management, and assessment capabilities. It can automatically generate quizzes from uploaded documents using AI, while also supporting manual quiz creation and editing. This becomes a permanent capability that any learning-related scenario can leverage.

## ✨ Key Features

- **AI-Powered Generation**: Create quizzes automatically from PDFs, text files, markdown, and Word documents
- **Multiple Question Types**: MCQ, True/False, Short Answer, Fill-in-the-Blank, Matching, and Ordering
- **Smart Question Bank**: Semantic search and reusable questions with quality scoring
- **Real-time Assessment**: Interactive quiz-taking with immediate feedback
- **Analytics Dashboard**: Track performance, identify knowledge gaps, and optimize learning
- **Export/Import**: Support for JSON, QTI, and Moodle formats
- **API & CLI Access**: Full programmatic control for integration with other scenarios

## 🏗️ Architecture

```
quiz-generator/
├── api/              # Go REST API server
├── cli/              # Command-line interface
├── ui/               # React/Vite/TypeScript web interface
├── initialization/   # Resource setup and workflows
│   ├── automation/   # n8n workflows
│   └── storage/      # Database schemas
├── docs/             # Documentation
└── tests/            # Test suites
```

## 🚀 Quick Start

### Prerequisites
- Vrooli CLI installed
- PostgreSQL, n8n, and Ollama resources available
- Node.js 18+ and Go 1.21+

### Installation

```bash
# 1. Setup the scenario
cd scenarios/quiz-generator
vrooli scenario setup quiz-generator

# 2. Start the scenario
vrooli scenario run quiz-generator

# 3. Install CLI (optional)
./cli/install.sh
```

### Access Points
- **Web UI**: http://localhost:3251
- **API**: http://localhost:3250/api/v1
- **CLI**: `quiz-generator help`

## 📖 Usage Examples

### Generate Quiz from Document
```bash
# Via CLI
quiz-generator generate document.pdf --questions 10 --difficulty medium

# Via API
curl -X POST http://localhost:3250/api/v1/quiz/generate \
  -H "Content-Type: application/json" \
  -d '{
    "content": "The solar system has eight planets...",
    "question_count": 10,
    "difficulty": "medium"
  }'
```

### Take a Quiz
```bash
# Interactive CLI mode
quiz-generator take <quiz-id>

# Web UI
Navigate to http://localhost:3251/quiz/<quiz-id>
```

### Search Question Bank
```bash
# Find related questions
quiz-generator search "solar system" --tags astronomy
```

### Export Quiz
```bash
# Export to JSON
quiz-generator export <quiz-id> --format json --output quiz.json

# Export to QTI for LMS
quiz-generator export <quiz-id> --format qti --output quiz.xml
```

## 🔌 Integration with Other Scenarios

The Quiz Generator provides APIs that other scenarios can leverage:

### For Course Builder
```javascript
// Embed quiz in course module
const response = await fetch('/api/v1/quiz/generate', {
  method: 'POST',
  body: JSON.stringify({
    content: chapterContent,
    question_count: 5,
    difficulty: 'medium'
  })
});
```

### For Study Buddy
```javascript
// Generate practice questions
const questions = await fetch('/api/v1/question-bank/search', {
  method: 'POST',
  body: JSON.stringify({
    query: studyTopic,
    limit: 10
  })
});
```

## 🎨 UI Features

The React-based UI provides:
- **Dashboard**: Overview of all quizzes and recent activity
- **Quiz Builder**: Drag-and-drop interface for manual quiz creation
- **Question Editor**: Rich text editing with media support
- **Live Preview**: See how quizzes appear to users
- **Analytics**: Visual charts showing performance metrics
- **Dark Mode**: Eye-friendly interface for extended use

## 🛠️ Configuration

### Environment Variables
```bash
# API Configuration
QUIZ_GENERATOR_API_URL=http://localhost:3250/api/v1
DATABASE_URL=postgres://user:pass@localhost/quiz_generator
REDIS_URL=localhost:6379

# AI Settings
OLLAMA_MODEL=llama2
QUESTION_GENERATION_TIMEOUT=30

# UI Settings
VITE_API_BASE_URL=http://localhost:3250
```

### CLI Configuration
Edit `~/.vrooli/quiz-generator/config.yaml`:
```yaml
api:
  base_url: http://localhost:3250/api/v1
defaults:
  question_count: 10
  difficulty: medium
```

## 📊 Performance Metrics

- **Quiz Generation**: < 5 seconds for 10 questions
- **API Response**: < 200ms for standard operations
- **Concurrent Users**: 100+ simultaneous quiz takers
- **Question Quality**: > 85% relevance score

## 🔄 Lifecycle Management

### Start
```bash
vrooli scenario run quiz-generator
```

### Stop
```bash
vrooli scenario stop quiz-generator
# Or press Ctrl+C in the terminal where it's running
```

### Reset Data
```bash
# Clear all quiz data (careful!)
resource-postgres query "TRUNCATE quiz_generator.quizzes CASCADE"
```

### View Logs
```bash
vrooli scenario logs quiz-generator
```

## 🧩 API Reference

### Core Endpoints
- `POST /api/v1/quiz/generate` - Generate quiz from content
- `GET /api/v1/quiz/:id` - Retrieve quiz for taking/editing
- `POST /api/v1/quiz/:id/submit` - Submit quiz answers
- `GET /api/v1/quizzes` - List all quizzes
- `POST /api/v1/question-bank/search` - Search questions
- `GET /api/v1/quiz/:id/export` - Export quiz

Full API documentation available at http://localhost:3250/api/docs when running.

## 🤝 Contributing

This scenario is designed to be extended. Key areas for contribution:
- Additional question types
- More export formats
- Advanced analytics
- Multi-language support
- Accessibility improvements

## 📈 Future Enhancements

### Version 2.0 Planned Features
- Adaptive testing algorithms
- Multimedia questions (images, audio, video)
- Real-time collaboration
- Advanced analytics with ML insights
- Mobile app support

## 🐛 Troubleshooting

### Common Issues

**Quiz generation fails**
```bash
# Check Ollama is running
resource-ollama status
# Check n8n workflow is active
resource-n8n list-workflows
```

**Database connection errors**
```bash
# Verify PostgreSQL is running
resource-postgres status
# Check schema is initialized
resource-postgres query "SELECT * FROM quiz_generator.quizzes LIMIT 1"
```

**UI not loading**
```bash
# Check if port is available
lsof -i :3251
# Rebuild UI
cd ui && npm install && npm run build
```

## 📝 License

Part of the Vrooli ecosystem - see main repository for license details.

## 🔗 Related Scenarios

- **course-builder**: Uses quiz-generator for course assessments
- **study-buddy**: Leverages question bank for practice
- **certification-manager**: Extends for professional exams
- **interview-prep**: Uses for technical interview questions

---

**Status**: Active Development  
**Version**: 1.0.0  
**Last Updated**: 2024-09-05  
**Maintainer**: AI Agent

For more information, see the [PRD.md](PRD.md) for detailed requirements and architecture.