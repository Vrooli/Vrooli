# Nutrition Tracker - Product Requirements Document (PRD)

## 1. Overview
The Nutrition Tracker is a comprehensive application for tracking nutrition intake, monitoring calories and macros, and receiving AI-powered meal suggestions to support healthy eating habits.

## 2. Target Audience
- Individuals focused on weight management
- Fitness enthusiasts tracking macros
- People with dietary restrictions
- Health-conscious users seeking meal planning assistance

## 3. Key Features
### 3.1 User Management
- User registration and authentication
- Profile management (goals, dietary preferences, allergies)

### 3.2 Food Logging
- Searchable food database
- Barcode scanning for packaged foods
- Custom food entry
- Meal categorization (breakfast, lunch, etc.)

### 3.3 Tracking & Analytics
- Daily/weekly/monthly calorie and macro tracking
- Progress charts and reports
- Goal setting and achievement tracking

### 3.4 AI-Powered Suggestions
- Personalized meal recommendations based on preferences and goals
- Nutritional analysis of logged meals
- Recipe suggestions with nutritional breakdowns

### 3.5 Integration
- Sync with fitness trackers (future)
- Export reports to PDF/CSV

## 4. Technical Requirements
- Backend: Go API with PostgreSQL
- Frontend: Node.js/React UI
- Automation: Direct API/CLI flows (shared n8n workflows removed)
- AI: Ollama for local inference
- Vector Search: Qdrant for food similarity

## 5. Non-Functional Requirements
- Responsive design for mobile and desktop
- Data privacy compliance (GDPR)
- Offline capability for food logging
- Performance: &lt;2s response time for searches

## 6. Success Metrics
- User retention: 70% after 30 days
- Daily active users: 500+
- Food log accuracy: 95%+
