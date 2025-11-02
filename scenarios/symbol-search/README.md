# Symbol Search

> **High-performance Unicode and ASCII symbol search engine with advanced filtering and cross-scenario API integration**

Symbol Search is a comprehensive utility scenario that provides instant access to 140K+ Unicode characters through a performant search interface, RESTful API, and command-line tools. Built for developer productivity and cross-scenario integration.

## 🎯 Core Capabilities

- **Universal Symbol Access**: Search 140K+ Unicode characters by name, category, block, or properties
- **Performance-Optimized**: Sub-50ms search queries with virtualized UI rendering for large datasets  
- **Cross-Scenario Integration**: RESTful API and CLI enable other scenarios to programmatically access symbol data
- **Advanced Filtering**: Filter by Unicode categories, blocks, versions, and character properties
- **Developer-Friendly**: Multiple output formats (Unicode, decimal, HTML entities, CSS, JavaScript)

## 🚀 Quick Start

### Prerequisites

- PostgreSQL database (managed by Vrooli resources)
- Go 1.21+ (for API compilation)
- Node.js 16+ (for UI development)
- curl and jq (for CLI functionality)

### Setup

```bash
# Initialize the scenario (includes database setup, API compilation, CLI installation)
vrooli scenario setup symbol-search

# Start the services
vrooli scenario run symbol-search
```

The API will be available at `http://localhost:15000` and the UI at `http://localhost:3000`.

### Basic Usage

**Web Interface:**
- Navigate to `http://localhost:3000`
- Search by name: "heart", "star", "arrow"
- Search by codepoint: "U+1F600", "128512"
- Apply filters for precise results

**Command Line:**
```bash
# Search for symbols
symbol-search search "heart" --category So

# Get character details  
symbol-search character U+1F600

# List categories and blocks
symbol-search categories
symbol-search blocks

# Get character ranges (for test generation)
symbol-search range U+2600 U+26FF
```

**API Integration:**
```bash
# Search symbols
curl "http://localhost:15000/api/search?q=heart&category=So&limit=10"

# Get character details
curl "http://localhost:15000/api/character/U+2764"

# Bulk character ranges
curl -X POST "http://localhost:15000/api/bulk/range" \
  -H "Content-Type: application/json" \
  -d '{"ranges": [{"start": "U+2600", "end": "U+26FF"}]}'
```

## 🏗️ Architecture

### Components

- **Go API Server**: High-performance REST API with PostgreSQL integration
- **React UI**: Virtualized interface optimized for large datasets
- **CLI Tool**: Bash-based command-line interface with full API parity
- **PostgreSQL Database**: Optimized Unicode character storage with full-text search indexes

### Performance Features

- **Database Optimization**: GIN indexes for full-text search, composite indexes for filters
- **UI Virtualization**: React Window for rendering 10K+ results without DOM performance issues
- **API Efficiency**: Sub-50ms response times, request debouncing, abort handling
- **Memory Management**: Connection pooling, result pagination, efficient data structures

### Cross-Scenario Integration

Other scenarios can leverage Symbol Search via:

**API Endpoints:**
- `/api/search` - Symbol search with filtering
- `/api/character/{codepoint}` - Detailed character information
- `/api/bulk/range` - Bulk character range retrieval
- `/api/categories` - Unicode category metadata
- `/api/blocks` - Unicode block metadata

**CLI Commands:**
```bash
# Test data generation example
symbol-search range U+0000 U+007F --json | jq '.characters[].decimal'

# Internationalization validation
symbol-search search --block "CJK Unified Ideographs" --limit 100
```

## 📖 Use Cases

### For Developers
- **Input Validation Testing**: Generate comprehensive test cases using Unicode boundary conditions
- **Internationalization**: Validate application support across writing systems
- **Documentation**: Find proper Unicode names and codes for technical writing

### For Designers  
- **Symbol Discovery**: Find appropriate icons and symbols for interfaces
- **Typography**: Explore character sets and special typography symbols
- **Emoji/Icon Selection**: Browse emoji and pictographic symbols with semantic search

### For Security Testing
- **Fuzzing**: Generate problematic character combinations for input sanitization testing
- **Edge Case Discovery**: Find characters that commonly break parsing or validation logic

### Cross-Scenario Examples
- **test-data-generator**: Uses bulk range API to create comprehensive Unicode test datasets
- **internationalization-validator**: Leverages block filtering to test global character support  
- **security-fuzzer**: Accesses boundary case characters for input validation testing

## 🎨 User Experience

Symbol Search features a clean, professional interface optimized for developer productivity:

- **Information-Dense Layout**: Maximum symbols visible with minimal scrolling
- **Instant Search**: Real-time results with 300ms debouncing
- **Advanced Filters**: Category, block, and version filtering with visual feedback
- **Detailed Character View**: Comprehensive character information with usage examples
- **Responsive Design**: Works seamlessly on desktop and mobile devices
- **Keyboard Navigation**: Full accessibility support with keyboard shortcuts

The interface prioritizes **function over form** while maintaining modern design standards and excellent usability.

## 🔧 Development

### Local Development

```bash
# Start API development server
cd api && go run main.go

# Start UI development server  
cd ui && npm run dev

# Run tests
vrooli scenario test symbol-search
```

### Performance Testing

```bash
# Run the full phased suite
./test/run-tests.sh

# Focus on a single phase
./test/run-tests.sh performance
```

