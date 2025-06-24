/**
 * Reasoning Response Fixtures
 * 
 * Pre-defined chain-of-thought and reasoning patterns for testing advanced AI capabilities.
 */

import type { AIMockConfig } from "../types.js";

/**
 * Simple chain of thought
 */
export const simpleReasoning = (): AIMockConfig => ({
    content: "Based on my analysis, the best approach is to use a factory pattern.",
    reasoning: "The user needs to create different types of objects dynamically. Factory pattern provides a clean interface for object creation without exposing instantiation logic.",
    confidence: 0.88,
    model: "gpt-4o"
});

/**
 * Step-by-step reasoning
 */
export const stepByStepReasoning = (): AIMockConfig => ({
    content: "After careful consideration, I recommend implementing a microservices architecture.",
    reasoning: `Let me think through this step by step:
1. Current monolithic architecture is becoming hard to maintain
2. Team size is growing, need independent deployment
3. Different components have different scaling needs
4. Microservices would allow team autonomy
5. However, this adds complexity in service communication
6. Given the team's experience and project requirements, the benefits outweigh the costs`,
    confidence: 0.85,
    model: "o1-mini"
});

/**
 * Mathematical reasoning
 */
export const mathematicalReasoning = (): AIMockConfig => ({
    content: "The optimal solution requires 7 servers to handle the peak load.",
    reasoning: `Calculating server requirements:
- Peak load: 10,000 requests/second
- Each server capacity: 1,500 requests/second
- Required servers: 10,000 / 1,500 = 6.67
- Rounding up for safety: 7 servers
- Adding 20% buffer for failover: 7 * 1.2 = 8.4
- Final recommendation: 7 active + 2 standby = 9 total servers`,
    confidence: 0.92,
    metadata: {
        calculationType: "capacity_planning",
        assumptions: ["uniform load distribution", "no caching layer"]
    }
});

/**
 * Comparative reasoning
 */
export const comparativeReasoning = (): AIMockConfig => ({
    content: "React is the better choice for this project compared to Angular or Vue.",
    reasoning: `Comparing the three frameworks:

React:
+ Large ecosystem and community
+ Flexible and unopinionated
+ Great for complex UIs
- Steeper learning curve for beginners

Angular:
+ Full framework with everything included
+ Strong typing with TypeScript
- Heavyweight for small projects
- Opinionated structure

Vue:
+ Gentle learning curve
+ Good documentation
- Smaller ecosystem
- Less suitable for very large apps

Given your team's experience and project requirements, React offers the best balance.`,
    confidence: 0.87
});

/**
 * Problem decomposition reasoning
 */
export const problemDecomposition = (): AIMockConfig => ({
    content: "I'll solve this complex problem by breaking it into manageable sub-problems.",
    reasoning: `Breaking down the problem:
1. Main problem: Optimize database performance
2. Sub-problems identified:
   a) Slow query execution (50% of issues)
   b) Inefficient indexing (30% of issues)
   c) Connection pool exhaustion (20% of issues)
3. Prioritizing by impact: Start with query optimization
4. Each sub-problem can be tackled independently
5. Solutions can be implemented incrementally`,
    confidence: 0.9,
    metadata: {
        problemComplexity: "high",
        decompositionDepth: 2
    }
});

/**
 * Causal reasoning
 */
export const causalReasoning = (): AIMockConfig => ({
    content: "The performance degradation is caused by memory leaks in the event handlers.",
    reasoning: `Tracing the causal chain:
1. Symptom: Application slows down over time
2. Observation: Memory usage increases linearly
3. Investigation: Event listeners are registered but not removed
4. Root cause: Missing cleanup in useEffect hooks
5. Effect: Each component mount adds listeners without removal
6. Result: Memory accumulation leading to performance issues`,
    confidence: 0.91
});

/**
 * Hypothetical reasoning
 */
export const hypotheticalReasoning = (): AIMockConfig => ({
    content: "If we implement caching, we could reduce database load by approximately 70%.",
    reasoning: `Hypothetical analysis:
- Assumption: 70% of requests are for frequently accessed data
- If we cache these requests with 1-hour TTL
- Then database hits would reduce from 1000/min to 300/min
- This would free up database connections
- Response time would improve from 200ms to 50ms for cached requests
- However, we'd need to handle cache invalidation carefully`,
    confidence: 0.82,
    model: "gpt-4o"
});

