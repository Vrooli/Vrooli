# AI Routine Creation System

This directory contains the AI-powered routine creation pipeline for Vrooli. It enables systematic generation, staging, and validation of routine definitions using AI assistance.

## Overview

The AI routine creation system provides a structured workflow for:
- **Ideation**: Collecting routine ideas and requirements in a backlog
- **Generation**: Using AI to convert ideas into complete routine definitions
- **Staging**: Organizing generated routines for review and testing
- **Validation**: Importing and testing routines in the local development environment

## Directory Structure

```
docs/ai-creation/routine/
├── README.md                    # This file
├── prompt.md                    # AI routine generation instructions
├── backlog.md                   # Queue of routine ideas to process
├── routine-reference.json       # Complete ID/name/type reference for all routines (JSON format)
├── staged/                      # Generated routine definitions ready for import
│   ├── subroutines/            # Generated reusable subroutines
│   ├── main-routines/          # Main routines using subroutines
│   └── [category folders]      # Categorized routine JSON files
├── cache/                       # Resolution system cache
│   ├── search-results.json     # Cached semantic search results
│   ├── staged-index.json       # Index of staged subroutines
│   └── resolution-map.json     # Capability to subroutine mappings
└── templates/                   # Common patterns (future)

scripts/
├── ai-creation/                   # AI-powered content generation and validation
│   ├── maintenance-agent.sh       # AI agent for maintenance tasks
│   ├── maintenance-supervisor.sh  # Supervisor for AI maintenance workflows
│   ├── routine-generate.sh        # Generate routines using AI
│   ├── routine-import.sh          # Import and validate staged routines
│   ├── routine-reference-generator.sh # Generate routine-reference.json from staged files
│   ├── validate-routine.sh        # Validate routine JSON structure and configuration
│   └── validate-subroutines.sh    # Validate subroutine references in routine files
└── main/
    ├── routine-generate-enhanced.sh # Smart multi-pass generation (recommended)
    └── routine-generate-direct.sh   # Direct prompt generation for manual use
```

## Quick Start

### 1. Add Routine Ideas
Edit `backlog.md` to add new routine concepts:

```markdown
### Your Routine Name
- **Category**: Productivity
- **Description**: What the routine does and why it's valuable
- **Inputs**: Required input data
- **Outputs**: Expected outputs
- **Strategy**: conversational|reasoning|deterministic
- **Priority**: High|Medium|Low
- **Notes**: Additional context
```

### 2. Generate Routines

#### Option A: Direct Generation with Claude (Recommended)
```bash
# Generate prompt for manual use with Claude
./scripts/main/routine-generate.sh --direct

# Or use the dedicated direct script with more options
./scripts/main/routine-generate-direct.sh --prompt-only --subroutines
```

This generates a prompt that you can copy and paste directly to Claude (web interface or Claude Code) without going through the maintenance-agent.sh script. This approach gives you the most control and visibility into the generation process. Options:
- `--prompt-only`: Generate and display the prompt
- `--subroutines`: Include subroutine discovery in the prompt
- `--output FILE`: Save prompt to a file
- `--validate`: Show validation instructions

#### Option B: Enhanced Generation with Smart Subroutine Resolution
```bash
./scripts/main/routine-generate-enhanced.sh
```

This enhanced system provides:
- **Semantic search** for existing subroutines to avoid duplication
- **Multi-pass generation** that creates missing subroutines automatically  
- **Smart dependency resolution** with proper reuse of components
- **Staged file scanning** to reuse already-generated subroutines
- **Hierarchical organization** with separate subroutines and main routines

#### Option C: Basic Generation with Claude Code CLI (Automated)
```bash
./scripts/ai-creation/routine-generate.sh
```

This calls `maintenance-agent.sh` with the AI creation workflow prompt to automatically read the backlog, generate complete JSON definitions, and save them to `staged/` (may include TODO placeholders for subroutines).

### 3. Import and Test
Import generated routines into your local Vrooli instance:

