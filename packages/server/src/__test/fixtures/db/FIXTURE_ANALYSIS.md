# Database Fixture Analysis Report

## Current State Summary
- **Prisma Models**: 66 total models
- **DbFactory Files**: 42 files
- **Simple Fixture Files**: 51 files
- **Total Coverage**: Significant gaps and redundancies identified

## Model-to-Fixture Mapping Analysis

### ✅ Models with Complete Coverage (DbFactory + Simple Fixtures)
| Model | DbFactory | Simple Fixtures | Status |
|-------|-----------|-----------------|--------|
| user | UserDbFactory.ts | userFixtures.ts | ✅ Complete |
| team | TeamDbFactory.ts | teamFixtures.ts | ✅ Complete |
| bookmark | BookmarkDbFactory.ts | bookmarkFixtures.ts | ✅ Complete |
| bookmark_list | BookmarkListDbFactory.ts | bookmarkListFixtures.ts | ✅ Complete |
| chat | ChatDbFactory.ts | chatFixtures.ts | ✅ Complete |
| chat_invite | ChatInviteDbFactory.ts | chatInviteFixtures.ts | ✅ Complete |
| chat_message | ChatMessageDbFactory.ts | chatMessageFixtures.ts | ✅ Complete |
| chat_participants | ChatParticipantDbFactory.ts | chatParticipantFixtures.ts | ✅ Complete |
| comment | CommentDbFactory.ts | commentFixtures.ts | ✅ Complete |
| issue | IssueDbFactory.ts | issueFixtures.ts | ✅ Complete |
| meeting | MeetingDbFactory.ts | meetingFixtures.ts | ✅ Complete |
| meeting_invite | MeetingInviteDbFactory.ts | meetingInviteFixtures.ts | ✅ Complete |
| notification | NotificationDbFactory.ts | notificationFixtures.ts | ✅ Complete |
| email | EmailDbFactory.ts | - | ✅ Complete (DbFactory only) |
| phone | PhoneDbFactory.ts | - | ✅ Complete (DbFactory only) |
| payment | PaymentDbFactory.ts | paymentFixtures.ts | ✅ Complete |
| session | SessionDbFactory.ts | sessionFixtures.ts | ✅ Complete |
| resource | ResourceDbFactory.ts | resourceFixtures.ts | ✅ Complete |
| resource_version | ResourceVersionDbFactory.ts | resourceVersionFixtures.ts | ✅ Complete |
| resource_version_relation | ResourceVersionRelationDbFactory.ts | resourceVersionRelationFixtures.ts | ✅ Complete |
| run | RunDbFactory.ts | runFixtures.ts | ✅ Complete |
| run_io | RunIODbFactory.ts | runIOFixtures.ts | ✅ Complete |
| run_step | RunStepDbFactory.ts | runStepFixtures.ts | ✅ Complete |
| schedule | ScheduleDbFactory.ts | scheduleFixtures.ts | ✅ Complete |
| schedule_exception | ScheduleExceptionDbFactory.ts | scheduleExceptionFixtures.ts | ✅ Complete |
| schedule_recurrence | ScheduleRecurrenceDbFactory.ts | scheduleRecurrenceFixtures.ts | ✅ Complete |
| reminder | ReminderDbFactory.ts | reminderFixtures.ts | ✅ Complete |
| tag | TagDbFactory.ts | tagFixtures.ts | ✅ Complete |
| view | ViewDbFactory.ts | viewFixtures.ts | ✅ Complete |
| report | ReportDbFactory.ts | reportFixtures.ts | ✅ Complete |
| report_response | ReportResponseDbFactory.ts | reportResponseFixtures.ts | ✅ Complete |
| reaction | ReactionDbFactory.ts | reactionFixtures.ts | ✅ Complete |
| reaction_summary | ReactionSummaryDbFactory.ts | - | ✅ Complete (DbFactory only) |

### ⚠️ Models with Simple Fixtures Only (Missing DbFactory)
| Model | Simple Fixtures | Missing DbFactory |
|-------|----------------|-------------------|
| api_key | apiKeyFixtures.ts | ❌ ApiKeyDbFactory.ts |
| api_key_external | apiKeyExternalFixtures.ts | ❌ ApiKeyExternalDbFactory.ts |
| award | awardFixtures.ts | ❌ AwardDbFactory.ts |
| credit_account | creditAccountFixtures.ts | ❌ CreditAccountDbFactory.ts |
| credit_ledger_entry | creditLedgerEntryFixtures.ts | ❌ CreditLedgerEntryDbFactory.ts |
| plan | planFixtures.ts | ❌ PlanDbFactory.ts |
| pull_request | pullRequestFixtures.ts | ❌ PullRequestDbFactory.ts |
| push_device | pushDeviceFixtures.ts | ❌ PushDeviceDbFactory.ts |
| reputation_history | reputationHistoryFixtures.ts | ❌ ReputationHistoryDbFactory.ts |
| stats_resource | statsResourceFixtures.ts | ❌ StatsResourceDbFactory.ts |
| stats_site | statsSiteFixtures.ts | ❌ StatsSiteDbFactory.ts |
| stats_team | statsTeamFixtures.ts | ❌ StatsTeamDbFactory.ts |
| stats_user | statsUserFixtures.ts | ❌ StatsUserDbFactory.ts |
| transfer | transferFixtures.ts | ❌ TransferDbFactory.ts |
| wallet | walletFixtures.ts | ❌ WalletDbFactory.ts |

