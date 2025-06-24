import { BaseToolFixture, ToolTestCase } from "./base.js";
import { MCPTool } from "../../../../services/mcp/tools.js";
import { EmergentAgent } from "../../../../services/execution/cross-cutting/agents/emergentAgent.js";
import { EventPublisher } from "../../../../services/execution/shared/EventPublisher.js";

export class SecurityToolFixture extends BaseToolFixture {
    private threatLog: ThreatEvent[] = [];
    private vulnerabilities: Vulnerability[] = [];
    private securityPolicies: SecurityPolicy[] = [];
    private adaptations: SecurityAdaptation[] = [];

    constructor(tool: MCPTool, eventPublisher: EventPublisher) {
        super(tool, eventPublisher);
        this.setupSecurityMonitoring();
    }

    /**
     * Test security tool's threat detection capabilities
     */
    async testThreatDetection(
        benignTraffic: SecurityEvent[],
        maliciousTraffic: SecurityEvent[],
    ): Promise<ThreatDetectionResult> {
        const results = {
            truePositives: 0,
            falsePositives: 0,
            trueNegatives: 0,
            falseNegatives: 0,
        };

        // Test with benign traffic
        for (const event of benignTraffic) {
            const result = await this.tool.execute({
                action: "analyze_traffic",
                event,
            });
            
            if (result.threatDetected) {
                results.falsePositives++;
            } else {
                results.trueNegatives++;
            }
        }

        // Test with malicious traffic
        for (const event of maliciousTraffic) {
            const result = await this.tool.execute({
                action: "analyze_traffic",
                event,
            });
            
            if (result.threatDetected) {
                results.truePositives++;
                this.threatLog.push({
                    ...result.threat,
                    timestamp: new Date(),
                });
            } else {
                results.falseNegatives++;
            }
        }

        const precision = results.truePositives / (results.truePositives + results.falsePositives || 1);
        const recall = results.truePositives / (results.truePositives + results.falseNegatives || 1);

        return {
            ...results,
            precision,
            recall,
            f1Score: 2 * (precision * recall) / (precision + recall || 1),
            detectedThreats: this.threatLog,
        };
    }

    /**
     * Test vulnerability scanning capabilities
     */
    async testVulnerabilityScanning(
        target: ScanTarget,
        knownVulnerabilities: string[],
    ): Promise<VulnerabilityScanResult> {
        const scanResult = await this.tool.execute({
            action: "scan_vulnerabilities",
            target,
            depth: "comprehensive",
        });

        const detected = scanResult.vulnerabilities || [];
        this.vulnerabilities.push(...detected);

        // Calculate detection metrics
        const detectedIds = detected.map(v => v.id);
        const foundKnown = knownVulnerabilities.filter(id => detectedIds.includes(id));
        const coverage = foundKnown.length / knownVulnerabilities.length;

        return {
            target: target.name,
            totalFound: detected.length,
            criticalCount: detected.filter(v => v.severity === "critical").length,
            highCount: detected.filter(v => v.severity === "high").length,
            mediumCount: detected.filter(v => v.severity === "medium").length,
            lowCount: detected.filter(v => v.severity === "low").length,
            knownVulnerabilityCoverage: coverage,
            scanDuration: scanResult.duration,
            recommendations: this.generateRecommendations(detected),
        };
    }

    /**
     * Test adaptive security response
     */
    async testAdaptiveSecurity(
        attackScenarios: AttackScenario[],
    ): Promise<AdaptiveSecurityResult> {
        const adaptationTimeline: AdaptationEvent[] = [];
        let successfulAdaptations = 0;

        for (const scenario of attackScenarios) {
            const startTime = Date.now();
            
            // Initial attack
            const initialResult = await this.simulateAttack(scenario);
            
            // Wait for adaptation
            await this.waitForAdaptation(scenario.attackType);
            
            // Repeat attack to test adaptation
            const adaptedResult = await this.simulateAttack(scenario);
            
            const adaptationTime = Date.now() - startTime;
            const adapted = adaptedResult.blocked && !initialResult.blocked;
            
            if (adapted) {
                successfulAdaptations++;
            }

            adaptationTimeline.push({
                scenario: scenario.name,
                timestamp: new Date(),
                initialSuccess: !initialResult.blocked,
                adaptedSuccess: adaptedResult.blocked,
                adaptationTime,
                adaptationType: this.getLatestAdaptation()?.type || "unknown",
            });
        }

        return {
            totalScenarios: attackScenarios.length,
            successfulAdaptations,
            adaptationRate: successfulAdaptations / attackScenarios.length,
            averageAdaptationTime: adaptationTimeline.reduce((sum, e) => sum + e.adaptationTime, 0) / adaptationTimeline.length,
            timeline: adaptationTimeline,
            finalSecurityPosture: await this.assessSecurityPosture(),
        };
    }

