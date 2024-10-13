import { CreateFormInputProps, FormBuildViewProps, FormDividerType, FormElement, FormHeaderType, FormInputType, FormRunViewProps, FormSchema, FormStructureType, FormViewProps, GridContainer, InputType, createFormInput, generateInitialValues, mergeDeep, noop, noopSubmit, preventFormSubmit, uuid } from "@local/shared";
import { Box, BoxProps, Divider, Grid, GridSpacing, IconButton, List, ListItem, ListItemIcon, ListItemText, ListSubheader, Popover, Stack, Typography, styled, useTheme } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FormDivider } from "components/inputs/form/FormDivider/FormDivider";
import { FormHeader } from "components/inputs/form/FormHeader/FormHeader";
import { FormInput } from "components/inputs/form/FormInput/FormInput";
import { Formik } from "formik";
import { FormErrorBoundary } from "forms/FormErrorBoundary/FormErrorBoundary";
import { usePopover } from "hooks/usePopover";
import { useWindowSize } from "hooks/useWindowSize";
import { CaseSensitiveIcon, DragIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, HeaderIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, MinusIcon, NumberIcon, ObjectIcon, SliderIcon, SwitchIcon, CaseSensitiveIcon as TextInputIcon, UploadIcon, VrooliIcon } from "icons";
import React, { Fragment, memo, useCallback, useMemo, useRef, useState } from "react";
import { DragDropContext, Draggable, DraggableProvided, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";

/**
 * Function to convert a FormSchema into an array of containers with start and end indices for rendering.
 * @param formSchema The schema defining the form layout, containers, and elements.
 * @returns An array of containers with start and end indices for the elements in each container. 
 * If no containers are provided, returns one container covering all elements. 
 * Ensures valid ranges for each container.
 */
export function normalizeFormContainers(
    formSchema: Partial<Pick<FormSchema, "containers" | "elements">>,
): GridContainer[] {
    if (typeof formSchema !== "object" || formSchema === null) {
        // If the schema is not an object, return an empty array
        return [];
    }

    const { elements, containers } = formSchema;

    if (!Array.isArray(elements) || elements.length === 0) {
        // If there are no elements, return an empty array
        return [];
    }

    if (!Array.isArray(containers) || containers.length === 0) {
        // If no containers are provided, return one container for all elements
        return [{
            totalItems: elements.length,
        }];
    }

    const gridContainers: GridContainer[] = [];
    let currentIndex = 0;

    containers.forEach(container => {
        if (currentIndex >= elements.length) {
            // If the current index is beyond the elements length, break the loop
            return;
        }

        const endIndex = Math.min(currentIndex + container.totalItems - 1, elements.length - 1);

        gridContainers.push({
            ...container,
            totalItems: endIndex - currentIndex + 1,
        });

        currentIndex = endIndex + 1;
    });

    // Ensure all elements are covered in case totalItems were more than elements length
    if (currentIndex < elements.length) {
        gridContainers.push({
            totalItems: elements.length - currentIndex,
        });
    }

    return gridContainers;
}

type PopoverListInput = {
    category: string;
    items: readonly { type: InputType; icon: React.ReactNode; label: string }[];
}[]

type PopoverListStructure = {
    category: string;
    items: readonly { type: FormStructureType; icon: React.ReactNode; label: string; tag?: FormHeaderType["tag"] }[];
}[]

/**
 * Applies restrictions to the available form element types in 
 * one of the form builder popover categories
 */
function filterCategory<T extends PopoverListInput | PopoverListStructure>(
    category: T[0],
    limits: FormBuildViewProps["limits"],
    popover: "inputs" | "structures",
) {
    if (limits?.[popover]?.types) {
        // Filter items based on the limits.input array
        const filteredItems = category.items.filter(item => limits?.[popover]?.types?.includes(item.type as never));

        // If filteredItems is not empty, return category with filtered items
        if (filteredItems.length > 0) {
            return { ...category, items: filteredItems };
        }
        // If filteredItems is empty, return null
        return null;
    }
    // Otherwise, return the category as-is
    return category;
}

interface ElementBuildOuterBoxProps extends BoxProps {
    isSelected: boolean;
}
const ElementBuildOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSelected",
})<ElementBuildOuterBoxProps>(({ isSelected, theme }) => ({
    display: "flex",
    background: theme.palette.background.paper,
    border: isSelected ? `4px solid ${theme.palette.secondary.main}` : "none",
    borderRadius: "8px",
    overflow: "overlay",
    height: "100%",
    alignItems: "stretch",
}));