#### Option A: Using the CLI (Recommended)
```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Build and install the CLI (first time only)
cd packages/cli
pnpm install
pnpm run build
npm link
cd ../..

# Import routines using CLI
vrooli auth login  # First time only

# Import all routines
vrooli routine import-dir ./docs/ai-creation/routine/staged/

# Or import by category
vrooli routine import-dir ./docs/ai-creation/routine/staged/productivity/
vrooli routine import-dir ./docs/ai-creation/routine/staged/personal/
```

#### Option B: Using Shell Script (Automated)
```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Import routines (automatically builds CLI if needed)
./scripts/ai-creation/routine-import.sh
```

### 4. Validate Subroutine References

Before importing routines, it's recommended to validate that all subroutine references are correct:

```bash
# Quick check for TODO placeholders (fastest)
./scripts/ai-creation/validate-subroutines.sh --todo-only

# Full validation: check TODOs, ID formats, and existence
./scripts/ai-creation/validate-subroutines.sh

# Validate specific file with detailed output
./scripts/ai-creation/validate-subroutines.sh --verbose docs/ai-creation/routine/staged/productivity/task-prioritizer.json

# Validate specific directory
./scripts/ai-creation/validate-subroutines.sh --directory docs/ai-creation/routine/staged/personal/

# List all available subroutines for reference
./scripts/ai-creation/validate-subroutines.sh --list-available

# Get help and see all options
./scripts/ai-creation/validate-subroutines.sh --help
```

#### What Gets Validated:
- **TODO Placeholders**: Identifies `"subroutineId": "TODO: ..."` that need real IDs
- **ID Format**: Ensures subroutine IDs are valid 19-digit snowflake IDs
- **ID Existence**: Verifies referenced subroutines exist in `routine-reference.json`
- **Multi-step Routines**: Extracts subroutine references from graph configurations

#### Common Issues and Fixes:
```bash
# Replace TODO placeholders with actual routine IDs
# Use these commands to find suitable subroutines:

# Search for subroutines by functionality
jq '.routines[] | select(.name | test("data analysis"; "i"))' routine-reference.json

# List all available routine IDs
jq -r '.routines[].id' routine-reference.json

# Find routines by category
jq '.byCategory.productivity.routines' routine-reference.json
```

### 5. Update Routine Reference (After Adding New Routines)

When new routine files are added to the `staged/` directory, update the reference file:

```bash
# Regenerate the routine reference with latest files
./scripts/ai-creation/routine-reference-generator.sh

# Or specify custom output location
./scripts/ai-creation/routine-reference-generator.sh docs/ai-creation/routine/my-reference.md
```

This creates/updates `docs/ai-creation/routine/routine-reference.json` with:
- All routine IDs, names, types, and descriptions in structured JSON format
- Routines grouped by category and type for easy filtering
- Built-in usage examples for jq queries
- Metadata including generation timestamp and total count

**Note**: Always regenerate the reference after adding new routine files to ensure subroutine builders have access to the latest routine IDs.

#### JSON Query Examples:
```bash
# List all routine types
jq '.byType | keys' docs/ai-creation/routine/routine-reference.json

# Find routine by exact ID
jq '.routines[] | select(.id == "7829564732190847634")' docs/ai-creation/routine/routine-reference.json

# Search routines by name pattern (case-insensitive)
jq '.routines[] | select(.name | test("habit"; "i"))' docs/ai-creation/routine/routine-reference.json

# Get all multi-step routines
jq '.byType.RoutineMultiStep' docs/ai-creation/routine/routine-reference.json

# List routines in productivity category
jq '.byCategory.productivity' docs/ai-creation/routine/routine-reference.json

# Count routines by type
jq '.byType | to_entries | map({type: .key, count: (.value | length)})' docs/ai-creation/routine/routine-reference.json
```

## Prerequisites

### For Generation (`routine-generate.sh`)
- `maintenance-agent.sh` available in project root
- Properly configured AI model access

### For Import
#### Using CLI (Recommended)
- Local Vrooli development environment running
- API accessible at `http://localhost:5329`
- Node.js and pnpm installed

#### Using Shell Script
- Local Vrooli development environment running
- API accessible at `http://localhost:5329`
- Test user account configured
- `curl` and `jq` installed

## Environment Variables

### For CLI
The CLI stores credentials securely in `~/.vrooli/config.json`. No environment variables needed.

### For Shell Script
Configure these environment variables for the import script:

