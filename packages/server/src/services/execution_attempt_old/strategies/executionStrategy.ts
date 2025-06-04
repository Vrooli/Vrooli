import { type Logger } from "winston";
import { type ContextBuilder } from "../../conversation/contextBuilder.js";
import { type MessageStore } from "../../conversation/messageStore.js";
import { type ReasoningEngine } from "../../conversation/responseEngine.js";
import { type ToolRunner } from "../../conversation/toolRunner.js";
import { type BotParticipant } from "../../conversation/types.js";
import { type Tool } from "../../mcp/types.js";
import { type ExecutionContext } from "../executionContext.js";
import { type SubroutineExecutionResult } from "../unifiedExecutionEngine.js";

/**
 * Dependencies provided to execution strategies.
 */
export interface ExecutionStrategyDependencies {
    /** The execution context */
    context: ExecutionContext;
    /** The reasoning engine for AI interactions */
    reasoningEngine: ReasoningEngine;
    /** The tool runner for executing tools */
    toolRunner: ToolRunner;
    /** The context builder for preparing prompts */
    contextBuilder: ContextBuilder;
    /** The message store for conversation history */
    messageStore: MessageStore;
    /** The bot participant executing this subroutine */
    botParticipant: BotParticipant;
    /** Available tools for this execution */
    availableTools: Tool[];
    /** Optional abort signal for cancellation */
    abortSignal?: AbortSignal;
    /** Logger instance */
    logger: Logger;
}

/**
 * Base interface for execution strategies.
 * 
 * Each strategy implements a different approach to executing subroutines:
 * - ReasoningStrategy: For meta-cognitive and self-review processes
 * - DeterministicStrategy: For strict, reliable executions
 * - ConversationalStrategy: For multi-turn AI interactions
 */
export interface ExecutionStrategy {
    /** Name of the strategy for logging and identification */
    readonly name: string;

    /** 
     * Executes a subroutine according to this strategy's approach.
     * 
     * @param deps Dependencies needed for execution
     * @returns The execution result with outputs, credits used, etc.
     */
    execute(deps: ExecutionStrategyDependencies): Promise<SubroutineExecutionResult>;

    /**
     * Checks if this strategy can handle a given routine type.
     * 
     * @param routineSubType The subtype of the routine
     * @param config Optional routine configuration
     * @returns True if this strategy can handle the routine
     */
    canHandle(routineSubType: string, config?: Record<string, unknown>): boolean;
} 
