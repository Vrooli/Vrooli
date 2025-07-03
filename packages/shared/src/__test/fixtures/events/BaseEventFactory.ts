/**
 * Base implementation of EventFixtureFactory with common patterns
 * All event fixture factories should extend this class
 */

import {
    type BaseFixtureEvent,
    type CorrelatedEvent,
    type EventEffect,
    type EventFactoryOptions,
    type EventFixtureFactory,
    type EventPattern,
    type SimulationOptions,
    type StateChangeLog,
    type StatefulEvent,
    type TestResult,
    type TimedEvent,
} from "./types.js";

export abstract class BaseEventFactory<TEvent extends BaseFixtureEvent, TData = unknown>
    implements EventFixtureFactory<TEvent, TData> {

    protected options: EventFactoryOptions<TData>;
    protected eventName: string;
    protected stateTracker: Map<string, Record<string, unknown>> = new Map();

    constructor(eventName: string, options: EventFactoryOptions<TData> = {}) {
        this.eventName = eventName;
        this.options = options;
    }

    // Abstract properties to be implemented by subclasses
    abstract single: TEvent;
    abstract sequence: TEvent[];
    abstract variants: Record<string, TEvent | TEvent[]>;

    /**
     * Create a single event with optional overrides
     */
    create(overrides?: Partial<TData>): TEvent {
        const baseData = this.extractData(this.single);
        const mergedData = this.mergeData(baseData, overrides);

        if (this.options.validation) {
            const validationResult = this.options.validation(mergedData);
            if (validationResult !== true) {
                throw new Error(`Event validation failed: ${validationResult}`);
            }
        }

        const transformedData = this.options.transform
            ? this.options.transform(mergedData)
            : mergedData;

        return {
            event: this.eventName,
            data: transformedData,
        } as TEvent;
    }

    /**
     * Create a sequence of events based on a pattern
     */
    createSequence(
        pattern: EventPattern,
        options?: { count?: number; interval?: number; data?: Partial<TData> },
    ): TEvent[] {
        const { count = 5, interval = 1000, data = {} } = options || {};
        const events: TEvent[] = [];

        switch (pattern) {
            case "single":
                events.push(this.create(data));
                break;

            case "burst":
                for (let i = 0; i < count; i++) {
                    events.push(this.create({ ...data, sequence: i } as unknown as Partial<TData>));
                }
                break;

            case "periodic":
                for (let i = 0; i < count; i++) {
                    const event = this.withDelay(
                        this.create({ ...data, sequence: i } as unknown as Partial<TData>),
                        i * interval,
                    );
                    events.push(event as unknown as TEvent);
                }
                break;

            case "random":
                for (let i = 0; i < count; i++) {
                    const randomDelay = Math.random() * interval * 2;
                    const event = this.withDelay(
                        this.create({ ...data, sequence: i } as unknown as Partial<TData>),
                        randomDelay,
                    );
                    events.push(event as unknown as TEvent);
                }
                break;

            case "escalating":
                for (let i = 0; i < count; i++) {
                    const escalatingInterval = interval / (i + 1);
                    const escalatingDelay = i === 0 ? 0 : i * escalatingInterval;
                    const event = this.withDelay(
                        this.create({ ...data, sequence: i } as unknown as Partial<TData>),
                        escalatingDelay,
                    );
                    events.push(event as unknown as TEvent);
                }
                break;

            case "degrading":
                for (let i = 0; i < count; i++) {
                    const degradingInterval = interval * (i + 1);
                    const degradingDelay = i === 0 ? 0 : i * degradingInterval;
                    const event = this.withDelay(
                        this.create({ ...data, sequence: i } as unknown as Partial<TData>),
                        degradingDelay,
                    );
                    events.push(event as unknown as TEvent);
                }
                break;
        }

        return events;
    }

    /**
     * Create correlated events with tracking metadata
     */
    createCorrelated(correlationId: string, events: TEvent[]): CorrelatedEvent<TData>[] {
        return events.map((event, index) => ({
            event: event.event,
            data: this.extractData(event),
            metadata: {
                correlationId,
                sequence: index,
                timestamp: Date.now() + (index * 100),
                causedBy: index > 0 ? events[index - 1].event : undefined,
                causes: index < events.length - 1 ? [events[index + 1].event] : [],
            },
        }));
    }

    /**
     * Add timing information to events
     */
    withTiming(events: TEvent[], intervals: number[]): TimedEvent<TData>[] {
        return events.map((event, index) => ({
            event: event.event,
            data: this.extractData(event),
            timing: {
                delay: intervals[index] || 0,
                timestamp: Date.now() + (intervals[index] || 0),
            },
        }));
    }

    /**
     * Add delay to a single event
     */
    withDelay(event: TEvent, delay: number): TimedEvent<TData> {
        return {
            event: event.event,
            data: this.extractData(event),
            timing: {
                delay,
                timestamp: Date.now() + delay,
            },
        };
    }

    /**
     * Add jitter to event timing
     */
    withJitter(events: TEvent[], baseDelay: number, jitter: number): TimedEvent<TData>[] {
        return events.map((event) => {
            const randomJitter = (Math.random() - 0.5) * 2 * jitter;
            const finalDelay = Math.max(0, baseDelay + randomJitter);

            return {
                event: event.event,
                data: this.extractData(event),
                timing: {
                    delay: finalDelay,
                    jitter: randomJitter,
                    timestamp: Date.now() + finalDelay,
                },
            };
        });
    }

    /**
     * Add state tracking to an event
     */
    withState(
        event: TEvent,
        state: { before: Record<string, unknown>; after: Record<string, unknown> },
    ): StatefulEvent<TData> {
        const changed = Object.keys(state.after).filter(
            key => state.before[key] !== state.after[key],
        );

        return {
            event: event.event,
            data: this.extractData(event),
            state: {
                before: state.before,
                after: state.after,
                changed,
            },
        };
    }

    /**
     * Track state changes across a sequence of events
     */
    trackStateChanges(events: TEvent[]): StateChangeLog {
        const changes: StateChangeLog["changes"] = [];
        let currentState: Record<string, unknown> = {};

        events.forEach((event) => {
            const stateBefore = { ...currentState };
            const stateAfter = this.applyEventToState(stateBefore, event);

            const diff = Object.keys(stateAfter)
                .filter(key => stateBefore[key] !== stateAfter[key])
                .map(key => ({
                    property: key,
                    before: stateBefore[key],
                    after: stateAfter[key],
                }));

            if (diff.length > 0) {
                changes.push({
                    event: event.event,
                    timestamp: Date.now(),
                    stateBefore,
                    stateAfter,
                    diff,
                });
            }

            currentState = stateAfter;
        });

        return { changes };
    }

    /**
     * Validate that events occur in the expected order
     */
    validateEventOrder(events: TEvent[]): boolean {
        // Check if events have timing information
        const timedEvents = events.filter((e) =>
            typeof e === "object" && e !== null && "timing" in e,
        ) as unknown as TimedEvent<TData>[];
        if (timedEvents.length === 0) return true;

        // Verify timestamps are in ascending order
        for (let i = 1; i < timedEvents.length; i++) {
            const prevEvent = timedEvents[i - 1];
            const currEvent = timedEvents[i];
            const prevTimestamp = prevEvent.timing?.timestamp || 0;
            const currTimestamp = currEvent.timing?.timestamp || 0;
            if (currTimestamp < prevTimestamp) return false;
        }

        return true;
    }

    /**
     * Simulate event flow with network conditions and timing
     */
    async simulateEventFlow(
        events: TEvent[],
        options?: SimulationOptions,
    ): Promise<TestResult> {
        const result: TestResult = {
            success: true,
            events: [],
            stateChanges: [],
            errors: [],
            duration: 0,
        };

        const startTime = Date.now();
        const state = { ...(options?.state || {}) };

        for (const event of events) {
            try {
                // Apply network conditions
                const processedEvent = await this.applyNetworkConditions(event, options);

                // Apply error simulation
                if (options?.errors && this.shouldInjectError(event, options.errors)) {
                    throw new Error(`Simulated error for event: ${event.event}`);
                }

                // Record event
                result.events.push({
                    event: processedEvent.event,
                    timestamp: Date.now(),
                    data: this.extractData(processedEvent),
                });

                // Track state changes
                const oldState = { ...state };
                Object.assign(state, this.applyEventToState(state, processedEvent));

                const stateChanges = Object.keys(state)
                    .filter(key => oldState[key] !== state[key])
                    .map(key => ({
                        timestamp: Date.now(),
                        property: key,
                        oldValue: oldState[key],
                        newValue: state[key],
                    }));

                result.stateChanges.push(...stateChanges);

                // Apply timing
                if (options?.timing !== "instant" && this.isTimedEvent(processedEvent)) {
                    const timedEvent = processedEvent as unknown as TimedEvent<TData>;
                    const delay = options?.timing === "fast"
                        ? timedEvent.timing.delay ? timedEvent.timing.delay / 10 : 0
                        : timedEvent.timing.delay || 0;

                    await new Promise(resolve => setTimeout(resolve, delay));
                }

            } catch (error) {
                result.success = false;
                result.errors.push(error as Error);
            }
        }

        result.duration = Date.now() - startTime;
        return result;
    }

    /**
     * Assert that an event produces expected effects
     */
    assertEventEffects(event: TEvent, expectedEffects: EventEffect[]): void {
        for (const effect of expectedEffects) {
            switch (effect.type) {
                case "state":
                    if (effect.property && effect.value !== undefined) {
                        const state = this.applyEventToState({}, event);
                        if (state[effect.property] !== effect.value) {
                            throw new Error(
                                `Expected state.${effect.property} to be ${effect.value}, ` +
                                `but got ${state[effect.property]}`,
                            );
                        }
                    }
                    break;

                case "event":
                    if (effect.event && event.event !== effect.event) {
                        throw new Error(
                            `Expected event type ${effect.event}, but got ${event.event}`,
                        );
                    }
                    break;

                case "timing":
                    if (this.isTimedEvent(event)) {
                        const timedEvent = event as unknown as TimedEvent<TData>;
                        const delay = timedEvent.timing.delay || 0;

                        if (effect.minDelay !== undefined && delay < effect.minDelay) {
                            throw new Error(
                                `Expected delay >= ${effect.minDelay}, but got ${delay}`,
                            );
                        }

                        if (effect.maxDelay !== undefined && delay > effect.maxDelay) {
                            throw new Error(
                                `Expected delay <= ${effect.maxDelay}, but got ${delay}`,
                            );
                        }
                    }
                    break;
            }
        }
    }

    // Protected helper methods

    protected extractData(event: TEvent | TimedEvent<TData> | CorrelatedEvent<TData>): TData {
        return (event as BaseFixtureEvent).data as TData;
    }

    protected mergeData(base: TData, overrides?: Partial<TData>): TData {
        return { ...base, ...overrides } as TData;
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: TEvent,
    ): Record<string, unknown> {
        // Override in subclasses to define how events modify state
        return { ...state, lastEvent: event.event, lastEventTime: Date.now() };
    }

    protected async applyNetworkConditions(
        event: TEvent,
        options?: SimulationOptions,
    ): Promise<TEvent> {
        if (!options?.network || options.network === "fast") return event;

        const conditions = this.getNetworkConditions(options.network);

        // Simulate packet loss
        if (Math.random() * 100 < conditions.loss) {
            throw new Error("Packet lost");
        }

        // Apply latency
        const delay = conditions.latency + (Math.random() - 0.5) * 2 * conditions.jitter;
        await new Promise(resolve => setTimeout(resolve, delay));

        return event;
    }

    protected getNetworkConditions(network: SimulationOptions["network"]): {
        latency: number;
        jitter: number;
        loss: number;
    } {
        if (typeof network === "object") return network;

        const presets = {
            fast: { latency: 5, jitter: 1, loss: 0 },
            slow: { latency: 200, jitter: 50, loss: 2 },
            flaky: { latency: 100, jitter: 100, loss: 10 },
            offline: { latency: 0, jitter: 0, loss: 100 },
        };

        return presets[network] || presets.fast;
    }

    protected shouldInjectError(event: TEvent, errors: SimulationOptions["errors"]): boolean {
        if (!errors) return false;

        const relevantError = errors.find(e => e.event === event.event);
        if (!relevantError) return false;

        return Math.random() < relevantError.probability;
    }

    // Helper to check if event is a TimedEvent
    private isTimedEvent(event: unknown): event is TimedEvent<TData> {
        return typeof event === "object" && event !== null && "timing" in event;
    }
}