```bash
export VROOLI_TEST_EMAIL="your-test@example.com"
export VROOLI_TEST_PASSWORD="your-test-password"
```

If not set, defaults to `test@example.com` / `password`.

## CLI vs Shell Script

### Why Use the CLI?

The Vrooli CLI provides several advantages over the shell scripts:

1. **Better Authentication**: Secure token storage with automatic refresh
2. **Progress Tracking**: Real-time progress bars and status updates
3. **Error Handling**: Detailed error messages and recovery
4. **Multi-Environment**: Easy switching between local/staging/production
5. **Validation**: Client-side validation before server calls
6. **Batch Operations**: Efficient handling of multiple routines

### CLI Commands for Routine Management

```bash
# Search for routines using semantic similarity (for finding subroutines)
vrooli routine search "analyze user behavior patterns" --format ids

# Discover all available routines for use as subroutines
vrooli routine discover --format mapping

# Validate routines before import
vrooli routine validate ./staged/my-routine.json

# Import with progress tracking
vrooli routine import-dir ./staged/ --fail-fast

# List imported routines
vrooli routine list --search "ai-generated"

# Export routine for backup
vrooli routine export <routine-id> -o backup.json

# Test routine execution
vrooli routine run <routine-id> --watch
```

## Workflow Details

### Routine Generation Process

1. **AI Workflow**: `routine-generate.sh` calls `maintenance-agent.sh` with a structured prompt
2. **Automatic Processing**: AI reads `backlog.md`, selects first unprocessed item
3. **Dynamic Subroutine Discovery**: 
   - If CLI is authenticated: Automatically discovers all available subroutines and saves to `available-subroutines.txt`
   - AI reads this file to find real subroutine IDs based on functionality needed
   - If unavailable: AI adds TODO comments where manual subroutine selection is needed
4. **Generation**: AI follows `prompt.md` instructions to create complete routine JSON with actual subroutine IDs
5. **Staging**: Generated routine is saved to `staged/` with descriptive filename
6. **Enhanced Validation**: 
   - Structure validation: Checks JSON schema, required fields, ID formats
   - Subroutine validation: Verifies all referenced subroutines exist in the database
   - TODO detection: Flags any placeholder subroutine references
7. **Backlog Management**: AI updates `backlog.md` to mark item as processed

### Import and Validation Process

1. **CLI Auto-Build**: Automatically builds and installs Vrooli CLI if not found
2. **Authentication Check**: Verifies CLI authentication status and prompts login if needed
3. **File Discovery**: Counts and lists JSON files in staged directory
4. **Pre-Import Validation**: Validates all routine JSON files before attempting import
5. **Batch Import**: Imports all valid routines using CLI with fail-fast mode
6. **Success Reporting**: Shows recently imported routines and provides feedback

## Routine Categories

### Enhanced Category Framework

Based on analysis of 98 routines (75 main + 23 subroutines), the system supports these comprehensive categories:

#### 🧠 **Meta-Intelligence & Systems**
- Memory maintenance, capability analysis, anomaly detection
- Strategy selection and optimization
- Self-reflection and continuous improvement

#### 🎯 **Personal Excellence**
- Introspective development, habit formation, wellness
- Mindfulness, personal branding, life planning

#### 💼 **Professional Mastery**
- Career navigation, interview prep, performance optimization
- Leadership development, team building, onboarding

#### 🚀 **Innovation & Creativity**
- Problem-solving frameworks, brainstorming facilitation
- Creative mentorship, design thinking, innovation workshops

#### 📊 **Research & Intelligence**
- Multi-source synthesis, fact-checking, competitive analysis
- Market research, documentation generation, knowledge processing

#### 💻 **Technical Excellence**
- Code generation/review, debugging, API documentation
- Data visualization, system integration, automation

#### 💰 **Financial Intelligence**
- Portfolio analysis, budget optimization, financial planning
- Investment strategy, economic modeling

#### 🎯 **Strategic Decision-Making**
- Crisis management, risk assessment, decision frameworks
- SWOT analysis, scenario planning, resource allocation

#### ⚡ **Productivity Engineering**
- Task prioritization, time blocking, workflow optimization
- Project management, meeting facilitation, review systems

