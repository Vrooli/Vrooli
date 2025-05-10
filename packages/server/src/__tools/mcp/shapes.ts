// packages/server/src/services/mcp/mcp_io_shapes.ts
import { CallDataApiConfigObject, CallDataCodeConfigObject, CallDataGenerateConfigObject, CallDataSmartContractConfigObject, CheckboxFormInputProps, CodeFormInputProps, DropzoneFormInputProps, GraphConfigObject, IntegerFormInputProps, LATEST_CONFIG_VERSION, LinkUrlFormInputProps, RadioFormInputProps, ResourceSubType, ResourceType, ResourceVersionSortBy, SelectorFormInputProps, SliderFormInputProps, SwitchFormInputProps, TeamConfigObject, TeamSortBy, TextFormInputProps, VisibilityType } from '@local/shared';

interface CommonFindFilters {
    /** Filter by specific IDs. */
    ids?: string[];
    /** Filter by visibility. */
    visibility?: VisibilityType;
    /** Number of items to return. */
    take?: number;
    /** Cursor for pagination. */
    after?: string;
}

interface CommonAttributes {
    /**
     * @default false
     */
    isPrivate: boolean;
    /**
     * Optional handle. Must be unique if provided.
     * @minLength 3
     * @maxLength 16
     * @pattern ^[a-zA-Z0-9_-]+$
     */
    handle?: string;
    /**
     * Tags to associate with the resource.
     */
    tagsConnect?: string[];
    /**
     * Tag IDs to remove from the resource.
     */
    tagsDisconnect?: string[];
}

type CheckboxInputProps = Omit<CheckboxFormInputProps, "color" | "row">;
type CodeInputProps = Omit<CodeFormInputProps, "disabled">;
type DropzoneInputProps = Omit<DropzoneFormInputProps, "cancelText" | "defaultValue" | "disabled" | "dropzoneText" | "showThumbs" | "uploadText">;
type IntegerInputProps = Omit<IntegerFormInputProps, "autoFocus" | "disabled" | "error" | "fullWidth" | "helperText" | "key" | "label" | "tooltip" | "zeroText">;
// type LinkItemInputProps = Omit<LinkItemFormInputProps, "placeholder">;
type LinkUrlInputProps = Omit<LinkUrlFormInputProps, "placeholder">;
type RadioInputProps = Omit<RadioFormInputProps, "row">;
type SelectorInputProps = Omit<SelectorFormInputProps<any>, "autoFocus" | "disabled" | "fullWidth" | "getOptionDescription" | "getOptionLabel" | "getOptionValue" | "inputAriaLabel" | "noneText" | "tabIndex">;
type SliderInputProps = Omit<SliderFormInputProps, "valueLabelDisplay">;
type SwitchInputProps = Omit<SwitchFormInputProps, "color" | "label" | "size">;
type TextInputProps = Omit<TextFormInputProps, "maxRows" | "minRows">;
type FormInputType = {
    type: "CheckboxInput" | "CodeInput" | "DropzoneInput" | "IntegerInput" | "LinkUrlInput" | "RadioInput" | "SelectorInput" | "SliderInput" | "SwitchInput" | "TextInput";
    /** ID for field */
    id: string;
    /** The name of the field, as will be used by formik */
    fieldName: string;
    /** The label to display for the field */
    label: string;
    /** Longer help text. Markdown OK */
    helpText?: string;
    /** If the field is required. Defaults to false. */
    isRequired?: boolean;
    /** Type-specific props */
    props: CheckboxInputProps | CodeInputProps | DropzoneInputProps | IntegerInputProps | LinkUrlInputProps | RadioInputProps | SelectorInputProps | SliderInputProps | SwitchInputProps | TextInputProps;
}
type FormInformationalType = {
    type: "Divider" | "Header" | "Image" | "QrCode" | "Tip" | "Video",
    /** ID for field */
    id: string;
    /** The label to display for the field */
    label: string;
    /** Longer help text. Markdown OK */
    helpText?: string;
    /** Header tag if type is Header */
    tag?: "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
    /** Icon type if type is Tip */
    icon?: "Error" | "Info" | "Warning";
    /** URL if type is Tip */
    link?: string;
    /** URL of type is Image, QrCode, or Video */
    url?: string;
}
type FormElement = FormInputType | FormInformationalType;
type SimpleFormSchema = {
    __version: typeof LATEST_CONFIG_VERSION;
    schema: {
        elements: FormElement[];
    };
}
type FormInput = SimpleFormSchema;
type FormOutput = SimpleFormSchema;

interface ResourceAddBase extends Pick<CommonAttributes, 'isPrivate'> {
    /**
     * @default false
     */
    isComplete?: boolean;
    /**
     * @example "1.0.0"
     */
    versionLabel: string;
}

