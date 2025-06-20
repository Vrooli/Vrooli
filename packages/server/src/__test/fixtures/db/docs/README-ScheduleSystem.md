# Schedule System Database Fixture Factories

This document describes the database fixture factories implemented for Vrooli's Scheduling System, which provides comprehensive support for creating test data for schedules, recurrences, and exceptions.

## Overview

The Schedule System consists of three main database models:

1. **Schedule** - Base schedule entity with timezone, start/end dates
2. **ScheduleRecurrence** - Defines recurring patterns (daily, weekly, monthly, yearly)  
3. **ScheduleException** - Handles one-time exceptions to recurring schedules

## Factory Files

### Primary Factories

- **`ScheduleDbFactory.ts`** - Enhanced schedule factory with RRULE support
- **`ScheduleExceptionDbFactory.ts`** - Exception factory for cancellations and rescheduling
- **`ScheduleRecurrenceDbFactory.ts`** - Recurrence factory with pattern support

### Supporting Files

- **`scheduleSystemExamples.ts`** - Complex usage examples
- **`demonstrateScheduleFactories.ts`** - Simple demonstration
- **`validateScheduleFactories.ts`** - Validation script

## Key Features

### 1. RRULE Support

The factories support RFC 5545 RRULE format for complex recurrence patterns:

```typescript
// Daily every 2 days
const rrule = RRuleHelpers.daily(2);
// Result: "FREQ=DAILY;INTERVAL=2"

// Weekly on Monday, Wednesday, Friday
const rrule = RRuleHelpers.weekly(["MO", "WE", "FR"]);
// Result: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR"

// Monthly on the 15th
const rrule = RRuleHelpers.monthlyByDate(15);
// Result: "FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15"

// Yearly on June 15th
const rrule = RRuleHelpers.yearly(6, 15);
// Result: "FREQ=YEARLY;INTERVAL=1;BYMONTH=6;BYMONTHDAY=15"
```

### 2. Schedule Creation

#### Basic Schedule
```typescript
const schedule = ScheduleDbFactory.createMinimal("meeting_123", "Meeting", {
    startTime: new Date("2025-07-15T14:00:00Z"),
    endTime: new Date("2025-07-15T15:30:00Z"),
    timezone: "America/New_York",
});
```

#### Recurring Schedule
```typescript
const schedule = ScheduleDbFactory.createRecurring(
    "project_456",
    "RunProject",
    {
        recurrenceType: "Weekly",
        interval: 1,
        dayOfWeek: 3, // Wednesday
        duration: 60,
    }
);
```

#### Schedule with RRULE
```typescript
const rrule = "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR";
const schedule = ScheduleDbFactory.createWithRRule(
    "routine_789",
    "RunRoutine",
    rrule
);
```

### 3. Exception Handling

#### Cancelled Occurrence
```typescript
const exception = ScheduleExceptionDbFactory.createCancellation(
    "schedule_123",
    new Date("2025-07-04T10:00:00Z") // Independence Day
);
```

#### Rescheduled Occurrence
```typescript
const exception = ScheduleExceptionDbFactory.createRescheduled(
    "schedule_456",
    new Date("2025-07-10T10:00:00Z"), // Original time
    new Date("2025-07-10T14:00:00Z"), // New start
    new Date("2025-07-10T15:00:00Z")  // New end
);
```

#### Extended Duration
```typescript
const exception = ScheduleExceptionDbFactory.createExtended(
    "schedule_789",
    new Date("2025-07-15T10:00:00Z"),
    2 // Extend by 2 hours
);
```

### 4. Recurrence Patterns

#### Standard Patterns
```typescript
// Daily standup
const recurrence = ScheduleRecurrenceDbFactory.createDaily("schedule_id", 1, {
    duration: 15
});

// Weekly team meeting
const recurrence = ScheduleRecurrenceDbFactory.createWeekly(
    "schedule_id", 
    3, // Wednesday
    1
);

// Monthly all-hands
const recurrence = ScheduleRecurrenceDbFactory.createMonthly(
    "schedule_id",
    1, // First of month
    1
);

// Annual review
const recurrence = ScheduleRecurrenceDbFactory.createYearly(
    "schedule_id",
    12, // December
    15, // 15th
    1
);
```

#### From RRULE String
```typescript
const recurrence = ScheduleRecurrenceDbFactory.createFromRRule(
    "schedule_id",
    "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=15" // Quarterly on 15th
);
```

### 5. Timezone Support

All schedules support global timezone handling:

```typescript
const timezones = [
    "UTC",
    "America/New_York",
    "Europe/London", 
    "Asia/Tokyo",
    "Pacific/Auckland"
];

timezones.forEach(tz => {
    const schedule = ScheduleDbFactory.createMinimal("event_id", "Meeting", {
        timezone: tz
    });
});
```

### 6. Occurrence Generation

Generate occurrences for testing:

```typescript
const schedule = {
    startTime: new Date("2025-07-01T10:00:00Z"),
    endTime: new Date("2025-07-01T11:00:00Z"),
    timezone: "UTC",
    recurrences: [{
        recurrenceType: "Weekly",
        interval: 1,
        dayOfWeek: 3, // Wednesday
    }],
};

const occurrences = ScheduleDbFactory.generateOccurrences(
    schedule,
    {
        start: new Date("2025-07-01T00:00:00Z"),
        end: new Date("2025-07-31T23:59:59Z"),
    }
);
// Returns array of { start: Date, end: Date } for each occurrence
```

## Common Scenarios

### 1. Daily Team Standup
```typescript
const standup = await prisma.schedule.create({
    data: ScheduleDbFactory.createRecurring(
        "team_meeting",
        "Meeting",
        scheduleRecurrencePatterns.dailyStandup("schedule_id")
    )
});
```

### 2. Conference Week Cancellations
```typescript
const exceptions = scheduleExceptionPatterns.conferenceWeek(
    "schedule_id",
    new Date("2025-08-04T00:00:00Z") // Week start
);

await Promise.all(
    exceptions.map(exception => 
        prisma.schedule_exception.create({ data: exception })
    )
);
```

### 3. Holiday Schedule
```typescript
const holidays = scheduleExceptionPatterns.usFederalHolidays("schedule_id", 2025);

await Promise.all(
    holidays.map(holiday => 
        prisma.schedule_exception.create({ data: holiday })
    )
);
```

### 4. Complex Multi-Pattern Schedule
```typescript
const schedule = ScheduleDbFactory.createWithRecurrences(
    "routine_id",
    "RunRoutine",
    [
        // Weekly on Tuesdays
        { recurrenceType: "Weekly", interval: 1, dayOfWeek: 2 },
        // Monthly on 1st
        { recurrenceType: "Monthly", interval: 1, dayOfMonth: 1 },
        // Quarterly using RRULE
        "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=15"
    ]
);
```

## Validation Features

All factories include comprehensive validation:

```typescript
const factory = new ScheduleDbFactory();
const schedule = factory.createMinimal();
const validation = factory.validateFixture(schedule);

if (!validation.isValid) {
    console.log("Errors:", validation.errors);
    console.log("Warnings:", validation.warnings);
}
```

## Edge Cases Supported

- **Leap year events** (February 29th)
- **End-of-month handling** (31st adjusts to last day)
- **Business days only** patterns
- **Multiple timezone scheduling**
- **Long-running events** (multi-day)
- **Frequency limits** (count and until date)

## Testing Integration

### With Prisma
```typescript
import { ScheduleDbFactory } from './fixtures/db';

describe('Schedule Tests', () => {
    it('should create recurring meeting', async () => {
        const scheduleData = ScheduleDbFactory.createRecurring(
            "meeting_123",
            "Meeting",
            { recurrenceType: "Weekly", dayOfWeek: 3 }
        );
        
        const schedule = await prisma.schedule.create({
            data: scheduleData,
            include: { recurrences: true }
        });
        
        expect(schedule.recurrences).toHaveLength(1);
        expect(schedule.recurrences[0].recurrenceType).toBe("Weekly");
    });
});
```

### Bulk Creation
```typescript
const schedules = await Promise.all([
    // One-time event
    prisma.schedule.create({
        data: ScheduleDbFactory.createMinimal("event_1", "Meeting")
    }),
    // Daily recurring
    prisma.schedule.create({
        data: ScheduleDbFactory.createRecurring(
            "standup_1", 
            "Meeting",
            scheduleRecurrencePatterns.dailyStandup("schedule_1")
        )
    }),
    // Weekly with exceptions
    prisma.schedule.create({
        data: ScheduleDbFactory.createWithExceptions(
            "weekly_1",
            "Meeting", 
            [{ 
                date: new Date("2025-07-04T10:00:00Z"),
                type: 'cancel'
            }]
        )
    })
]);
```

## Error Scenarios

Factories include comprehensive error test cases:

- **Missing required fields**
- **Invalid data types**
- **Constraint violations**
- **Business logic errors**
- **Timezone validation**
- **Date range validation**

## Performance Considerations

- Factories generate fresh IDs to avoid conflicts
- Validation is optional for performance-critical tests
- Bulk operations supported for large datasets
- Minimal data structures for basic use cases

## Migration from Legacy Fixtures

The new enhanced factories are backward compatible with existing fixture methods:

```typescript
// Legacy (still works)
const schedule = ScheduleDbFactory.createMinimal("id", "type");

// Enhanced (new features)
const schedule = new ScheduleDbFactory()
    .createWithRelationships({
        withAuth: true,
        overrides: { timezone: "America/New_York" }
    });
```

## Future Enhancements

Planned features for future versions:

- Custom business calendar support
- Advanced RRULE features (BYSETPOS, WKST)
- Schedule conflict detection
- Capacity and resource management
- Integration with external calendar systems

## Conclusion

The Schedule System fixture factories provide comprehensive support for testing all aspects of Vrooli's scheduling functionality, from simple one-time events to complex recurring patterns with exceptions and timezone handling. The factories follow the enhanced pattern with validation, error scenarios, and extensive customization options.