    /**
     * Test zero-day threat emergence
     */
    async testZeroDayResponse(
        normalBehavior: BehaviorProfile,
        zeroDayAttack: ZeroDayScenario,
    ): Promise<ZeroDayResponseResult> {
        // Establish baseline
        await this.establishBehaviorBaseline(normalBehavior);

        // Execute zero-day attack
        const detectionTime = await this.measureDetectionTime(zeroDayAttack);
        
        // Analyze response
        const response = await this.analyzeZeroDayResponse(zeroDayAttack);

        return {
            attack: zeroDayAttack.name,
            detected: response.detected,
            detectionTime,
            containmentTime: response.containmentTime,
            damageAssessment: response.damage,
            mitigationStrategy: response.mitigation,
            learningOutcome: response.learning,
        };
    }

    /**
     * Test security policy evolution
     */
    async testPolicyEvolution(
        initialPolicies: SecurityPolicy[],
        threats: ThreatEvent[],
        duration: number,
    ): Promise<PolicyEvolutionResult> {
        // Set initial policies
        await this.setPolicies(initialPolicies);
        const snapshots: PolicySnapshot[] = [this.takePolicySnapshot("initial")];

        const startTime = Date.now();
        let threatIndex = 0;

        while (Date.now() - startTime < duration) {
            // Apply threats gradually
            if (threatIndex < threats.length) {
                await this.processThreat(threats[threatIndex]);
                threatIndex++;
            }

            // Check for policy evolution
            const currentPolicies = await this.getCurrentPolicies();
            if (this.policiesChanged(this.securityPolicies, currentPolicies)) {
                this.securityPolicies = currentPolicies;
                snapshots.push(this.takePolicySnapshot(`evolution-${snapshots.length}`));
            }

            await new Promise(resolve => setTimeout(resolve, 1000));
        }

        snapshots.push(this.takePolicySnapshot("final"));

        return {
            initialPolicyCount: initialPolicies.length,
            finalPolicyCount: this.securityPolicies.length,
            evolutionSteps: snapshots.length - 2,
            snapshots,
            improvements: this.analyzePolicyImprovements(snapshots),
            threatsCovered: this.calculateThreatCoverage(this.securityPolicies, threats),
        };
    }

    protected async executeWithRetry(input: any, maxRetries: number): Promise<any> {
        let lastError: any;
        
        for (let i = 0; i < maxRetries; i++) {
            try {
                return await this.tool.execute(input);
            } catch (error) {
                lastError = error;
                // Security tools might need longer retry delays
                if (i < maxRetries - 1) {
                    await new Promise(resolve => setTimeout(resolve, 2000 * Math.pow(2, i)));
                }
            }
        }
        
        throw lastError;
    }

    protected captureResourceUsage(): { cpu: number; memory: number } {
        // Security tools typically use more resources
        return {
            cpu: Math.random() * 70 + 20, // 20-90% CPU
            memory: Math.random() * 500 + 100, // 100-600 MB
        };
    }

    protected async applyConstraints(constraints: any[]): Promise<void> {
        for (const constraint of constraints) {
            if (constraint.type === "permission") {
                await this.tool.execute({
                    action: "set_permissions",
                    permissions: constraint.value,
                });
            }
        }
    }