### ❌ Models with No Fixtures At All
| Model | Status | Reason |
|-------|--------|--------|
| chat_translation | ❌ Missing | Translation model - not implemented |
| comment_translation | ❌ Missing | Translation model - not implemented |
| issue_translation | ❌ Missing | Translation model - not implemented |
| meeting_translation | ❌ Missing | Translation model - not implemented |
| tag_translation | ❌ Missing | Translation model - not implemented |
| team_translation | ❌ Missing | Translation model - not implemented |
| user_translation | ❌ Missing | Translation model - not implemented |
| pull_request_translation | ❌ Missing | Translation model - not implemented |
| resource_translation | ❌ Missing | Translation model - not implemented |
| meeting_attendees | ❌ Missing | Junction table - need fixture |
| member | ❌ Missing | Has memberFixtures.ts but no DbFactory |
| member_invite | ❌ Missing | Has memberInviteFixtures.ts but no DbFactory |
| notification_subscription | ❌ Missing | Completely missing |
| resource_tag | ❌ Missing | Junction table - completely missing |
| team_tag | ❌ Missing | Junction table - completely missing |
| user_auth | ✅ Covered by AuthDbFactory | Mapped to auth concept |

### 🚫 Extra Fixture Files That Don't Map to Models
| File | Issue | Action Required |
|------|-------|-----------------|
| premiumFixtures.ts | No 'premium' model exists | ❌ DELETE |
| statsFixtures.ts | Too generic - conflicts with specific stats models | ❌ DELETE or MERGE |

### 🔄 Files Needing Rename/Correction
| Current File | Should Be | Reason |
|-------------|-----------|---------|
| ScheduleRecurrenceEnhancedDbFactory.ts | ScheduleRecurrenceDbFactory.ts | Naming inconsistency |
| ApiKeyDbFactory.ts | Should exist | Currently missing |
| AwardDbFactory.ts | Should exist | Currently missing |

## Correction Plan Summary

### Phase 1: Delete Extra Files
```bash
rm /root/Vrooli/packages/server/src/__test/fixtures/db/premiumFixtures.ts
# Consider merging statsFixtures.ts into specific stats model fixtures
```

### Phase 2: Create Missing DbFactory Files (15 files)
1. ApiKeyDbFactory.ts
2. ApiKeyExternalDbFactory.ts  
3. AwardDbFactory.ts
4. CreditAccountDbFactory.ts
5. CreditLedgerEntryDbFactory.ts
6. PlanDbFactory.ts
7. PullRequestDbFactory.ts
8. PushDeviceDbFactory.ts
9. ReputationHistoryDbFactory.ts
10. StatsResourceDbFactory.ts
11. StatsSiteDbFactory.ts
12. StatsTeamDbFactory.ts
13. StatsUserDbFactory.ts
14. TransferDbFactory.ts
15. WalletDbFactory.ts

### Phase 3: Create Missing Translation Model Fixtures (18 files total - 9 DbFactory + 9 simple fixtures)
1. ChatTranslationDbFactory.ts + chatTranslationFixtures.ts
2. CommentTranslationDbFactory.ts + commentTranslationFixtures.ts
3. IssueTranslationDbFactory.ts + issueTranslationFixtures.ts
4. MeetingTranslationDbFactory.ts + meetingTranslationFixtures.ts
5. TagTranslationDbFactory.ts + tagTranslationFixtures.ts
6. TeamTranslationDbFactory.ts + teamTranslationFixtures.ts
7. UserTranslationDbFactory.ts + userTranslationFixtures.ts
8. PullRequestTranslationDbFactory.ts + pullRequestTranslationFixtures.ts
9. ResourceTranslationDbFactory.ts + resourceTranslationFixtures.ts

### Phase 4: Create Missing Junction/Relationship Model Fixtures (10 files total - 5 DbFactory + 5 simple fixtures)
1. MeetingAttendeesDbFactory.ts + meetingAttendeesFixtures.ts
2. MemberDbFactory.ts (memberFixtures.ts exists)
3. MemberInviteDbFactory.ts (memberInviteFixtures.ts exists)
4. NotificationSubscriptionDbFactory.ts + notificationSubscriptionFixtures.ts
5. ResourceTagDbFactory.ts + resourceTagFixtures.ts
6. TeamTagDbFactory.ts + teamTagFixtures.ts

### Phase 5: Update Index.ts Exports
Add proper exports for all new DbFactory files and simple fixtures.

## Final Target State
- **66 Prisma Models** → **66 corresponding fixture sets**
- **DbFactory pattern**: All models should have a DbFactory class
- **Simple fixtures**: All models should have simple fixture files for backward compatibility
- **Proper exports**: All fixtures properly exported from index.ts
- **Clean organization**: No extra files, consistent naming

## Type Safety Verification Results
Type checking revealed **widespread structural issues** across existing fixtures:

### Critical Issues Found:
1. **Import Failures**: `generatePK` import from `@vrooli/shared` failing across multiple files
2. **Prisma Model Names**: Many fixtures use incorrect Prisma type names (PascalCase vs snake_case)
3. **BigInt Support**: ES2020 target issues with BigInt literals
4. **Interface Mismatches**: Factory classes not properly implementing base interfaces

### Recommendation:
**STOP** creating new DbFactory files until core infrastructure is fixed. The existing fixtures need:
1. Fix shared package imports
2. Update TypeScript config for ES2020+ support
3. Correct Prisma model type references
4. Resolve base factory interface inconsistencies

Creating new fixtures on top of this broken foundation would compound the issues.