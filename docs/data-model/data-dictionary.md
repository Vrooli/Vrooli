# Data Dictionary

Comprehensive reference for data types, enums, field specifications, and validation rules used throughout Vrooli's database schema.

## 📊 Core Data Types

### **Primary Key Pattern**
```sql
-- All entities use BigInt primary keys
id BIGINT PRIMARY KEY
```
- **Size**: 64-bit signed integer
- **Range**: -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807
- **Generation**: Application-generated using Snowflake algorithm
- **Benefits**: Sortable, time-ordered, distributed-safe

### **Public Identifier Pattern**
```sql
-- URL-safe public identifiers
publicId VARCHAR(12) UNIQUE NOT NULL
```
- **Format**: Base62 encoded string
- **Length**: 12 characters
- **Characters**: `[A-Za-z0-9]`
- **Example**: `aB3xY9mK2nQ7`
- **Purpose**: URL-safe external references

### **Timestamp Pattern**
```sql
-- Consistent timestamp handling
createdAt TIMESTAMPTZ DEFAULT NOW() NOT NULL
updatedAt TIMESTAMPTZ DEFAULT NOW() NOT NULL
```
- **Type**: `TIMESTAMPTZ` (timezone-aware)
- **Precision**: Microseconds
- **Default**: Current timestamp
- **Updates**: Automatic via triggers or application logic

### **JSON Configuration Pattern**
```sql
-- Flexible configuration storage
config JSON DEFAULT '{}' NOT NULL
permissions JSON DEFAULT '{}' NOT NULL
```
- **Type**: `JSON` (validated JSON)
- **Default**: Empty object `{}`
- **Validation**: Schema validation at application level
- **Indexing**: GIN indexes for complex queries

## 📋 Enum Definitions

### **RunStatus**
Execution status for run instances.

```typescript
enum RunStatus {
  Scheduled = 'Scheduled',     // Queued for execution
  InProgress = 'InProgress',   // Currently executing
  Paused = 'Paused',           // Execution paused
  Completed = 'Completed',     // Successfully finished
  Failed = 'Failed',           // Execution failed
  Cancelled = 'Cancelled'      // User cancelled
}
```

**Database Definition:**
```sql
CREATE TYPE "RunStatus" AS ENUM (
  'Scheduled',
  'InProgress',
  'Paused',
  'Completed',
  'Failed',
  'Cancelled'
);
```

### **RunStepStatus**
Status for individual execution steps.

```typescript
enum RunStepStatus {
  InProgress = 'InProgress',   // Currently executing
  Completed = 'Completed',     // Step completed
  Skipped = 'Skipped'          // Step skipped
}
```

### **ResourceType**
Types of resources in the system.

```typescript
enum ResourceType {
  Api = 'Api',                 // External API integration
  Code = 'Code',               // Code snippets/libraries
  Project = 'Project',         // Project containers
  Routine = 'Routine',         // Automation workflows
  Standard = 'Standard',       // Reusable patterns
  Team = 'Team',               // Team profiles
  User = 'User'                // User profiles
}
```

### **InviteStatus**
Status for invitations.

```typescript
enum InviteStatus {
  Pending = 'Pending',         // Invitation sent
  Accepted = 'Accepted',       // Invitation accepted
  Declined = 'Declined',       // Invitation declined
  Expired = 'Expired'          // Invitation expired
}
```

### **IssueStatus**
Status for issues and bug reports.

```typescript
enum IssueStatus {
  Draft = 'Draft',             // Work in progress
  Open = 'Open',               // Ready for review
  InProgress = 'InProgress',   // Being worked on
  Closed = 'Closed',           // Resolved
  ClosedResolved = 'ClosedResolved', // Resolved and closed
  ClosedNotPlanned = 'ClosedNotPlanned' // Closed without resolution
}
```

### **StatPeriodType**
Statistical period types.

```typescript
enum StatPeriodType {
  Daily = 'Daily',             // Daily statistics
  Weekly = 'Weekly',           // Weekly aggregates
  Monthly = 'Monthly',         // Monthly aggregates
  Yearly = 'Yearly'            // Yearly aggregates
}
```

### **PaymentStatus**
Payment transaction status.

```typescript
enum PaymentStatus {
  Pending = 'Pending',         // Payment initiated
  Processing = 'Processing',   // Payment processing
  Succeeded = 'Succeeded',     // Payment successful
  Failed = 'Failed',           // Payment failed
  Canceled = 'Canceled',       // Payment cancelled
  Refunded = 'Refunded'        // Payment refunded
}
```