    protected async runAgentScenario(
        agent: EmergentAgent,
        scenario: any,
    ): Promise<void> {
        // Security scenario with agent
        for (const event of scenario.securityEvents || []) {
            await agent.executeTool(this.tool.id, {
                action: "analyze_event",
                event,
            });
            
            // Check for emergent security behaviors
            const behaviors = agent.getEmergentBehaviors();
            if (behaviors.includes("proactive_threat_hunting")) {
                await agent.executeTool(this.tool.id, {
                    action: "hunt_threats",
                    scope: "expanded",
                });
            }
        }
    }

    protected async applyStimulus(stimulus: any): Promise<void> {
        switch (stimulus.type) {
            case "new_threat":
                await this.introduceNewThreat(stimulus.data);
                break;
            case "attack_pattern":
                await this.simulateAttackPattern(stimulus.data);
                break;
            case "policy_violation":
                await this.triggerPolicyViolation(stimulus.data);
                break;
        }
    }

    protected async takeSnapshot(name: string): Promise<any> {
        const stats = await this.tool.execute({
            action: "get_security_stats",
        });

        return {
            name,
            timestamp: new Date(),
            performance: {
                latency: stats.avgDetectionTime || 50,
                accuracy: stats.detectionAccuracy || 0.95,
                throughput: stats.eventsPerSecond || 10000,
                errorRate: stats.falsePositiveRate || 0.02,
            },
            capabilities: stats.capabilities || ["basic_detection"],
            threatsCovered: stats.threatTypes || [],
        };
    }

    private setupSecurityMonitoring() {
        this.eventPublisher.on("threat:detected", (threat: ThreatEvent) => {
            this.threatLog.push(threat);
        });

        this.eventPublisher.on("security:adapted", (adaptation: SecurityAdaptation) => {
            this.adaptations.push(adaptation);
        });
    }

    private generateRecommendations(vulnerabilities: Vulnerability[]): string[] {
        const recommendations: string[] = [];
        
        const critical = vulnerabilities.filter(v => v.severity === "critical");
        if (critical.length > 0) {
            recommendations.push(`Immediately patch ${critical.length} critical vulnerabilities`);
        }
        
        const highRisk = vulnerabilities.filter(v => v.riskScore > 8);
        if (highRisk.length > 0) {
            recommendations.push(`Address ${highRisk.length} high-risk vulnerabilities within 24 hours`);
        }
        
        return recommendations;
    }

    private async simulateAttack(scenario: AttackScenario): Promise<AttackResult> {
        const result = await this.tool.execute({
            action: "simulate_attack",
            attackType: scenario.attackType,
            payload: scenario.payload,
        });

        return {
            blocked: result.blocked || false,
            damage: result.damage || 0,
            detectionTime: result.detectionTime || 0,
        };
    }

    private async waitForAdaptation(attackType: string): Promise<void> {
        const maxWait = 5000;
        const startTime = Date.now();
        
        while (Date.now() - startTime < maxWait) {
            const adaptation = this.adaptations.find(a => a.threatType === attackType);
            if (adaptation) return;
            
            await new Promise(resolve => setTimeout(resolve, 100));
        }
    }

    private getLatestAdaptation(): SecurityAdaptation | undefined {
        return this.adaptations[this.adaptations.length - 1];
    }

    private async assessSecurityPosture(): Promise<SecurityPosture> {
        const assessment = await this.tool.execute({
            action: "assess_posture",
        });

        return {
            score: assessment.score || 0,
            strengths: assessment.strengths || [],
            weaknesses: assessment.weaknesses || [],
            recommendations: assessment.recommendations || [],
        };
    }

    private async establishBehaviorBaseline(profile: BehaviorProfile): Promise<void> {
        await this.tool.execute({
            action: "establish_baseline",
            profile,
            duration: profile.duration,
        });
    }

    private async measureDetectionTime(attack: ZeroDayScenario): Promise<number> {
        const startTime = Date.now();
        
        await this.tool.execute({
            action: "execute_attack",
            attack: attack.payload,
            stealth: attack.stealthLevel,
        });

        // Wait for detection
        return new Promise((resolve) => {
            const handler = (event: any) => {
                if (event.type === "zero_day_detected") {
                    this.eventPublisher.off("threat:detected", handler);
                    resolve(Date.now() - startTime);
                }
            };
            
            this.eventPublisher.on("threat:detected", handler);
            
            // Timeout after 30 seconds
            setTimeout(() => {
                this.eventPublisher.off("threat:detected", handler);
                resolve(-1); // Not detected
            }, 30000);
        });
    }

