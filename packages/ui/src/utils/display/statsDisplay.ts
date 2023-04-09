/**
 * Aggregate stats object
 */
export type StatsAggregate<T extends { [key: string]: any; }> = Omit<T, '__typename' | 'id' | 'periodStart' | 'periodEnd' | 'periodType'>;

/**
 * Stats object shaped for generating line graphs. 
 */
export type StatsVisual<T extends { [key: string]: any; }> = {
    [key in keyof StatsAggregate<T>]: number[];
};

/**
 * Converts a list of stats to aggregate (e.g. sums, averages) and 
 * visual (i.e. line graph) data
 * @param stats Stats to convert
 * @returns Aggregate and visual data for stats
 */
export const statsDisplay = <Stat extends { [key: string]: any; }>(stats: Stat[]): {
    aggregate: StatsAggregate<Stat>,
    visual: StatsVisual<Stat>
} => {
    // Initialize result
    const aggregate: StatsAggregate<Stat> = {} as any;
    const visual: StatsVisual<Stat> = {} as any;
    // Loop through stats
    for (const stat of stats) {
        // Loop through fields in stat
        for (const key of Object.keys(stat)) {
            // Ignore id, periodStart, periodEnd, and periodType
            if (['__typename', 'id', 'periodStart', 'periodEnd', 'periodType'].includes(key)) continue;
            // Ignore if not a number (should all be numbers at this point, but you never know)
            if (typeof stat[key] !== 'number') continue;
            // If field starts with 'active' (e.g. 'activeUsers'), set aggregate value if it is greater than the current value
            // This will give us the highest occurrence (max) of the field
            if (key.startsWith('active')) {
                if (aggregate[key] === undefined || stat[key] > aggregate[key]) {
                    (aggregate as any)[key] = stat[key];
                }
                // Also add to visual data
                (visual as any)[key] = [...((visual as any)[key] ?? []), stat[key]];
            }
            // Otherwise, sum value
            else {
                (aggregate as any)[key] = ((aggregate as any)[key] ?? 0) + stat[key];
                (visual as any)[key] = [...((visual as any)[key] ?? []), stat[key]];
            }
        }
    }
    // Loop through result fields
    for (const key of Object.keys(aggregate)) {
        // If field ends with 'Average', divide by number of stats
        if (key.endsWith('Average')) {
            (aggregate as any)[key] = (aggregate as any)[key] / stats.length;
        }
    }
    // Return result
    return { aggregate, visual };
}