### **AccountStatus**
User account status.

```typescript
enum AccountStatus {
  Unlocked = 'Unlocked',       // Normal active account
  SoftLocked = 'SoftLocked',   // Temporarily restricted
  HardLocked = 'HardLocked',   // Permanently suspended
  Deleted = 'Deleted'          // Account marked for deletion
}
```

### **InviteStatus**
Status for invitations.

```typescript
enum InviteStatus {
  Pending = 'Pending',         // Invitation sent
  Accepted = 'Accepted',       // Invitation accepted
  Declined = 'Declined',       // Invitation declined
  Expired = 'Expired'          // Invitation expired
}
```

## 🔤 Field Specifications

### **User Fields**

> **Note**: For complete entity definitions and relationships, see [Entity Model](entities.md#user-entity).

#### **Core Identity Fields**
```sql
-- Email address (unique, case-insensitive)
email CITEXT UNIQUE NOT NULL
-- Display name
name VARCHAR(128)
-- Unique handle (@username)
handle CITEXT UNIQUE
-- User biography
bio VARCHAR(2048)
```

**Validation Rules:**
- Email: RFC 5322 compliant, maximum 254 characters
- Name: 1-128 characters, UTF-8 supported
- Handle: 3-50 characters, alphanumeric + underscore/hyphen
- Bio: Maximum 2048 characters, markdown supported

#### **Profile Settings**
```sql
-- Profile visibility
isPrivate BOOLEAN DEFAULT FALSE NOT NULL
-- Bot account indicator
isBot BOOLEAN DEFAULT FALSE NOT NULL
-- Bot depicting real person
isBotDepictingPerson BOOLEAN DEFAULT FALSE NOT NULL
-- Preferred languages
languages TEXT[] DEFAULT '{}' NOT NULL
```

**Validation Rules:**
- Languages: ISO 639-1 language codes
- Bot accounts: Cannot have private profiles
- Depicting person: Only valid for bot accounts

#### **Privacy Controls**
```sql
-- Privacy settings for various content types
isPrivateMemberships BOOLEAN DEFAULT FALSE NOT NULL
isPrivatePullRequests BOOLEAN DEFAULT FALSE NOT NULL
isPrivateResources BOOLEAN DEFAULT FALSE NOT NULL
isPrivateResourcesCreated BOOLEAN DEFAULT FALSE NOT NULL
isPrivateTeamsCreated BOOLEAN DEFAULT FALSE NOT NULL
isPrivateBookmarks BOOLEAN DEFAULT FALSE NOT NULL
isPrivateVotes BOOLEAN DEFAULT FALSE NOT NULL
```

### **Team Fields**

#### **Team Identity**
```sql
-- Team name
name VARCHAR(128) NOT NULL
-- Team description
description VARCHAR(2048)
-- Unique team handle
handle CITEXT UNIQUE
-- Team visibility
isPrivate BOOLEAN DEFAULT FALSE NOT NULL
-- Open membership
isOpenToNewMembers BOOLEAN DEFAULT FALSE NOT NULL
```

**Validation Rules:**
- Name: 1-128 characters, required
- Description: Maximum 2048 characters
- Handle: 3-50 characters, globally unique
- Open membership: Only valid for public teams

#### **Team Hierarchy**
```sql
-- Parent team for hierarchies
parentId BIGINT REFERENCES team(id) ON DELETE SET NULL
```

**Business Rules:**
- Maximum hierarchy depth: 5 levels
- Circular references prevented
- Child teams inherit parent permissions

### **Resource Fields**

#### **Resource Metadata**
```sql
-- Resource type
resourceType VARCHAR(32) NOT NULL
-- Completion status
hasCompleteVersion BOOLEAN DEFAULT FALSE NOT NULL
-- Completion timestamp
completedAt TIMESTAMPTZ
-- Transfer timestamp
transferredAt TIMESTAMPTZ
-- Soft delete flag
isDeleted BOOLEAN DEFAULT FALSE NOT NULL
```

**Validation Rules:**
- ResourceType: Must match enum values
- CompletedAt: Set when hasCompleteVersion becomes true
- TransferredAt: Set when ownership changes

#### **Community Metrics**
```sql
-- Community score
score INTEGER DEFAULT 0 NOT NULL
-- Bookmark count
bookmarks INTEGER DEFAULT 0 NOT NULL
-- View count
views INTEGER DEFAULT 0 NOT NULL
```

**Business Rules:**
- Score: Calculated from community interactions
- Bookmarks: Denormalized count for performance
- Views: Unique views per user per day

### **Run Fields**

#### **Execution Metadata**
```sql
-- Execution name/description
name VARCHAR(128) NOT NULL
-- Current status
status "RunStatus" DEFAULT 'Scheduled' NOT NULL
-- Execution data/context
data VARCHAR(16384)
-- Automatic execution flag
wasRunAutomatically BOOLEAN DEFAULT FALSE NOT NULL
```

**Validation Rules:**
- Name: Required, 1-128 characters
- Data: JSON string, maximum 16KB
- Status transitions: Must follow valid state machine

#### **Timing Information**
```sql
-- Execution start time
startedAt TIMESTAMPTZ
-- Execution completion time
completedAt TIMESTAMPTZ
-- Execution duration in milliseconds
timeElapsed INTEGER
```

**Business Rules:**
- StartedAt: Set when status changes to InProgress
- CompletedAt: Set when status changes to Completed/Failed
- TimeElapsed: Calculated as completedAt - startedAt

#### **Performance Metrics**
```sql
-- Complexity score
completedComplexity INTEGER DEFAULT 0 NOT NULL
-- Context switch count
contextSwitches INTEGER DEFAULT 0 NOT NULL
```

**Calculation Rules:**
- Complexity: Based on executed steps and operations
- Context switches: Count of tier transitions during execution

### **Chat Fields**

#### **Chat Configuration**
```sql
-- Chat visibility
isPrivate BOOLEAN DEFAULT TRUE NOT NULL
-- Invite-based access
openToAnyoneWithInvite BOOLEAN DEFAULT FALSE NOT NULL
-- Chat configuration
config JSON DEFAULT '{}' NOT NULL
```

**Configuration Schema:**
```typescript
interface ChatConfig {
  maxParticipants?: number;        // Maximum participants (default: 100)
  messageRetentionDays?: number;   // Message retention (default: 365)
  allowFileUploads?: boolean;      // File upload permission (default: true)
  moderationEnabled?: boolean;     // Content moderation (default: false)
  aiAssistantEnabled?: boolean;    // AI assistant access (default: true)
}
```

### **Message Fields**

#### **Message Content**
```sql
-- Message language
language VARCHAR(3) NOT NULL
-- Message text content
text VARCHAR(32768) NOT NULL
-- Message configuration
config JSON DEFAULT '{}' NOT NULL
-- Message score
score INTEGER DEFAULT 0 NOT NULL
-- Version index for edits
versionIndex INTEGER DEFAULT 0 NOT NULL
```

**Validation Rules:**
- Language: ISO 639-2/3 language code (2-3 characters)
- Text: 1-32768 characters, markdown supported
- VersionIndex: Incremented on each edit

**Message Config Schema:**
```typescript
interface MessageConfig {
  edited?: boolean;                // Message was edited
  editedAt?: string;              // Last edit timestamp
  mentions?: string[];            // Mentioned user IDs
  attachments?: MessageAttachment[]; // File attachments
  reactions?: MessageReaction[];   // Message reactions
}
```

## 🔍 Validation Patterns

### **String Validation**
```typescript
// Email validation
const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Handle validation (alphanumeric + underscore/hyphen)
const handleRegex = /^[a-zA-Z0-9_-]{3,50}$/;

// Public ID validation (Base62)
const publicIdRegex = /^[A-Za-z0-9]{12}$/;

// Language code validation (ISO 639-2/3)
const languageRegex = /^[a-z]{2,3}$/;
```

### **Numeric Validation**
```typescript
// Score validation
const isValidScore = (score: number): boolean => {
  return Number.isInteger(score) && score >= 0;
};

// Time elapsed validation
const isValidTimeElapsed = (time: number): boolean => {
  return Number.isInteger(time) && time >= 0 && time <= 86400000; // Max 24 hours
};

// Version index validation
const isValidVersionIndex = (index: number): boolean => {
  return Number.isInteger(index) && index >= 0;
};
```

### **JSON Schema Validation**
```typescript
// User preferences schema
const userPreferencesSchema = {
  type: 'object',
  properties: {
    theme: { enum: ['light', 'dark', 'auto'] },
    language: { type: 'string', pattern: '^[a-z]{2,3}$' },
    notifications: {
      type: 'object',
      properties: {
        email: { type: 'boolean' },
        push: { type: 'boolean' },
        sms: { type: 'boolean' }
      }
    },
    accessibility: {
      type: 'object',
      properties: {
        highContrast: { type: 'boolean' },
        reduceMotion: { type: 'boolean' },
        fontSize: { enum: ['small', 'medium', 'large'] }
      }
    }
  }
};

// Team permissions schema
const teamPermissionsSchema = {
  type: 'object',
  properties: {
    canCreateResources: { type: 'boolean', default: true },
    canInviteMembers: { type: 'boolean', default: false },
    canModifySettings: { type: 'boolean', default: false },
    canManageRoles: { type: 'boolean', default: false },
    resourceTypes: {
      type: 'array',
      items: { enum: ['Api', 'Code', 'Project', 'Routine', 'Standard'] }
    }
  }
};
```

## 📐 Constraint Definitions

### **Check Constraints**
```sql
-- Score constraints
ALTER TABLE resource ADD CONSTRAINT check_resource_score 
CHECK (score >= 0);

ALTER TABLE chat_message ADD CONSTRAINT check_message_score 
CHECK (score >= -1000 AND score <= 1000);

-- Time constraints
ALTER TABLE run ADD CONSTRAINT check_run_time_elapsed 
CHECK (timeElapsed IS NULL OR timeElapsed >= 0);

ALTER TABLE run_step ADD CONSTRAINT check_step_time_elapsed 
CHECK (timeElapsed IS NULL OR timeElapsed >= 0);

-- Version constraints
ALTER TABLE resource_version ADD CONSTRAINT check_version_index 
CHECK (versionIndex >= 0);

ALTER TABLE chat_message ADD CONSTRAINT check_version_index 
CHECK (versionIndex >= 0);

-- Step constraints
ALTER TABLE run_step ADD CONSTRAINT check_step_number 
CHECK (step > 0);

-- Count constraints
ALTER TABLE resource ADD CONSTRAINT check_bookmarks_count 
CHECK (bookmarks >= 0);

ALTER TABLE resource ADD CONSTRAINT check_views_count 
CHECK (views >= 0);
```

### **Unique Constraints**
```sql
-- User constraints
ALTER TABLE user ADD CONSTRAINT unique_user_email UNIQUE (email);
ALTER TABLE user ADD CONSTRAINT unique_user_handle UNIQUE (handle);
ALTER TABLE user ADD CONSTRAINT unique_user_public_id UNIQUE (publicId);

-- Team constraints
ALTER TABLE team ADD CONSTRAINT unique_team_handle UNIQUE (handle);
ALTER TABLE team ADD CONSTRAINT unique_team_public_id UNIQUE (publicId);

-- Relationship constraints
ALTER TABLE member ADD CONSTRAINT unique_member_user_team 
UNIQUE (userId, teamId);

ALTER TABLE chat_participants ADD CONSTRAINT unique_participant_user_chat 
UNIQUE (userId, chatId);

ALTER TABLE resource_tag ADD CONSTRAINT unique_resource_tag 
UNIQUE (resourceId, tagId);
```

### **Foreign Key Constraints**
```sql
-- Strong ownership (CASCADE)
ALTER TABLE member ADD CONSTRAINT fk_member_user 
FOREIGN KEY (userId) REFERENCES user(id) ON DELETE CASCADE;

ALTER TABLE resource_version ADD CONSTRAINT fk_resource_version_root 
FOREIGN KEY (rootId) REFERENCES resource(id) ON DELETE CASCADE;

-- Weak references (SET NULL)
ALTER TABLE resource ADD CONSTRAINT fk_resource_created_by 
FOREIGN KEY (createdById) REFERENCES user(id) ON DELETE SET NULL;

ALTER TABLE run ADD CONSTRAINT fk_run_resource_version 
FOREIGN KEY (resourceVersionId) REFERENCES resource_version(id) ON DELETE SET NULL;

-- Self-references (SET NULL)
ALTER TABLE team ADD CONSTRAINT fk_team_parent 
FOREIGN KEY (parentId) REFERENCES team(id) ON DELETE SET NULL;

ALTER TABLE chat_message ADD CONSTRAINT fk_chat_message_parent 
FOREIGN KEY (parentId) REFERENCES chat_message(id) ON DELETE SET NULL;
```

## 🔧 Data Quality Rules

### **Consistency Rules**
```sql
-- Resource completion consistency
-- If hasCompleteVersion is true, at least one version must be complete
CREATE OR REPLACE FUNCTION check_resource_completion()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.hasCompleteVersion = TRUE THEN
    IF NOT EXISTS (
      SELECT 1 FROM resource_version 
      WHERE rootId = NEW.id AND isComplete = TRUE
    ) THEN
      RAISE EXCEPTION 'Resource marked as complete but no complete version exists';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Version latest consistency
-- Only one version per resource should be marked as latest
CREATE OR REPLACE FUNCTION enforce_single_latest_version()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.isLatest = TRUE THEN
    UPDATE resource_version 
    SET isLatest = FALSE 
    WHERE rootId = NEW.rootId AND id != NEW.id;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### **Audit Triggers**
```sql
-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updatedAt = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updatedAt
CREATE TRIGGER trigger_user_updated_at
  BEFORE UPDATE ON user
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_resource_updated_at
  BEFORE UPDATE ON resource
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

## 🔒 Security Implementation

### **Data Encryption**
```typescript
// Encrypt sensitive fields before storage
export async function encryptPersonalData(data: string): Promise<string> {
  const algorithm = 'aes-256-gcm';
  const key = crypto.scryptSync(process.env.ENCRYPTION_KEY!, 'salt', 32);
  const iv = crypto.randomBytes(16);
  
  const cipher = crypto.createCipher(algorithm, key, { iv });
  let encrypted = cipher.update(data, 'utf8', 'hex');
  encrypted += cipher.final('hex');
  
  const authTag = cipher.getAuthTag();
  return `${iv.toString('hex')}:${authTag.toString('hex')}:${encrypted}`;
}

// Decrypt sensitive fields on retrieval
export async function decryptPersonalData(encryptedData: string): Promise<string> {
  const [ivHex, authTagHex, encrypted] = encryptedData.split(':');
  const algorithm = 'aes-256-gcm';
  const key = crypto.scryptSync(process.env.ENCRYPTION_KEY!, 'salt', 32);
  
  const decipher = crypto.createDecipher(algorithm, key, {
    iv: Buffer.from(ivHex, 'hex'),
    authTag: Buffer.from(authTagHex, 'hex')
  });
  
  let decrypted = decipher.update(encrypted, 'hex', 'utf8');
  decrypted += decipher.final('utf8');
  return decrypted;
}
```

### **Access Control Implementation**
```typescript
// Role-based access control via permissions JSON
interface UserPermissions {
  canCreateResources: boolean;
  canInviteMembers: boolean;
  canModifySettings: boolean;
  resourceTypes: ResourceType[];
  maxResourcesPerDay?: number;
}

// Validate user permissions
export function hasPermission(
  userPermissions: UserPermissions,
  action: string,
  resourceType?: ResourceType
): boolean {
  switch (action) {
    case 'CREATE_RESOURCE':
      return userPermissions.canCreateResources && 
             (!resourceType || userPermissions.resourceTypes.includes(resourceType));
    case 'INVITE_MEMBER':
      return userPermissions.canInviteMembers;
    case 'MODIFY_SETTINGS':
      return userPermissions.canModifySettings;
    default:
      return false;
  }
}
```

## 🎯 Vector Data Types

### **Embedding Storage**
```sql
-- Vector embedding fields for AI features (Prisma schema)
embedding Unsupported("vector(1536)")

-- Actual PostgreSQL type
embedding vector(1536)  -- OpenAI embedding dimensions
```

**Vector Data Specifications:**
- **Dimensions**: 1536 (OpenAI text-embedding-ada-002)
- **Data Type**: `vector(1536)` (pgvector extension)
- **Prisma Type**: `Unsupported("vector(1536)")` 
- **Storage**: ~6KB per embedding (float32 × 1536)
- **Indexing**: HNSW index for similarity search

### **Vector Operations**
```typescript
// Store embeddings for semantic search
export async function storeResourceEmbedding(
  resourceId: string,
  content: string
): Promise<void> {
  const embedding = await generateEmbedding(content);
  
  await prisma.resourceVersion.update({
    where: { id: BigInt(resourceId) },
    data: {
      embedding: `[${embedding.join(',')}]` // Store as vector
    }
  });
}

// Semantic similarity search
export async function findSimilarResources(
  queryEmbedding: number[],
  limit: number = 10
): Promise<Resource[]> {
  return prisma.$queryRaw`
    SELECT r.*, rv.embedding <-> ${`[${queryEmbedding.join(',')}]`}::vector as distance
    FROM resource r
    JOIN resource_version rv ON r.id = rv.rootId AND rv.isLatest = true
    WHERE rv.embedding IS NOT NULL
    ORDER BY distance ASC
    LIMIT ${limit}
  `;
}
```

---

**Related Documentation:**
- [Entity Model](entities.md) - Entity definitions and relationships
- [Database Architecture](architecture.md) - Infrastructure configuration
- [Schema Evolution](schema-evolution.md) - Migration and versioning