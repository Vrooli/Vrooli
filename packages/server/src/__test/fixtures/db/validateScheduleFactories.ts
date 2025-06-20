/**
 * Validation script for Schedule System fixture factories
 * Can be run directly to verify factory functionality
 */

import { 
    ScheduleDbFactory, 
    ScheduleExceptionDbFactory, 
    ScheduleRecurrenceDbFactory,
    RRuleHelpers
} from "./index.js";

/**
 * Run validation tests for the Schedule System factories
 */
export function validateScheduleSystemFactories(): { 
    success: boolean; 
    results: string[]; 
    errors: string[];
} {
    const results: string[] = [];
    const errors: string[] = [];

    try {
        // Test 1: Basic Schedule creation
        results.push("✓ Testing ScheduleDbFactory.createMinimal...");
        const schedule = ScheduleDbFactory.createMinimal("meeting_123", "Meeting");
        if (!schedule.id || !schedule.startTime || !schedule.timezone) {
            throw new Error("Missing required schedule fields");
        }
        results.push("  - Schedule has required fields: id, startTime, timezone");

        // Test 2: Schedule with recurrence
        results.push("✓ Testing ScheduleDbFactory.createRecurring...");
        const recurringSchedule = ScheduleDbFactory.createRecurring(
            "project_456",
            "RunProject",
            {
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 3,
            }
        );
        if (!recurringSchedule.recurrences?.create) {
            throw new Error("Recurring schedule missing recurrence data");
        }
        results.push("  - Recurring schedule has recurrence data");

        // Test 3: RRULE generation
        results.push("✓ Testing RRuleHelpers...");
        const dailyRule = RRuleHelpers.daily(2);
        if (dailyRule !== "FREQ=DAILY;INTERVAL=2") {
            throw new Error(`Expected daily RRULE, got: ${dailyRule}`);
        }
        
        const weeklyRule = RRuleHelpers.weekly(["MO", "WE", "FR"]);
        if (!weeklyRule.includes("BYDAY=MO,WE,FR")) {
            throw new Error(`Expected MWF RRULE, got: ${weeklyRule}`);
        }
        results.push("  - RRULE generation works correctly");

        // Test 4: RRULE parsing
        results.push("✓ Testing RRULE parsing...");
        const parsed = RRuleHelpers.parseToRecurrence("FREQ=WEEKLY;INTERVAL=1;BYDAY=TU");
        if (parsed.recurrenceType !== "Weekly" || parsed.dayOfWeek !== 2) {
            throw new Error(`RRULE parsing failed: ${JSON.stringify(parsed)}`);
        }
        results.push("  - RRULE parsing works correctly");

        // Test 5: Schedule Exception creation
        results.push("✓ Testing ScheduleExceptionDbFactory...");
        const exception = ScheduleExceptionDbFactory.createCancellation(
            "schedule_789",
            new Date("2025-07-04T10:00:00Z")
        );
        if (exception.newStartTime !== null || exception.newEndTime !== null) {
            throw new Error("Cancellation should have null new times");
        }
        results.push("  - Exception creation works correctly");

        // Test 6: Schedule Recurrence creation
        results.push("✓ Testing ScheduleRecurrenceDbFactory...");
        const recurrence = ScheduleRecurrenceDbFactory.createWeekly(
            "schedule_101",
            5, // Friday
            1
        );
        if (recurrence.recurrenceType !== "Weekly" || recurrence.dayOfWeek !== 5) {
            throw new Error("Weekly recurrence creation failed");
        }
        results.push("  - Recurrence creation works correctly");

        // Test 7: Validation
        results.push("✓ Testing factory validation...");
        const factory = new ScheduleDbFactory();
        const validSchedule = factory.createMinimal();
        const validation = factory.validateFixture(validSchedule);
        if (!validation.isValid) {
            throw new Error(`Valid schedule failed validation: ${validation.errors.join(", ")}`);
        }
        results.push("  - Factory validation works correctly");

        // Test 8: Occurrence generation
        results.push("✓ Testing occurrence generation...");
        const testSchedule = {
            startTime: new Date("2025-07-01T10:00:00Z"),
            endTime: new Date("2025-07-01T11:00:00Z"),
            timezone: "UTC",
            recurrences: [{
                recurrenceType: "Weekly",
                interval: 1,
                dayOfWeek: 2, // Tuesday
                dayOfMonth: null,
                month: null,
                endDate: null,
            }],
        };
        
        const occurrences = ScheduleDbFactory.generateOccurrences(
            testSchedule,
            {
                start: new Date("2025-07-01T00:00:00Z"),
                end: new Date("2025-07-31T23:59:59Z"),
            }
        );
        
        if (occurrences.length === 0) {
            throw new Error("No occurrences generated");
        }
        results.push(`  - Generated ${occurrences.length} occurrences for July 2025`);

        // Test 9: Complex RRULE from string
        results.push("✓ Testing RRULE from string...");
        const complexRRule = "FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=15";
        const complexRecurrence = ScheduleRecurrenceDbFactory.createFromRRule(
            "schedule_202",
            complexRRule
        );
        if (complexRecurrence.recurrenceType !== "Monthly" || 
            complexRecurrence.interval !== 3 || 
            complexRecurrence.dayOfMonth !== 15) {
            throw new Error("Complex RRULE parsing failed");
        }
        results.push("  - Complex RRULE parsing works correctly");

        // Test 10: Multi-day weekly pattern
        results.push("✓ Testing multi-day weekly pattern...");
        const multiDayRecurrences = ScheduleRecurrenceDbFactory.createMultiDayWeekly(
            "schedule_303",
            [1, 3, 5], // MWF
            { duration: 60 }
        );
        if (multiDayRecurrences.length !== 3) {
            throw new Error("Multi-day weekly pattern should create 3 recurrences");
        }
        results.push("  - Multi-day weekly pattern works correctly");

        results.push("\n🎉 All Schedule System factory tests passed!");
        return { success: true, results, errors };

    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        errors.push(`❌ Test failed: ${errorMessage}`);
        return { success: false, results, errors };
    }
}

// Run validation if this file is executed directly
if (import.meta.url.endsWith(process.argv[1])) {
    const { success, results, errors } = validateScheduleSystemFactories();
    
    console.log("Schedule System Factory Validation");
    console.log("=".repeat(40));
    
    results.forEach(result => console.log(result));
    
    if (errors.length > 0) {
        console.log("\nErrors:");
        errors.forEach(error => console.log(error));
    }
    
    console.log(`\nResult: ${success ? "✅ SUCCESS" : "❌ FAILURE"}`);
    process.exit(success ? 0 : 1);
}