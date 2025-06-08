import { CodeLanguage, InputType, RoutineVersionConfig, type FormInputType, type PassableLogger, type ResourceVersion, type StandardVersionConfigObject } from "@vrooli/shared";
import { RelatedResourceLabel, RelatedResourceUtils, type RelatedVersionLink } from "@vrooli/shared/utils";

/**
 * Internal representation of a routine's input configuration,
 * combining form definition with an optional linked standard.
 * Migrated from executor.ts
 */
interface InternalRoutineInputConfig {
    name: string;
    isRequired: boolean;
    description?: string;
    formDefinition: FormInputType;
    standardVersion?: {
        codeLanguage?: CodeLanguage | string; // string for custom, CodeLanguage for known
        props: Record<string, unknown>;
    };
}

/**
 * Internal representation of a routine's output configuration.
 * Migrated from executor.ts
 */
interface InternalRoutineOutputConfig {
    name: string;
    description?: string;
    formDefinition: FormInputType;
    standardVersion?: {
        codeLanguage?: CodeLanguage | string;
        props: Record<string, unknown>;
    };
}

/**
 * Display information for a subroutine input/output.
 * Temporary type until proper types are imported.
 */
interface SubroutineIODisplayInfo {
    defaultValue: unknown | undefined;
    description: string | undefined;
    name: string | undefined;
    props: SubroutineOutputDisplayInfoProps | undefined;
    value: unknown | undefined;
}

interface SubroutineInputDisplayInfo extends SubroutineIODisplayInfo {
    isRequired: boolean;
}

// Use the base type directly since the output type has no additional properties
type SubroutineOutputDisplayInfo = SubroutineIODisplayInfo;

interface SubroutineOutputDisplayInfoProps {
    type: string;
    schema: string | undefined;
    [key: string]: unknown; // Allow additional properties
}

/**
 * A mapping of input and output names to their current value and display information.
 */
interface SubroutineIOMapping {
    inputs: Record<string, SubroutineInputDisplayInfo>;
    outputs: Record<string, SubroutineOutputDisplayInfo>;
}

/**
 * I/O Processing component extracted from legacy SubroutineExecutor.
 * Handles complex form analysis, standards integration, and input validation.
 */
