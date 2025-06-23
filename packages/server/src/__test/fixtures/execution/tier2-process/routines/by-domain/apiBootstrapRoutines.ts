/**
 * API Bootstrap Routine Fixtures
 * 
 * Demonstrates how agents can discover and create new API integrations
 * through intelligent routine generation and composition.
 */

import { ResourceSubType } from "@vrooli/shared";
import type { RoutineFixture } from "./types.js";

/**
 * API integration routines created by agents
 */
export const API_BOOTSTRAP_ROUTINES: Record<string, RoutineFixture> = {
    /**
     * Stripe API Integration Bootstrap
     * Agent-generated routine for Stripe payment processing
     */
    STRIPE_INTEGRATION_BOOTSTRAP: {
        id: "stripe_api_bootstrap_v1",
        name: "Stripe API Integration Bootstrap",
        description: "Agent-generated routine for complete Stripe payment processing integration",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        createdBy: "api_integration_creator",
        generatedFrom: {
            apiSpec: "https://stripe.com/docs/api",
            analysisDate: new Date("2024-09-01"),
            confidence: 0.92,
            generationAgent: "api_integration_creator",
        },

        config: {
            __version: "1.0",
            strategy: "deterministic",

            // Generated sub-routines
            subRoutines: {
                // Authentication routine
                authenticate: {
                    id: "stripe_auth",
                    name: "Stripe Authentication",
                    strategy: "deterministic",
                    config: {
                        method: "bearer_token",
                        headerName: "Authorization",
                        tokenPrefix: "Bearer sk_",
                        securityLevel: "secret",
                        validation: {
                            tokenFormat: /^sk_(test|live)_[a-zA-Z0-9]+$/,
                            errorHandling: "immediate",
                        },
                    },
                },

                // Payment creation routine
                createPayment: {
                    id: "stripe_create_payment",
                    name: "Create Stripe Payment Intent",
                    strategy: "deterministic",
                    inputs: [
                        { name: "amount", type: "number", required: true, validation: { min: 50 } },
                        { name: "currency", type: "string", default: "usd", validation: { enum: ["usd", "eur", "gbp"] } },
                        { name: "description", type: "string", required: false, maxLength: 500 },
                        { name: "metadata", type: "object", required: false },
                    ],
                    apiCall: {
                        endpoint: "/v1/payment_intents",
                        method: "POST",
                        bodyMapping: {
                            amount: "{{input.amount}}",
                            currency: "{{input.currency}}",
                            description: "{{input.description}}",
                            metadata: "{{input.metadata}}",
                        },
                        headers: {
                            "Content-Type": "application/x-www-form-urlencoded",
                            "Idempotency-Key": "{{generateIdempotencyKey()}}",
                        },
                    },
                    errorHandling: {
                        "rate_limit_error": {
                            strategy: "exponential_backoff",
                            initialDelay: 1000,
                            maxRetries: 5,
                        },
                        "invalid_request_error": {
                            strategy: "validation_error",
                            logLevel: "error",
                        },
                        "authentication_error": {
                            strategy: "reauth_and_retry",
                            maxRetries: 1,
                        },
                        "api_error": {
                            strategy: "circuit_breaker",
                            threshold: 5,
                            timeout: 60000,
                        },
                    },
                },

                // Webhook handler routine
                handleWebhook: {
                    id: "stripe_webhook_handler",
                    name: "Handle Stripe Webhooks",
                    strategy: "reasoning",
                    inputs: [
                        { name: "payload", type: "string", required: true },
                        { name: "signature", type: "string", required: true },
                    ],
                    webhookConfig: {
                        verificationMethod: "stripe_signature",
                        secretKey: "{{env.STRIPE_WEBHOOK_SECRET}}",
                        eventTypes: [
                            "payment_intent.succeeded",
                            "payment_intent.failed",
                            "payment_intent.canceled",
                            "charge.dispute.created",
                        ],
                        processingStrategy: "async_queue",
                    },
                },

                // Refund processing routine
                processRefund: {
                    id: "stripe_process_refund",
                    name: "Process Stripe Refund",
                    strategy: "deterministic",
                    inputs: [
                        { name: "paymentIntentId", type: "string", required: true },
                        { name: "amount", type: "number", required: false },
                        { name: "reason", type: "string", required: false },
                    ],
                    apiCall: {
                        endpoint: "/v1/refunds",
                        method: "POST",
                        bodyMapping: {
                            payment_intent: "{{input.paymentIntentId}}",
                            amount: "{{input.amount}}",
                            reason: "{{input.reason}}",
                        },
                    },
                },
            },

            // Auto-generated test suite
            testSuite: {
                unitTests: [
                    {
                        name: "test_authentication",
                        type: "auth_validation",
                        testData: { apiKey: "sk_test_example" },
                        expected: { status: 200, authenticated: true },
                    },
                    {
                        name: "test_payment_creation",
                        type: "api_call",
                        mockData: {
                            amount: 1000,
                            currency: "usd",
                            description: "Test payment",
                        },
                        expected: {
                            status: "requires_payment_method",
                            amount: 1000,
                        },
                    },
                    {
                        name: "test_webhook_verification",
                        type: "webhook_test",
                        mockPayload: { type: "payment_intent.succeeded" },
                        expected: { verified: true, processed: true },
                    },
                ],
                integrationTests: [
                    {
                        name: "end_to_end_payment_flow",
                        steps: [
                            "authenticate",
                            "create_payment",
                            "simulate_payment_completion",
                            "verify_webhook",
                        ],
                        timeout: 30000,
                        cleanupRequired: true,
                    },
                ],
                performanceTests: [
                    {
                        name: "payment_creation_performance",
                        operation: "createPayment",
                        iterations: 100,
                        expectedLatency: { p50: 200, p95: 500, p99: 1000 },
                    },
                ],
            },

            // Learning patterns for improvement
            learningPatterns: {
                errorPatterns: {
                    trackingEnabled: true,
                    adaptiveRetry: true,
                    patternDetection: ["rate_limits", "service_outages", "validation_errors"],
                },
                performanceOptimization: {
                    caching: ["authentication_tokens", "idempotency_keys"],
                    batching: ["bulk_refunds", "bulk_captures"],
                    connectionPooling: true,
                },
                usageAnalytics: {
                    trackSuccessRate: true,
                    trackLatency: true,
                    trackErrorTypes: true,
                    reportingInterval: "hourly",
                },
            },
        },

        // Evolution potential
        evolutionPotential: {
            suggestedEnhancements: [
                "Add subscription management routines",
                "Implement customer portal integration",
                "Add fraud detection preprocessing",
                "Implement multi-currency optimization",
            ],
            compatibleIntegrations: ["stripe_billing", "stripe_connect", "stripe_radar"],
            performanceBaseline: {
                averageLatency: 250,
                successRate: 0.98,
                errorRate: 0.02,
                throughput: 1000, // requests per minute
            },
        },
    },

    /**
     * Twilio SMS Integration Bootstrap
     * Agent-generated routine for SMS communication
     */
    TWILIO_SMS_BOOTSTRAP: {
        id: "twilio_sms_bootstrap_v1",
        name: "Twilio SMS Integration Bootstrap",
        description: "Agent-generated routine for complete Twilio SMS integration",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        createdBy: "api_integration_creator",
        generatedFrom: {
            apiSpec: "https://www.twilio.com/docs/sms/api",
            analysisDate: new Date("2024-09-05"),
            confidence: 0.89,
            generationAgent: "api_integration_creator",
        },

        config: {
            __version: "1.0",
            strategy: "deterministic",

            subRoutines: {
                // Authentication
                authenticate: {
                    id: "twilio_auth",
                    name: "Twilio Authentication",
                    strategy: "deterministic",
                    config: {
                        method: "basic_auth",
                        accountSid: "{{env.TWILIO_ACCOUNT_SID}}",
                        authToken: "{{env.TWILIO_AUTH_TOKEN}}",
                        validation: {
                            sidFormat: /^AC[a-zA-Z0-9]{32}$/,
                        },
                    },
                },

                // Send SMS
                sendSMS: {
                    id: "twilio_send_sms",
                    name: "Send SMS via Twilio",
                    strategy: "deterministic",
                    inputs: [
                        { name: "to", type: "string", required: true, validation: { pattern: /^\+[1-9]\d{1,14}$/ } },
                        { name: "body", type: "string", required: true, maxLength: 1600 },
                        { name: "from", type: "string", default: "{{env.TWILIO_PHONE_NUMBER}}" },
                        { name: "mediaUrl", type: "array", required: false },
                    ],
                    apiCall: {
                        endpoint: "/2010-04-01/Accounts/{{accountSid}}/Messages.json",
                        method: "POST",
                        bodyMapping: {
                            To: "{{input.to}}",
                            From: "{{input.from}}",
                            Body: "{{input.body}}",
                            MediaUrl: "{{input.mediaUrl}}",
                        },
                    },
                    rateLimiting: {
                        messagesPerSecond: 1,
                        burstCapacity: 10,
                        queueStrategy: "priority",
                    },
                    compliance: {
                        optOutDetection: true,
                        contentFiltering: ["spam", "phishing"],
                        regionCompliance: ["gdpr", "tcpa"],
                    },
                },

                // Status callback handler
                handleStatusCallback: {
                    id: "twilio_status_callback",
                    name: "Handle SMS Status Updates",
                    strategy: "reasoning",
                    webhookConfig: {
                        statusTypes: ["sent", "delivered", "failed", "undelivered"],
                        retryStrategy: "exponential_backoff",
                        persistenceRequired: true,
                    },
                },

                // Batch SMS sending
                sendBatchSMS: {
                    id: "twilio_batch_sms",
                    name: "Send Batch SMS",
                    strategy: "routing",
                    config: {
                        parallelization: {
                            maxConcurrent: 10,
                            throttleMs: 100,
                        },
                        errorHandling: {
                            continueOnError: true,
                            collectErrors: true,
                        },
                        reporting: {
                            trackDelivery: true,
                            generateReport: true,
                        },
                    },
                },
            },

            // Compliance and monitoring
            complianceFeatures: {
                optOut: {
                    keywords: ["STOP", "UNSUBSCRIBE", "CANCEL"],
                    autoProcess: true,
                    maintainList: true,
                },
                rateLimit: {
                    perPhoneNumber: { daily: 200, hourly: 20 },
                    perAccount: { daily: 10000, hourly: 1000 },
                },
                contentModeration: {
                    enabled: true,
                    filters: ["profanity", "spam", "phishing"],
                    action: "flag_for_review",
                },
            },
        },
    },

    /**
     * SendGrid Email Integration Bootstrap
     * Agent-generated routine for email automation
     */
    SENDGRID_EMAIL_BOOTSTRAP: {
        id: "sendgrid_email_bootstrap_v1",
        name: "SendGrid Email Integration Bootstrap",
        description: "Agent-generated routine for comprehensive email automation via SendGrid",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        createdBy: "api_integration_creator",
        generatedFrom: {
            apiSpec: "https://docs.sendgrid.com/api-reference",
            analysisDate: new Date("2024-09-10"),
            confidence: 0.94,
            generationAgent: "api_integration_creator",
        },

        config: {
            __version: "1.0",
            strategy: "deterministic",

            subRoutines: {
                // Send single email
                sendEmail: {
                    id: "sendgrid_send_email",
                    name: "Send Email via SendGrid",
                    strategy: "deterministic",
                    inputs: [
                        { name: "to", type: "array", required: true },
                        { name: "subject", type: "string", required: true },
                        { name: "content", type: "object", required: true },
                        { name: "from", type: "object", required: true },
                        { name: "templateId", type: "string", required: false },
                        { name: "dynamicTemplateData", type: "object", required: false },
                    ],
                    apiCall: {
                        endpoint: "/v3/mail/send",
                        method: "POST",
                        headers: {
                            "Authorization": "Bearer {{env.SENDGRID_API_KEY}}",
                            "Content-Type": "application/json",
                        },
                        bodyMapping: {
                            personalizations: [{
                                to: "{{input.to}}",
                                dynamic_template_data: "{{input.dynamicTemplateData}}",
                            }],
                            from: "{{input.from}}",
                            subject: "{{input.subject}}",
                            content: "{{input.content}}",
                            template_id: "{{input.templateId}}",
                        },
                    },
                    deliverability: {
                        trackOpens: true,
                        trackClicks: true,
                        subscriptionTracking: true,
                        spamCheck: true,
                    },
                },

                // Manage templates
                manageTemplates: {
                    id: "sendgrid_template_manager",
                    name: "Manage Email Templates",
                    strategy: "reasoning",
                    operations: {
                        create: {
                            endpoint: "/v3/templates",
                            method: "POST",
                        },
                        update: {
                            endpoint: "/v3/templates/{{templateId}}",
                            method: "PATCH",
                        },
                        listVersions: {
                            endpoint: "/v3/templates/{{templateId}}/versions",
                            method: "GET",
                        },
                    },
                    versionControl: {
                        enabled: true,
                        autoBackup: true,
                        maxVersions: 10,
                    },
                },

                // Analytics and reporting
                getAnalytics: {
                    id: "sendgrid_analytics",
                    name: "Retrieve Email Analytics",
                    strategy: "deterministic",
                    endpoints: {
                        stats: "/v3/stats",
                        suppressions: "/v3/suppression",
                        bounces: "/v3/suppression/bounces",
                        spamReports: "/v3/suppression/spam_reports",
                    },
                    aggregation: {
                        timeWindows: ["day", "week", "month"],
                        metrics: ["delivered", "opens", "clicks", "bounces", "spam_reports"],
                    },
                },

                // List management
                manageContactLists: {
                    id: "sendgrid_list_manager",
                    name: "Manage Contact Lists",
                    strategy: "reasoning",
                    features: {
                        segmentation: {
                            criteria: ["engagement", "location", "preferences"],
                            autoUpdate: true,
                        },
                        importExport: {
                            formats: ["csv", "json"],
                            validation: true,
                            deduplication: true,
                        },
                        compliance: {
                            gdprFields: ["consent_date", "consent_source"],
                            autoUnsubscribe: true,
                        },
                    },
                },
            },

            // Advanced features
            advancedFeatures: {
                ipWarmup: {
                    enabled: true,
                    schedule: "gradual",
                    monitoring: true,
                },
                domainAuthentication: {
                    spf: true,
                    dkim: true,
                    dmarc: true,
                },
                eventWebhook: {
                    endpoint: "{{env.SENDGRID_WEBHOOK_URL}}",
                    events: ["delivered", "opened", "clicked", "bounced", "unsubscribed"],
                    security: "signature_verification",
                },
            },
        },
    },

    /**
     * GitHub API Integration Bootstrap
     * Agent-generated routine for repository automation
     */
    GITHUB_API_BOOTSTRAP: {
        id: "github_api_bootstrap_v1",
        name: "GitHub API Integration Bootstrap",
        description: "Agent-generated routine for GitHub repository automation and CI/CD",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        createdBy: "api_integration_creator",
        generatedFrom: {
            apiSpec: "https://docs.github.com/en/rest",
            analysisDate: new Date("2024-09-15"),
            confidence: 0.91,
            generationAgent: "api_integration_creator",
        },

        config: {
            __version: "1.0",
            strategy: "reasoning",

            subRoutines: {
                // Repository management
                manageRepository: {
                    id: "github_repo_manager",
                    name: "Manage GitHub Repository",
                    strategy: "deterministic",
                    operations: {
                        create: {
                            endpoint: "/repos",
                            method: "POST",
                            requiredScopes: ["repo"],
                        },
                        update: {
                            endpoint: "/repos/{{owner}}/{{repo}}",
                            method: "PATCH",
                            requiredScopes: ["repo"],
                        },
                        createBranch: {
                            endpoint: "/repos/{{owner}}/{{repo}}/git/refs",
                            method: "POST",
                            requiredScopes: ["repo"],
                        },
                        protectBranch: {
                            endpoint: "/repos/{{owner}}/{{repo}}/branches/{{branch}}/protection",
                            method: "PUT",
                            requiredScopes: ["repo", "admin:repo"],
                        },
                    },
                },

                // Pull request automation
                automatedPR: {
                    id: "github_pr_automation",
                    name: "Automated Pull Request Management",
                    strategy: "reasoning",
                    workflow: {
                        create: {
                            validation: ["branch_exists", "no_conflicts", "tests_pass"],
                            autoAssign: true,
                            labels: ["auto-generated", "needs-review"],
                        },
                        review: {
                            checksRequired: ["ci_pass", "code_coverage", "security_scan"],
                            autoMerge: {
                                enabled: true,
                                strategy: "squash",
                                deleteAfterMerge: true,
                            },
                        },
                    },
                },

                // CI/CD integration
                cicdIntegration: {
                    id: "github_cicd",
                    name: "CI/CD Workflow Management",
                    strategy: "routing",
                    workflows: {
                        triggerWorkflow: {
                            endpoint: "/repos/{{owner}}/{{repo}}/actions/workflows/{{workflow_id}}/dispatches",
                            method: "POST",
                            inputs: {
                                ref: "{{branch}}",
                                inputs: "{{workflow_inputs}}",
                            },
                        },
                        monitorRuns: {
                            endpoint: "/repos/{{owner}}/{{repo}}/actions/runs",
                            method: "GET",
                            polling: {
                                interval: 30000,
                                maxAttempts: 60,
                            },
                        },
                        downloadArtifacts: {
                            endpoint: "/repos/{{owner}}/{{repo}}/actions/artifacts",
                            method: "GET",
                            storage: "s3",
                        },
                    },
                },

                // Issue and project management
                projectAutomation: {
                    id: "github_project_automation",
                    name: "Project Board Automation",
                    strategy: "reasoning",
                    features: {
                        issueTriaging: {
                            autoLabel: true,
                            prioritization: ["critical", "high", "medium", "low"],
                            assignment: "round_robin",
                        },
                        boardAutomation: {
                            moveOnStatus: true,
                            createFromTemplate: true,
                            syncWithMilestones: true,
                        },
                        reporting: {
                            velocity: true,
                            burndown: true,
                            cycleTime: true,
                        },
                    },
                },
            },

            // Security and compliance
            securityFeatures: {
                secretScanning: {
                    enabled: true,
                    customPatterns: ["api_key", "private_key", "password"],
                },
                dependencyScanning: {
                    enabled: true,
                    autoUpdate: true,
                    securityAlerts: true,
                },
                codeScanning: {
                    enabled: true,
                    tools: ["codeql", "sonarqube"],
                    blockOnHighSeverity: true,
                },
            },
        },
    },

    /**
     * Slack API Integration Bootstrap
     * Agent-generated routine for team communication automation
     */
    SLACK_API_BOOTSTRAP: {
        id: "slack_api_bootstrap_v1",
        name: "Slack API Integration Bootstrap",
        description: "Agent-generated routine for Slack workspace automation and bot creation",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        createdBy: "api_integration_creator",
        generatedFrom: {
            apiSpec: "https://api.slack.com/docs",
            analysisDate: new Date("2024-09-20"),
            confidence: 0.88,
            generationAgent: "api_integration_creator",
        },

        config: {
            __version: "1.0",
            strategy: "conversational",

            subRoutines: {
                // Message handling
                messageHandler: {
                    id: "slack_message_handler",
                    name: "Slack Message Handler",
                    strategy: "reasoning",
                    features: {
                        sendMessage: {
                            endpoint: "/chat.postMessage",
                            formatting: ["markdown", "blocks", "attachments"],
                            threading: true,
                        },
                        updateMessage: {
                            endpoint: "/chat.update",
                            preserveThreads: true,
                        },
                        ephemeralMessage: {
                            endpoint: "/chat.postEphemeral",
                            timeout: 300000,
                        },
                    },
                    interactivity: {
                        buttons: true,
                        selectMenus: true,
                        datePickers: true,
                        modals: true,
                    },
                },

                // Bot personality
                botPersonality: {
                    id: "slack_bot_personality",
                    name: "Conversational Bot Behavior",
                    strategy: "conversational",
                    personality: {
                        tone: "friendly_professional",
                        contextAwareness: true,
                        humor: "occasional",
                        helpfulness: "proactive",
                    },
                    capabilities: {
                        naturalLanguage: true,
                        commandParsing: true,
                        contextMemory: true,
                        learningEnabled: true,
                    },
                },

                // Workflow automation
                workflowBuilder: {
                    id: "slack_workflow_builder",
                    name: "Slack Workflow Automation",
                    strategy: "routing",
                    workflows: {
                        onboarding: {
                            triggers: ["user_joined"],
                            steps: ["welcome_message", "resource_share", "buddy_assignment"],
                        },
                        approval: {
                            triggers: ["approval_request"],
                            steps: ["notify_approvers", "collect_responses", "execute_action"],
                        },
                        incident: {
                            triggers: ["incident_declared"],
                            steps: ["create_channel", "notify_team", "start_timeline"],
                        },
                    },
                },

                // Analytics and insights
                workspaceAnalytics: {
                    id: "slack_analytics",
                    name: "Workspace Analytics",
                    strategy: "deterministic",
                    metrics: {
                        engagement: ["messages_sent", "reactions", "threads_created"],
                        productivity: ["response_time", "resolution_time", "automation_usage"],
                        health: ["active_users", "channel_activity", "integration_usage"],
                    },
                    reporting: {
                        schedule: "weekly",
                        format: "dashboard",
                        distribution: ["#analytics", "leadership"],
                    },
                },
            },
        },
    },
};

