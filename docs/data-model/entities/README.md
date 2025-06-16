# Entity Documentation

Comprehensive documentation of all database entities in Vrooli, organized by business domain for easy navigation and maintenance.

## 📁 **Entity Modules**

### **✅ Core Platform Entities**
- **[core.md](core.md)** - **Users, Teams, Resources, Runs**
  - Foundation entities that form the backbone of the platform
  - User management, team organization, resource creation, and execution tracking
  - **Models**: User, Team, Resource, ResourceVersion, Run, RunStep, RunIO, Member

### **✅ Communication Systems**
- **[communication.md](communication.md)** - **Chats, Messages, Notifications, Emails**
  - Real-time messaging, notifications, and email management
  - Multi-language support and AI embeddings for search
  - **Models**: Chat, ChatMessage, ChatParticipants, ChatInvite, ChatTranslation, Email, Notification, NotificationSubscription

### **✅ Content Management**
- **[content.md](content.md)** - **Comments, Issues, Pull Requests, Reactions**
  - User-generated content, collaboration, and community features
  - Threaded discussions, issue tracking, code review, and content moderation
  - **Models**: Comment, CommentTranslation, Issue, IssueTranslation, PullRequest, PullRequestTranslation, Reaction, ReactionSummary, Report, ReportResponse

### **✅ Commerce & Billing**
- **[commerce.md](commerce.md)** - **Payments, Plans, Credits, Wallets**
  - Subscription management, payment processing, and credit systems
  - Multi-currency support and detailed transaction logging
  - **Models**: Plan, Payment, CreditAccount, CreditLedgerEntry, Wallet

### **✅ Organization & Scheduling**
- **[organization.md](organization.md)** - **Invites, Reminders, Schedules, Transfers**
  - Team invitations, reminder management, scheduling, and resource transfers
  - Time management, workflow automation, and team coordination
  - **Models**: MemberInvite, ReminderList, Reminder, ReminderItem, Schedule, ScheduleRecurrence, ScheduleException, Transfer

### **✅ Analytics & Internationalization**
- **[analytics.md](analytics.md)** - **Statistics, Translations, Tags**
  - Platform analytics, multi-language support, and content categorization
  - Performance metrics, global accessibility, and tag management
  - **Models**: StatsResource, StatsSite, StatsTeam, StatsUser, ResourceTranslation, TeamTranslation, UserTranslation, TagTranslation, TeamTag

### **✅ Security & Authentication**
- **[security.md](security.md)** - **Sessions, API Keys, Auth, Achievements**
  - Authentication systems, API access control, and user achievements
  - Security management, external integrations, and gamification
  - **Models**: Session, UserAuth, ApiKey, ApiKeyExternal, Award, View

## 📊 **Coverage Statistics**

| Module | Models | Status | Lines | Priority |
|--------|--------|---------|-------|----------|
| Core | 10 | ✅ Complete | ~500 | Critical |
| Communication | 14 | ✅ Complete | ~650 | Critical |
| Content | 10 | ✅ Complete | ~500 | High |
| Commerce | 5 | ✅ Complete | ~400 | High |
| Organization | 8 | ✅ Complete | ~450 | Medium |
| Analytics | 9 | ✅ Complete | ~400 | Medium |
| Security | 6 | ✅ Complete | ~350 | Medium |
| **Total** | **62** | **62 Done** | **3250** | - |

**Schema Coverage**: 100% (62/62 models documented)

## 🔗 **Cross-References**

Each entity module includes:
- **Entity Relationship Diagrams** - Visual representation of relationships
- **TypeScript Interfaces** - Complete type definitions matching schema
- **Usage Patterns** - Common query patterns and business logic
- **Related Documentation** - Links to other relevant modules

## 🎯 **Navigation Tips**

### **For New Developers**
1. Start with **[Core Entities](core.md)** to understand the foundation
2. Move to your area of interest (Communication, Content, Commerce)
3. Use cross-references to explore related entities

### **For Feature Development**
1. Identify the relevant module for your feature
2. Review entity definitions and relationships
3. Check usage patterns for implementation guidance

### **For Schema Changes**
1. Update the relevant entity module
2. Check cross-references in other modules
3. Update relationships documentation if needed

## 📋 **Contribution Guidelines**

When adding new entity documentation:
1. **Follow Module Structure**: Use consistent formatting and sections
2. **Include All Fields**: Match actual schema.prisma definitions exactly
3. **Add Usage Patterns**: Provide practical query examples
4. **Update Cross-References**: Link to related entities in other modules
5. **Test Examples**: Ensure all code examples work with current schema

## 🔧 **File Organization**

```
docs/data-model/entities/
├── README.md              # This navigation file
├── core.md               # Foundation entities (User, Team, Resource, etc.)
├── communication.md      # Messaging, notifications, meetings
├── content.md           # User-generated content and collaboration
├── commerce.md          # Billing, payments, and credits
├── organization.md      # Scheduling, reminders, and transfers
├── analytics.md         # Statistics, translations, and tags
└── security.md          # Authentication, API keys, and achievements
```

---

**Related Documentation:**
- [Data Model Overview](../README.md) - Main data model documentation hub
- [Entity Relationships](../relationships.md) - Cross-entity relationships and constraints
- [Data Dictionary](../data-dictionary.md) - Field specifications and data types