/**
 * Diagnostic reasoning
 */
export const diagnosticReasoning = (): AIMockConfig => ({
    content: "The issue appears to be a race condition in the authentication flow.",
    reasoning: `Diagnostic process:
1. Symptoms: Intermittent login failures (occurs ~15% of attempts)
2. Pattern: Failures more common under high load
3. Hypothesis: Timing-dependent issue
4. Test: Added delays between operations - issue disappeared
5. Analysis: Token validation happens before token is fully written
6. Conclusion: Classic race condition between write and read operations`,
    confidence: 0.89
});

/**
 * Strategic reasoning
 */
export const strategicReasoning = (): AIMockConfig => ({
    content: "The optimal strategy is to migrate gradually using the strangler fig pattern.",
    reasoning: `Strategic analysis:
- Goal: Migrate from legacy system to new architecture
- Constraints: Cannot have downtime, limited resources
- Options considered:
  1. Big bang migration: Too risky
  2. Parallel run: Too expensive
  3. Strangler fig: Gradual, low risk
- Strategy: Route new features to new system, migrate old features incrementally
- Benefits: Continuous delivery, easy rollback, learn as we go
- Timeline: 6-month migration with monthly milestones`,
    confidence: 0.86
});

/**
 * Analogical reasoning
 */
export const analogicalReasoning = (): AIMockConfig => ({
    content: "Think of the event bus like a postal service for your application components.",
    reasoning: `Drawing an analogy:
- Event bus = Postal service
- Events = Letters/packages
- Publishers = Senders
- Subscribers = Recipients
- Event types = Different mail categories
- Just as postal service decouples senders and recipients, event bus decouples components
- This mental model helps understand why components don't need direct references`,
    confidence: 0.84
});

/**
 * Probabilistic reasoning
 */
export const probabilisticReasoning = (): AIMockConfig => ({
    content: "There's an 85% chance this deployment will succeed without issues.",
    reasoning: `Probability calculation:
- Base success rate from history: 90%
- Risk factors identified:
  * Major dependency update (-3%)
  * New team member's first deployment (-2%)
  * But we have improved test coverage (+1%)
  * And it's during low-traffic hours (+1%)
- Adjusted probability: 90% - 3% - 2% + 1% + 1% = 87%
- Conservative estimate: 85%`,
    confidence: 0.78,
    metadata: {
        calculationMethod: "bayesian_adjustment"
    }
});

/**
 * Self-correction reasoning
 */
export const selfCorrectionReasoning = (): AIMockConfig => ({
    content: "Actually, let me revise my previous suggestion. A message queue would be better than direct API calls.",
    reasoning: `Initial thought: Use direct API calls for service communication
Wait, reconsidering:
- Direct API calls create tight coupling
- What if a service is down? The whole flow fails
- Message queue provides:
  * Decoupling of services
  * Built-in retry mechanism
  * Better fault tolerance
  * Ability to handle traffic spikes
Correction: Message queue is the better architectural choice here`,
    confidence: 0.91,
    model: "o1-mini"
});

/**
 * Meta-reasoning (reasoning about reasoning)
 */
export const metaReasoning = (): AIMockConfig => ({
    content: "I need more information to provide a confident recommendation.",
    reasoning: `Analyzing my reasoning process:
- I have partial information about the system
- My current confidence would be ~60% with available data
- Missing critical information:
  * Current traffic patterns
  * Budget constraints
  * Team expertise
- Making a recommendation now would be premature
- Better to ask clarifying questions first`,
    confidence: 0.95,
    metadata: {
        reasoningType: "meta-cognitive",
        informationCompleteness: 0.6
    }
});

/**
 * Creative reasoning
 */
export const creativeReasoning = (): AIMockConfig => ({
    content: "Here's an unconventional approach: use the database as a message queue.",
    reasoning: `Creative problem-solving process:
- Constraint: Cannot add new infrastructure (message queue)
- But we need queue-like functionality
- Insight: PostgreSQL has LISTEN/NOTIFY
- We can create a 'tasks' table with status flags
- Workers poll the table and claim tasks with row locks
- This gives us queue semantics using existing infrastructure
- Unconventional but solves the immediate problem`,
    confidence: 0.79,
    model: "gpt-4o"
});