export class IOProcessor {
    /**
     * Migrated from SubroutineExecutor.buildIOMapping()
     * Complex form analysis and standards integration
     */
    async buildIOMapping(
        routine: ResourceVersion,
        providedInputs: Record<string, unknown>,
        logger: PassableLogger,
    ): Promise<SubroutineIOMapping> {
        // Initialize the mapping
        const mapping: SubroutineIOMapping = { inputs: {}, outputs: {} };
        const parsedRoutine = RoutineVersionConfig.parse(routine, logger, { useFallbacks: true });
        const { formInput, formOutput } = parsedRoutine;
        const formInputSchema = formInput?.schema;
        const formOutputSchema = formOutput?.schema;

        const inputsConfig: InternalRoutineInputConfig[] = [];
        if (formInputSchema?.elements) {
            for (const element of formInputSchema.elements) {
                if ("fieldName" in element && element.fieldName) {
                    // element is FormInputType. It should have fieldName, label, and props.
                    const specificProps = element.props as { isRequired?: boolean; placeholder?: string;[key: string]: any };
                    let standardVersionDef: InternalRoutineInputConfig["standardVersion"] = undefined;

                    if (routine.relatedVersions) {
                        for (const link of routine.relatedVersions) {
                            const adaptedLink: RelatedVersionLink = {
                                targetVersionId: link.toVersion.id,
                                labels: link.labels,
                                targetVersionObject: link.toVersion,
                            };
                            const fieldNameFromLink = RelatedResourceUtils.getFieldIdentifierFromLink(adaptedLink, RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD);
                            if (fieldNameFromLink === element.fieldName) {
                                const relatedStandard = link.toVersion;
                                if (relatedStandard) {
                                    const lang = relatedStandard.codeLanguage;
                                    const standardConfig = relatedStandard.config as StandardVersionConfigObject | undefined;
                                    standardVersionDef = {
                                        codeLanguage: lang === null ? undefined : lang,
                                        props: standardConfig?.props ?? {},
                                    };
                                }
                                break;
                            }
                        }
                    }
                    inputsConfig.push({
                        name: element.fieldName,
                        isRequired: !!(specificProps?.isRequired || (element as { isRequired?: boolean }).isRequired),
                        description: element.label ?? specificProps?.placeholder,
                        formDefinition: element, // element is already FormInputType
                        standardVersion: standardVersionDef,
                    });
                }
            }
        }

        const outputsConfig: InternalRoutineOutputConfig[] = [];
        if (formOutputSchema?.elements) {
            for (const element of formOutputSchema.elements) {
                if ("fieldName" in element && element.fieldName) {
                    // element is FormInputType. It should have fieldName, label, and props.
                    const specificProps = element.props as { placeholder?: string;[key: string]: any };
                    let standardVersionDef: InternalRoutineOutputConfig["standardVersion"] = undefined;

                    if (routine.relatedVersions) {
                        for (const link of routine.relatedVersions) {
                            const adaptedLink: RelatedVersionLink = {
                                targetVersionId: link.toVersion.id,
                                labels: link.labels,
                                targetVersionObject: link.toVersion,
                            };
                            const fieldNameFromLink = RelatedResourceUtils.getFieldIdentifierFromLink(adaptedLink, RelatedResourceLabel.DEFINES_STANDARD_FOR_OUTPUT_FIELD);
                            if (fieldNameFromLink === element.fieldName) {
                                const relatedStandard = link.toVersion;
                                if (relatedStandard) {
                                    const lang = relatedStandard.codeLanguage;
                                    const standardConfig = relatedStandard.config as StandardVersionConfigObject | undefined;
                                    standardVersionDef = {
                                        codeLanguage: lang === null ? undefined : lang,
                                        props: standardConfig?.props ?? {},
                                    };
                                }
                                break;
                            }
                        }
                    }
                    outputsConfig.push({
                        name: element.fieldName,
                        description: element.label ?? specificProps?.placeholder,
                        formDefinition: element, // element is already FormInputType
                        standardVersion: standardVersionDef,
                    });
                }
            }
        }

        // Process inputs
        inputsConfig.forEach((input) => {
            if (!input.name) {
                return;
            }

            const formElement = input.formDefinition;
            const { defaultValue, props } = this.findElementStructure(formElement, input);

            const description = input.description ?? formElement?.description ?? undefined;
            const isRequired = input.isRequired ?? false;
            const name = input.name;
            const value = providedInputs[name];

            mapping.inputs[name] = {
                defaultValue,
                description,
                isRequired,
                name,
                props,
                value,
            };
        });

        // Process outputs
        outputsConfig.forEach((output) => {
            if (!output.name) {
                return;
            }

            const formElement = output.formDefinition;
            const { defaultValue, props } = this.findElementStructure(formElement, output);

            const description = output.description ?? formElement?.description ?? undefined;
            const name = output.name;
            const value = providedInputs[name];

            mapping.outputs[name] = {
                defaultValue,
                description,
                name,
                props,
                value,
            };
        });

        return mapping;
    }

