// AI_CHECK: TYPE_SAFETY=fixed-stats-display-types | LAST: 2025-06-28
// Fixed type safety issues: replaced 'any' types with proper Record types, eliminated 
// unsafe type assertions, improved generic constraints, and added proper type casting
/**
 * Aggregate stats object
 */
export type StatsAggregate<T extends Record<string, unknown>> = Omit<T, "__typename" | "id" | "periodStart" | "periodEnd" | "periodType">;

/**
 * Stats object shaped for generating line graphs. 
 */
export type StatsVisual<T extends Record<string, unknown>> = {
    [key in keyof StatsAggregate<T>]: number[];
};

/**
 * Converts a list of stats to aggregate (e.g. sums, averages) and 
 * visual (i.e. line graph) data
 * @param stats Stats to convert
 * @returns Aggregate and visual data for stats
 */
export const statsDisplay = <Stat extends Record<string, unknown>>(stats: Stat[]): {
    aggregate: StatsAggregate<Stat>,
    visual: StatsVisual<Stat>
} => {
    // Initialize result
    const aggregate = {} as Record<string, number>;
    const visual = {} as Record<string, number[]>;
    // Loop through stats
    for (const stat of stats) {
        // Loop through fields in stat
        for (const key of Object.keys(stat)) {
            // Ignore id, periodStart, periodEnd, and periodType
            if (["__typename", "id", "periodStart", "periodEnd", "periodType"].includes(key)) continue;
            // Ignore if not a number (should all be numbers at this point, but you never know)
            if (typeof stat[key] !== "number") continue;
            // If field starts with 'active' (e.g. 'activeUsers'), set aggregate value if it is greater than the current value
            // This will give us the highest occurrence (max) of the field
            if (key.startsWith("active")) {
                if (aggregate[key] === undefined || stat[key] > aggregate[key]) {
                    aggregate[key] = stat[key] as number;
                }
                // Also add to visual data
                visual[key] = [...(visual[key] ?? []), stat[key] as number];
            }
            // Otherwise, sum value
            else {
                aggregate[key] = (aggregate[key] ?? 0) + (stat[key] as number);
                visual[key] = [...(visual[key] ?? []), stat[key] as number];
            }
        }
    }
    // Loop through result fields
    for (const key of Object.keys(aggregate)) {
        // If field ends with 'Average', divide by number of stats
        if (key.endsWith("Average")) {
            aggregate[key] = aggregate[key] / stats.length;
        }
    }
    // Return result
    return { 
        aggregate: aggregate as StatsAggregate<Stat>, 
        visual: visual as StatsVisual<Stat>, 
    };
};
