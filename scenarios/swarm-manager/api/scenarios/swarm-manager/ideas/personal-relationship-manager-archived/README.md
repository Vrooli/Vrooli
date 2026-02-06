# Personal Relationship Manager

## Overview
A thoughtful relationship management system that helps you nurture and maintain personal connections. Track important dates, notes, gift ideas, and relationship histories in a warm, intuitive interface.

## Purpose
- **Remember Important Dates**: Never forget birthdays, anniversaries, or special occasions
- **Track Relationship Details**: Store notes, preferences, and meaningful conversations
- **Gift Suggestions**: AI-powered gift recommendations based on interests and past gifts
- **Connection Reminders**: Proactive nudges to reach out and maintain relationships
- **Relationship Insights**: Understand patterns and strengthen connections over time

## Key Features
- Birthday and event reminders with customizable notification preferences
- Gift idea generator with budget considerations and interest matching
- Relationship timeline showing interaction history and milestones
- Notes and tags for organizing contacts by groups (family, friends, colleagues)
- Automated web research for gift ideas based on person's interests
- Integration with calendar systems for event tracking

## UX Design Philosophy
**Warm & Personal**: Soft color palette with pastel tones, rounded corners, and friendly typography. Think of a digital version of a personal journal or scrapbook - intimate, caring, and thoughtful. Uses subtle animations and delightful micro-interactions to make relationship management feel less like a chore and more like a caring practice.

## Dependencies
- **Shared Workflows**: Uses ollama.json for AI interactions
- **Resources**: PostgreSQL for data, N8n for automation, Ollama for AI suggestions
- **Integrations**: Web search for gift ideas, calendar systems for reminders

## Business Value
- **Consumer App**: Direct-to-consumer SaaS for individuals ($5-10/month)
- **Enterprise**: Team relationship management for sales/customer success
- **Freemium Model**: Basic features free, premium for AI suggestions and unlimited contacts

## Technical Notes
- Stores all data locally for privacy
- Uses vector embeddings for intelligent gift matching
- Implements smart notification scheduling to avoid reminder fatigue