const ElementButton = styled("button")(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
    cursor: "pointer",
}));

const DragBox = styled(Box)(({ theme }) => ({
    cursor: "pointer",
    display: "flex",
    flexDirection: "column",
    padding: "4px",
    background: theme.palette.secondary.main,
}));

const dragIconStyle = { marginTop: "auto", marginBottom: "auto" } as const;

interface ToolbarBoxProps extends BoxProps {
    formElementsCount: number;
}
const ToolbarBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "formElementsCount",
})<ToolbarBoxProps>(({ formElementsCount, theme }) => ({
    display: "flex",
    justifyContent: "space-around",
    margin: "-4px",
    padding: "4px",
    background: theme.palette.primary.main,
    color: theme.palette.secondary.contrastText,
    borderBottomLeftRadius: formElementsCount === 0 ? "8px" : "0px",
    borderBottomRightRadius: formElementsCount === 0 ? "8px" : "0px",
}));

const toolbarLargeButtonStyle = { cursor: "pointer" } as const;

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

interface PopoverListItemProps {
    icon: React.ReactNode;
    key: string;
    label: string;
    tag?: FormHeaderType["tag"];
    type: FormElement["type"];
    onAddDivider: () => unknown;
    onAddHeader: (headerData: Partial<FormHeaderType>) => unknown;
    onAddInput: (InputData: Omit<Partial<FormInputType>, "type"> & { type: InputType; }) => unknown;
}

const PopoverListItem = memo(function PopoverListItemMemo({
    icon,
    key,
    label,
    onAddDivider,
    onAddHeader,
    onAddInput,
    tag,
    type,
}: PopoverListItemProps) {
    const handleClick = useCallback(() => {
        if (type === "Divider") {
            onAddDivider();
        } else if (type === "Header") {
            if (tag) {
                onAddHeader({ tag });
            } else {
                console.error("Missing tag for header - cannot add header to form.");
            }
        } else {
            onAddInput({ type });
        }
    }, [onAddDivider, onAddHeader, onAddInput, tag, type]);

    return (
        <ListItem
            button
            key={key}
            onClick={handleClick}
        >
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText primary={label} />
        </ListItem>
    );
});

const FormHelperText = styled(Typography)(({ theme }) => ({
    textAlign: "center",
    color: theme.palette.text.secondary,
    padding: "20px",
}));

const formDividerStyle = { marginBottom: 2 } as const;

