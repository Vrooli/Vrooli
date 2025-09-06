# Social Media Scheduler - Product Requirements Document

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Social Media Scheduler provides intelligent multi-platform content scheduling and distribution with AI-powered platform optimization. It automatically adapts content for each social media platform's unique constraints, audience, and culture while maintaining brand consistency and optimal posting times across Twitter, Instagram, LinkedIn, Facebook, and TikTok.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This scenario transforms every content-generating scenario in Vrooli into a multi-platform distribution powerhouse. Any scenario that creates content (blog posts, announcements, campaigns) can now automatically reach audiences across all social platforms. The platform-specific optimization intelligence becomes reusable knowledge for all future content scenarios, and the scheduling patterns become foundational for any time-based automation.

### Recursive Value
**What new scenarios become possible after this exists?**
- Auto-publishing blog scenario: Generates posts + automatically distributes them across platforms
- Event promotion orchestrator: Manages entire launch sequences with platform-specific timing
- Crisis communication manager: Rapid multi-platform message distribution with appropriate tone adaptation
- Influencer collaboration hub: Manages sponsored content scheduling with approval workflows
- Product launch coordinator: Orchestrates complex launch campaigns across all channels
- A/B testing orchestrator: Tests content variants across platforms and optimizes based on engagement
- Brand consistency monitor: Ensures all distributed content maintains brand guidelines

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Multi-platform content scheduling (Twitter, Instagram, LinkedIn, Facebook)
  - [ ] Platform-specific content optimization and preview system
  - [ ] Visual calendar interface with drag-and-drop scheduling
  - [ ] Direct API integration with social platforms (no n8n overhead)
  - [ ] Redis-based job queue with retry logic and rate limiting
  - [ ] Integration with scenario-authenticator for multi-tenant SaaS capability
  - [ ] Integration with campaign-content-studio for content generation
  - [ ] Integration with brand-manager for consistent visual identity
  - [ ] Media management with auto-resizing and format optimization
  - [ ] Real-time posting status and analytics dashboard
  
- **Should Have (P1)**
  - [ ] Bulk scheduling from CSV/Google Sheets imports
  - [ ] AI-powered optimal posting time recommendations
  - [ ] Content approval workflows for team collaboration
  - [ ] Advanced timezone handling for global audiences
  - [ ] Cross-platform performance analytics and engagement tracking
  - [ ] Content recycling for high-performing posts
  - [ ] Hashtag research and trending analysis integration
  - [ ] White-label capabilities for agency users
  
- **Nice to Have (P2)**
  - [ ] TikTok integration with video content optimization
  - [ ] Auto-reposting of viral content with smart timing
  - [ ] RSS/blog feed auto-posting integration
  - [ ] Advanced A/B testing with engagement optimization
  - [ ] Social listening integration for trending topic adaptation
  - [ ] Competitor analysis and benchmarking
  - [ ] AI-powered content performance prediction

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Posting Speed | < 2 seconds from queue to platform | Redis job monitoring |
| Platform Preview Accuracy | > 99% match with actual post appearance | Automated UI testing |
| Scheduling Reliability | > 99.9% posts published on time | Job success rate monitoring |
| Content Optimization Time | < 5 seconds per platform adaptation | AI processing benchmarks |
| Multi-tenant Performance | Support 1000+ concurrent users | Load testing |
| API Response Time | < 500ms for all UI interactions | Performance monitoring |

### Business Validation
| Metric | Target | Validation Method |
|--------|--------|-------------------|
| User Adoption | 100+ active schedulers within 3 months | Usage analytics |
| Content Volume | 10,000+ posts scheduled per month | Platform metrics |
| Revenue Generation | $10K MRR within 6 months | SaaS billing integration |
| Platform Coverage | 95%+ of user's desired platforms supported | User surveys |
| User Retention | 85%+ monthly retention rate | Cohort analysis |

## 🏗️ Technical Architecture

### Resource Dependencies
- **postgres**: User accounts, scheduled posts, analytics, OAuth tokens
  - purpose: Primary data store for all scheduling and user data
  - integration_pattern: Direct connection for complex queries and transactions
  - access_method: Go database/sql with pgx driver

