import { z } from 'zod';

/**
 * Scenario-related Zod schemas for API response validation.
 */

// Scenario schema
export const ScenarioSchema = z.object({
  name: z.string(),
  description: z.string(),
  status: z.string(),
});

// List scenarios response schema
export const ListScenariosResponseSchema = z.object({
  scenarios: z.array(ScenarioSchema),
});

// Export types
export type Scenario = z.infer<typeof ScenarioSchema>;
export type ListScenariosResponse = z.infer<typeof ListScenariosResponseSchema>;
