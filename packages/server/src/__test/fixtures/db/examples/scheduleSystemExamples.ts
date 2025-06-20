/**
 * Examples demonstrating the usage of Schedule System fixture factories
 * Shows how to create complex scheduling scenarios for testing
 */

import { 
    ScheduleDbFactory, 
    ScheduleExceptionDbFactory, 
    ScheduleRecurrenceDbFactory,
    RRuleHelpers,
    scheduleExceptionPatterns,
    scheduleRecurrencePatterns
} from "../index.js";

/**
 * Example 1: One-time event
 */
export const oneTimeEventExample = () => {
    const meetingId = "meeting_123";
    
    return ScheduleDbFactory.createMinimal(meetingId, "Meeting", {
        startTime: new Date("2025-07-15T14:00:00Z"),
        endTime: new Date("2025-07-15T15:30:00Z"),
        timezone: "America/New_York",
    });
};

/**
 * Example 2: Daily recurring meeting with exceptions
 */
export const dailyRecurringWithExceptionsExample = async (prisma: any) => {
    const meetingId = "meeting_456";
    
    // Create base schedule with daily recurrence
    const schedule = await prisma.schedule.create({
        data: ScheduleDbFactory.createRecurring(
            meetingId,
            "Meeting",
            {
                recurrenceType: "Daily",
                interval: 1,
                duration: 60,
                endDate: new Date("2025-12-31T23:59:59Z"),
            },
            {
                startTime: new Date("2025-07-01T09:00:00Z"),
                endTime: new Date("2025-07-01T10:00:00Z"),
                timezone: "UTC",
            }
        ),
    });
    
    // Add exceptions for holidays
    const exceptions = await Promise.all([
        prisma.schedule_exception.create({
            data: ScheduleExceptionDbFactory.createHolidayException(
                schedule.id,
                new Date("2025-07-04T09:00:00Z"),
                "Independence Day"
            ),
        }),
        prisma.schedule_exception.create({
            data: ScheduleExceptionDbFactory.createRescheduled(
                schedule.id,
                new Date("2025-07-15T09:00:00Z"),
                new Date("2025-07-15T14:00:00Z"), // Moved to afternoon
                new Date("2025-07-15T15:00:00Z")
            ),
        }),
    ]);
    
    return { schedule, exceptions };
};

/**
 * Example 3: Complex schedule with RRULE
 */
export const complexRRuleExample = () => {
    const projectId = "project_789";
    
    // Every Monday, Wednesday, Friday at 2 PM
    const rrule = RRuleHelpers.weekly(["MO", "WE", "FR"], 1);
    const withEndDate = RRuleHelpers.withUntil(rrule, new Date("2025-12-31T23:59:59Z"));
    
    return ScheduleDbFactory.createWithRRule(
        projectId,
        "RunProject",
        withEndDate,
        {
            startTime: new Date("2025-07-01T14:00:00Z"),
            endTime: new Date("2025-07-01T15:00:00Z"),
            timezone: "Europe/London",
        }
    );
};

/**
 * Example 4: Monthly schedule on specific day
 */
export const monthlyScheduleExample = () => {
    const focusModeId = "focus_mode_101";
    
    // Every month on the 15th
    return ScheduleDbFactory.createRecurring(
        focusModeId,
        "FocusMode",
        {
            recurrenceType: "Monthly",
            interval: 1,
            dayOfMonth: 15,
            duration: 120,
        },
        {
            startTime: new Date("2025-07-15T10:00:00Z"),
            endTime: new Date("2025-07-15T12:00:00Z"),
            timezone: "Asia/Tokyo",
        }
    );
};

/**
 * Example 5: Complex schedule with multiple recurrences
 */
export const multipleRecurrencesExample = async (prisma: any) => {
    const routineId = "routine_202";
    
    // Schedule with both weekly and monthly patterns
    const scheduleData = ScheduleDbFactory.createWithRecurrences(
        routineId,
        "RunRoutine",
        [
            // Weekly on Tuesdays
            {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 2,
                duration: 60,
            },
            // Monthly on the 1st
            {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 1,
                duration: 180,
            },
            // From RRULE: Quarterly on 15th
            "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=15",
        ],
        {
            startTime: new Date("2025-07-01T09:00:00Z"),
            endTime: new Date("2025-07-01T10:00:00Z"),
            timezone: "Pacific/Auckland",
        }
    );
    
    return prisma.schedule.create({ data: scheduleData });
};

/**
 * Example 6: Generate occurrences for testing
 */