//TODO: Allow titles to collapse sections below them. Need to update the way inputs are rendered so that they're nested within the header using a collapsibetext component
export function FormBuildView({
    fieldNamePrefix,
    limits,
    onSchemaChange,
    schema,
}: FormBuildViewProps) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [selectedElementIndex, setSelectedElementIndex] = useState<number | null>(null);
    const elementRefs = useRef<(HTMLElement | null)[]>([]);

    const [inputAnchorEl, openInputPopover, closeInputPopover, isInputPopoverOpen] = usePopover();
    const handleInputPopoverOpen = useCallback(function handleInputPopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openInputPopover(event);
    }, [openInputPopover]);

    const [structureAnchorEl, openStructurePopover, closeStructurePopover, isStructurePopoverOpen] = usePopover();
    const handleStructurePopoverOpen = useCallback(function handleStructurePopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openStructurePopover(event);
    }, [openStructurePopover]);

    const closePopovers = useCallback(function closePopoversCallback() {
        closeInputPopover();
        closeStructurePopover();
    }, [closeStructurePopover, closeInputPopover]);

    const handleAddElement = useCallback(function handleAddElementCallback<T extends FormElement>(element: Omit<T, "id">) {
        const newElement = {
            ...element,
            id: uuid(),
        } as unknown as FormElement;
        const newElements = [...schema.elements ?? []];
        if (selectedElementIndex !== null) {
            newElements.splice(selectedElementIndex + 1, 0, newElement);
        } else {
            newElements.push(newElement);
        }
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
        setSelectedElementIndex(newElements.length - 1);
    }, [onSchemaChange, schema, selectedElementIndex]);

    const handleDeleteElement = useCallback(function handleDeleteElementCallback(index: number) {
        const newElements = [...schema.elements ?? []];
        newElements.splice(index, 1);
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
        // Set the selection to the next element if there is one
        if (index < newElements.length) {
            setSelectedElementIndex(index);
        } else if (index > 0) {
            setSelectedElementIndex(index - 1);
        } else {
            setSelectedElementIndex(null);
        }
    }, [schema, onSchemaChange]);

    function selectElement(index: number) {
        setSelectedElementIndex(index);
    }

    const inputItems = useMemo(function inputItemsMemo() {
        return ([
            {
                category: "Text Inputs",
                items: [
                    { type: InputType.Text, icon: <CaseSensitiveIcon />, label: "Text" },
                    { type: InputType.JSON, icon: <ObjectIcon />, label: "JSON (structured text)" }, //TODO
                ],
            },
            {
                category: "Selection Inputs",
                items: [
                    { type: InputType.Checkbox, icon: <ListCheckIcon />, label: "Checkbox (Select multiple from list)" },
                    { type: InputType.Radio, icon: <ListBulletIcon />, label: "Radio (Select one from list)" },
                    { type: InputType.Selector, icon: <ListIcon />, label: "Selector (Select one from list)" }, // TODO waiting on InputType.JSON to define item shape (standard) and InputType.LinkItem to connect standard
                    { type: InputType.Switch, icon: <SwitchIcon />, label: "Switch (Toggle on/off or true/false)" },
                ],
            },
            {
                category: "Link Inputs",
                items: [
                    { type: InputType.LinkUrl, icon: <LinkIcon />, label: "Link any URL" }, //TODO
                    { type: InputType.LinkItem, icon: <VrooliIcon />, label: "Link Vrooli object" }, //TODO
                ],
            },
            {
                category: "Numeric Inputs",
                items: [
                    { type: InputType.IntegerInput, icon: <NumberIcon />, label: "Number" },
                    { type: InputType.Slider, icon: <SliderIcon />, label: "Slider (Select a number from a range)" },
                ],
            },
            {
                category: "File Inputs",
                items: [
                    { type: InputType.Dropzone, icon: <UploadIcon />, label: "Dropzone (file upload)" },
                ],
            },
        ] as const).reduce((acc, category) => {
            const filteredCategory = filterCategory<PopoverListInput>(category, limits, "inputs");
            if (filteredCategory) {
                acc.push(filteredCategory);
            }
            return acc;
        }, [] as PopoverListInput);
    }, [limits]);

    const structureItems = useMemo(function structureItemsMemo() {
        return ([
            {
                category: "Headers",
                items: [
                    { type: FormStructureType.Header, tag: "h1", icon: <Header1Icon />, label: "Title (Largest)" },
                    { type: FormStructureType.Header, tag: "h2", icon: <Header2Icon />, label: "Subtitle (Large)" },
                    { type: FormStructureType.Header, tag: "h3", icon: <Header3Icon />, label: "Header (Medium)" },
                    { type: FormStructureType.Header, tag: "h4", icon: <Header4Icon />, label: "Subheader (Small)" },
                ],
            }, {
                category: "Page Elements",
                items: [
                    { type: FormStructureType.Divider, icon: <MinusIcon />, label: "Divider" },
                ],
            },
        ] as const).reduce((acc, category) => {
            const filteredCategory = filterCategory<PopoverListStructure>(category, limits, "structures");
            if (filteredCategory) {
                acc.push(filteredCategory);
            }
            return acc;
        }, [] as PopoverListStructure);
    }, [limits]);

    const handleAddInput = useCallback(function handleAddInputCallback(data: Omit<Partial<FormInputType>, "type"> & { type: InputType }) {
        const FIELD_NAME_RANDOM_LENGTH = 10;
        const newElement = createFormInput({
            fieldName: `field-${randomString(FIELD_NAME_RANDOM_LENGTH)}`,
            label: `Input #${selectedElementIndex !== null ? selectedElementIndex + 1 : schema.elements.length + 1}`,
            ...data,
        } as CreateFormInputProps);
        if (!newElement) {
            console.error("Failed to create form input", data);
            return;
        }
        handleAddElement(newElement);
        closeInputPopover();
    }, [handleAddElement, closeInputPopover, selectedElementIndex, schema.elements.length]);

    const handleUpdateInput = useCallback(function handleUpdateInputCallback(index: number, data: Partial<FormInputType>) {
        const element = mergeDeep(data, schema.elements[index] as FormInputType);
        const newElements = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)] as FormElement[];
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
    }, [onSchemaChange, schema]);

    const handleAddHeader = useCallback(function handleAddHeaderCallback(data: Partial<FormHeaderType>) {
        const tag = data.tag ?? "h1";
        handleAddElement<FormHeaderType>({
            type: FormStructureType.Header,
            label: data.label ?? `New ${tag.toUpperCase()}`,
            tag,
            ...data,
        });
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const handleUpdateHeader = useCallback(function handleUpdateHeaderCallback(index: number, data: Partial<FormHeaderType>) {
        const element = {
            ...schema.elements[index],
            ...data,
        };
        const newElements = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)] as FormElement[];
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
    }, [onSchemaChange, schema]);

    const handleAddDivider = useCallback(function handleAddDividerCallback() {
        handleAddElement<FormDividerType>({
            type: FormStructureType.Divider,
            label: "",
        });
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination || result.type !== "formElement") return;
        const newElements = [...schema.elements];
        const [removed] = newElements.splice(source.index, 1);
        newElements.splice(destination.index, 0, removed);
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
        setSelectedElementIndex(destination.index);
    }, [onSchemaChange, schema]);

    function renderElement(element: FormElement, index: number, providedDrag: DraggableProvided) {
        const isSelected = selectedElementIndex === index;

        function handleElementButtonRef(ref: HTMLElement | null) {
            elementRefs.current[index] = ref;
        }
        function onElementButtonClick() {
            selectElement(index);
        }
        function onElementButtonKeyDown(e: React.KeyboardEvent) {
            if (!isSelected && (e.key === "Enter" || e.key === " ")) {
                e.preventDefault();
                selectElement(index);
            }
        }

        function onFormHeaderUpdate(data: Partial<FormHeaderType>) {
            if (!isSelected) return;
            handleUpdateHeader(index, data);
        }
        function onFormInputConfigUpdate(updatedInput: Partial<FormInputType>) {
            if (!isSelected) return;
            handleUpdateInput(index, updatedInput);
        }
        function onFormElementDelete() {
            if (!isSelected) return;
            handleDeleteElement(index);
        }

        return (
            <ElementBuildOuterBox
                isSelected={isSelected}
                key={element.id}
                ref={providedDrag.innerRef}
                {...providedDrag.draggableProps}
            >
                <ElementButton
                    ref={handleElementButtonRef}
                    onClick={onElementButtonClick}
                    onKeyDown={onElementButtonKeyDown}
                    onSubmit={preventFormSubmit} // Form elements should never submit the form
                    type="button"
                >
                    {element.type === FormStructureType.Header && (
                        <FormHeader
                            element={element as FormHeaderType}
                            isEditing={isSelected}
                            onUpdate={onFormHeaderUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Divider && (
                        <FormDivider
                            isEditing={isSelected}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {!(element.type in FormStructureType) && (
                        <FormInput
                            fieldData={element as FormInputType}
                            fieldNamePrefix={fieldNamePrefix}
                            index={index}
                            isEditing={isSelected}
                            onConfigUpdate={onFormInputConfigUpdate}
                            onDelete={onFormElementDelete}
                        />)}
                    {isSelected ? Toolbar : null}
                </ElementButton>
                {isSelected && (
                    <DragBox
                        {...providedDrag.dragHandleProps}
                    >
                        <DragIcon
                            fill={palette.secondary.contrastText}
                            width="24px"
                            height="24px"
                            style={dragIconStyle}
                        />
                    </DragBox>
                )}
            </ElementBuildOuterBox>
        );
    }

    const Toolbar = useMemo(() => {
        const numInputItems = inputItems.reduce((acc, { items }) => acc + items.length, 0);
        const firstInputItem = numInputItems === 1 ? inputItems[0].items[0] : null;
        const DisplayedInputIcon = firstInputItem ? (() => firstInputItem.icon) : TextInputIcon;
        function inputOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstInputItem) {
                handleAddInput({ type: firstInputItem.type });
            } else {
                handleInputPopoverOpen(event);
            }
        }

        const numStructureItems = structureItems.reduce((acc, { items }) => acc + items.length, 0);
        const firstStructureItem = numStructureItems === 1 ? structureItems[0].items[0] : null;
        const DisplayedStructureIcon = firstStructureItem ? (() => firstStructureItem.icon) : HeaderIcon;
        function structureOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstStructureItem) {
                if (firstStructureItem.type === "Divider") {
                    handleAddDivider();
                } else {
                    handleAddHeader({ tag: firstStructureItem.tag });
                }
            } else {
                handleStructurePopoverOpen(event);
            }
        }

        return (
            <ToolbarBox formElementsCount={schema.elements.length}>
                {numInputItems > 0 && <>
                    {isMobile ? <IconButton onClick={inputOnClick}>
                        <DisplayedInputIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={inputOnClick}>
                        {numInputItems === 1 && firstInputItem ? `Add ${firstInputItem.label.toLowerCase()}` : "Add input"}
                    </Typography>}
                </>}
                {numStructureItems > 0 && <>
                    {isMobile ? <IconButton onClick={structureOnClick}>
                        <DisplayedStructureIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={structureOnClick}>
                        {numStructureItems === 1 && firstStructureItem ? `Add ${firstStructureItem.label.toLowerCase()}` : "Add structure"}
                    </Typography>}
                </>}
            </ToolbarBox>
        );
    }, [inputItems, structureItems, schema.elements.length, isMobile, handleAddInput, handleInputPopoverOpen, handleAddDivider, handleAddHeader, handleStructurePopoverOpen]);

    const initialValues = useMemo(function initialValuesMemo() {
        return generateInitialValues(schema.elements, fieldNamePrefix);
    }, [fieldNamePrefix, schema.elements]);

    return (
        <div>
            <Popover
                open={isInputPopoverOpen}
                anchorEl={inputAnchorEl}
                onClose={closeInputPopover}
                anchorOrigin={popoverAnchorOrigin}
            >

                {inputItems.map(({ category, items }) => (
                    <Fragment key={category}>
                        <ListSubheader>{category}</ListSubheader>
                        {items.map((item) => (
                            <PopoverListItem
                                key={item.type}
                                icon={item.icon}
                                label={item.label}
                                type={item.type as PopoverListItemProps["type"]}
                                onAddDivider={handleAddDivider}
                                onAddHeader={handleAddHeader}
                                onAddInput={handleAddInput}
                            />
                        ))}
                        <Divider />
                    </Fragment>
                ))}
            </Popover>
            <Popover
                open={isStructurePopoverOpen}
                anchorEl={structureAnchorEl}
                onClose={closeStructurePopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {structureItems.map(({ category, items }) => (
                        <Fragment key={category}>
                            <ListSubheader>{category}</ListSubheader>
                            {items.map((item) => (
                                <PopoverListItem
                                    key={item.type}
                                    icon={item.icon}
                                    label={item.label}
                                    tag={item.tag}
                                    type={item.type}
                                    onAddDivider={handleAddDivider}
                                    onAddHeader={handleAddHeader}
                                    onAddInput={handleAddInput}
                                />
                            ))}
                            <Divider />
                        </Fragment>
                    ))}
                </List>
            </Popover>
            {schema.elements.length === 0 && <FormHelperText variant="body1">Use the options below to populate the form.</FormHelperText>}
            {schema.elements.length > 0 && <Divider sx={formDividerStyle} />}
            {/* Formik defined here to cancel out any formik context higher-up in the tree */}
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
            >
                {() => (
                    <FormErrorBoundary onError={closePopovers}> {/* Error boundary to catch elements that fail to render */}
                        {/* Drag/Drop context to reorder elements */}
                        <DragDropContext onDragEnd={onDragEnd}>
                            <Droppable droppableId="form-builder-elements" direction="vertical" type="formElement">
                                {(providedDrop) => (
                                    <Box ref={providedDrop.innerRef} {...providedDrop.droppableProps}> {/* Droppable area for form elements */}
                                        {schema.elements.map((element, index) => (
                                            <Draggable key={`${element.id}-drag`} draggableId={`${element.id}-drag`} index={index}>
                                                {(providedDrag) => renderElement(element, index, providedDrag)}
                                            </Draggable>
                                        ))}
                                        {providedDrop.placeholder}
                                    </Box>
                                )}
                            </Droppable>
                        </DragDropContext>
                    </FormErrorBoundary>
                )}
            </Formik>
            {schema.elements.length === 0 || selectedElementIndex === null ? Toolbar : null}
        </div>
    );
}