- **redis**: Job queue, rate limiting, caching, session management  
  - purpose: High-performance job processing and caching layer
  - integration_pattern: Direct connection for sub-second job processing
  - access_method: Go Redis client with pub/sub for real-time updates

- **minio**: Media storage, image processing, asset management
  - purpose: Scalable storage for all user-uploaded media and generated assets
  - integration_pattern: Direct S3-compatible API calls
  - access_method: S3 SDK for Go with CDN-style serving

- **ollama**: Platform-specific content adaptation and optimization
  - purpose: AI-powered content transformation for each platform's culture and constraints
  - integration_pattern: Direct API calls for real-time optimization
  - access_method: HTTP REST API with streaming for large content

- **browserless**: Fallback posting mechanism and screenshot verification
  - purpose: Backup posting method when APIs fail, plus visual verification
  - integration_pattern: Automated browser actions for complex posting flows
  - access_method: REST API with custom automation scripts

### Resource Integrations
integration_priorities:
  1_direct_api:
    - justification: Maximum speed and reliability for real-time scheduling
    - endpoints: 
      - Twitter API v2: https://api.twitter.com/2/
      - Meta Graph API: https://graph.facebook.com/v18.0/
      - LinkedIn API: https://api.linkedin.com/rest/
      - PostgreSQL: postgres://localhost:5432/social_scheduler
      - Redis: redis://localhost:6379/
      - MinIO: http://localhost:9000/

  2_shared_workflows:
    - justification: None - avoiding n8n for performance reasons
    
  3_scenario_integration:
    - scenario_authenticator: OAuth management and user authentication
    - campaign_content_studio: Content generation and campaign management
    - brand_manager: Brand asset application and consistency checking

### Data Models
data_entities:
  - name: User
    storage: postgres
    schema: |
      CREATE TABLE users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        subscription_tier VARCHAR(50) DEFAULT 'free',
        timezone VARCHAR(100) DEFAULT 'UTC',
        created_at TIMESTAMP DEFAULT NOW(),
        updated_at TIMESTAMP DEFAULT NOW()
      );

  - name: SocialAccount
    storage: postgres
    schema: |
      CREATE TABLE social_accounts (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        platform VARCHAR(50) NOT NULL, -- twitter, instagram, linkedin, facebook
        platform_user_id VARCHAR(255) NOT NULL,
        username VARCHAR(255) NOT NULL,
        access_token TEXT NOT NULL,
        refresh_token TEXT,
        token_expires_at TIMESTAMP,
        account_data JSONB DEFAULT '{}',
        is_active BOOLEAN DEFAULT true,
        created_at TIMESTAMP DEFAULT NOW(),
        UNIQUE(user_id, platform, platform_user_id)
      );

  - name: ScheduledPost
    storage: postgres
    schema: |
      CREATE TABLE scheduled_posts (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        campaign_id UUID, -- Optional reference to campaign-content-studio
        title VARCHAR(500) NOT NULL,
        content TEXT NOT NULL,
        platform_variants JSONB DEFAULT '{}', -- Platform-specific adaptations
        media_urls TEXT[] DEFAULT '{}',
        platforms VARCHAR(50)[] NOT NULL,
        scheduled_at TIMESTAMP NOT NULL,
        timezone VARCHAR(100) NOT NULL,
        status VARCHAR(50) DEFAULT 'scheduled', -- scheduled, posted, failed, cancelled
        posted_at TIMESTAMP,
        analytics_data JSONB DEFAULT '{}',
        created_at TIMESTAMP DEFAULT NOW(),
        updated_at TIMESTAMP DEFAULT NOW()
      );

  - name: PostJob
    storage: redis
    schema: |
      {
        "id": "uuid",
        "post_id": "uuid",
        "user_id": "uuid", 
        "platform": "twitter|instagram|linkedin|facebook",
        "content": "optimized content for platform",
        "media_urls": ["url1", "url2"],
        "scheduled_at": "2024-01-01T12:00:00Z",
        "retry_count": 0,
        "max_retries": 3,
        "status": "pending|processing|completed|failed"
      }