#### 📚 **Learning Acceleration**
- Adaptive learning paths, study optimization, skill assessment
- Language learning, research assistance, knowledge retention

#### 📝 **Communication Mastery**
- Email campaigns, report writing, content creation
- Meeting summaries, social media planning, documentation

#### 🏗️ **Infrastructure & Operations**
- System monitoring, process optimization, team coordination
- Resource management, capability mapping, gap analysis

### Meta-Level Categories

#### 🔄 **Meta-Routine Engineering**
- Routine schema builders and validators
- Routine optimization (conversational → deterministic/API conversion)
- Routine performance analyzers and success rate monitoring
- Routine migration tools for schema updates
- Routine dependency mappers and relationship optimization
- Self-improving routine generation systems

#### 🤖 **AI Agent Orchestration**
- Task decomposition engines for breaking large tasks into manageable chunks
- Progress checkpoint managers with state persistence
- Regression prevention frameworks and quality gate controllers
- Agent session managers with time-boxed work cycles
- Automated testing loops and rollback mechanisms
- Agent memory persistence across session boundaries

#### 💼 **Automation Service Delivery**
- Client requirements analyzers and technical scope estimators
- API development pipelines and agent development frameworks
- Workflow automation builders for business processes
- Deliverable quality assurance and project documentation generators
- Proposal generation and project management automation
- Revenue-generating automation service templates

### Executive & Managerial Categories

#### 👔 **Executive Leadership & Governance**
- CEO decision frameworks and strategic vision synthesis
- Board meeting orchestration, agenda creation, governance compliance
- Executive performance reviews and succession planning
- Crisis leadership protocols and stakeholder communication
- Strategic vision alignment and company-wide goal cascading
- Executive calendar optimization and stakeholder access management

#### 📊 **Product Strategy & Development**
- Product roadmap architects and market-driven feature prioritization
- Product-market fit analyzers and customer feedback synthesis
- Product launch orchestrators and go-to-market strategy builders
- Product performance dashboards and KPI monitoring
- Competitive product intelligence and market positioning
- Product portfolio optimizers and lifecycle management

#### ⚡ **Agile & Scrum Leadership**
- Sprint master coordinators and sprint planning automation
- Scrum health monitors and team velocity tracking
- Agile transformation guides and team coaching frameworks
- Release planning engines and multi-team coordination
- Agile metrics analyzers and process optimization
- Stakeholder alignment tools and demo coordination

#### 💼 **Business Development & Partnerships**
- Partnership pipeline managers and opportunity identification
- Deal structure optimizers and contract negotiation frameworks
- Market expansion strategists and geographic analysis
- Revenue growth engines and channel development
- Strategic alliance builders and partnership evaluation
- Business model innovation and revenue stream analysis

#### 🏢 **Organizational Design & Operations**
- Org chart optimizers and reporting relationship analysis
- Culture development frameworks and values alignment
- Change management orchestrators and transformation planning
- Performance management systems and goal setting automation
- Talent pipeline builders and succession planning
- Operational excellence monitors and process optimization

#### 💰 **Financial Strategy & Planning**
- CFO dashboard builders and financial health monitoring
- Budget planning engines and resource allocation
- Investment decision frameworks and ROI analysis
- Financial risk assessors and mitigation strategy development
- Investor relations managers and pitch deck generation
- M&A analysis tools and due diligence frameworks

#### 🎯 **Strategic Planning & Execution**
- Strategy development workshops and SWOT analysis
- OKR management systems and objective setting
- Strategic initiative trackers and project portfolio monitoring
- Market intelligence synthesizers and competitive analysis
- Strategic communication builders and strategy rollout
- Performance review aggregators and company-wide synthesis

#### 🤝 **Stakeholder Management & Communications**
- Stakeholder mapping tools and influence analysis
- Executive communication builders and all-hands preparation
- Investor update generators and performance summaries
- Crisis communication managers and media response
- Customer success orchestrators and executive relationship management
- Board relations optimizers and governance compliance

#### 🚀 **Innovation & Growth Management**
- Innovation pipeline managers and idea evaluation
- Growth strategy builders and market opportunity analysis
- Digital transformation leaders and technology adoption
- Acquisition integration managers and post-merger integration
- Venture investment analyzers and startup evaluation
- Ecosystem development tools and platform strategy

