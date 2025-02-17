import { BranchProgress, DecisionOption, DecisionStrategyType, DeferredDecisionData, ResolvedDecisionData, ResolvedDecisionDataChooseMultiple, ResolvedDecisionDataChooseOne, RunProgress, SubroutineContext } from "./types.js";

/**
 * Handles making decisions when there are multiple possible next steps.
 * 
 * This allows for different strategies to be used at runtime, such as 
 * user choice, random selection, or LLM-based selection.
 */
export abstract class DecisionStrategy {
    abstract __type: DecisionStrategyType;

    /**
     * Stores the current decision options for a run, both resolved and deferred.
     * 
     * This should be updated before every usage of this strategy.
     */
    protected decisions: RunProgress["decisions"] = [];

    /**
     * Updates the decision options for a run.
     * 
     * This should be called before every usage of this strategy.
     * 
     * @param runProgress The current run progress
     * @param newDecisions Any new decisions to add to the list
     * @returns The updated decision options
     */
    public updateDecisionOptions(runProgress: Pick<RunProgress, "decisions">, newDecisions: (ResolvedDecisionData | DeferredDecisionData)[] = []) {
        // Filter out any existing decisions with the same branchCompositeId
        this.decisions = runProgress.decisions.filter(decision => !newDecisions.some(newDecision => newDecision.key === decision.key));
        // Add the new decisions
        this.decisions.push(...newDecisions);
        return this.decisions;
    }

    /**
     * Checks the provided RunProgress for an already resolved decision for this composite key.
     * 
     * NOTE: We specifically ignore deferred decisions here because we don't want to block the run 
     * if it's resumed with existing deferred decisions. Any time a deferred decision is created, the branch 
     * is paused (meaning this method won't be used again) until the decision is resolved. If the branch ends up being resumed with the decision not 
     * resolved, then we should ignore the old deferred decision and create a new one.
     *
     * @param decisionKey The unique identifier for the decision
     * @returns The resolved decision if it exists, or null if not.
     */
    private findResolvedDecision(decisionKey: string): ResolvedDecisionData | null {
        // Find the decision in the stored `decisions` array
        const decision = this.decisions.find(decision => decision.key === decisionKey);
        if (!decision) {
            return null;
        }
        // Only return the decision if it's resolved
        if (decision.__type === "Resolved") {
            return decision;
        }
        return null;
    }

    /**
     * Generates a composite key for a decision based on the branch composite ID and a context string.
     * This key should uniquely identify a decision point within a run.
     *
     * @param branch Information about the branch the decision is being made for
     * @param decisionContext A string identifying the specific decision point (e.g., the calling method's name).
     * @returns A composite key to uniquely identify the decision
     */
    public generateDecisionKey(branch: BranchProgress, decisionContext: string): string {
        return `${branch.branchId}.${decisionContext}`;
    }

    /**
     * Picks one option from a list of possible next steps, 
     * without checking if a resolved decision already exists.
     * 
     * @param options The possible next steps
     * @param decisionKey The unique identifier for the decision
     * @param subcontext The current context for the subroutine
     * @returns The ID of the chosen next step
     */
    abstract pickOne(
        options: DecisionOption[],
        decisionKey: string,
        subcontext: SubroutineContext,
    ): Promise<ResolvedDecisionDataChooseOne | DeferredDecisionData>;

    /**
     * Picks multiple options from a list of possible next steps, 
     * without checking if a resolved decision already exists.
     * 
     * @param options The possible next steps
     * @param decisionKey The unique identifier for the decision
     * @param subcontext The current context for the subroutine
     * @returns The IDs of the chosen next steps
     */
    abstract pickMultiple(
        options: DecisionOption[],
        decisionKey: string,
        subcontext: SubroutineContext,
    ): Promise<ResolvedDecisionDataChooseMultiple | DeferredDecisionData>;

    /**
     * Chooses one option from a list of possible next steps.
     * 
     * If a resolved decision already exists, it will be returned immediately.
     * 
     * @param options The possible next steps
     * @param decisionKey The unique identifier for the decision
     * @param subcontext The current context for the subroutine
     * @returns The ID of the chosen next step
     */
    public async chooseOne(
        options: DecisionOption[],
        decisionKey: string,
        subcontext: SubroutineContext,
    ): Promise<ResolvedDecisionDataChooseOne | DeferredDecisionData> {
        const resolvedDecision = this.findResolvedDecision(decisionKey);
        if (resolvedDecision && resolvedDecision.decisionType === "chooseOne") {
            return resolvedDecision;
        }
        return this.pickOne(options, decisionKey, subcontext);
    }

    /**
     * Chooses multiple options from a list of possible next steps.
     * 
     * If a resolved decision already exists, it will be returned immediately.
     * 
     * @param options The possible next steps
     * @param decisionKey The unique identifier for the decision
     * @param subcontext The current context for the subroutine
     * @returns The IDs of the chosen next steps
     */
    public async chooseMultiple(
        options: DecisionOption[],
        decisionKey: string,
        subcontext: SubroutineContext,
    ): Promise<ResolvedDecisionDataChooseMultiple | DeferredDecisionData> {
        const resolvedDecision = this.findResolvedDecision(decisionKey);
        if (resolvedDecision && resolvedDecision.decisionType === "chooseMultiple") {
            return resolvedDecision;
        }
        return this.pickMultiple(options, decisionKey, subcontext);
    }
}

/**
 * Barebones decision strategy that picks the first option.
 */
export class AutoPickFirstStrategy extends DecisionStrategy {
    __type = DecisionStrategyType.AutoPickFirst;

    async pickOne(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseOne> {
        const nextNode = options[0];
        if (!nextNode) {
            throw new Error("No valid next nodes");
        }
        return {
            __type: "Resolved" as const,
            decisionType: "chooseOne" as const,
            key: decisionKey,
            result: nextNode.nodeId,
        };
    }

    async pickMultiple(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseMultiple> {
        const result = options.map(option => option.nodeId);
        return {
            __type: "Resolved" as const,
            decisionType: "chooseMultiple" as const,
            key: decisionKey,
            result,
        };
    }
}

/**
 * Decision strategy that picks a random option.
 */
export class AutoPickRandomStrategy extends DecisionStrategy {
    __type = DecisionStrategyType.AutoPickRandom;

    async pickOne(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseOne> {
        const nextNode = options[Math.floor(Math.random() * options.length)];
        if (!nextNode) {
            throw new Error("No valid next nodes");
        }
        return {
            __type: "Resolved" as const,
            decisionType: "chooseOne" as const,
            key: decisionKey,
            result: nextNode.nodeId,
        };
    }

    async pickMultiple(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseMultiple> {
        const result = options.filter(() => Math.random() < 0.5).map(option => option.nodeId);
        return {
            __type: "Resolved" as const,
            decisionType: "chooseMultiple" as const,
            key: decisionKey,
            result,
        };
    }
}