    private async analyzeZeroDayResponse(attack: ZeroDayScenario): Promise<any> {
        const response = await this.tool.execute({
            action: "analyze_zero_day_response",
            attackId: attack.id,
        });

        return response;
    }

    private async setPolicies(policies: SecurityPolicy[]): Promise<void> {
        this.securityPolicies = policies;
        await this.tool.execute({
            action: "set_policies",
            policies,
        });
    }

    private takePolicySnapshot(name: string): PolicySnapshot {
        return {
            name,
            timestamp: new Date(),
            policies: [...this.securityPolicies],
            metrics: {
                totalPolicies: this.securityPolicies.length,
                activeRules: this.securityPolicies.reduce((sum, p) => sum + p.rules.length, 0),
                coverage: this.calculatePolicyCoverage(),
            },
        };
    }

    private async processThreat(threat: ThreatEvent): Promise<void> {
        await this.tool.execute({
            action: "process_threat",
            threat,
        });
    }

    private async getCurrentPolicies(): Promise<SecurityPolicy[]> {
        const result = await this.tool.execute({
            action: "get_policies",
        });
        return result.policies || [];
    }

    private policiesChanged(old: SecurityPolicy[], current: SecurityPolicy[]): boolean {
        if (old.length !== current.length) return true;
        
        // Check for policy modifications
        return !old.every((oldPolicy, i) => 
            JSON.stringify(oldPolicy) === JSON.stringify(current[i]),
        );
    }

    private analyzePolicyImprovements(snapshots: PolicySnapshot[]): PolicyImprovement[] {
        const improvements: PolicyImprovement[] = [];
        
        for (let i = 1; i < snapshots.length; i++) {
            const prev = snapshots[i - 1];
            const curr = snapshots[i];
            
            if (curr.metrics.coverage > prev.metrics.coverage) {
                improvements.push({
                    type: "coverage_increase",
                    from: prev.metrics.coverage,
                    to: curr.metrics.coverage,
                    timestamp: curr.timestamp,
                });
            }
            
            if (curr.metrics.activeRules > prev.metrics.activeRules) {
                improvements.push({
                    type: "rule_addition",
                    from: prev.metrics.activeRules,
                    to: curr.metrics.activeRules,
                    timestamp: curr.timestamp,
                });
            }
        }
        
        return improvements;
    }

    private calculateThreatCoverage(policies: SecurityPolicy[], threats: ThreatEvent[]): number {
        let coveredThreats = 0;
        
        for (const threat of threats) {
            const covered = policies.some(policy => 
                policy.threatTypes.includes(threat.type),
            );
            if (covered) coveredThreats++;
        }
        
        return coveredThreats / threats.length;
    }

    private calculatePolicyCoverage(): number {
        // Simplified coverage calculation
        const knownThreatTypes = ["malware", "intrusion", "ddos", "injection", "xss"];
        const coveredTypes = new Set<string>();
        
        this.securityPolicies.forEach(policy => {
            policy.threatTypes.forEach(type => coveredTypes.add(type));
        });
        
        return coveredTypes.size / knownThreatTypes.length;
    }

    private async introduceNewThreat(threat: any): Promise<void> {
        await this.tool.execute({
            action: "introduce_threat",
            threat,
        });
    }

    private async simulateAttackPattern(pattern: any): Promise<void> {
        await this.tool.execute({
            action: "simulate_pattern",
            pattern,
        });
    }

    private async triggerPolicyViolation(violation: any): Promise<void> {
        await this.tool.execute({
            action: "violate_policy",
            violation,
        });
    }
}

// Type definitions
interface ThreatEvent {
    id: string;
    type: string;
    severity: "low" | "medium" | "high" | "critical";
    timestamp: Date;
    source: string;
    target: string;
    payload?: any;
}

interface Vulnerability {
    id: string;
    name: string;
    severity: "low" | "medium" | "high" | "critical";
    riskScore: number;
    description: string;
    remediation: string;
}

