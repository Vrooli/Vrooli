import { McpToolName } from "../../../consts/mcp.js";
import { InputType } from "../../../consts/model.js";
import { type RoutineVersionConfigObject } from "../../../shape/configs/routine.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

/**
 * Routine configuration fixtures for testing various routine types
 */
export const routineConfigFixtures: ConfigTestFixtures<RoutineVersionConfigObject> & {
    // Additional categorized fixtures for different routine types
    action: {
        simple: RoutineVersionConfigObject;
        withInputMapping: RoutineVersionConfigObject;
        withOutputMapping: RoutineVersionConfigObject;
        withMachine: RoutineVersionConfigObject;
    };
    generate: {
        basic: RoutineVersionConfigObject;
        withCustomModel: RoutineVersionConfigObject;
        withComplexPrompt: RoutineVersionConfigObject;
    };
    multiStep: {
        sequential: RoutineVersionConfigObject;
        withBranching: RoutineVersionConfigObject;
        complexWorkflow: RoutineVersionConfigObject;
    };
} = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        graph: {
            __version: LATEST_CONFIG_VERSION,
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="multiStepProcess" isExecutable="true">
    <startEvent id="start"/>
    <task id="step1" name="Data Collection">
      <incoming>flow1</incoming>
      <outgoing>flow2</outgoing>
    </task>
    <task id="step2" name="Process Data">
      <incoming>flow2</incoming>
      <outgoing>flow3</outgoing>
    </task>
    <endEvent id="end">
      <incoming>flow3</incoming>
    </endEvent>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="step1"/>
    <sequenceFlow id="flow2" sourceRef="step1" targetRef="step2"/>
    <sequenceFlow id="flow3" sourceRef="step2" targetRef="end"/>
  </process>
</definitions>`,
                activityMap: {
                    step1: {
                        subroutineId: "subroutine_1",
                        inputMap: { input1: "stepInput1" },
                        outputMap: { output1: "stepOutput1" },
                    },
                    step2: {
                        subroutineId: "subroutine_2", 
                        inputMap: { input2: "stepInput2" },
                        outputMap: { output2: "stepOutput2" },
                    },
                },
                rootContext: {
                    inputMap: { routineInput: "processInput" },
                    outputMap: { routineOutput: "processOutput" },
                },
            },
        },
        resources: [{
            link: "https://example.com/routine-guide",
            usedFor: "Tutorial",
            translations: [{
                language: "en",
                name: "Routine Guide",
                description: "How to use this routine",
            }],
        }],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        resources: [],
    },

    invalid: {
        missingVersion: {
        },
        invalidVersion: {
            __version: "0.1",
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            // Invalid: mixing incompatible config types
            callDataAction: {},
            callDataApi: {},
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            // @ts-expect-error - Testing invalid type for callDataAction  
            callDataAction: "not an object",
        },
    },

    variants: {
        simpleApiCall: {
            __version: LATEST_CONFIG_VERSION,
            callDataApi: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    endpoint: "/api/v1/users",
                    method: "GET",
                },
            },
        },

        textGeneration: {
            __version: LATEST_CONFIG_VERSION,
            callDataGenerate: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    model: {
                        name: "GPT_4_Name",
                        value: "gpt-4-0125-preview",
                    },
                    prompt: "Generate a summary of: {{text}}",
                    maxTokens: 500,
                },
            },
        },

        dataTransformation: {
            __version: LATEST_CONFIG_VERSION,
            callDataCode: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    inputTemplate: {
                        name: "{{input.name}}",
                        age: "{{input.age}}",
                    },
                    outputMappings: [{
                        schemaIndex: 0,
                        mapping: {
                            fullName: "output.name",
                            isAdult: "output.adult",
                        },
                    }],
                },
            },
        },
    },

    // Action routine fixtures
    action: {
        simple: {
            __version: LATEST_CONFIG_VERSION,
            callDataAction: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    toolName: McpToolName.ResourceManage,
                    inputTemplate: JSON.stringify({
                        op: "find",
                        resource_type: "Note",
                    }),
                    outputMapping: {},
                },
            },
        },

        withInputMapping: {
            __version: LATEST_CONFIG_VERSION,
            callDataApi: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    endpoint: "/api/v1/process",
                    method: "POST",
                    body: {
                        userInput: "{{input.data}}",
                        userId: "{{input.userId}}",
                        timestamp: "{{now()}}",
                    },
                },
            },
        },

        withOutputMapping: {
            __version: LATEST_CONFIG_VERSION,
            callDataApi: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    endpoint: "/api/v1/analyze",
                    method: "POST",
                    body: {},
                },
            },
            formOutput: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    containers: [],
                    elements: [
                        {
                            id: "result",
                            fieldName: "result",
                            label: "Analysis Result",
                            type: InputType.Text,
                            props: {
                                placeholder: "Analysis result will appear here",
                                maxChars: 1000,
                            },
                        },
                        {
                            id: "confidence",
                            fieldName: "confidence",
                            label: "Confidence Score",
                            type: InputType.IntegerInput,
                            props: {
                                min: 0,
                                max: 100,
                                defaultValue: 0,
                            },
                        },
                    ],
                },
            },
        },

        withMachine: {
            __version: LATEST_CONFIG_VERSION,
            callDataAction: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    // @ts-expect-error - Testing invalid tool name
                    toolName: "machine_execute",
                    inputTemplate: JSON.stringify({
                        machineId: "machine_123456789",
                        operation: "process",
                        parameters: {
                            mode: "advanced",
                            timeout: 30000,
                        },
                    }),
                    outputMapping: {},
                },
            },
        },
    },

    // Generate routine fixtures
    generate: {
        basic: {
            __version: LATEST_CONFIG_VERSION,
            callDataGenerate: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    model: {
                        name: "GPT_4o_Mini_Name",
                        value: "gpt-4o-mini-2024-07-18",
                    },
                    prompt: "Answer the following question: {{input.question}}",
                },
            },
        },

        withCustomModel: {
            __version: LATEST_CONFIG_VERSION,
            callDataGenerate: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    model: {
                        name: "Claude_3_Opus_Name",
                        value: "claude-3-opus-20240229",
                    },
                    prompt: "Analyze the following code and suggest improvements: {{input.code}}",
                    maxTokens: 2000,
                },
            },
        },

        withComplexPrompt: {
            __version: LATEST_CONFIG_VERSION,
            callDataGenerate: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    model: {
                        name: "GPT_4_Name",
                        value: "gpt-4-0125-preview",
                    },
                    prompt: `Based on the following context:
                User Profile: {{input.userProfile}}
                Previous Interactions: {{input.history}}
                Current Request: {{input.request}}
                
                Generate a personalized response that:
                1. Addresses the user's specific needs
                2. Takes into account their history
                3. Provides actionable next steps`,
                    maxTokens: 1000,
                },
            },
        },
    },

    // Multi-step routine fixtures
    multiStep: {
        sequential: {
            __version: LATEST_CONFIG_VERSION,
            graph: {
                __version: LATEST_CONFIG_VERSION,
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="sequentialProcess" isExecutable="true">
    <startEvent id="start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="fetch"/>
    <callActivity id="fetch" name="Fetch Data" calledElement="fetchData"/>
    <sequenceFlow id="flow2" sourceRef="fetch" targetRef="process"/>
    <callActivity id="process" name="Process Data" calledElement="processData"/>
    <sequenceFlow id="flow3" sourceRef="process" targetRef="save"/>
    <callActivity id="save" name="Save Results" calledElement="saveResults"/>
    <sequenceFlow id="flow4" sourceRef="save" targetRef="end"/>
    <endEvent id="end"/>
  </process>
</definitions>`,
                    activityMap: {
                        "fetch": {
                            subroutineId: "sub_fetch_123",
                            inputMap: {
                                "url": "input.dataUrl",
                            },
                            outputMap: {
                                "data": "fetchedData",
                            },
                        },
                        "process": {
                            subroutineId: "sub_process_456",
                            inputMap: {
                                "rawData": "fetchedData",
                            },
                            outputMap: {
                                "summary": "processedSummary",
                            },
                        },
                        "save": {
                            subroutineId: "sub_save_789",
                            inputMap: {
                                "content": "processedSummary",
                            },
                            outputMap: {
                                "id": "savedId",
                            },
                        },
                    },
                    rootContext: {
                        inputMap: {
                            "dataUrl": "url",
                        },
                        outputMap: {
                            "resultId": "savedId",
                        },
                    },
                },
            },
        },

        withBranching: {
            __version: LATEST_CONFIG_VERSION,
            graph: {
                __version: LATEST_CONFIG_VERSION,
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="branchingProcess" isExecutable="true">
    <startEvent id="start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="check"/>
    <exclusiveGateway id="check" name="Check Condition"/>
    <sequenceFlow id="flow2" sourceRef="check" targetRef="highValue">
      <conditionExpression>value > 100</conditionExpression>
    </sequenceFlow>
    <sequenceFlow id="flow3" sourceRef="check" targetRef="lowValue">
      <conditionExpression>value <= 100</conditionExpression>
    </sequenceFlow>
    <callActivity id="highValue" name="Process High Value" calledElement="processHighValue"/>
    <callActivity id="lowValue" name="Process Low Value" calledElement="processLowValue"/>
    <sequenceFlow id="flow4" sourceRef="highValue" targetRef="end"/>
    <sequenceFlow id="flow5" sourceRef="lowValue" targetRef="end"/>
    <endEvent id="end"/>
  </process>
</definitions>`,
                    activityMap: {
                        "highValue": {
                            subroutineId: "sub_high_value_123",
                            inputMap: {
                                "value": "input.value",
                            },
                            outputMap: {
                                "result": "highResult",
                            },
                        },
                        "lowValue": {
                            subroutineId: "sub_low_value_456",
                            inputMap: {
                                "value": "input.value",
                            },
                            outputMap: {
                                "result": "lowResult",
                            },
                        },
                    },
                    rootContext: {
                        inputMap: {
                            "value": "inputValue",
                        },
                        outputMap: {
                            "result": "finalResult",
                        },
                    },
                },
            },
        },

        complexWorkflow: {
            __version: LATEST_CONFIG_VERSION,
            graph: {
                __version: LATEST_CONFIG_VERSION,
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="complexWorkflow" isExecutable="true">
    <startEvent id="start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="init"/>
    <callActivity id="init" name="Initialize" calledElement="initialize"/>
    <sequenceFlow id="flow2" sourceRef="init" targetRef="fetchSources"/>
    <callActivity id="fetchSources" name="Fetch Data Sources" calledElement="fetchDataSources"/>
    <sequenceFlow id="flow3" sourceRef="fetchSources" targetRef="analyzeEach"/>
    <callActivity id="analyzeEach" name="Analyze Each Source" calledElement="analyzeSource"/>
    <sequenceFlow id="flow4" sourceRef="analyzeEach" targetRef="aggregate"/>
    <callActivity id="aggregate" name="Aggregate Results" calledElement="aggregateResults"/>
    <sequenceFlow id="flow5" sourceRef="aggregate" targetRef="generateReport"/>
    <callActivity id="generateReport" name="Generate Report" calledElement="generateReport"/>
    <sequenceFlow id="flow6" sourceRef="generateReport" targetRef="notify"/>
    <callActivity id="notify" name="Send Notifications" calledElement="sendNotifications"/>
    <sequenceFlow id="flow7" sourceRef="notify" targetRef="end"/>
    <endEvent id="end"/>
  </process>
</definitions>`,
                    activityMap: {
                        "init": {
                            subroutineId: "sub_init_001",
                            inputMap: {},
                            outputMap: {
                                "context": "workflowContext",
                            },
                        },
                        "fetchSources": {
                            subroutineId: "sub_fetch_002",
                            inputMap: {
                                "context": "workflowContext",
                            },
                            outputMap: {
                                "sources": "dataSources",
                            },
                        },
                        "analyzeEach": {
                            subroutineId: "sub_analyze_003",
                            inputMap: {
                                "sources": "dataSources",
                            },
                            outputMap: {
                                "analyses": "sourceAnalyses",
                            },
                        },
                        "aggregate": {
                            subroutineId: "sub_aggregate_004",
                            inputMap: {
                                "analyses": "sourceAnalyses",
                            },
                            outputMap: {
                                "aggregated": "aggregatedData",
                            },
                        },
                        "generateReport": {
                            subroutineId: "sub_report_005",
                            inputMap: {
                                "data": "aggregatedData",
                            },
                            outputMap: {
                                "report": "finalReport",
                            },
                        },
                        "notify": {
                            subroutineId: "sub_notify_006",
                            inputMap: {
                                "report": "finalReport",
                            },
                            outputMap: {
                                "notificationStatus": "status",
                            },
                        },
                    },
                    rootContext: {
                        inputMap: {
                            "input": "workflowInput",
                        },
                        outputMap: {
                            "report": "finalReport",
                            "status": "notificationStatus",
                        },
                    },
                },
            },
        },
    },
};

