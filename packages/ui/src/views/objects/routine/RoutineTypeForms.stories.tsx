import { BotStyle, FormBuilder, InputType, LATEST_ROUTINE_CONFIG_VERSION, RoutineType, RoutineVersionConfig, noop, noopSubmit, uuid } from "@local/shared";
import { Box } from "@mui/material";
import { Formik } from "formik";
import { useMemo } from "react";
import { PageContainer } from "../../../components/Page/Page.js";
import { ScrollBox } from "../../../styles.js";
import {
    RoutineApiForm,
    RoutineDataConverterForm,
    RoutineDataForm,
    type RoutineFormPropsBase,
    RoutineGenerateForm,
    RoutineInformationalForm,
    RoutineSmartContractForm,
} from "./RoutineTypeForms.js";

export default {
    title: "Views/Objects/Routine/RoutineTypeForms",
};

const mockFormProps: Omit<RoutineFormPropsBase, "config" | "display"> = {
    disabled: false,
    handleClearRun: noop,
    handleCompleteStep: noop,
    handleRunStep: noop,
    hasErrors: false,
    isCompleteStepDisabled: false,
    isPartOfMultiStepRoutine: false,
    isRunStepDisabled: false,
    isRunningStep: false,
    onCallDataActionChange: noop,
    onCallDataApiChange: noop,
    onCallDataCodeChange: noop,
    onCallDataGenerateChange: noop,
    onCallDataSmartContractChange: noop,
    onFormInputChange: noop,
    onFormOutputChange: noop,
    onGraphChange: noop,
    onRunChange: noop,
    routineId: uuid(),
    routineName: "Test Routine",
    run: null,
};

const emptyFormParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({ __version: LATEST_ROUTINE_CONFIG_VERSION }),
};

const apiFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        callDataApi: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                endpoint: "https://api.example.com/v1/create",
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: {
                    name: "John Doe",
                    email: "john.doe@example.com",
                },
                meta: {
                    retry: false,
                    timeoutMs: 10000,
                },
            },
        },
        formInput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "name",
                        type: InputType.Text,
                        label: "Name",
                        fieldName: "name",
                        description: "Enter your name",
                        props: {
                            defaultValue: "",
                            placeholder: "John Doe",
                        },
                    },
                    {
                        id: "email",
                        type: InputType.Text,
                        label: "Email",
                        fieldName: "email",
                        description: "Enter your email",
                        props: {
                            defaultValue: "",
                            placeholder: "john.doe@example.com",
                        },
                    },
                    {
                        id: "retryEnabled",
                        type: InputType.Switch,
                        label: "Enable Retry",
                        fieldName: "retry",
                        description: "Enable retrying failed requests",
                        props: {
                            defaultValue: false,
                            color: "primary",
                        },
                    },
                    {
                        id: "timeout",
                        type: InputType.IntegerInput,
                        label: "Timeout (ms)",
                        fieldName: "timeoutMs",
                        description: "Request timeout in milliseconds",
                        props: {
                            defaultValue: 10000,
                            min: 1000,
                            max: 60000,
                            step: 1000,
                        },
                    },
                ],
                containers: [
                    {
                        title: "API Configuration",
                        description: "Configure the API request settings",
                        totalItems: 4,
                    },
                ],
            },
        },
        formOutput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "statusCode",
                        type: InputType.IntegerInput,
                        label: "Status Code",
                        fieldName: "statusCode",
                        description: "HTTP status code from the response",
                        props: {
                            defaultValue: 200,
                            min: 100,
                            max: 599,
                        },
                    },
                    {
                        id: "responseBody",
                        type: InputType.Text,
                        label: "Response Body",
                        fieldName: "body",
                        description: "Response body content",
                        props: {
                            isMarkdown: true,
                            minRows: 4,
                            maxRows: 10,
                            placeholder: "Response content will appear here",
                        },
                    },
                    {
                        id: "responseHeaders",
                        type: InputType.JSON,
                        label: "Response Headers",
                        fieldName: "headers",
                        description: "HTTP headers from the response",
                        props: {
                            defaultValue: "{}",
                        },
                    },
                    {
                        id: "error",
                        type: InputType.Text,
                        label: "Error",
                        fieldName: "error",
                        description: "Error message if request failed",
                        props: {
                            defaultValue: "",
                            placeholder: "Any error messages will appear here",
                        },
                    },
                ],
                containers: [
                    {
                        title: "API Response",
                        description: "Results from the API request",
                        totalItems: 4,
                    },
                ],
            },
        },
    }),
};

const dataConverterFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        callDataCode: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                // Example function that converts data:
                // function convertData(data: {
                //   input: string | object,
                //   targetFormat: "json" | "csv" | "xml"  
                // }) {
                //   // Convert logic here
                //   return {
                //     converted: string,
                //     format: string
                //   }
                // }
                inputTemplate: [
                    "{{input.routineInputA}}",
                    "{{input.routineInputB}}",
                ],
                outputMappings: [
                    {
                        schemaIndex: 0,
                        mapping: {
                            routineOutputX: "converted",
                            routineOutputY: "format",
                        },
                    },
                ],
            },
        },
        formInput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "input",
                        type: InputType.Text,
                        label: "Input Data",
                        fieldName: "input",
                        description: "Data to be converted (as JSON or string)",
                        props: {
                            defaultValue: "{}",
                        },
                    },
                    {
                        id: "format",
                        type: InputType.Selector,
                        label: "Target Format",
                        fieldName: "format",
                        description: "Select output format",
                        props: {
                            options: [
                                { label: "JSON", value: "json" },
                                { label: "CSV", value: "csv" },
                                { label: "XML", value: "xml" },
                            ],
                            defaultValue: "json",
                        },
                    },
                ],
                containers: [
                    {
                        title: "Data Conversion Input",
                        description: "Enter data and select output format",
                        totalItems: 2,
                    },
                ],
            },
        },
        formOutput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "convertedData",
                        type: InputType.Text,
                        label: "Converted Data",
                        fieldName: "convertedData",
                        description: "Data in converted format",
                        props: {
                            defaultValue: "",
                        },
                    },
                    {
                        id: "outputFormat",
                        type: InputType.Text,
                        label: "Output Format",
                        fieldName: "outputFormat",
                        description: "Format of converted data",
                        props: {
                            defaultValue: "",
                        },
                    },
                ],
                containers: [
                    {
                        title: "Conversion Result",
                        description: "Results of the data conversion",
                        totalItems: 2,
                    },
                ],
            },
        },
    }),
};

const dataFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        formOutput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "sampleText",
                        type: InputType.Text,
                        label: "Sample Text",
                        fieldName: "sampleText",
                        description: "Example text output",
                        props: {
                            defaultValue: "This is sample output text",
                            isMarkdown: true,
                            minRows: 2,
                            maxRows: 4,
                        },
                    },
                    {
                        id: "sampleNumber",
                        type: InputType.IntegerInput,
                        label: "Sample Number",
                        fieldName: "sampleNumber",
                        description: "Example numeric output",
                        props: {
                            defaultValue: 42,
                            min: 0,
                            max: 100,
                        },
                    },
                    {
                        id: "sampleJson",
                        type: InputType.JSON,
                        label: "Sample JSON",
                        fieldName: "sampleJson",
                        description: "Example JSON output",
                        props: {
                            defaultValue: `{
  "key1": "value1",
  "key2": 123,
  "key3": true
}`,
                        },
                    },
                ],
                containers: [
                    {
                        title: "Sample Output",
                        description: "Example form output data",
                        totalItems: 3,
                    },
                ],
            },
        },
    }),
};

const generateFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        callDataGenerate: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                prompt: "Write a blog post about {{input.topic}} in {{input.style}} style",
                model: null,
                botStyle: BotStyle.Default,
                maxTokens: 1000,
            },
        },
        formInput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "topic",
                        type: InputType.Text,
                        label: "Topic",
                        fieldName: "topic",
                        description: "What would you like the AI to write about?",
                        props: {
                            defaultValue: "",
                            placeholder: "artificial intelligence",
                            minRows: 1,
                            maxRows: 3,
                        },
                    },
                    {
                        id: "style",
                        type: InputType.Selector,
                        label: "Writing Style",
                        fieldName: "style",
                        description: "Select the writing style",
                        props: {
                            options: [
                                { label: "Professional", value: "professional" },
                                { label: "Casual", value: "casual" },
                                { label: "Academic", value: "academic" },
                                { label: "Creative", value: "creative" },
                            ],
                            defaultValue: "professional",
                            getOptionLabel: (option) => option.label,
                            getOptionValue: (option) => option.value,
                            getOptionDescription: () => "",
                        },
                    },
                ],
                containers: [
                    {
                        title: "Generation Settings",
                        description: "Configure how the AI generates content",
                        totalItems: 2,
                    },
                ],
            },
        },
        formOutput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "response",
                        type: InputType.Text,
                        label: "Generated Content",
                        fieldName: "response",
                        description: "AI-generated content based on your inputs",
                        props: {
                            isMarkdown: true,
                            minRows: 4,
                            maxRows: 20,
                            placeholder: "Generated content will appear here",
                        },
                    },
                ],
                containers: [
                    {
                        title: "Generated Output",
                        description: "Results from the AI generation",
                        totalItems: 1,
                    },
                ],
            },
        },
    }),
};

const informationalFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        formInput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "title",
                        type: InputType.Text,
                        label: "Title",
                        fieldName: "title",
                        description: "Title of the information",
                        props: {
                            defaultValue: "System Requirements & Prerequisites",
                        },
                    },
                    {
                        id: "content",
                        type: InputType.Text,
                        label: "Content",
                        fieldName: "content",
                        description: "Main content text",
                        props: {
                            defaultValue: "# Requirements\n\n- Node.js v16 or higher\n- NPM v7 or higher\n- 4GB RAM minimum\n- 10GB free disk space\n\n# Setup Steps\n\n1. Install dependencies\n2. Configure environment\n3. Run initialization script",
                            isMarkdown: true,
                            minRows: 4,
                            maxRows: 20,
                        },
                    },
                    {
                        id: "tags",
                        type: InputType.JSON,
                        label: "Tags",
                        fieldName: "tags",
                        description: "Categorization tags (as JSON array)",
                        props: {
                            defaultValue: "[\"requirements\", \"setup\", \"technical\"]",
                        },
                    },
                ],
                containers: [
                    {
                        title: "Information Content",
                        description: "Enter the information to be displayed",
                        totalItems: 3,
                    },
                ],
            },
        },
    }),
};

const smartContractFormWithDataParams = {
    ...mockFormProps,
    config: new RoutineVersionConfig({
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        callDataSmartContract: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                contractAddress: "0x1234567890abcdef1234567890abcdef12345678",
                chainId: 1,
                abi: [
                    {
                        "inputs": [
                            {
                                "name": "tokenId",
                                "type": "uint256",
                            },
                        ],
                        "name": "ownerOf",
                        "outputs": [
                            {
                                "name": "owner",
                                "type": "address",
                            },
                        ],
                        "stateMutability": "view",
                        "type": "function",
                    },
                    {
                        "inputs": [
                            {
                                "name": "to",
                                "type": "address",
                            },
                            {
                                "name": "tokenId",
                                "type": "uint256",
                            },
                        ],
                        "name": "transfer",
                        "outputs": [],
                        "stateMutability": "nonpayable",
                        "type": "function",
                    },
                ],
                functionName: "ownerOf",
                functionArgs: ["{{input.tokenId}}"],
                value: "0",
                gasLimit: "100000",
            },
        },
        formInput: {
            __version: LATEST_ROUTINE_CONFIG_VERSION,
            schema: {
                elements: [
                    {
                        id: "tokenId",
                        type: InputType.IntegerInput,
                        label: "Token ID",
                        fieldName: "tokenId",
                        description: "Enter the NFT token ID to check ownership",
                        props: {
                            defaultValue: 1,
                            min: 1,
                            label: "Enter token ID",
                        },
                    },
                    {
                        id: "network",
                        type: InputType.Selector,
                        label: "Network",
                        fieldName: "network",
                        description: "Blockchain network",
                        props: {
                            options: [
                                { label: "Ethereum Mainnet", value: "1" },
                                { label: "Goerli Testnet", value: "5" },
                                { label: "Sepolia Testnet", value: "11155111" },
                            ],
                            defaultValue: "1",
                            getOptionLabel: (option) => option.label,
                            getOptionValue: (option) => option.value,
                        },
                    },
                ],
                containers: [
                    {
                        title: "Smart Contract Parameters",
                        description: "Configure the parameters for the smart contract call",
                        totalItems: 3,
                    },
                ],
            },
        },
    }),
};

const outerStyle = {
    width: "min(500px, 100%)",
    padding: "20px",
    border: "1px solid #ccc",
    paddingBottom: "100px",
} as const;
function Outer({ children }: { children: React.ReactNode }) {
    return (
        <PageContainer>
            <ScrollBox>
                <Box sx={outerStyle}>
                    {children}
                </Box>
            </ScrollBox>
        </PageContainer>
    );
}