### Customer Relationship Management (CRM) Categories

#### 🎯 **Customer Lifecycle Management**
- Customer journey orchestration and touchpoint optimization
- Customer onboarding automation and welcome sequences
- Customer success management and churn prevention
- Customer retention strategies and loyalty program management
- Customer lifecycle analytics and lifetime value calculation
- Customer milestone tracking and achievement recognition

#### 📊 **Sales Pipeline & Process Automation**
- Lead management systems and lead scoring automation
- Sales process orchestration and deal stage advancement
- Proposal & quote generation with dynamic pricing
- Sales performance analytics and pipeline health monitoring
- Territory & account management optimization
- Sales enablement tools and performance coaching

#### 💙 **Customer Experience & Support**
- Customer support automation and ticket routing
- Customer feedback management and sentiment analysis
- Customer communication orchestration across channels
- Customer self-service optimization and chatbot coordination
- Customer experience analytics and satisfaction tracking
- Customer advocacy programs and referral management

#### 🔍 **Customer Intelligence & Analytics**
- Customer data integration and profile enrichment
- Customer behavior analytics and predictive modeling
- Customer segmentation engines and persona development
- Customer health monitoring and risk assessment
- Customer insight generation and voice of customer analysis
- Customer privacy management and compliance tracking

#### 🎨 **Marketing Automation & Campaigns**
- Campaign management systems and multi-channel orchestration
- Content personalization and dynamic delivery engines
- Marketing attribution and touch point analysis
- Lead nurturing automation and behavior-triggered sequences
- Event marketing management and attendee coordination
- Marketing operations and workflow automation

#### 💰 **Revenue Operations & Growth**
- Revenue forecasting and pipeline prediction
- Pricing optimization and competitor analysis
- Upsell/cross-sell automation and opportunity identification
- Customer acquisition cost management and channel effectiveness
- Revenue attribution and growth driver analysis
- Subscription management and renewal automation

### Modern Lifestyle & Specialized Categories

#### 🎬 **Digital Content Creation & Creator Economy**
- Video content production (TikTok, YouTube, video essays)
- Visual content design (thumbnails, social graphics, video editing)
- Content strategy & planning (calendars, trend analysis, viral optimization)
- Creator monetization (revenue streams, brand partnerships, merchandise)
- Audience development (community building, engagement, growth strategies)
- Platform optimization (algorithms, posting schedules, cross-platform distribution)

#### 🎓 **E-Learning & Online Education**
- Course development (curriculum design, learning objectives, assessments)
- Student engagement (interactive content, progress tracking, motivation)
- Educational content creation (lesson planning, multimedia, accessibility)
- Learning analytics (performance analysis, completion optimization)
- Educational technology (LMS integration, tool selection, virtual classrooms)
- Certification & credentialing (assessment design, verification, validation)

#### 🎮 **Gaming & Esports**
- Competitive gaming strategy (team composition, meta analysis, tournaments)
- Content creator gaming (stream planning, highlights, content optimization)
- Esports management (team coordination, tournaments, sponsorships)
- Game development support (feedback analysis, beta testing, player behavior)
- Gaming community management (Discord, fan engagement, events)
- Gaming monetization (streaming revenue, sponsorships, merchandise)

#### 🧠 **Mental Health & Wellness**
- Stress management (assessment, coping strategies, relaxation techniques)
- Mental health monitoring (mood tracking, trigger identification, check-ins)
- Therapy support (session prep, goal tracking, progress monitoring)
- Workplace mental health (burnout prevention, work-life balance, team wellness)
- Crisis support systems (resource identification, support networks, interventions)
- Wellness program design (initiatives, assistance programs, challenges)

#### 👨‍👩‍👧‍👦 **Parenting & Family Management**
- Child development tracking (milestones, activities, educational support)
- Family schedule coordination (activities, school, calendar optimization)
- Parenting resource management (material curation, expert advice, decisions)
- Family financial planning (education savings, budgeting, expense tracking)
- Family communication (conflict resolution, meetings, skill building)
- Special needs support (resources, therapy coordination, advocacy)