/**
 * Create a simple action routine config
 */
export function createActionRoutineConfig(
    toolName = McpToolName.ResourceManage,
    inputTemplate = JSON.stringify({ op: "find", resource_type: "Note" }),
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        callDataAction: {
            __version: LATEST_CONFIG_VERSION,
            schema: {
                toolName,
                inputTemplate,
                outputMapping: {},
            },
        },
    };
}

/**
 * Create a generate routine config
 */
export function createGenerateRoutineConfig(
    prompt: string,
    model = {
        name: "GPT_4o_Mini_Name",
        value: "gpt-4o-mini-2024-07-18",
    },
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        callDataGenerate: {
            __version: LATEST_CONFIG_VERSION,
            schema: {
                model,
                prompt,
            },
        },
    };
}

/**
 * Create a multi-step routine config
 */
export function createMultiStepRoutineConfig(
    graphData = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="multiStepProcess" isExecutable="true">
    <startEvent id="start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="step1"/>
    <callActivity id="step1" name="Step 1" calledElement="step1"/>
    <sequenceFlow id="flow2" sourceRef="step1" targetRef="end"/>
    <endEvent id="end"/>
  </process>
</definitions>`,
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        graph: {
            __version: LATEST_CONFIG_VERSION,
            __type: "BPMN-2.0",
            schema: {
                __format: "xml",
                data: graphData,
                activityMap: {},
                rootContext: {
                    inputMap: {},
                    outputMap: {},
                },
            },
        },
    };
}