/**
 * Calculates size of grid item based on the number of items in the grid. 
 * 1 item is { xs: 12 }, 
 * 2 items is { xs: 12, sm: 6 },
 * 3 items is { xs: 12, sm: 6, md: 4 },
 * 4+ items is { xs: 12, sm: 6, md: 4, lg: 3 }
 * @returns Size of grid item
 */
export function calculateGridItemSize(numItems: number): { [key: string]: GridSpacing } {
    switch (numItems) {
        case 1:
            return { xs: 12 };
        // eslint-disable-next-line no-magic-numbers
        case 2:
            return { xs: 12, sm: 6 };
        // eslint-disable-next-line no-magic-numbers
        case 3:
            return { xs: 12, sm: 6, md: 4 };
        default:
            return { xs: 12, sm: 6, md: 4, lg: 3 };
    }
}

type GeneratedFormItemProps = {
    /** The child form input or form structure element */
    children: JSX.Element | null | undefined;
    /** The number of fields in the grid, for calculating grid item size */
    fieldsInGrid: number;
    /** Whether to wrap the child in a grid item */
    isInGrid: boolean;
}

/**
 * A wrapper for a form item that can be wrapped in a grid item
 */
export function GeneratedGridItem({
    children,
    fieldsInGrid,
    isInGrid,
}: GeneratedFormItemProps) {
    return isInGrid ? <Grid item {...calculateGridItemSize(fieldsInGrid)}>{children}</Grid> : children;
}

