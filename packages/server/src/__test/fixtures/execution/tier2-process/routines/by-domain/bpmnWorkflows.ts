/**
 * BPMN-based multi-step workflow fixtures
 * 
 * Complex workflows that combine multiple routines for comprehensive processing
 */

import type { 
    RoutineVersionConfigObject,
    GraphBpmnConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
} from "@vrooli/shared";
import { ResourceSubType, InputType } from "@vrooli/shared";
import type { RoutineFixture, RoutineFixtureCollection } from "./types.js";
// Import sequential routines for reference in activity maps
import { SECURITY_ROUTINES } from "./securityRoutines.js";
import { MEDICAL_ROUTINES } from "./medicalRoutines.js";
import { PERFORMANCE_ROUTINES } from "./performanceRoutines.js";
import { SYSTEM_ROUTINES } from "./systemRoutines.js";

/**
 * Comprehensive Security Audit Workflow
 * Combines multiple security checks into a comprehensive audit
 * Used by: Security swarms
 */
export const COMPREHENSIVE_SECURITY_AUDIT: RoutineFixture = {
    id: "comprehensive_security_audit",
    name: "Comprehensive Security Audit Workflow",
    description: "Multi-step security audit combining API security, data privacy, and threat analysis",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  xmlns:vrooli="http://vrooli.com/bpmn/extensions">
  <bpmn:process id="comprehensive_security_audit_process" name="Comprehensive Security Audit">
    
    <bpmn:startEvent id="start" name="Start Audit">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:parallelGateway id="split_analysis" name="Split Analysis">
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
      <bpmn:outgoing>flow3</bpmn:outgoing>
      <bpmn:outgoing>flow4</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:callActivity id="api_security_check" name="API Security Check" calledElement="api_security_scan">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="apiLogs" fromContext="root.systemLogs" />
          <vrooli:input name="timeRange" fromContext="root.auditTimeRange" />
          <vrooli:output name="threats" />
          <vrooli:output name="anomalies" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow5</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="data_privacy_check" name="GDPR Compliance Check" calledElement="gdpr_data_audit">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="processingRecords" fromContext="root.dataProcessingRecords" />
          <vrooli:input name="retentionPolicies" fromContext="root.retentionPolicies" />
          <vrooli:output name="complianceScore" />
          <vrooli:output name="violations" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow6</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="threat_analysis" name="Threat Pattern Analysis" calledElement="trading_pattern_analysis">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="tradingData" fromContext="root.activityData" />
          <vrooli:input name="timeWindow" fromContext="root.auditTimeRange" />
          <vrooli:output name="suspiciousPatterns" />
          <vrooli:output name="riskScore" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow4</bpmn:incoming>
      <bpmn:outgoing>flow7</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:parallelGateway id="join_analysis" name="Join Results">
      <bpmn:incoming>flow5</bpmn:incoming>
      <bpmn:incoming>flow6</bpmn:incoming>
      <bpmn:incoming>flow7</bpmn:incoming>
      <bpmn:outgoing>flow8</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:scriptTask id="generate_report" name="Generate Audit Report">
      <bpmn:incoming>flow8</bpmn:incoming>
      <bpmn:outgoing>flow9</bpmn:outgoing>
      <bpmn:script>
        // Combine results from all security checks
        const report = {
          timestamp: new Date(),
          apiSecurity: {
            threats: api_security_check.threats,
            anomalies: api_security_check.anomalies
          },
          dataPrivacy: {
            complianceScore: data_privacy_check.complianceScore,
            violations: data_privacy_check.violations
          },
          threatAnalysis: {
            patterns: threat_analysis.suspiciousPatterns,
            riskScore: threat_analysis.riskScore
          },
          overallRiskScore: calculateOverallRisk(),
          recommendations: generateRecommendations()
        };
        return report;
      </bpmn:script>
    </bpmn:scriptTask>
    
    <bpmn:endEvent id="end" name="Audit Complete">
      <bpmn:incoming>flow9</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="split_analysis" />
    <bpmn:sequenceFlow id="flow2" sourceRef="split_analysis" targetRef="api_security_check" />
    <bpmn:sequenceFlow id="flow3" sourceRef="split_analysis" targetRef="data_privacy_check" />
    <bpmn:sequenceFlow id="flow4" sourceRef="split_analysis" targetRef="threat_analysis" />
    <bpmn:sequenceFlow id="flow5" sourceRef="api_security_check" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow6" sourceRef="data_privacy_check" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow7" sourceRef="threat_analysis" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow8" sourceRef="join_analysis" targetRef="generate_report" />
    <bpmn:sequenceFlow id="flow9" sourceRef="generate_report" targetRef="end" />
    
  </bpmn:process>
</bpmn:definitions>`,
                activityMap: {
                    "api_security_check": {
                        subroutineId: SECURITY_ROUTINES.API_SECURITY_SCAN.id,
                        inputMap: {
                            "apiLogs": "apiLogs",
                            "timeRange": "timeRange",
                        },
                        outputMap: {
                            "threats": "threats",
                            "anomalies": "anomalies",
                        },
                    },
                    "data_privacy_check": {
                        subroutineId: SECURITY_ROUTINES.GDPR_DATA_AUDIT.id,
                        inputMap: {
                            "processingRecords": "processingRecords",
                            "retentionPolicies": "retentionPolicies",
                        },
                        outputMap: {
                            "complianceScore": "complianceScore",
                            "violations": "violations",
                        },
                    },
                    "threat_analysis": {
                        subroutineId: SECURITY_ROUTINES.TRADING_PATTERN_ANALYSIS.id,
                        inputMap: {
                            "tradingData": "tradingData",
                            "timeWindow": "timeWindow",
                        },
                        outputMap: {
                            "suspiciousPatterns": "suspiciousPatterns",
                            "riskScore": "riskScore",
                        },
                    },
                },
                rootContext: {
                    inputMap: {
                        "systemLogs": "systemLogs",
                        "auditTimeRange": "auditTimeRange",
                        "dataProcessingRecords": "dataProcessingRecords",
                        "retentionPolicies": "retentionPolicies",
                        "activityData": "activityData",
                    },
                    outputMap: {
                        "auditReport": "auditReport",
                    },
                },
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "systemLogs",
                        id: "system_logs",
                        label: "System Logs",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "auditTimeRange",
                        id: "audit_time_range",
                        label: "Audit Time Range",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "24h",
                        props: {},
                    },
                    {
                        fieldName: "dataProcessingRecords",
                        id: "data_processing_records",
                        label: "Data Processing Records",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "retentionPolicies",
                        id: "retention_policies",
                        label: "Data Retention Policies",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "activityData",
                        id: "activity_data",
                        label: "User Activity Data",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "auditReport",
                        id: "audit_report",
                        label: "Security Audit Report",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
        subroutineStrategies: {
            "api_security_check": "deterministic",
            "data_privacy_check": "reasoning",
            "threat_analysis": "deterministic",
        },
    } as RoutineVersionConfigObject,
};

/**
 * Medical Treatment Validation Workflow
 * Multi-step validation for AI-generated medical treatments
 * Used by: Medical safety swarms
 */
export const MEDICAL_TREATMENT_VALIDATION: RoutineFixture = {
    id: "medical_treatment_validation",
    name: "Medical Treatment Validation Workflow",
    description: "Comprehensive validation of AI-generated medical treatment plans",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="medical_treatment_validation_process">
    
    <bpmn:startEvent id="start" name="Start Validation">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:callActivity id="hipaa_check" name="HIPAA Compliance Check" calledElement="hipaa_compliance_check">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="aiOutput" fromContext="root.treatmentPlan" />
          <vrooli:input name="confidenceThreshold" fromContext="root.complianceThreshold" />
          <vrooli:output name="phiDetected" />
          <vrooli:output name="complianceStatus" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:exclusiveGateway id="compliance_gateway" name="Compliance Check">
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow3</bpmn:outgoing>
      <bpmn:outgoing>flow_reject</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:callActivity id="clinical_validation" name="Clinical Validation" calledElement="medical_diagnosis_validation">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="diagnosis" fromContext="root.diagnosis" />
          <vrooli:input name="demographics" fromContext="root.patientDemographics" />
          <vrooli:input name="symptoms" fromContext="root.symptoms" />
          <vrooli:output name="validationReport" toRootContext="clinicalValidation" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow4</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="bias_check" name="Bias Detection" calledElement="output_quality_assessment">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="output" fromContext="root.treatmentPlan" />
          <vrooli:input name="criteria" fromContext="root.biasCriteria" />
          <vrooli:output name="biasReport" toRootContext="biasAssessment" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow4</bpmn:incoming>
      <bpmn:outgoing>flow5</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:endEvent id="end_approved" name="Treatment Approved">
      <bpmn:incoming>flow5</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:endEvent id="end_rejected" name="Treatment Rejected">
      <bpmn:incoming>flow_reject</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="hipaa_check" />
    <bpmn:sequenceFlow id="flow2" sourceRef="hipaa_check" targetRef="compliance_gateway" />
    <bpmn:sequenceFlow id="flow3" sourceRef="compliance_gateway" targetRef="clinical_validation">
      <bpmn:conditionExpression>complianceStatus == "compliant"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="flow_reject" sourceRef="compliance_gateway" targetRef="end_rejected">
      <bpmn:conditionExpression>complianceStatus != "compliant"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="flow4" sourceRef="clinical_validation" targetRef="bias_check" />
    <bpmn:sequenceFlow id="flow5" sourceRef="bias_check" targetRef="end_approved" />
    
  </bpmn:process>
</bpmn:definitions>`,
                activityMap: {
                    "hipaa_check": {
                        subroutineId: SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK.id,
                        inputMap: {
                            "aiOutput": "aiOutput",
                            "confidenceThreshold": "confidenceThreshold",
                        },
                        outputMap: {
                            "phiDetected": "phiDetected",
                            "complianceStatus": "complianceStatus",
                        },
                    },
                    "clinical_validation": {
                        subroutineId: MEDICAL_ROUTINES.MEDICAL_DIAGNOSIS_VALIDATION.id,
                        inputMap: {
                            "diagnosis": "diagnosis",
                            "demographics": "demographics",
                            "symptoms": "symptoms",
                        },
                        outputMap: {
                            "validationReport": "response",
                        },
                    },
                    "bias_check": {
                        subroutineId: PERFORMANCE_ROUTINES.OUTPUT_QUALITY_ASSESSMENT.id,
                        inputMap: {
                            "output": "output",
                            "criteria": "criteria",
                        },
                        outputMap: {
                            "biasReport": "response",
                        },
                    },
                },
                rootContext: {
                    inputMap: {
                        "treatmentPlan": "treatmentPlan",
                        "complianceThreshold": "complianceThreshold",
                        "diagnosis": "diagnosis",
                        "patientDemographics": "patientDemographics",
                        "symptoms": "symptoms",
                        "biasCriteria": "biasCriteria",
                    },
                    outputMap: {
                        "clinicalValidation": "clinicalValidation",
                        "biasAssessment": "biasAssessment",
                    },
                },
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "treatmentPlan",
                        id: "treatment_plan",
                        label: "AI Treatment Plan",
                        type: InputType.Text,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "complianceThreshold",
                        id: "compliance_threshold",
                        label: "Compliance Threshold",
                        type: InputType.Slider,
                        isRequired: true,
                        defaultValue: 0.9,
                        props: { min: 0.5, max: 1.0, step: 0.05 },
                    },
                    {
                        fieldName: "diagnosis",
                        id: "diagnosis",
                        label: "Diagnosis",
                        type: InputType.Text,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "patientDemographics",
                        id: "patient_demographics",
                        label: "Patient Demographics",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "symptoms",
                        id: "symptoms",
                        label: "Symptoms",
                        type: InputType.Text,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "biasCriteria",
                        id: "bias_criteria",
                        label: "Bias Detection Criteria",
                        type: InputType.JSON,
                        isRequired: true,
                        defaultValue: {
                            biasThreshold: 0.15,
                            checkDemographic: true,
                            checkCultural: true,
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "clinicalValidation",
                        id: "clinical_validation",
                        label: "Clinical Validation Report",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "biasAssessment",
                        id: "bias_assessment",
                        label: "Bias Assessment Report",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
        allowStrategyOverride: true,
        subroutineStrategies: {
            "hipaa_check": "deterministic",
            "clinical_validation": "reasoning",
            "bias_check": "reasoning",
        },
    } as RoutineVersionConfigObject,
};