interface SecurityPolicy {
    id: string;
    name: string;
    threatTypes: string[];
    rules: PolicyRule[];
    enabled: boolean;
}

interface PolicyRule {
    id: string;
    condition: string;
    action: string;
}

interface SecurityAdaptation {
    timestamp: Date;
    threatType: string;
    type: string;
    description: string;
}

interface SecurityEvent {
    type: string;
    source: string;
    destination: string;
    payload: any;
    timestamp: Date;
}

interface ThreatDetectionResult {
    truePositives: number;
    falsePositives: number;
    trueNegatives: number;
    falseNegatives: number;
    precision: number;
    recall: number;
    f1Score: number;
    detectedThreats: ThreatEvent[];
}

interface ScanTarget {
    name: string;
    type: "network" | "application" | "system";
    address: string;
    ports?: number[];
}

interface VulnerabilityScanResult {
    target: string;
    totalFound: number;
    criticalCount: number;
    highCount: number;
    mediumCount: number;
    lowCount: number;
    knownVulnerabilityCoverage: number;
    scanDuration: number;
    recommendations: string[];
}

interface AttackScenario {
    name: string;
    attackType: string;
    payload: any;
    expectedDamage: number;
}

interface AttackResult {
    blocked: boolean;
    damage: number;
    detectionTime: number;
}

interface AdaptationEvent {
    scenario: string;
    timestamp: Date;
    initialSuccess: boolean;
    adaptedSuccess: boolean;
    adaptationTime: number;
    adaptationType: string;
}

interface AdaptiveSecurityResult {
    totalScenarios: number;
    successfulAdaptations: number;
    adaptationRate: number;
    averageAdaptationTime: number;
    timeline: AdaptationEvent[];
    finalSecurityPosture: SecurityPosture;
}

interface SecurityPosture {
    score: number;
    strengths: string[];
    weaknesses: string[];
    recommendations: string[];
}

interface BehaviorProfile {
    duration: number;
    normalPatterns: any[];
}

interface ZeroDayScenario {
    id: string;
    name: string;
    payload: any;
    stealthLevel: number;
}

interface ZeroDayResponseResult {
    attack: string;
    detected: boolean;
    detectionTime: number;
    containmentTime: number;
    damageAssessment: any;
    mitigationStrategy: string;
    learningOutcome: any;
}

interface PolicySnapshot {
    name: string;
    timestamp: Date;
    policies: SecurityPolicy[];
    metrics: {
        totalPolicies: number;
        activeRules: number;
        coverage: number;
    };
}

interface PolicyImprovement {
    type: string;
    from: number;
    to: number;
    timestamp: Date;
}

interface PolicyEvolutionResult {
    initialPolicyCount: number;
    finalPolicyCount: number;
    evolutionSteps: number;
    snapshots: PolicySnapshot[];
    improvements: PolicyImprovement[];
    threatsCovered: number;
}

/**
 * Standard security test cases
 */
export const SECURITY_TEST_CASES: ToolTestCase[] = [
    {
        name: "sql-injection-detection",
        description: "Detect SQL injection attempt",
        input: {
            action: "analyze_request",
            request: {
                method: "POST",
                path: "/login",
                body: {
                    username: "admin' OR '1'='1",
                    password: "password",
                },
            },
        },
        expectedOutput: {
            threat: true,
            type: "sql_injection",
        },
    },
    {
        name: "port-scan-detection",
        description: "Detect port scanning activity",
        input: {
            action: "analyze_traffic",
            events: [
                { port: 22, timestamp: Date.now() },
                { port: 23, timestamp: Date.now() + 100 },
                { port: 80, timestamp: Date.now() + 200 },
                { port: 443, timestamp: Date.now() + 300 },
                { port: 3389, timestamp: Date.now() + 400 },
            ],
        },
        expectedOutput: {
            threat: true,
            type: "port_scan",
        },
    },
    {
        name: "vulnerability-scan",
        description: "Scan for known vulnerabilities",
        input: {
            action: "scan",
            target: {
                type: "application",
                address: "localhost:3000",
            },
        },
        expectedBehavior: {
            executionTime: { min: 1000, max: 30000 },
            sideEffects: ["vulnerability_report_generated"],
        },
    },
];