# Contact Book - Social Intelligence Engine

> **Transform every scenario from contact-aware to relationship-intelligent**

Contact Book is a foundational Vrooli scenario that provides sophisticated contact management and social intelligence capabilities to all other scenarios. It's not just an address book—it's a **social graph engine** that enables relationship-aware computing across the entire Vrooli ecosystem.

## 🎯 What It Does

### Core Capabilities
- **Rich Contact Management**: Store comprehensive contact information with metadata, preferences, and social context
- **Relationship Graph**: Model complex social relationships with strength scoring and temporal tracking
- **Social Analytics**: Compute closeness scores, maintenance priorities, and relationship insights
- **Cross-Scenario Integration**: Provide social intelligence to other scenarios via API and CLI
- **Privacy-First Design**: Consent-based data management with granular permission controls

### Intelligence Amplification
Every scenario becomes socially intelligent:
- **Wedding Planner** → Smart seating algorithms using relationship graphs
- **Email Assistant** → Personalized tone based on relationship context  
- **Personal Digital Twin** → Social fabric awareness for better recommendations
- **Birthday Reminders** → Gift suggestions based on interests and relationship strength

## 🚀 Quick Start

### Prerequisites
- PostgreSQL (primary data storage) - automatically managed by Vrooli
- Qdrant (semantic search and embeddings) - optional, falls back to SQL search
- Go 1.21+ (for API compilation) - required for building

### Installation

```bash
# Run the scenario
vrooli scenario run contact-book

# Or set up manually
cd scenarios/contact-book
vrooli setup
./scripts/manage.sh setup
```

### Basic Usage

```bash
# List all contacts
contact-book list

# Search for specific people
contact-book search "sarah chen"

# Add a new contact
contact-book add --name "John Doe" --email "john@example.com" --tags "colleague,friend"

# Create relationships
contact-book connect alice-id bob-id --type friend --strength 0.8

# Check social analytics
contact-book analytics

# Find contacts needing attention
contact-book maintenance --limit 5
```

### Web Profile UI

When this service runs alongside **scenario-authenticator**, it serves a lightweight profile manager at `http://localhost:${API_PORT}/profile`. Point the authenticator UI at this URL (set `CONTACT_BOOK_URL` on the auth UI) so end users can edit the contact record linked to their login.

### For Other Scenarios

```bash
# Get contacts as JSON for programmatic use
contact-book search "wedding guests" --json | jq '.results[].id'

# Find close relationships for personal recommendations
contact-book analytics --json | jq '.analytics[] | select(.overall_closeness_score > 0.7)'

# Get birthday information
contact-book list --json | jq '.persons[] | select(.metadata.birthday)'
```

## 📊 Data Model

### Core Entities

**Person** - Rich contact profiles with:
- Basic info (names, emails, phones, pronouns)
- Temporal metadata (birthday, timezone, preferences)
- Social context (interests, dietary restrictions, accessibility needs)
- Communication preferences (channels, tone, response patterns)
- Computed signals (closeness scores, affinity vectors)

**Relationship** - Graph edges with:
- Relationship type and strength (0.0 - 1.0)
- Temporal tracking (last contact, recency decay)
- Shared interests and affinity scoring
- Introduction context and mutual connections

**Social Analytics** - Computed insights:
- Overall closeness scores
- Maintenance priority rankings
- Communication frequency analysis
- Interest similarity vectors

### Time-Bounded Data
All addresses, organization memberships, and relationships support temporal validity (`valid_from`/`valid_to`), enabling accurate historical relationship modeling.

### Privacy Controls
Granular consent management per person and data type:
- Dietary information consent
- Communication analysis permissions  
- Calendar integration authorization
- Photo storage consent

## 🔌 Integration Examples

### Wedding Planner Integration
```bash
# Get guest relationships for seating
contact-book relationships --json | jq '.relationships[] | select(.strength > 0.6)'

# Find dietary restrictions for catering
contact-book list --json | jq '.persons[] | select(.metadata.dietary_restrictions | length > 0)'

# Identify potential conflicts
contact-book relationships --type "ex-partner" --json
```

### Email Assistant Integration
```bash
# Get communication preferences
contact-book get "$contact_id" --json | jq '.communication_preferences'

# Check relationship strength for tone adjustment
contact-book analytics --person-id "$contact_id" --json | jq '.overall_closeness_score'
```

### Personal Digital Twin Integration
```bash
# Find contacts needing relationship maintenance
contact-book maintenance --json | jq '.[] | select(.maintenance_priority_score > 0.7)'

# Get mutual connections
contact-book relationships "$person_id" --json | jq '.relationships[] | .shared_interests'
```

## 🏗️ Architecture

### Technology Stack
- **API**: Go with Gin framework
- **Database**: PostgreSQL with rich graph modeling
- **Search**: Qdrant for semantic contact search
- **Storage**: MinIO for photos/documents (optional)
- **Cache**: Redis for performance optimization (optional)

### API Endpoints

```
GET    /health                    # Health check
GET    /api/v1/contacts           # List contacts
GET    /api/v1/contacts/:id       # Get specific contact
POST   /api/v1/contacts           # Create contact
POST   /api/v1/search             # Search contacts
GET    /api/v1/relationships      # List relationships
POST   /api/v1/relationships      # Create relationship
GET    /api/v1/analytics          # Social analytics
```

### Database Schema Highlights