/**
 * System Failure Analysis Workflow
 * Multi-step analysis of system failures to identify patterns
 * Used by: Pattern Learning Agent (Resilience)
 */
export const SYSTEM_FAILURE_ANALYSIS_WORKFLOW: RoutineFixture = {
    id: "system_failure_analysis_workflow",
    name: "System Failure Analysis Workflow",
    description: "Analyzes system failure logs to identify patterns and correlations",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="system_failure_analysis_process">
    
    <bpmn:startEvent id="start" name="Start Analysis">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <!-- Search for error logs -->
    <bpmn:callActivity id="search_error_logs" name="Search Error Logs" calledElement="system_log_search">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="searchPattern" fromContext="root.errorPattern" />
          <vrooli:input name="timeRange" fromContext="root.analysisWindow" />
          <vrooli:input name="severity" value="error" />
          <vrooli:output name="logs" toContext="errorLogs" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:callActivity>
    
    <!-- Analyze patterns using LLM -->
    <bpmn:serviceTask id="analyze_patterns" name="Analyze Failure Patterns">
      <bpmn:extensionElements>
        <vrooli:generate>
          <vrooli:prompt>Analyze these system failure logs for patterns:
{{errorLogs}}

Identify:
1. Common failure patterns
2. Correlations between failures
3. Likely root causes
4. Predictions for future failures

Format as JSON with keys: patterns, correlations, rootCauses, predictions</vrooli:prompt>
          <vrooli:output name="analysisResult" />
        </vrooli:generate>
      </bpmn:extensionElements>
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow3</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:endEvent id="end" name="Analysis Complete">
      <bpmn:incoming>flow3</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="search_error_logs" />
    <bpmn:sequenceFlow id="flow2" sourceRef="search_error_logs" targetRef="analyze_patterns" />
    <bpmn:sequenceFlow id="flow3" sourceRef="analyze_patterns" targetRef="end" />
    
  </bpmn:process>
</bpmn:definitions>`,
                activityMap: {
                    "search_error_logs": {
                        subroutineId: SYSTEM_ROUTINES.SYSTEM_LOG_SEARCH.id,
                        inputMap: {
                            "searchPattern": "searchPattern",
                            "timeRange": "timeRange",
                            "severity": "severity",
                        },
                        outputMap: {
                            "logs": "logs",
                        },
                    },
                },
                rootContext: {
                    inputMap: {
                        "errorPattern": "errorPattern",
                        "analysisWindow": "analysisWindow",
                    },
                    outputMap: {
                        "patterns": "patterns",
                        "correlations": "correlations",
                        "rootCauses": "rootCauses",
                        "predictions": "predictions",
                    },
                },
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "errorPattern",
                        id: "error_pattern",
                        label: "Error Pattern to Search",
                        type: InputType.Text,
                        isRequired: false,
                        defaultValue: "error|failure|exception",
                        props: {},
                    },
                    {
                        fieldName: "analysisWindow",
                        id: "analysis_window",
                        label: "Analysis Window",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "24h",
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "patterns",
                        id: "patterns",
                        label: "Detected Failure Patterns",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "correlations",
                        id: "correlations",
                        label: "Failure Correlations",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "rootCauses",
                        id: "root_causes",
                        label: "Likely Root Causes",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "predictions",
                        id: "predictions",
                        label: "Failure Predictions",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
    } as RoutineVersionConfigObject,
};

/**
 * System Resilience Optimization Workflow
 * Analyzes failures and optimizes system resilience
 * Used by: Resilience agent swarms
 */
export const RESILIENCE_OPTIMIZATION_WORKFLOW: RoutineFixture = {
    id: "resilience_optimization_workflow",
    name: "System Resilience Optimization",
    description: "Comprehensive workflow for analyzing failures and optimizing system resilience",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="resilience_optimization_process">
    
    <bpmn:startEvent id="start" name="Start Analysis">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:callActivity id="failure_analysis" name="Analyze Failures" calledElement="system_failure_analysis_workflow">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="failureLogs" fromContext="root.systemLogs" />
          <vrooli:input name="analysisWindow" fromContext="root.timeWindow" />
          <vrooli:output name="patterns" />
          <vrooli:output name="correlations" />
          <vrooli:output name="predictions" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="performance_analysis" name="Performance Analysis" calledElement="performance_bottleneck_detection">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="metrics" fromContext="root.performanceMetrics" />
          <vrooli:input name="config" fromContext="root.systemConfig" />
          <vrooli:input name="threshold" fromContext="root.performanceThreshold" />
          <vrooli:output name="bottlenecks" />
          <vrooli:output name="recommendations" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow3</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:scriptTask id="generate_optimization_plan" name="Generate Optimization Plan">
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow4</bpmn:outgoing>
      <bpmn:script>
        const plan = {
          failurePatterns: failure_analysis.patterns,
          correlations: failure_analysis.correlations,
          predictions: failure_analysis.predictions,
          bottlenecks: performance_analysis.bottlenecks,
          recommendations: performance_analysis.recommendations,
          optimizationStrategy: generateStrategy(),
          implementationPlan: createImplementationPlan()
        };
        return plan;
      </bpmn:script>
    </bpmn:scriptTask>
    
    <bpmn:endEvent id="end" name="Plan Ready">
      <bpmn:incoming>flow4</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="failure_analysis" />
    <bpmn:sequenceFlow id="flow2" sourceRef="failure_analysis" targetRef="performance_analysis" />
    <bpmn:sequenceFlow id="flow3" sourceRef="performance_analysis" targetRef="generate_optimization_plan" />
    <bpmn:sequenceFlow id="flow4" sourceRef="generate_optimization_plan" targetRef="end" />
    
  </bpmn:process>
</bpmn:definitions>`,
                activityMap: {
                    "failure_analysis": {
                        subroutineId: "system_failure_analysis_workflow",
                        inputMap: {
                            "errorPattern": "errorPattern",
                            "analysisWindow": "analysisWindow",
                        },
                        outputMap: {
                            "patterns": "patterns",
                            "correlations": "correlations",
                            "predictions": "predictions",
                        },
                    },
                    "performance_analysis": {
                        subroutineId: PERFORMANCE_ROUTINES.PERFORMANCE_BOTTLENECK_DETECTION.id,
                        inputMap: {
                            "metrics": "metrics",
                            "config": "config",
                            "threshold": "threshold",
                        },
                        outputMap: {
                            "bottlenecks": "bottlenecks",
                            "recommendations": "recommendations",
                        },
                    },
                },
                rootContext: {
                    inputMap: {
                        "errorPattern": "errorPattern",
                        "analysisWindow": "analysisWindow",
                        "performanceMetrics": "performanceMetrics",
                        "systemConfig": "systemConfig",
                        "performanceThreshold": "performanceThreshold",
                    },
                    outputMap: {
                        "optimizationPlan": "optimizationPlan",
                    },
                },
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "systemLogs",
                        id: "system_logs",
                        label: "System Failure Logs",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "timeWindow",
                        id: "time_window",
                        label: "Analysis Time Window",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "24h",
                        props: {},
                    },
                    {
                        fieldName: "performanceMetrics",
                        id: "performance_metrics",
                        label: "Performance Metrics",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "systemConfig",
                        id: "system_config",
                        label: "System Configuration",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "performanceThreshold",
                        id: "performance_threshold",
                        label: "Performance Threshold (ms)",
                        type: InputType.IntegerInput,
                        isRequired: true,
                        defaultValue: 1000,
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "optimizationPlan",
                        id: "optimization_plan",
                        label: "Resilience Optimization Plan",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
        subroutineStrategies: {
            "failure_analysis": "reasoning",
            "performance_analysis": "deterministic",
        },
    } as RoutineVersionConfigObject,
};