const ElementRunOuterBox = styled("button")(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
    cursor: "pointer",
}));

const sectionsStackStyle = { width: "100%" } as const;
const formViewDividerStyle = {
    paddingTop: 2,
} as const;

export function FormRunView({
    disabled,
    fieldNamePrefix,
    schema,
}: FormRunViewProps) {

    const renderElement = useCallback(function renderElementMemo(element: FormElement, index: number) {
        return (
            <ElementRunOuterBox
                key={element.id}
            >
                {element.type === FormStructureType.Header && (
                    <FormHeader
                        element={element as FormHeaderType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
                    />
                )}
                {element.type === FormStructureType.Divider && (
                    <FormDivider
                        isEditing={false}
                        onDelete={noop}
                    />
                )}
                {!(element.type in FormStructureType) && (
                    <FormInput
                        disabled={disabled}
                        fieldData={element as FormInputType}
                        fieldNamePrefix={fieldNamePrefix}
                        index={index}
                        isEditing={false}
                        onConfigUpdate={noop}
                        onDelete={noop}
                    />)}
            </ElementRunOuterBox>
        );
    }, [disabled, fieldNamePrefix]);

    // TODO build view should also group into sections, where sections are created automatically based on headers and page dividers, or manually somehow (when you want to display inputs on the same line, for example). Can possibly update normalizeFormContainers to handle this
    const sections = useMemo(() => {
        // Normalize/heal containers to ensure they cover all elements
        const gridContainers = normalizeFormContainers(schema);
        // Render each container as a stack or grid, depending on configuration
        const sections: JSX.Element[] = [];
        let currentIndex = 0;

        for (let i = 0; i < gridContainers.length; i++) {
            const currContainer: GridContainer = gridContainers[i];
            const containerProps = {
                direction: currContainer?.direction ?? "column",
                key: `form-section-container-${i}`,
                spacing: 2,
            };
            // Use grid for horizontal layout, and stack for vertical layout
            const useGrid = containerProps.direction === "row";
            // Generate component for each field in the grid
            const gridItems: JSX.Element[] = [];
            const endIndex = currentIndex + currContainer.totalItems - 1;

            for (let j = currentIndex; j <= endIndex; j++) {
                const fieldData = schema.elements[j] as FormInputType;
                gridItems.push(
                    <GeneratedGridItem
                        key={`grid-item-${fieldData.id}`}
                        fieldsInGrid={currContainer.totalItems}
                        isInGrid={useGrid}
                    >
                        {renderElement(fieldData, j)}
                    </GeneratedGridItem>,
                );
            }

            const itemsContainer = useGrid ? (
                <Grid container {...containerProps}>
                    {gridItems}
                </Grid>
            ) : (
                <Stack {...containerProps}>
                    {gridItems}
                </Stack>
            );

            // If a title is provided, the items are wrapped in a collapsible container
            if (currContainer?.title) {
                sections.push(
                    <ContentCollapse
                        disableCollapse={currContainer.disableCollapse}
                        helpText={currContainer.description ?? undefined}
                        title={currContainer.title}
                        titleComponent="legend"
                    >
                        {itemsContainer}
                        {i < gridContainers.length - 1 && <Divider sx={formViewDividerStyle} />}
                    </ContentCollapse>,
                );
            } else {
                if (i < gridContainers.length - 1) {
                    sections.push(<div>
                        {itemsContainer}
                        <Divider sx={formViewDividerStyle} />
                    </div>);
                } else {
                    sections.push(itemsContainer);
                }
            }

            currentIndex = endIndex + 1;
        }
        return sections;
    }, [renderElement, schema]);

    return (
        <div>
            {schema.elements.length === 0 && <FormHelperText variant="body1">The form is empty.</FormHelperText>}
            {/* Don't use formik here, since it should be provided by parent */}
            <FormErrorBoundary> {/* Error boundary to catch elements that fail to render */}
                <Stack
                    direction={"column"}
                    key={"form-container"}
                    spacing={4}
                    sx={sectionsStackStyle}
                >
                    {sections}
                </Stack>
            </FormErrorBoundary>
        </div>
    );
}

export function FormView({
    isEditing,
    ...props
}: FormViewProps) {
    if (isEditing) {
        return <FormBuildView {...props} />;
    }
    return <FormRunView {...props} />;
}
