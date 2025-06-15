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

## 🚧 **Planned Modules**

### **Organization & Productivity**
- **[organization.md]** - **Bookmarks, Tags, Schedules, Meetings**
  - Content organization, categorization, and time management
  - **Models**: bookmark, bookmark_list, tag, tag_translation, schedule, schedule_exception, schedule_recurrence, meeting, meeting_attendees, meeting_invite, meeting_translation, reminder, reminder_item, reminder_list

### **Analytics & Gamification**
- **[analytics.md]** - **Statistics, Awards, Reputation, Views**
  - Usage analytics, performance metrics, and gamification systems
  - **Models**: stats_resource, stats_site, stats_team, stats_user, award, reputation_history, view

### **Authentication & Security**
- **[auth.md]** - **Sessions, API Keys, User Auth, Devices**
  - Authentication systems, API access control, and device management
  - **Models**: session, api_key, api_key_external, user_auth, user_translation, phone, push_device, transfer

## 📊 **Coverage Statistics**

| Module | Models | Status | Lines | Priority |
|--------|--------|---------|-------|----------|
| Core | 8 | ✅ Complete | ~400 | Critical |
| Communication | 8 | ✅ Complete | ~350 | Critical |
| Content | 10 | ✅ Complete | ~500 | High |
| Commerce | 5 | ✅ Complete | ~400 | High |
| Organization | 12 | 🚧 Planned | - | Medium |
| Analytics | 8 | 🚧 Planned | - | Medium |
| Authentication | 8 | 🚧 Planned | - | Medium |
| **Total** | **59** | **31 Done** | **1650** | - |

**Schema Coverage**: 47% (31/66 models documented)

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
├── core.md               # Foundation entities
├── communication.md      # Messaging and notifications
├── content.md           # User-generated content
├── commerce.md          # Billing and payments
├── organization.md      # [Planned] Bookmarks, tags, schedules
├── analytics.md         # [Planned] Statistics and gamification
└── auth.md             # [Planned] Authentication and security
```

---

**Related Documentation:**
- [Data Model Overview](../README.md) - Main data model documentation hub
- [Entity Relationships](../relationships.md) - Cross-entity relationships and constraints
- [Data Dictionary](../data-dictionary.md) - Field specifications and data types