export const generateOccurrencesExample = () => {
    const schedule = {
        startTime: new Date("2025-07-01T10:00:00Z"),
        endTime: new Date("2025-07-01T11:00:00Z"),
        timezone: "UTC",
        recurrences: [
            {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 3, // Wednesday
                dayOfMonth: null,
                month: null,
                endDate: null,
            },
        ],
    };
    
    // Get all occurrences for July 2025
    const occurrences = ScheduleDbFactory.generateOccurrences(
        schedule,
        {
            start: new Date("2025-07-01T00:00:00Z"),
            end: new Date("2025-07-31T23:59:59Z"),
        }
    );
    
    return occurrences; // Will return 5 Wednesday occurrences
};

/**
 * Example 7: Conference week with all meetings cancelled
 */
export const conferenceWeekExample = async (prisma: any) => {
    const meetingId = "meeting_303";
    
    // Create recurring meeting
    const schedule = await prisma.schedule.create({
        data: ScheduleDbFactory.createRecurring(
            meetingId,
            "Meeting",
            scheduleRecurrencePatterns.weeklyTeamMeeting(meetingId)
        ),
    });
    
    // Cancel entire conference week
    const conferenceWeekStart = new Date("2025-08-04T00:00:00Z");
    const exceptions = scheduleExceptionPatterns.conferenceWeek(
        schedule.id,
        conferenceWeekStart
    );
    
    await Promise.all(
        exceptions.map(exception => 
            prisma.schedule_exception.create({ data: exception })
        )
    );
    
    return { schedule, exceptionsCount: exceptions.length };
};

/**
 * Example 8: Yearly event with timezone handling
 */
export const yearlyEventWithTimezoneExample = () => {
    const eventId = "event_404";
    
    return ScheduleDbFactory.createRecurring(
        eventId,
        "Meeting",
        {
            recurrenceType: "Yearly",
            interval: 1,
            month: 6, // June
            dayOfMonth: 15,
            duration: 480, // All day (8 hours)
        },
        {
            startTime: new Date("2025-06-15T00:00:00Z"),
            endTime: new Date("2025-06-15T08:00:00Z"),
            timezone: "Pacific/Honolulu", // Different timezone
        }
    );
};

/**
 * Example 9: Bulk creation for testing
 */
export const bulkScheduleCreationExample = async (prisma: any) => {
    const schedules = [];
    
    // Create 10 different types of schedules
    for (let i = 0; i < 10; i++) {
        const scheduleType = i % 4;
        let scheduleData;
        
        switch (scheduleType) {
            case 0: // One-time
                scheduleData = ScheduleDbFactory.createMinimal(
                    `meeting_${i}`,
                    "Meeting"
                );
                break;
            case 1: // Daily
                scheduleData = ScheduleDbFactory.createRecurring(
                    `routine_${i}`,
                    "RunRoutine",
                    scheduleRecurrencePatterns.dailyStandup(`schedule_${i}`)
                );
                break;
            case 2: // Weekly
                scheduleData = ScheduleDbFactory.createRecurring(
                    `project_${i}`,
                    "RunProject",
                    scheduleRecurrencePatterns.weeklyTeamMeeting(`schedule_${i}`)
                );
                break;
            case 3: // Monthly
                scheduleData = ScheduleDbFactory.createRecurring(
                    `focus_${i}`,
                    "FocusMode",
                    scheduleRecurrencePatterns.monthlyAllHands(`schedule_${i}`)
                );
                break;
        }
        
        const schedule = await prisma.schedule.create({ data: scheduleData });
        schedules.push(schedule);
    }
    
    return schedules;
};

/**
 * Example 10: Testing edge cases
 */
export const edgeCaseExamples = () => {
    const examples = [];
    
    // Leap year event
    examples.push(
        ScheduleDbFactory.createRecurring(
            "event_leap",
            "Meeting",
            {
                recurrenceType: "Yearly",
                interval: 1,
                month: 2,
                dayOfMonth: 29, // Feb 29
                duration: 60,
            }
        )
    );
    
    // Very frequent recurrence
    examples.push(
        ScheduleDbFactory.createRecurring(
            "routine_frequent",
            "RunRoutine",
            {
                recurrenceType: "Daily",
                interval: 1,
                duration: 30,
                endDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 1 week
            }
        )
    );
    
    // Monthly on last day (31st will adjust to last day of month)
    examples.push(
        ScheduleDbFactory.createRecurring(
            "project_lastday",
            "RunProject",
            {
                recurrenceType: "Monthly",
                interval: 1,
                dayOfMonth: 31,
                duration: 120,
            }
        )
    );
    
    return examples;
};