/**
 * Market Analysis Routing Workflow
 * Demonstrates intelligent routing with parallel execution
 */
export const MARKET_ANALYSIS_ROUTING_WORKFLOW: RoutineFixture = {
    id: "market_analysis_routing_workflow",
    name: "Intelligent Market Analysis Routing",
    description: "Coordinates multiple specialized analysis routines based on market conditions using routing strategy",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        strategy: "routing",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:process id="market_analysis_routing" name="Market Analysis Routing">
    <!-- Initial market condition assessment -->
    <bpmn:serviceTask id="assess_conditions" name="Assess Market Conditions">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:output name="marketVolatility" />
            <vrooli:output name="sectorFocus" />
            <vrooli:output name="analysisDepth" />
        </bpmn:extensionElements>
    </bpmn:serviceTask>
    
    <!-- Intelligent routing gateway -->
    <bpmn:exclusiveGateway id="route_analysis" name="Route by Conditions">
        <bpmn:extensionElements>
            <vrooli:routingLogic>
                <vrooli:rule condition="marketVolatility > 0.8" target="crisis_analysis" />
                <vrooli:rule condition="marketVolatility < 0.3" target="standard_analysis" />
                <vrooli:rule condition="default" target="adaptive_analysis" />
            </vrooli:routingLogic>
        </bpmn:extensionElements>
    </bpmn:exclusiveGateway>
    
    <!-- Parallel sub-workflow execution -->
    <bpmn:subProcess id="adaptive_analysis" name="Adaptive Analysis">
        <bpmn:parallelGateway id="split_adaptive" />
        
        <bpmn:callActivity id="technical_analysis" calledElement="technical_indicators_v3">
            <vrooli:contextSharing>selective</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:callActivity id="sentiment_analysis" calledElement="market_sentiment_v2">
            <vrooli:contextSharing>selective</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:callActivity id="fundamental_analysis" calledElement="fundamental_metrics_v3">
            <vrooli:contextSharing>full</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:parallelGateway id="join_adaptive" />
    </bpmn:subProcess>
    
    <!-- Result aggregation -->
    <bpmn:serviceTask id="aggregate_results" name="Aggregate Analysis Results">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:aggregation>
                <vrooli:method>weighted_merge</vrooli:method>
                <vrooli:weights>
                    <vrooli:technical>0.3</vrooli:technical>
                    <vrooli:sentiment>0.2</vrooli:sentiment>
                    <vrooli:fundamental>0.5</vrooli:fundamental>
                </vrooli:weights>
            </vrooli:aggregation>
        </bpmn:extensionElements>
    </bpmn:serviceTask>
</bpmn:process>`,
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "marketData",
                        id: "market_data",
                        label: "Market Data Feed",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "analysisTimeframe",
                        id: "analysis_timeframe",
                        label: "Analysis Timeframe",
                        type: InputType.Dropzone,
                        isRequired: true,
                        props: {
                            options: ["1h", "4h", "1d", "1w", "1m"],
                        },
                    },
                    {
                        fieldName: "riskTolerance",
                        id: "risk_tolerance",
                        label: "Risk Tolerance Level",
                        type: InputType.Slider,
                        isRequired: true,
                        defaultValue: 0.5,
                        props: {
                            min: 0,
                            max: 1,
                            step: 0.1,
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "analysisReport",
                        id: "analysis_report",
                        label: "Comprehensive Market Analysis",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "tradingSignals",
                        id: "trading_signals",
                        label: "Trading Signals",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "confidenceScore",
                        id: "confidence_score",
                        label: "Analysis Confidence",
                        type: InputType.Slider,
                        isRequired: true,
                        props: {
                            min: 0,
                            max: 1,
                        },
                    },
                ],
            },
        } as FormOutputConfigObject,
    } as RoutineVersionConfigObject,
};

/**
 * Customer Support Routing Workflow
 * Demonstrates scatter-gather pattern for customer support
 */
export const CUSTOMER_SUPPORT_ROUTING_WORKFLOW: RoutineFixture = {
    id: "customer_support_routing_workflow",
    name: "Customer Support Intelligent Routing",
    description: "Routes customer inquiries to specialized sub-routines based on issue type",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        strategy: "routing",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:process id="customer_support_routing" name="Customer Support Routing">
    <!-- Issue classification -->
    <bpmn:serviceTask id="classify_issue" name="Classify Customer Issue">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:classification>
                <vrooli:categories>technical,billing,account,general</vrooli:categories>
                <vrooli:confidence>0.85</vrooli:confidence>
            </vrooli:classification>
        </bpmn:extensionElements>
    </bpmn:serviceTask>
    
    <!-- Routing based on classification -->
    <bpmn:eventBasedGateway id="route_by_type" name="Route by Issue Type">
        <bpmn:outgoing>to_technical</bpmn:outgoing>
        <bpmn:outgoing>to_billing</bpmn:outgoing>
        <bpmn:outgoing>to_account</bpmn:outgoing>
        <bpmn:outgoing>to_escalation</bpmn:outgoing>
    </bpmn:eventBasedGateway>
    
    <!-- Parallel execution of specialized handlers -->
    <bpmn:subProcess id="technical_handler" name="Technical Support Handler">
        <bpmn:callActivity id="diagnose_issue" calledElement="technical_diagnosis_v3" />
        <bpmn:callActivity id="solution_lookup" calledElement="knowledge_base_search_v2" />
        <bpmn:callActivity id="generate_solution" calledElement="solution_generator_v2" />
    </bpmn:subProcess>
    
    <bpmn:subProcess id="billing_handler" name="Billing Support Handler">
        <bpmn:callActivity id="account_lookup" calledElement="account_retrieval_v3" />
        <bpmn:callActivity id="billing_analysis" calledElement="billing_analyzer_v2" />
        <bpmn:callActivity id="payment_options" calledElement="payment_resolver_v2" />
    </bpmn:subProcess>
    
    <!-- Context propagation and result synthesis -->
    <bpmn:serviceTask id="synthesize_response" name="Synthesize Final Response">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:contextMerge>
                <vrooli:preserveCustomer>true</vrooli:preserveCustomer>
                <vrooli:combineResults>intelligent</vrooli:combineResults>
                <vrooli:personalizeResponse>true</vrooli:personalizeResponse>
            </vrooli:contextMerge>
        </bpmn:extensionElements>
    </bpmn:serviceTask>
</bpmn:process>`,
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "customerInquiry",
                        id: "customer_inquiry",
                        label: "Customer Inquiry",
                        type: InputType.TextArea,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "customerId",
                        id: "customer_id",
                        label: "Customer ID",
                        type: InputType.Text,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "inquiryHistory",
                        id: "inquiry_history",
                        label: "Previous Inquiries",
                        type: InputType.JSON,
                        isRequired: false,
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "resolution",
                        id: "resolution",
                        label: "Issue Resolution",
                        type: InputType.TextArea,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "nextSteps",
                        id: "next_steps",
                        label: "Recommended Next Steps",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "satisfactionPrediction",
                        id: "satisfaction_prediction",
                        label: "Predicted Satisfaction Score",
                        type: InputType.Slider,
                        isRequired: true,
                        props: {
                            min: 0,
                            max: 5,
                        },
                    },
                ],
            },
        } as FormOutputConfigObject,
    } as RoutineVersionConfigObject,
};