interface ResourceUpdateBase extends Pick<ResourceAddBase, "isComplete" | 'isPrivate' | "versionLabel"> {
    // Add fields as needed
}

interface ResourceRootAddBase extends Pick<CommonAttributes, 'isPrivate' | 'tagsConnect'> {
    // Add fields as needed
}

interface ResourceRootUpdateBase extends Pick<CommonAttributes, "isPrivate" | "tagsConnect" | "tagsDisconnect"> {
    // Add fields as needed
}

/**
 * Represents a single member invitation item for MCP operations.
 * @title MCP Member Invite Item
 */
interface MemberInviteItem {
    /** User ID of the member to invite. */
    userConnect: string;
    /** Optional invitation message. */
    message?: string;
    /** Grant admin privileges to the invited member upon joining. */
    willBeAdmin?: boolean;
}

// type RoutineMultiStepConfig = Pick<RoutineVersionConfigObject, "graph" | "formInput" | "formOutput">;
type RoutineMultiStepConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    graph: GraphConfigObject;
    formInput: FormInput;
    formOutput: FormOutput;
}
/**
 * Attributes for adding a `RoutineMultiStep` resource via MCP.
 * @title MCP Routine Multi‑Step Add Attributes
 * @description Defines the minimal fields required to create a new multi‑step routine. The server
 *              supplies ID and other server‑controlled data automatically.
 */
export interface RoutineMultiStepAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineMultiStepConfig;
    resourceSubType: ResourceSubType.RoutineMultiStep;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineMultiStep` resource via MCP.
 * @title MCP Routine Multi‑Step Update Attributes
 * @description Defines the properties for updating a multi‑step routine through the MCP.
 */
export interface RoutineMultiStepUpdateAttributes extends ResourceUpdateBase {
    /** 
     * Updated configuration object for the team. 
     * This must be the FULL config object, not a partial update.
     */
    config?: RoutineMultiStepConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineApiConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    formInput: FormInput;
    formOutput: FormOutput;
    /** Connection details for the target API */
    callDataApi: CallDataApiConfigObject;
};
/**
 * Attributes for adding a `RoutineApi` resource via MCP.
 * @title MCP Routine API Add Attributes
 * @description Defines the properties for adding a new API routine through the MCP.
 */
export interface RoutineApiAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineApiConfig;
    resourceSubType: ResourceSubType.RoutineApi;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineApi` resource via MCP.
 * @title MCP Routine API Update Attributes
 * @description Defines the properties for updating an API routine through the MCP.
 */
export interface RoutineApiUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineApiConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineCodeConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    formInput: FormInput;
    formOutput: FormOutput;
    /** Linked code object used for conversion*/
    callDataCode: CallDataCodeConfigObject;
};
/**
 * Attributes for adding a `RoutineCode` resource via MCP.
 * @title MCP Routine Code Add Attributes
 * @description Defines the properties for adding a new Code routine through the MCP.
 */
export interface RoutineCodeAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineCodeConfig;
    resourceSubType: ResourceSubType.RoutineCode;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineCode` resource via MCP.
 * @title MCP Routine Code Update Attributes
 * @description Defines the properties for updating a Code routine through the MCP.
 */
export interface RoutineCodeUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineCodeConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineDataConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    /** Pure-data routines only expose outputs. */
    formOutput: FormOutput;
};
/**
 * Attributes for adding a `RoutineData` resource via MCP.
 * @title MCP Routine Data Add Attributes
 * @description Defines the properties for adding a new Data routine through the MCP.
 */
export interface RoutineDataAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineDataConfig;
    resourceSubType: ResourceSubType.RoutineData;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineData` resource via MCP.
 * @title MCP Routine Data Update Attributes
 * @description Defines the properties for updating a Data routine through the MCP.
 */
export interface RoutineDataUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineDataConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineGenerateConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    formInput: FormInput;
    formOutput: FormOutput;
    /** Holds model, bot style, etc. */
    callDataGenerate: CallDataGenerateConfigObject;
};
/**
 * Attributes for adding a `RoutineGenerate` resource via MCP.
 * @title MCP Routine Generate Add Attributes
 * @description Defines the properties for adding a new Generate routine through the MCP.
 */
export interface RoutineGenerateAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineGenerateConfig;
    resourceSubType: ResourceSubType.RoutineGenerate;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineGenerate` resource via MCP.
 * @title MCP Routine Generate Update Attributes
 * @description Defines the properties for updating a Generate routine through the MCP.
 */
export interface RoutineGenerateUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineGenerateConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineInformationalConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    /** This routine type only collects information, using the formInput */
    formInput: FormInput;
};
/**
 * Attributes for adding a `RoutineInformational` resource via MCP.
 * @title MCP Routine Informational Add Attributes
 * @description Defines the properties for adding a new Informational routine through the MCP.
 */
export interface RoutineInformationalAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineInformationalConfig;
    resourceSubType: ResourceSubType.RoutineInformational;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineInformational` resource via MCP.
 * @title MCP Routine Informational Update Attributes
 * @description Defines the properties for updating an Informational routine through the MCP.
 */