### API Endpoints
endpoints:
  - method: POST
    path: /api/v1/posts/schedule
    description: Schedule a new post across multiple platforms
    auth_required: true
    request_body: |
      {
        "title": "Post title",
        "content": "Base content to be adapted per platform",
        "platforms": ["twitter", "instagram", "linkedin"],
        "scheduled_at": "2024-01-01T12:00:00Z",
        "timezone": "America/New_York",
        "campaign_id": "optional-campaign-uuid",
        "media_files": ["file1.jpg", "file2.png"],
        "auto_optimize": true
      }

  - method: GET
    path: /api/v1/posts/calendar
    description: Get scheduled posts for calendar view
    auth_required: true
    query_params:
      - start_date: ISO date string
      - end_date: ISO date string
      - platforms: comma-separated platform filter

  - method: POST  
    path: /api/v1/posts/{id}/optimize
    description: AI-optimize content for specific platforms
    auth_required: true
    request_body: |
      {
        "platforms": ["twitter", "linkedin"],
        "optimization_goals": ["engagement", "reach", "brand_consistency"]
      }

  - method: GET
    path: /api/v1/analytics/overview
    description: Get posting analytics and performance metrics
    auth_required: true
    query_params:
      - period: "7d|30d|90d"
      - platforms: comma-separated platform filter

## 🎨 UI Design Philosophy

### Design Principles
- **Calendar-First Interface**: Visual scheduling with intuitive drag-and-drop functionality
- **Platform-Native Previews**: Accurate preview of how content will appear on each platform
- **Speed-Optimized Workflow**: Minimize clicks from content creation to scheduling
- **Multi-Platform Awareness**: Clear visual distinction between platform requirements and constraints
- **Real-time Feedback**: Live posting status, engagement metrics, and system health indicators

### Visual Design System

#### Color Palette
- **Primary Gradient**: Linear gradient from `#4285f4` to `#34a853` (Google-inspired professional blue to green)
- **Platform Colors**: Platform-specific accent colors (Twitter blue, Instagram gradient, LinkedIn blue, Facebook blue)
- **Background**: Clean white with subtle gray (`#f8f9fa`) for calendar grid
- **Cards**: White with subtle shadow and platform-colored left border
- **Status Indicators**: Green (posted), Orange (scheduled), Red (failed), Gray (draft)

#### Typography
- **Font Family**: Inter (modern, professional, excellent readability)
- **Hierarchy**: 
  - Page titles: 2.5rem, font-weight: 600
  - Section headers: 1.8rem, font-weight: 500  
  - Content text: 1rem, font-weight: 400
  - Captions: 0.875rem, font-weight: 400

#### Layout System
- **Header**: Logo, user menu, quick actions (+ New Post), notifications
- **Sidebar**: Navigation (Calendar, Analytics, Settings), connected accounts overview
- **Main Content**: Calendar view with post cards, detailed post editor modal
- **Responsive**: Mobile-first design with breakpoints at 768px, 1024px, 1440px

### Core Features & UI Specifications

#### 1. Calendar Interface
- **Monthly View**: 7x6 grid with proper month navigation
- **Post Cards**: Platform icons, content preview, scheduled time, status indicator
- **Drag & Drop**: Move posts between dates and times
- **Multi-Platform Posts**: Show as connected cards with platform icons
- **Time Slot Indicators**: Visual grid showing optimal posting times per platform

#### 2. Post Scheduling Modal
- **Content Editor**: Rich text editor with character count per platform
- **Platform Selection**: Checkboxes with live preview for each selected platform  
- **Media Upload**: Drag-and-drop with automatic resizing preview
- **Schedule Controls**: Date/time picker with timezone selector
- **AI Optimization**: Toggle for automatic content adaptation per platform
- **Preview Tabs**: Accurate platform-specific preview (mobile and desktop views)

#### 3. Analytics Dashboard
- **Performance Overview**: Posts published, engagement rates, reach metrics
- **Platform Comparison**: Side-by-side performance across platforms
- **Time Analysis**: Best posting times discovered through data
- **Content Performance**: Top-performing posts with engagement breakdowns

#### 4. Connected Accounts Management
- **Platform Cards**: Visual cards for each connected social platform
- **Connection Status**: Green (active), Yellow (token expiring), Red (disconnected)
- **Account Switching**: For users with multiple accounts per platform
- **Permission Management**: Granular control over posting permissions