/**
 * Healthcare Triage Routing Workflow
 * Demonstrates priority-based routing with context preservation
 */
export const HEALTHCARE_TRIAGE_ROUTING_WORKFLOW: RoutineFixture = {
    id: "healthcare_triage_routing_workflow",
    name: "Healthcare Triage Intelligent Routing",
    description: "Routes patient cases to appropriate specialists based on urgency and symptoms",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineMultiStep,
    config: {
        __version: "1.0",
        strategy: "routing",
        graph: {
            __version: "1.0",
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:process id="healthcare_triage_routing" name="Healthcare Triage Routing">
    <!-- Initial triage assessment -->
    <bpmn:serviceTask id="triage_assessment" name="Initial Triage Assessment">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:priority>
                <vrooli:levels>critical,urgent,standard,routine</vrooli:levels>
                <vrooli:hipaaCompliant>true</vrooli:hipaaCompliant>
            </vrooli:priority>
        </bpmn:extensionElements>
    </bpmn:serviceTask>
    
    <!-- Priority-based routing -->
    <bpmn:exclusiveGateway id="priority_routing" name="Route by Priority">
        <bpmn:extensionElements>
            <vrooli:priorityRouting>
                <vrooli:critical>immediate_care_path</vrooli:critical>
                <vrooli:urgent>expedited_care_path</vrooli:urgent>
                <vrooli:standard>standard_care_path</vrooli:standard>
                <vrooli:routine>scheduled_care_path</vrooli:routine>
            </vrooli:priorityRouting>
        </bpmn:extensionElements>
    </bpmn:exclusiveGateway>
    
    <!-- Parallel specialist consultations -->
    <bpmn:subProcess id="specialist_consultations" name="Specialist Consultations">
        <bpmn:parallelGateway id="split_specialists" />
        
        <bpmn:callActivity id="symptom_analysis" calledElement="medical_symptom_analyzer_v3">
            <vrooli:contextSharing>phi_compliant</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:callActivity id="lab_analysis" calledElement="lab_result_interpreter_v2">
            <vrooli:contextSharing>selective</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:callActivity id="imaging_analysis" calledElement="imaging_analyzer_v2">
            <vrooli:contextSharing>secure</vrooli:contextSharing>
        </bpmn:callActivity>
        
        <bpmn:parallelGateway id="join_specialists" />
    </bpmn:subProcess>
    
    <!-- Treatment recommendation synthesis -->
    <bpmn:serviceTask id="synthesize_treatment" name="Synthesize Treatment Plan">
        <bpmn:extensionElements>
            <vrooli:strategy>reasoning</vrooli:strategy>
            <vrooli:compliance>
                <vrooli:hipaa>true</vrooli:hipaa>
                <vrooli:medicalStandards>true</vrooli:medicalStandards>
                <vrooli:auditTrail>complete</vrooli:auditTrail>
            </vrooli:compliance>
        </bpmn:extensionElements>
    </bpmn:serviceTask>
</bpmn:process>`,
            },
        } as GraphBpmnConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "patientSymptoms",
                        id: "patient_symptoms",
                        label: "Patient Symptoms",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "vitalSigns",
                        id: "vital_signs",
                        label: "Vital Signs",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "medicalHistory",
                        id: "medical_history",
                        label: "Medical History",
                        type: InputType.JSON,
                        isRequired: false,
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "triageLevel",
                        id: "triage_level",
                        label: "Triage Priority Level",
                        type: InputType.Dropzone,
                        isRequired: true,
                        props: {
                            options: ["critical", "urgent", "standard", "routine"],
                        },
                    },
                    {
                        fieldName: "treatmentPlan",
                        id: "treatment_plan",
                        label: "Recommended Treatment Plan",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "specialistReferrals",
                        id: "specialist_referrals",
                        label: "Specialist Referrals",
                        type: InputType.JSON,
                        isRequired: false,
                        props: {},
                    },
                ],
            },
        } as FormOutputConfigObject,
    } as RoutineVersionConfigObject,
};

export const BPMN_WORKFLOWS: RoutineFixtureCollection<"COMPREHENSIVE_SECURITY_AUDIT" | "MEDICAL_TREATMENT_VALIDATION" | "SYSTEM_FAILURE_ANALYSIS_WORKFLOW" | "RESILIENCE_OPTIMIZATION_WORKFLOW" | "MARKET_ANALYSIS_ROUTING_WORKFLOW" | "CUSTOMER_SUPPORT_ROUTING_WORKFLOW" | "HEALTHCARE_TRIAGE_ROUTING_WORKFLOW"> = {
    COMPREHENSIVE_SECURITY_AUDIT,
    MEDICAL_TREATMENT_VALIDATION,
    SYSTEM_FAILURE_ANALYSIS_WORKFLOW,
    RESILIENCE_OPTIMIZATION_WORKFLOW,
    MARKET_ANALYSIS_ROUTING_WORKFLOW,
    CUSTOMER_SUPPORT_ROUTING_WORKFLOW,
    HEALTHCARE_TRIAGE_ROUTING_WORKFLOW,
};