    /**
     * Migrated from SubroutineExecutor.findElementStructure()
     * Convert form elements to LLM-understandable structures
     */
    findElementStructure(
        formElement: FormInputType | undefined,
        io: InternalRoutineInputConfig | InternalRoutineOutputConfig,
    ): { defaultValue: unknown | undefined, props: SubroutineOutputDisplayInfoProps } {
        const result: { defaultValue: unknown | undefined, props: SubroutineOutputDisplayInfoProps } = {
            defaultValue: undefined,
            props: { type: "Text", schema: undefined },
        };

        // There are three cases:
        // 1. The io has a standard version attached, which defines a specific schema to adhere to
        // 2. The formElement exists, which can be used to infer the type and schema based on the element's type and properties
        // 3. Neither are present, so we fallback to plain text
        //
        // Case 1
        if (io.standardVersion) {
            const { codeLanguage, props } = io.standardVersion;
            result.props.type = codeLanguage === CodeLanguage.Json ? "JSON schema"
                : codeLanguage === CodeLanguage.Yaml ? "YAML schema"
                    : typeof codeLanguage === "string" && codeLanguage ? codeLanguage // Use custom string if provided
                        : "Unknown"; // Fallback for unknown or undefined codeLanguage
            result.props.schema = typeof props === "object" && props !== null ? JSON.stringify(props) : String(props);
        }
        // Case 2
        else if (formElement) {
            switch (formElement.type) {
                case InputType.Text: {
                    result.props.type = "Text";
                    const { defaultValue, maxChars } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxChars !== undefined) {
                        result.props.maxLength = maxChars;
                    }
                    break;
                }
                case InputType.Switch: {
                    result.props.type = "Boolean";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                case InputType.IntegerInput: {
                    result.props.type = "Number";
                    const { allowDecimal, defaultValue, max, min } = formElement.props;
                    if (allowDecimal === false) {
                        result.props.type = "Integer";
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (max !== undefined) {
                        result.props.max = max;
                    }
                    if (min !== undefined) {
                        result.props.min = min;
                    }
                    break;
                }
                case InputType.Slider: {
                    result.props.type = "Number";
                    const { defaultValue, max, min } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (max !== undefined) {
                        result.props.max = max;
                    }
                    if (min !== undefined) {
                        result.props.min = min;
                    }
                    break;
                }
                case InputType.Checkbox: {
                    result.props.type = "List of values from the provided options";
                    const { allowCustomValues, defaultValue, options, maxCustomValues, maxSelection, minSelection } = formElement.props;
                    if (allowCustomValues === true) {
                        const supportsMultipleCustomValues = maxCustomValues !== undefined && maxCustomValues > 1;
                        result.props.type = supportsMultipleCustomValues
                            ? "List of values, using the provided options and custom values"
                            : "List of values from the provided options, with up to one custom value";
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxCustomValues !== undefined) {
                        result.props.maxCustomValues = maxCustomValues;
                    }
                    if (maxSelection !== undefined) {
                        result.props.maxSelection = maxSelection;
                    }
                    if (minSelection !== undefined) {
                        result.props.minSelection = minSelection;
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.Radio: {
                    result.props.type = "One of the values from the provided options";
                    const { defaultValue, options } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.Selector: {
                    result.props.type = "One of the values from the provided options";
                    const { defaultValue, multiple, options, noneOption } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (multiple === true) {
                        result.props.type = "List of values from the provided options";
                    }
                    if (noneOption === true) {
                        result.props.type = "One of the values from the provided options, or null";
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.LanguageInput: {
                    result.props.type = "List of language codes";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                // This is a special case, as we'll be finding the link later using search 
                // instead of generating it with an LLM. But we still want to grab the data 
                // required to perform the search.
                case InputType.LinkItem: {
                    result.props.type = "Item ID (e.g. RoutineVersion:123-456-789)";
                    const { defaultValue, limitTo } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (limitTo) {
                        result.props.limitTo = limitTo;
                    }
                    break;
                }
                case InputType.LinkUrl: {
                    result.props.type = "URL";
                    const { acceptedHosts, defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (acceptedHosts) {
                        result.props.acceptedHosts = acceptedHosts;
                    }
                    break;
                }
                case InputType.TagSelector: {
                    result.props.type = "List of alphanumeric tags (a.k.a. hashtags, keywords)";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                case InputType.JSON: {
                    result.props.type = "JSON object"; // Defaults to JSON unless otherwise specified
                    const { defaultValue, limitTo } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (limitTo && Array.isArray(limitTo) && limitTo.length > 0) {
                        result.props.type = "Code in one of these languages: " + limitTo.join(", ");
                    }
                    break;
                }
                // Like LinkItem, we can't use generation for Dropzone. We'll have to support this with search instead.
                case InputType.Dropzone: {
                    result.props.type = "File";
                    const { acceptedFileTypes, defaultValue, maxFiles } = formElement.props;
                    if (acceptedFileTypes) {
                        result.props.acceptedFileTypes = acceptedFileTypes;
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxFiles !== undefined && maxFiles > 1) {
                        result.props.type = "Files";
                        result.props.maxFiles = maxFiles;
                    }
                    break;
                }
                default: {
                    // Add error logging here if needed
                    break;
                }
            }
        }
        return result;
    }

    /**
     * Find missing required inputs from an I/O mapping
     */
    findMissingRequiredInputs(ioMapping: SubroutineIOMapping): string[] {
        return Object.entries(ioMapping.inputs)
            .filter(([, input]) => input.isRequired && input.value === undefined)
            .map(([name]) => name);
    }
} 