## 🔗 Integration Specifications

### Campaign Content Studio Integration
- **Campaign Import**: Directly import generated campaigns with multiple posts
- **Brand Consistency**: Automatically apply brand guidelines from brand-manager
- **Content Templates**: Access campaign content templates for quick scheduling
- **Bulk Operations**: Schedule entire campaign across platforms with one click

### Scenario Authenticator Integration  
- **Multi-Tenant Architecture**: Full SaaS capability with user isolation
- **OAuth Management**: Secure social platform authentication flow
- **Subscription Management**: Freemium model with usage-based limitations
- **Team Collaboration**: Shared scheduling for agency and enterprise users

### Brand Manager Integration
- **Visual Identity**: Automatically apply brand colors, fonts, and logos
- **Content Guidelines**: Enforce brand voice and messaging consistency
- **Asset Library**: Access to brand-approved images and graphics
- **Compliance Checking**: Validate all content against brand guidelines

## 🚀 Success Criteria

### Validation Gates
All gates must pass for successful deployment:
- **Multi-Platform Posting**: Successfully post to Twitter, Instagram, LinkedIn, and Facebook
- **Content Optimization**: AI-powered platform-specific content adaptation working
- **Scheduling Accuracy**: 99.9%+ posts published within 30 seconds of scheduled time
- **Calendar UI**: Intuitive drag-and-drop scheduling with accurate previews
- **Analytics Tracking**: Real-time engagement and performance metrics
- **Integration Testing**: Seamless workflow with campaign-content-studio and brand-manager
- **Load Testing**: Support for 100+ concurrent users scheduling posts

### Business Metrics
- **User Adoption**: 50+ active schedulers within first month
- **Content Volume**: 1,000+ posts scheduled per week
- **Platform Coverage**: Average user connects 3+ social platforms
- **Retention Rate**: 80%+ monthly active user retention
- **Revenue Target**: $5K MRR within 3 months of launch

### Technical Benchmarks
- **API Response Time**: <300ms for 95% of requests
- **Job Processing Speed**: Posts processed and queued in <5 seconds
- **System Reliability**: 99.95% uptime for scheduling system
- **Data Accuracy**: Zero data loss during platform API failures
- **Security Compliance**: Full OAuth 2.0 implementation with encrypted token storage

## 📋 Implementation Phases

### Phase 1: Core Infrastructure (Weeks 1-2)
- PostgreSQL schema design and implementation
- Redis job queue system with retry logic
- Basic Go API with authentication
- Platform API integrations (Twitter, LinkedIn, Facebook, Instagram)
- OAuth flow implementation

### Phase 2: Scheduling Engine (Weeks 3-4)  
- Calendar-based scheduling system
- Content optimization pipeline with Ollama integration
- Media processing and storage system
- Job processing workers with rate limiting
- Basic web UI with calendar view

### Phase 3: Advanced Features (Weeks 5-6)
- Platform-specific preview system
- Analytics dashboard and engagement tracking
- Integration with campaign-content-studio and brand-manager
- Bulk scheduling and CSV import
- Team collaboration features

### Phase 4: Polish & Launch (Weeks 7-8)
- UI/UX refinement and mobile optimization
- Performance optimization and load testing
- Documentation and API reference
- User onboarding flow
- Production deployment and monitoring

## 🔒 Security & Compliance

### Data Protection
- **OAuth Token Encryption**: All social platform tokens encrypted at rest
- **User Data Isolation**: Multi-tenant architecture with strict user separation
- **API Rate Limiting**: Per-user rate limits to prevent abuse
- **Audit Logging**: Complete audit trail for all posting activities
- **GDPR Compliance**: User data export and deletion capabilities

### Platform Compliance
- **API Terms Compliance**: Adherence to each platform's developer terms
- **Content Policy Checking**: Basic content moderation to prevent policy violations  
- **Rate Limit Respect**: Intelligent rate limiting to stay within platform quotas
- **Error Handling**: Graceful failure handling with user notifications

This PRD defines a comprehensive social media scheduling scenario that amplifies Vrooli's content generation capabilities while establishing a foundation for multi-platform distribution intelligence that future scenarios can leverage.