#### 🏠 **Home Improvement & DIY**
- Project planning (renovation, contractor coordination, permits)
- DIY guidance (instructions, tool recommendations, safety protocols)
- Home maintenance (seasonal schedules, preventive care, repair prioritization)
- Smart home integration (device coordination, automation, energy optimization)
- Home value optimization (improvement ROI, market tracking, selling prep)
- Sustainability integration (efficiency upgrades, materials, environmental impact)

#### 🐕 **Pet Care & Animal Management**
- Pet health management (vaccination tracking, health monitoring, veterinary)
- Pet behavior training (programs, modification, socialization planning)
- Pet care scheduling (feeding, exercise planning, grooming coordination)
- Pet emergency preparedness (contacts, first aid, disaster planning)
- Pet travel & boarding (preparation, selection, pet-friendly planning)
- Multi-pet household (introductions, resource management, harmony)

#### 📚 **Instructional Content Creation**
- Tutorial & guide creation (step-by-step instructions, visual guides, process documentation)
- How-to content development (practical guides, troubleshooting instructions, user manuals)
- Training material development (onboarding guides, skill-building modules, competency frameworks)
- Documentation systems (user manuals, FAQ creation, knowledge base organization)
- Interactive learning tools (worksheets, exercises, hands-on activities, simulation design)
- Knowledge transfer optimization (instructional methodology, content accessibility, learning styles)

#### 📊 **Assessment & Evaluation Design**
- Quiz & test generation (knowledge checks, competency assessments, certification prep)
- Questionnaire development (surveys, feedback forms, data collection instruments)
- Evaluation frameworks (performance reviews, skill assessments, progress tracking)
- Research instruments (survey design, interview guides, data collection protocols)
- Quality assurance tools (compliance checklists, audit frameworks, validation tests)
- Measurement & analytics (assessment scoring, performance metrics, outcome tracking)

### Additional Potential Categories

#### 🏥 **Health & Medical Support**
- Symptom pattern analysis (non-diagnostic), health data trends
- Medical appointment preparation, health insurance navigation
- Clinical trial matching assistance

#### 🎮 **Gaming & Entertainment**
- Game strategy optimization, character build planners
- Tournament bracket analysis, virtual economy analyzers
- Game narrative generators, speedrun route optimization

#### 🏛️ **Legal & Compliance**
- Contract clause analysis, regulatory compliance checklists
- Legal document summarization, IP portfolio management
- Privacy policy generators, compliance audit frameworks

#### 🎨 **Art & Design**
- Color palette generators, design critique frameworks
- Art movement analyzers, portfolio curation assistants
- Creative brief generators, design system auditors

#### 🌱 **Agriculture & Food**
- Crop rotation planners, recipe nutrition optimizers
- Food waste reduction analyzers, farm resource management
- Seasonal planting guides, food supply chain trackers

#### 🚗 **Transportation & Logistics**
- Route optimization algorithms, fleet management analyzers
- Delivery scheduling systems, traffic pattern predictors
- Vehicle maintenance schedulers, supply chain bottleneck detectors

#### 🏘️ **Community & Social Impact**
- Community needs assessments, volunteer coordination systems
- Social impact measurement, neighborhood resource mapping
- Civic engagement frameworks, mutual aid network builders

#### 🔬 **Science & Research**
- Experiment design assistants, literature review synthesizers
- Hypothesis generators, data collection frameworks
- Research collaboration tools, grant proposal assistants

#### 🏠 **Real Estate & Property**
- Property valuation analyzers, rental yield calculators
- Neighborhood comparison tools, maintenance schedule generators
- Investment property analyzers, tenant screening frameworks

#### 🎭 **Performance & Entertainment**
- Performance preparation routines, audience engagement analyzers
- Set list optimizers, tour planning assistants
- Media kit generators, fan engagement strategies

#### ⚡ **Energy & Utilities**
- Energy consumption analyzers, renewable energy calculators
- Utility bill optimizers, smart home configuration
- Energy audit frameworks, grid stability monitors

#### 🛡️ **Security & Privacy**
- Digital footprint analyzers, privacy audit tools
- Security posture assessments, incident response frameworks
- Threat modeling assistants, data breach response plans

