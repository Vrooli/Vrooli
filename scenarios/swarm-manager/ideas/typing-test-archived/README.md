# Typing Test

## Purpose
Interactive typing speed and accuracy test with AI-powered personalized coaching to help users improve their typing skills.

## Key Features
- Real-time WPM and accuracy tracking
- Personalized AI coaching based on performance
- Progress tracking with achievements and leaderboards
- Custom practice texts targeting weak areas
- CLI tool for quick typing tests

## How It Helps Other Scenarios
- Provides typing assessment API for productivity scenarios
- Offers gamification patterns reusable by other skill-training apps
- Shares achievement system that can be adapted for other scenarios

## Dependencies
- **n8n**: Workflow automation for AI coaching and stats processing
- **PostgreSQL**: Stores user progress and leaderboard data
- **Shared Ollama workflow**: AI-powered coaching advice

## UI Style
Retro arcade terminal aesthetic with CRT effects, scanlines, and neon green styling. Features a nostalgic 80s computer theme that makes typing practice feel like playing a classic arcade game. Includes visual keyboard guide, combo system, and gamified scoring.

## CLI Usage
```bash
# Quick typing test
typing-test

# View personal stats
typing-test stats

# Get coaching advice
typing-test coach
```

## Integration Points
- API endpoints for typing assessment integration
- Webhook support for progress tracking in other apps
- Exportable typing metrics for productivity dashboards