### Database Management

The scenario includes comprehensive Unicode data seeding:

```bash
# Check database population
psql -c "SELECT COUNT(*) FROM characters"

# Refresh indexes
psql -c "REINDEX DATABASE vrooli"
```

## 📊 Performance Specifications

| Metric | Target | Measurement |
|--------|--------|-------------|
| Search Response Time | < 50ms | API monitoring |
| UI Rendering | < 200ms for 10K+ results | Frontend performance testing |
| Database Queries | < 25ms for indexed searches | PostgreSQL query analysis |
| Memory Usage | < 512MB at 100 concurrent users | Load testing |
| Character Data Accuracy | 100% Unicode 15.1 compliance | Unicode validation suite |

## 🤝 Integration Examples

### Test Data Generator Integration
```javascript
// Generate test cases using Unicode boundary conditions
const response = await fetch('/api/bulk/range', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    ranges: [
      { start: 'U+0000', end: 'U+001F' }, // Control characters
      { start: 'U+D800', end: 'U+DFFF' }, // Surrogate pairs
      { start: 'U+FDD0', end: 'U+FDEF' }  // Noncharacters
    ]
  })
})
```

### Security Fuzzer Integration
```bash
# Get problematic character combinations
symbol-search range U+202A U+202E --json | # Bidirectional overrides
  jq -r '.characters[].decimal' |
  xargs -I {} printf "\\$(printf %o {})"
```

### Internationalization Validator Integration
```bash
# Test application with various writing systems
for block in "Arabic" "Chinese" "Devanagari" "Cyrillic"; do
  symbol-search search --block "$block" --limit 50 --json |
    jq -r '.characters[].decimal' |
    # Feed to application test suite
done
```

## 🔗 API Reference

### Search Endpoints

**`GET /api/search`**
```
Query Parameters:
- q: string           // Search term
- category: string    // Unicode category filter
- block: string       // Unicode block filter  
- unicode_version: string // Unicode version filter
- limit: number       // Result limit (1-1000)
- offset: number      // Pagination offset

Response:
{
  "characters": Character[],
  "total": number,
  "query_time_ms": number,
  "filters_applied": object
}
```

**`GET /api/character/{codepoint}`**
```
Parameters:
- codepoint: string   // U+1F600 or decimal

Response:
{
  "character": Character,
  "related_characters": Character[],
  "usage_examples": string[]
}
```

**`POST /api/bulk/range`**
```
Body:
{
  "ranges": [{
    "start": string,    // U+0000 or decimal
    "end": string,      // U+007F or decimal
    "format": string    // Optional: unicode|decimal|html
  }]
}

Response:
{
  "characters": Character[],
  "total_characters": number,
  "ranges_processed": number
}
```

### Metadata Endpoints

**`GET /api/categories`** - List Unicode categories with counts
**`GET /api/blocks`** - List Unicode blocks with ranges and counts

## 📚 Technical Details

### Database Schema

```sql
-- Core tables
CREATE TABLE categories (code VARCHAR(2) PRIMARY KEY, name VARCHAR(50), description TEXT);
CREATE TABLE character_blocks (id SERIAL PRIMARY KEY, name VARCHAR(50) UNIQUE, start_codepoint INTEGER, end_codepoint INTEGER);
CREATE TABLE characters (codepoint VARCHAR(10) PRIMARY KEY, decimal INTEGER UNIQUE, name TEXT, category VARCHAR(2), block VARCHAR(50), unicode_version VARCHAR(10), description TEXT, html_entity VARCHAR(20), css_content VARCHAR(20), properties JSONB);

-- Performance indexes
CREATE INDEX idx_characters_name ON characters USING gin(to_tsvector('english', name));
CREATE INDEX idx_characters_category ON characters(category);
CREATE INDEX idx_characters_block ON characters(block);
CREATE INDEX idx_characters_fulltext ON characters USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));
```

### UI Performance Optimizations

- **Virtual Scrolling**: React Window renders only visible items
- **Request Debouncing**: 300ms delay prevents excessive API calls  
- **Request Cancellation**: Abort outdated requests to prevent race conditions
- **Memoization**: React.memo prevents unnecessary re-renders
- **Code Splitting**: Lazy-loaded components reduce initial bundle size

### Security Considerations

- **Input Sanitization**: All API parameters validated and sanitized
- **Rate Limiting**: Per-IP request limits prevent API abuse
- **CORS Configuration**: Appropriate cross-origin resource sharing settings
- **No Sensitive Data**: All Unicode data is publicly available

## 🎯 Future Enhancements

- **Semantic Search**: AI-powered symbol recommendations based on usage context
- **Visual Similarity**: Clustering symbols by visual appearance for icon discovery  
- **Usage Analytics**: Track symbol usage patterns across scenarios
- **Real-time Updates**: Automatic sync with new Unicode standard releases
- **Browser Extension**: Direct integration with code editors and design tools

## 💝 Contributing

Symbol Search is part of the Vrooli scenario ecosystem. Improvements to search performance, UI usability, or cross-scenario integration patterns are welcome.

Key areas for contribution:
- Unicode data completeness and accuracy
- Search performance optimization  
- UI/UX improvements for developer workflows
- Integration patterns for other scenarios
- Performance testing and benchmarking

---

**Symbol Search** - Making Unicode accessible to everyone, one character at a time. 🔍✨
