import { type PassableLogger } from "../consts/commonTypes.js";
import { SubroutineContextManager } from "./context.js";
import { type RunProgress } from "./types.js";

/**
 * Handles external event delivery for BPMN boundary events and intermediate events.
 * This class is designed to be used by event bus listeners, webhooks, and other
 * external systems that need to deliver messages or signals to running workflows.
 */
export class BpmnEventBus {
    /**
     * Delivers a message to all relevant subroutine instances that might be waiting for it.
     * This is typically called when a BPMN message is received from an external system.
     * 
     * @param messageId The ID of the message to deliver (should match messageRef.id in BPMN)
     * @param targetInstances Optional list of specific subroutine instance IDs to deliver to.
     *                       If not provided, delivers to all instances in the run.
     * @param run The run progress containing all subroutine contexts
     * @param logger Logger for error reporting
     */
    static deliverMessage(
        messageId: string,
        run: RunProgress,
        targetInstances?: string[],
        logger?: PassableLogger,
    ): void {
        const instances = targetInstances || Object.keys(run.subcontexts);

        for (const instanceId of instances) {
            const context = run.subcontexts[instanceId];
            if (context) {
                SubroutineContextManager.addRuntimeEvent(context, "message", messageId);
                logger?.info(`Delivered message ${messageId} to subroutine instance ${instanceId}`);
            } else {
                logger?.error(`Subroutine instance ${instanceId} not found when delivering message ${messageId}`);
            }
        }
    }

    /**
     * Delivers a signal to all relevant subroutine instances that might be waiting for it.
     * Signals are typically broadcast to all running instances, unlike messages which
     * are usually targeted to specific instances.
     * 
     * @param signalId The ID of the signal to deliver (should match signalRef.id in BPMN)
     * @param run The run progress containing all subroutine contexts
     * @param logger Logger for error reporting
     */
    static deliverSignal(
        signalId: string,
        run: RunProgress,
        logger?: PassableLogger,
    ): void {
        for (const [instanceId, context] of Object.entries(run.subcontexts)) {
            SubroutineContextManager.addRuntimeEvent(context, "signal", signalId);
            logger?.info(`Delivered signal ${signalId} to subroutine instance ${instanceId}`);
        }
    }

    /**
     * Delivers an error event to a specific subroutine instance.
     * This is typically called when a BPMN error is thrown from within a subroutine
     * and needs to be caught by error boundary events.
     * 
     * @param errorCode The BPMN error code to deliver
     * @param instanceId The specific subroutine instance ID to deliver the error to
     * @param run The run progress containing all subroutine contexts
     * @param logger Logger for error reporting
     */
    static deliverError(
        errorCode: string,
        instanceId: string,
        run: RunProgress,
        logger?: PassableLogger,
    ): void {
        const context = run.subcontexts[instanceId];
        if (context) {
            SubroutineContextManager.addRuntimeEvent(context, "error", errorCode);
            logger?.info(`Delivered error ${errorCode} to subroutine instance ${instanceId}`);
        } else {
            logger?.error(`Subroutine instance ${instanceId} not found when delivering error ${errorCode}`);
        }
    }

    /**
     * Delivers an escalation event to a specific subroutine instance.
     * This is typically called when a BPMN escalation is raised from within a subroutine
     * and needs to be caught by escalation boundary events.
     * 
     * @param escalationCode The BPMN escalation code to deliver
     * @param instanceId The specific subroutine instance ID to deliver the escalation to
     * @param run The run progress containing all subroutine contexts
     * @param logger Logger for error reporting
     */
    static deliverEscalation(
        escalationCode: string,
        instanceId: string,
        run: RunProgress,
        logger?: PassableLogger,
    ): void {
        const context = run.subcontexts[instanceId];
        if (context) {
            SubroutineContextManager.addRuntimeEvent(context, "escalation", escalationCode);
            logger?.info(`Delivered escalation ${escalationCode} to subroutine instance ${instanceId}`);
        } else {
            logger?.error(`Subroutine instance ${instanceId} not found when delivering escalation ${escalationCode}`);
        }
    }

    /**
     * Batch delivery method for multiple events at once.
     * Useful for external systems that might queue up multiple events.
     * 
     * @param events Object containing arrays of events to deliver
     * @param run The run progress containing all subroutine contexts
     * @param targetInstances Optional list of specific subroutine instance IDs to deliver to
     * @param logger Logger for error reporting
     */
    static deliverMultipleEvents(
        events: {
            messages?: { id: string; targetInstances?: string[] }[];
            signals?: string[];
            errors?: { code: string; instanceId: string }[];
            escalations?: { code: string; instanceId: string }[];
        },
        run: RunProgress,
        logger?: PassableLogger,
    ): void {
        // Deliver messages
        if (events.messages) {
            for (const message of events.messages) {
                this.deliverMessage(message.id, run, message.targetInstances, logger);
            }
        }

        // Deliver signals
        if (events.signals) {
            for (const signalId of events.signals) {
                this.deliverSignal(signalId, run, logger);
            }
        }

        // Deliver errors
        if (events.errors) {
            for (const error of events.errors) {
                this.deliverError(error.code, error.instanceId, run, logger);
            }
        }

        // Deliver escalations
        if (events.escalations) {
            for (const escalation of events.escalations) {
                this.deliverEscalation(escalation.code, escalation.instanceId, run, logger);
            }
        }
    }

    /**
     * Webhook handler for external BPMN message delivery.
     * This method can be used as a foundation for webhook endpoints that receive
     * external messages to be delivered to running workflows.
     * 
     * Example usage:
     * ```typescript
     * app.post('/webhook/bpmn-message', (req, res) => {
     *     const { messageId, runId, instanceIds } = req.body;
     *     const run = await loadRunProgress(runId);
     *     BpmnEventBus.handleMessageWebhook(messageId, run, instanceIds);
     *     res.json({ success: true });
     * });
     * ```
     * 
     * @param messageId The message ID to deliver
     * @param run The run progress to deliver the message to
     * @param targetInstances Optional specific instance IDs to target
     * @param logger Logger for error reporting
     */
    static handleMessageWebhook(
        messageId: string,
        run: RunProgress,
        targetInstances?: string[],
        logger?: PassableLogger,
    ): void {
        logger?.info(`Webhook received BPMN message: ${messageId} for run ${run.runId}`);
        this.deliverMessage(messageId, run, targetInstances, logger);
    }

    /**
     * Webhook handler for external BPMN signal delivery.
     * Similar to handleMessageWebhook but for signals.
     * 
     * @param signalId The signal ID to deliver
     * @param run The run progress to deliver the signal to
     * @param logger Logger for error reporting
     */
    static handleSignalWebhook(
        signalId: string,
        run: RunProgress,
        logger?: PassableLogger,
    ): void {
        logger?.info(`Webhook received BPMN signal: ${signalId} for run ${run.runId}`);
        this.deliverSignal(signalId, run, logger);
    }
} 