```sql
-- Rich person model with JSONB metadata
CREATE TABLE persons (
    id UUID PRIMARY KEY,
    full_name TEXT NOT NULL,
    emails TEXT[],
    metadata JSONB DEFAULT '{}',  -- Birthday, preferences, etc.
    computed_signals JSONB DEFAULT '{}',  -- Analytics
    ...
);

-- Weighted relationship graph  
CREATE TABLE relationships (
    from_person_id UUID REFERENCES persons(id),
    to_person_id UUID REFERENCES persons(id),
    relationship_type TEXT NOT NULL,
    strength DECIMAL(3,2) CHECK (strength BETWEEN 0 AND 1),
    last_contact_date DATE,
    shared_interests TEXT[],
    ...
);

-- Computed social intelligence
CREATE TABLE social_analytics (
    person_id UUID REFERENCES persons(id),
    overall_closeness_score DECIMAL(5,4),
    maintenance_priority_score DECIMAL(3,2),
    affinity_vector JSONB,  -- For similarity calculations
    ...
);
```

## 🧠 Social Intelligence Features

### Relationship Strength Scoring
Computed from multiple signals:
- Communication frequency and recency
- Mutual connections and shared contexts
- Explicit relationship declarations
- Event attendance patterns

### Maintenance Priorities
Identifies relationships needing attention based on:
- Time since last interaction
- Relationship importance (strength score)
- Historical communication patterns
- Life event triggers (birthdays, job changes)

### Affinity Matching
TF-IDF vectors enable:
- "Find people interested in hiking"
- "Who else likes jazz music?"
- Interest-based event recommendations
- Social introduction suggestions

### Communication Intelligence  
Learns preferences from interaction metadata:
- Preferred communication channels
- Response time patterns
- Message length preferences
- Tone adaptation (formal vs casual)

## 🔒 Privacy & Consent

### Privacy-First Design
- **Metadata Only**: Communication history stores patterns, never content
- **Consent Granularity**: Per-person, per-data-type permission controls
- **Data Minimization**: Store only what's necessary for social intelligence
- **Audit Trails**: Complete logging of data access and modifications

### Consent Management
```bash
# Check what data can be stored for a person
contact-book consent check "$person_id"

# Grant specific permissions
contact-book consent grant "$person_id" --scope dietary --scope communication_analysis

# View consent history
contact-book consent history "$person_id"
```

## 📈 Performance & Scale

### Optimization Strategies
- **Materialized Views**: Pre-computed relationship queries
- **Redis Caching**: Frequent contact lookups and analytics
- **Database Indexing**: Optimized for graph traversal queries
- **Batch Processing**: Nightly analytics computation

### Scale Targets
- **10K+ Contacts**: Efficient search and relationship queries
- **100K+ Relationships**: Graph traversal under 100ms
- **50+ Concurrent Scenarios**: Simultaneous API access
- **Sub-200ms Response**: 95% of contact queries

## 🧪 Testing & Validation

### Comprehensive Test Suite
```bash
# Run all integration tests
./test/test-database-integration.sh

# Validate scenario structure
vrooli scenario test contact-book

# Performance benchmarking
./test/performance-test.sh
```

### Validation Criteria
- ✅ All API endpoints functional
- ✅ CLI commands work with JSON output
- ✅ Database schema supports complex queries
- ✅ Cross-scenario integration examples work
- ✅ Privacy controls enforce consent properly

## 🔮 Future Vision

### Planned Enhancements (v2.0)
- **Real-time Analytics**: Event-driven relationship scoring
- **Advanced Graph Algorithms**: Community detection, influence scoring
- **Predictive Intelligence**: Optimal introduction timing, relationship maintenance suggestions
- **Cross-Platform Sync**: LinkedIn, Gmail, phone contact integration

### Long-term Possibilities
- **AI Social Orchestration**: Automated event planning based on relationship optimization
- **Temporal Relationship Modeling**: Relationship evolution prediction
- **Social Sentiment Analysis**: Relationship health monitoring
- **Group Dynamics Intelligence**: Multi-person interaction optimization

## 🤝 Contributing

### Development Setup
```bash
cd scenarios/contact-book
go mod download  # Install Go dependencies
./cli/install.sh  # Install CLI globally
vrooli setup      # Initialize resources
```

### Adding Features
1. Update database schema in `initialization/postgres/schema.sql`
2. Add API endpoints in `api/main.go`
3. Extend CLI commands in `cli/contact-book`
4. Add integration tests in `test/`
5. Update PRD.md with new capabilities

## 📚 Resources

- **[PRD.md](./PRD.md)**: Comprehensive product requirements
- **[API Documentation](./api/README.md)**: Detailed endpoint specifications
- **[Schema Documentation](./initialization/postgres/schema.sql)**: Database design rationale
- **[CLI Reference](./cli/contact-book)**: Complete command documentation

## 🌟 Key Benefits

### For Users
- **Unified Social Intelligence**: One source of truth for all relationship data
- **Privacy Control**: Granular consent management with transparent data practices
- **Cross-Scenario Value**: Every scenario becomes socially aware and personalized

### For Developers  
- **Rich API**: Comprehensive social intelligence capabilities via REST
- **CLI Integration**: Easy programmatic access for other scenarios
- **Extensible Design**: Plugin architecture for custom analytics and integrations

### For the Vrooli Ecosystem
- **Compound Intelligence**: Each scenario interaction enriches the social graph
- **Recursive Learning**: Better relationships → smarter scenarios → better relationships
- **Network Effects**: Value increases exponentially with adoption across scenarios

---

**Contact Book transforms Vrooli from a collection of AI tools into a socially-intelligent system that understands and optimizes human relationships.**

🚀 **Ready to make every scenario relationship-aware? Get started with `vrooli scenario run contact-book`**
