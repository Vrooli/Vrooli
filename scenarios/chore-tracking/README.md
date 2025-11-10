# ChoreQuest - Gamified Chore Tracking

## 🎮 Purpose
ChoreQuest transforms household chores into a fun, gamified experience with points, achievements, and a cute, quirky UI. It helps families and individuals track cleaning tasks, build healthy habits, and make mundane activities more engaging.

## 🌟 Core Features
- **Smart Scheduling**: AI-powered weekly chore scheduling that learns from your completion patterns
- **Gamification**: Points system, achievements, leaderboards, and rewards for completing chores  
- **Family Support**: Multiple user profiles with age-appropriate chores and rewards
- **Habit Building**: Track streaks, set reminders, and visualize progress over time

## 🔗 Dependencies & Integration
- **Shared Workflows**: Uses `initialization/n8n/ollama.json` for AI-powered scheduling
- **Storage**: PostgreSQL for user data, Redis for real-time leaderboards
- **Resources**: N8n for automation workflows

## 🎨 UX Style
- **Vibe**: Playful, colorful, and encouraging - like a mix between Animal Crossing and a productivity app
- **Elements**: Animated confetti on task completion, cute character mascots, satisfying sound effects
- **Colors**: Bright pastels with pops of vibrant colors for achievements
- **Mobile-First**: Optimized for quick task checking on phones

## 🚀 Usage
```bash
# CLI Commands
chore-tracking add "Clean kitchen" --difficulty easy --points 10
chore-tracking complete <chore-id>
chore-tracking leaderboard
chore-tracking schedule --week

# API Endpoints
GET  /api/chores           # List all chores
POST /api/chores           # Create new chore
POST /api/chores/:id/complete  # Mark chore complete
GET  /api/users/:id/stats  # Get user statistics
```

## 🧩 Integration Points
Other scenarios can leverage ChoreQuest for:
- **Task Management**: Any scenario needing recurring task tracking
- **Gamification Engine**: Points and achievements system
- **Schedule Generation**: AI-powered scheduling logic
- **Habit Tracking**: Streak and completion rate monitoring

## 📊 Data Model
- **Users**: Family members with profiles, preferences, and stats
- **Chores**: Tasks with difficulty, points, frequency, and categories
- **Assignments**: Scheduled chores with status and completion data
- **Achievements**: Unlockable badges and rewards based on activity
- **Rewards**: Custom rewards that can be "purchased" with points
