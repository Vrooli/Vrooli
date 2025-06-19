/**
 * Medical and healthcare routine fixtures
 * 
 * Routines for medical diagnosis validation, safety checks, and clinical compliance
 */

import type { 
    RoutineVersionConfigObject,
    CallDataGenerateConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
} from "@vrooli/shared";
import { ResourceSubType, InputType, BotStyle } from "@vrooli/shared";
import type { RoutineFixture, RoutineFixtureCollection } from "./types.js";

/**
 * Medical Diagnosis Validation Routine
 * Used by: Medical Safety Agent
 * Purpose: Validates AI-generated diagnoses against clinical guidelines
 */
export const MEDICAL_DIAGNOSIS_VALIDATION: RoutineFixture = {
    id: "medical_diagnosis_validation",
    name: "Medical Diagnosis Validation",
    description: "Validates AI-generated medical diagnoses against clinical guidelines and checks for bias",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineGenerate,
    config: {
        __version: "1.0",
        callDataGenerate: {
            __version: "1.0",
            schema: {
                botStyle: BotStyle.Academic,
                maxTokens: 1000,
                model: null,
                prompt: "Validate the following medical diagnosis:\n\nDiagnosis: {{input.diagnosis}}\nPatient Demographics: {{input.demographics}}\nSymptoms: {{input.symptoms}}\n\nCheck for:\n1. Clinical accuracy against latest guidelines\n2. Potential demographic bias\n3. Missing critical symptoms\n4. Safety concerns\n\nProvide a detailed validation report.",
                respondingBot: null
            }
        } as CallDataGenerateConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "diagnosis",
                        id: "diagnosis",
                        label: "AI Diagnosis",
                        type: InputType.Text,
                        isRequired: true,
                        props: {}
                    },
                    {
                        fieldName: "demographics",
                        id: "demographics",
                        label: "Patient Demographics",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {}
                    },
                    {
                        fieldName: "symptoms",
                        id: "symptoms",
                        label: "Symptoms",
                        type: InputType.Text,
                        isRequired: true,
                        props: {}
                    }
                ]
            }
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "response",
                        id: "response",
                        label: "Validation Report",
                        type: InputType.Text,
                        props: {
                            disabled: true
                        }
                    }
                ]
            }
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
    } as RoutineVersionConfigObject,
};

export const MEDICAL_ROUTINES: RoutineFixtureCollection<"MEDICAL_DIAGNOSIS_VALIDATION"> = {
    MEDICAL_DIAGNOSIS_VALIDATION,
} as const;