/**
 * Helper function to get all API bootstrap routines
 */
export function getAllAPIBootstrapRoutines(): RoutineFixture[] {
    return Object.values(API_BOOTSTRAP_ROUTINES);
}

/**
 * Get bootstrap routine by API provider
 */
export function getBootstrapRoutineByProvider(provider: string): RoutineFixture | undefined {
    return Object.values(API_BOOTSTRAP_ROUTINES).find(
        routine => routine.name.toLowerCase().includes(provider.toLowerCase()),
    );
}

/**
 * Get routines by generation confidence
 */
export function getRoutinesByConfidence(minConfidence: number): RoutineFixture[] {
    return Object.values(API_BOOTSTRAP_ROUTINES).filter(
        routine => (routine.generatedFrom?.confidence || 0) >= minConfidence,
    );
}

/**
 * Example of how agents evolve API integrations
 */
export const API_EVOLUTION_EXAMPLES = {
    STRIPE_TO_BILLING_SUITE: {
        from: "stripe_api_bootstrap_v1",
        to: "billing_automation_suite_v1",
        evolvingAgent: "domain_specialist_agent",
        additions: [
            "subscription_management",
            "usage_based_billing",
            "invoice_automation",
            "revenue_recognition",
            "dunning_management",
        ],
        reasoning: "Identified patterns of billing-related routine requests, created comprehensive suite",
        performanceImprovement: {
            automationLevel: 0.85, // 85% of billing tasks automated
            errorReduction: 0.72, // 72% fewer billing errors
            timeToInvoice: -0.90, // 90% faster invoicing
        },
    },
};