#### 🌐 **Cultural & Language**
- Cultural adaptation guides, translation quality checkers
- Localization frameworks, cross-cultural communication
- Language exchange matchers, cultural event planners

#### 🎯 **Sports & Fitness**
- Training program generators, performance metric analyzers
- Nutrition plan optimizers, recovery protocol builders
- Competition preparation, team formation optimizers

#### 💑 **Relationships & Social**
- Relationship milestone planners, conflict resolution frameworks
- Social skill builders, dating profile optimizers
- Family event coordinators, friendship maintenance systems

#### 🎓 **Academic & Scholarly**
- Citation management systems, peer review assistants
- Conference abstract writers, academic CV builders
- Research collaboration matchers, thesis structure planners

#### 🛍️ **Retail & E-commerce**
- Product recommendation engines, inventory optimization
- Customer journey analyzers, pricing strategy tools
- Review sentiment analysis, market trend predictors

#### 🌊 **Marine & Ocean**
- Tide prediction systems, marine navigation aids
- Ocean conservation trackers, fishing regulation guides
- Marine ecosystem monitors, coastal erosion analyzers

#### 🚀 **Space & Astronomy**
- Stargazing planners, satellite tracking systems
- Space mission analyzers, astronomical event alerts
- Space weather monitors, orbital mechanics calculators

## File Formats

### Backlog Items (`backlog.md`)
Simple markdown format with structured metadata for each routine idea.

### Staged Routines (`staged/*.json`)
Complete Vrooli routine definitions following the database schema:

```json
{
  "id": "1234567890123456789",
  "publicId": "routine-name-v1",
  "resourceType": "Routine",
  "versions": [
    {
      "id": "1234567890123456790",
      "resourceSubType": "RoutineMultiStep",
      "config": { /* Complete routine configuration */ },
      "translations": [ /* Name, description, instructions */ ]
    }
  ]
}
```

## Key Insights for Future Routine Creation

When creating new routines, follow these proven patterns and principles:

### 1. **Composability is King**
- Focus on creating modular subroutines that can be mixed and matched
- Design for reusability across different contexts
- Keep subroutines focused on single responsibilities

### 2. **Strategy Alignment**
- Use **conversational** for human-centric, adaptive tasks (coaching, creative work)
- Use **reasoning** for analysis, synthesis, and creative problem-solving
- Use **deterministic** for calculations, rules, and predictable workflows

### 3. **Power Patterns**
- **Multi-step workflows**: Build on intermediate results for complex outcomes
- **Parallel execution**: Run independent analyses simultaneously
- **Context preservation**: Maintain coherence across workflow steps
- **Rich outputs**: Capture multiple perspectives and formats

### 4. **Untapped Potential**
- **Cross-domain routines**: Combine technical + financial analysis, etc.
- **Adaptive routines**: Modify execution based on intermediate results
- **Collaborative routines**: Designed for team/swarm execution
- **Real-time monitoring**: Continuous adjustment and optimization

### 5. **Design Principles**
- **Start simple**: Begin with basic functionality, then enhance
- **Think workflows**: How will this routine compose with others?
- **User-centric**: Clear inputs, actionable outputs, helpful guidance
- **Error-resilient**: Handle edge cases and provide fallback options

## Quality Assurance

### Generated Routine Validation

Use `./scripts/ai-creation/validate-routine.sh` for structural validation and `./scripts/ai-creation/validate-subroutines.sh` for subroutine reference validation:

```bash
# Validate JSON structure, fields, and configuration
./scripts/ai-creation/validate-routine.sh docs/ai-creation/routine/staged/your-routine.json

# Validate subroutine references and TODO placeholders
./scripts/ai-creation/validate-subroutines.sh docs/ai-creation/routine/staged/your-routine.json

# Quick validation workflow
./scripts/ai-creation/validate-subroutines.sh --todo-only  # Check for TODOs
./scripts/ai-creation/validate-routine.sh docs/ai-creation/routine/staged/*.json    # Validate structure
```