export interface RoutineInformationalUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineInformationalConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

type RoutineSmartContractConfig = {
    __version: typeof LATEST_CONFIG_VERSION;
    formInput: FormInput;
    formOutput: FormOutput;
    /** Contract ABI / network / method info */
    callDataSmartContract: CallDataSmartContractConfigObject;
};
/**
 * Attributes for adding a `RoutineSmartContract` resource via MCP.
 * @title MCP Routine Smart Contract Add Attributes
 * @description Defines the properties for adding a new Smart Contract routine through the MCP.
 */
export interface RoutineSmartContractAddAttributes extends ResourceAddBase {
    /**
     * Optional routine configuration object.
     */
    config?: RoutineSmartContractConfig;
    resourceSubType: ResourceSubType.RoutineSmartContract;
    /**
     * Root resource information
     */
    rootCreate: ResourceRootAddBase & {
        resourceType: ResourceType.Routine;
    }
}
/**
 * Attributes for updating a `RoutineSmartContract` resource via MCP.
 * @title MCP Routine Smart Contract Update Attributes
 * @description Defines the properties for updating a Smart Contract routine through the MCP.
 */
export interface RoutineSmartContractUpdateAttributes extends ResourceUpdateBase {
    /**
     * Updated configuration object for the routine.
     */
    config?: RoutineSmartContractConfig;
    /**
     * Root resource information
     */
    rootUpdate: ResourceRootUpdateBase & {
        resourceType: ResourceType.Routine;
    }
}

/**
 * Filters for finding resources via MCP.
 * @title MCP Resource Find Filters
 */
export interface ResourceFindFilters extends CommonFindFilters {
    /** If this is the latest public version. */
    isLatest?: boolean;
    /** ID of the team that owns the root resource. */
    ownedByTeamIdRoot?: string[];
    /** ID of the user that owns the root resource. */
    ownedByUserdRoot?: string;
    /** ID of the root resource. */
    rootId?: string;
    /** Filter teams by associated tags. */
    tags?: string[];
    /** Sort order for the results. */
    sortBy?: ResourceVersionSortBy;
}

/**
 * Filters for finding 'Code' resources via MCP.
 * @title MCP Code Find Filters
 */
export interface CodeFindFilters extends ResourceFindFilters {
    /** Language of the code. */
    codeLanguage?: string;
}

/**
 * Attributes for adding a 'Team' resource via MCP.
 * @title MCP Team Add Attributes
 * @description Defines the properties for creating a new team through the MCP.
 *              The 'id' is typically server-generated and not part of the add attributes.
 */
export interface TeamAddAttributes extends Pick<CommonAttributes, "handle" | "isPrivate" | "tagsConnect"> {
    /**
     * Optional team configuration object.
     */
    config?: TeamConfigObject;
    /**
     * Optional list of members to invite initially.
     * User IDs are expected.
     */
    memberInvitesCreate?: MemberInviteItem[];
}

/**
 * Attributes for updating a 'Team' resource via MCP.
 * All properties are optional for an update.
 * @title MCP Team Update Attributes
 */
export interface TeamUpdateAttributes extends Pick<CommonAttributes, "handle" | "isPrivate" | "tagsConnect" | "tagsDisconnect"> {
    /** 
     * Updated configuration object for the team. 
     * This must be the FULL config object, not a partial update.
     */
    config?: TeamConfigObject;
    /**
     * Optional list of new members to invite.
     * User IDs are expected.
     */
    memberInvitesCreate?: MemberInviteItem[];
    /**
     * Optional list of member invite IDs to delete.
     */
    memberInvitesDelete?: string[];
    /**
     * Optional list of member IDs to remove from the team.
     */
    membersDelete?: string[];
}

/**
 * Filters for finding 'Team' resources via MCP.
 * @title MCP Team Find Filters
 */
export interface TeamFindFilters extends CommonFindFilters {
    /** Filter by team handle (exact match). */
    handle?: string;
    /** Find teams containing specific user IDs as members. */
    memberUserIds?: string[];
    /** Filter teams by associated tags. */
    tags?: string[];
    /** Filter teams by whether they are open to new members. */
    isOpenToNewMembers?: boolean;
    /** Sort order for the results. */
    sortBy?: TeamSortBy;
}

/**
 * Attributes for adding a 'Note' resource via MCP.
 * @title MCP Note Add Attributes
 */
export interface NoteAddAttributes extends Pick<CommonAttributes, "tagsConnect"> {
    /**
     * The name or title of the note.
     */
    name: string;
    /**
     * The content of the note.
     */
    content: string;
}