export function ApiFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.Api,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineApiForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
ApiFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the API form before any data has been entered.",
        },
    },
};
export function ApiFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(apiFormWithDataParams.config),
            routineType: RoutineType.Api,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineApiForm {...apiFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
ApiFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the API form after data has been entered.",
        },
    },
};
export function ApiFormEdit() {
    return (
        <Outer>
            <RoutineApiForm {...apiFormWithDataParams} display="edit" />
        </Outer>
    );
}
ApiFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the API form in edit mode.",
        },
    },
};

export function DataConverterFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.Code,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineDataConverterForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
DataConverterFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the data converter form before any data has been entered.",
        },
    },
};
export function DataConverterFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(dataConverterFormWithDataParams.config),
            routineType: RoutineType.Code,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineDataConverterForm {...dataConverterFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
DataConverterFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the data converter form after data has been entered.",
        },
    },
};
export function DataConverterFormEdit() {
    return (
        <Outer>
            <RoutineDataConverterForm {...dataConverterFormWithDataParams} display="edit" />
        </Outer>
    );
}
DataConverterFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the data converter form in edit mode.",
        },
    },
};

export function DataFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.Data,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineDataForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
DataFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the default data form.",
        },
    },
};
export function DataFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(dataFormWithDataParams.config),
            routineType: RoutineType.Data,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineDataForm {...dataFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
DataFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the data form after data has been entered.",
        },
    },
};
export function DataFormEdit() {
    return (
        <Outer>
            <RoutineDataForm {...dataFormWithDataParams} display="edit" />
        </Outer>
    );
}
DataFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the data form in edit mode.",
        },
    },
};

export function GenerateFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.Generate,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineGenerateForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
GenerateFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the generate form before any data has been entered.",
        },
    },
};
export function GenerateFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(generateFormWithDataParams.config),
            routineType: RoutineType.Generate,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineGenerateForm {...generateFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
GenerateFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the generate form after data has been entered.",
        },
    },
};
export function GenerateFormEdit() {
    return (
        <Outer>
            <RoutineGenerateForm {...generateFormWithDataParams} display="edit" />
        </Outer>
    );
}
GenerateFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the generate form in edit mode.",
        },
    },
};

export function InformationalFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.Informational,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineInformationalForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
InformationalFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the informational form before any data has been entered.",
        },
    },
};
export function InformationalFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(informationalFormWithDataParams.config),
            routineType: RoutineType.Informational,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineInformationalForm {...informationalFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
InformationalFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the informational form after data has been entered.",
        },
    },
};
export function InformationalFormEdit() {
    return (
        <Outer>
            <RoutineInformationalForm {...informationalFormWithDataParams} display="edit" />
        </Outer>
    );
}
InformationalFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the informational form in edit mode.",
        },
    },
};

export function SmartContractFormEmpty() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(emptyFormParams.config),
            routineType: RoutineType.SmartContract,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineSmartContractForm {...emptyFormParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
SmartContractFormEmpty.parameters = {
    docs: {
        description: {
            story: "Displays the smart contract form before any data has been entered.",
        },
    },
};
export function SmartContractFormWithData() {
    const routine = useMemo(function routineMemo() {
        return {
            config: JSON.stringify(smartContractFormWithDataParams.config),
            routineType: RoutineType.SmartContract,
        };
    }, []);

    const { initialValues, validationSchema } = useMemo(function initialValuesMemo() {
        const config = RoutineVersionConfig.parse(routine, console);
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, routine.routineType);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, routine.routineType);
        return { initialValues, validationSchema };
    }, [routine]);

    return (
        <Outer>
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
                validationSchema={validationSchema}
            >
                {() => (
                    <RoutineSmartContractForm {...smartContractFormWithDataParams} display="view" />
                )}
            </Formik>
        </Outer>
    );
}
SmartContractFormWithData.parameters = {
    docs: {
        description: {
            story: "Displays the smart contract form after data has been entered.",
        },
    },
};
export function SmartContractFormEdit() {
    return (
        <Outer>
            <RoutineSmartContractForm {...smartContractFormWithDataParams} display="edit" />
        </Outer>
    );
}
SmartContractFormEdit.parameters = {
    docs: {
        description: {
            story: "Displays the smart contract form in edit mode.",
        },
    },
};