**Validation Checklist:**
- ✅ Valid JSON structure (`scripts/ai-creation/validate-routine.sh`)
- ✅ Required fields present (`scripts/ai-creation/validate-routine.sh`)
- ✅ Correct ID formats (19-digit snowflake IDs) (`scripts/ai-creation/validate-routine.sh`)
- ✅ Valid subroutine references (`scripts/ai-creation/validate-subroutines.sh`)
- ✅ No TODO placeholders (`scripts/ai-creation/validate-subroutines.sh --todo-only`)
- ✅ Proper data flow mapping (`scripts/ai-creation/validate-routine.sh`)
- ✅ Complete form definitions (`scripts/ai-creation/validate-routine.sh`)

### Runtime Validation
- ✅ Successful API import
- ✅ Database storage
- ✅ Execution smoke tests
- ✅ Error handling verification

## Troubleshooting

### Generation Issues
- **"Could not find maintenance-agent.sh"**: Ensure you're in the project root and the script exists
- **AI generation fails**: Check AI model configuration and token limits
- **Empty staged directory**: Verify backlog has unprocessed items

### Import Issues (CLI)
- **"Vrooli CLI not found"**: Build and install the CLI (see Quick Start section)
- **"Not authenticated"**: Run `vrooli auth login` to authenticate
- **"No response from server"**: Check if API is running on port 5329
- **"Validation failed"**: Run `vrooli routine validate <file>` for detailed errors
- **"Import failed"**: Check server logs and use `--debug` flag for details

### Import Issues (Shell Script)
- **"API is not responding"**: Start local development environment
- **"Authentication failed"**: Check test user credentials exist in local database
- **"Import failed"**: Review routine JSON structure and validation errors
- **"Routine execution test failed"**: Check subroutine references and data mappings

### Validation Issues
- **"TODO placeholders found"**: Use `./scripts/ai-creation/validate-subroutines.sh --todo-only` to find them, then replace with real IDs
- **"Subroutine ID not found"**: Use `./scripts/ai-creation/validate-subroutines.sh --list-available` to see available routines
- **"Invalid ID format"**: Ensure subroutine IDs are 19-digit numeric strings

### Common Fixes
```bash
# Restart development environment
./scripts/main/develop.sh --target docker --detached yes

# Check API health (CLI method)
vrooli auth status

# Check API health (manual method)
curl http://localhost:5329/health

# CLI: Debug authentication issues
vrooli auth login --debug

# CLI: Validate routine before import
vrooli routine validate staged/your-routine.json

# CLI: Test import with dry run
vrooli routine import staged/your-routine.json --dry-run

# Verify test user exists
docker compose exec db psql -U postgres -d vrooli -c "SELECT email FROM users WHERE email = 'test@example.com';"

# Validate routine structure and subroutines
./scripts/ai-creation/validate-routine.sh docs/ai-creation/routine/staged/your-routine.json
./scripts/ai-creation/validate-subroutines.sh docs/ai-creation/routine/staged/your-routine.json

# Quick subroutine validation workflow
./scripts/ai-creation/validate-subroutines.sh --todo-only               # Find TODO placeholders
./scripts/ai-creation/validate-subroutines.sh --list-available          # See available subroutines
jq '.routines[] | select(.name | test("pattern"; "i"))' docs/ai-creation/routine/routine-reference.json  # Search by name

# Check routine JSON structure
jq . staged/your-routine.json
```

## Contributing

When adding new routine ideas to the backlog:

1. **Be Specific**: Provide clear inputs, outputs, and use cases
2. **Choose Appropriate Strategy**: Match the execution strategy to the routine's purpose
3. **Consider Reusability**: Design routines that can be used in multiple contexts
4. **Follow Naming Conventions**: Use descriptive, lowercase names with hyphens

When modifying the system:

1. **Test End-to-End**: Verify the complete pipeline from backlog to import
2. **Update Documentation**: Keep this README current with any changes
3. **Validate Generated Routines**: Ensure they import and execute successfully
4. **Consider Error Handling**: Add robust error checking and user feedback

## Related Documentation

- [`prompt.md`](./prompt.md) - Complete routine generation instructions
- [`backlog.md`](./backlog.md) - Current routine ideas and requirements
- [`/docs/plans/routine-generation-prompt.md`](../../plans/routine-generation-prompt.md) - Original prompt development
- [`/docs/README.md`](../../README.md) - General documentation index