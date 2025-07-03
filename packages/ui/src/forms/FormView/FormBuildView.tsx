// AI_CHECK: TYPE_SAFETY=fixed-double-casting-patterns | LAST: 2025-06-30
import { DragDropContext, Draggable, Droppable, type DraggableProvided, type DropResult } from "@hello-pangea/dnd";
import { useTheme } from "@mui/material";
import { Box } from "../../components/layout/Box.js";
import { Divider } from "../../components/layout/Divider.js";
import List from "@mui/material/List";
import ListSubheader from "@mui/material/ListSubheader";
import Popover from "@mui/material/Popover";
import Typography from "@mui/material/Typography";
import { FormBuilder, FormStructureType, InputType, createFormInput, mergeDeep, nanoid, noop, noopSubmit, preventFormSubmit, type CreateFormInputProps, type FormBuildViewProps, type FormDividerType, type FormElement, type FormHeaderType, type FormImageType, type FormInformationalType, type FormInputType, type FormQrCodeType, type FormTipType, type FormVideoType } from "@vrooli/shared";
import { Formik } from "formik";
import React, { Fragment, memo, useCallback, useMemo, useRef, useState } from "react";
import { IconButton } from "../../components/buttons/IconButton.js";
import { FormDivider } from "../../components/inputs/form/FormDivider.js";
import { FORM_HEADER_SIZE_OPTIONS, FormHeader } from "../../components/inputs/form/FormHeader.js";
import { FormImage } from "../../components/inputs/form/FormImage.js";
import { FormInput } from "../../components/inputs/form/FormInput.js";
import { FormQrCode } from "../../components/inputs/form/FormQrCode.js";
import { FormTip } from "../../components/inputs/form/FormTip.js";
import { FormVideo } from "../../components/inputs/form/FormVideo.js";
import { FormErrorBoundary } from "../../forms/FormErrorBoundary/FormErrorBoundary.js";
import { usePopover } from "../../hooks/usePopover.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { Icon, IconCommon } from "../../icons/Icons.js";
import { randomString } from "../../utils/codes.js";
import { PopoverListItem } from "./PopoverListItem.js";
import { filterCategory, normalizeFormContainers } from "./FormView.utils.js";
import { DragBox, ElementBuildOuterBox, ElementButton, FormHelperText, ToolbarBox, dragIconStyle, formDividerStyle, popoverAnchorOrigin, toolbarLargeButtonStyle } from "./FormView.styles.js";
import type { PopoverListInput, PopoverListStructure, PopoverListItemProps } from "./FormView.types.js";

export const FormBuildView = memo(function FormBuildViewMemo({
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
        const newElement: T = {
            ...element,
            id: nanoid(),
        } as T;
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
                    { type: InputType.Text, iconInfo: { name: "CaseSensitive", type: "Text" } as const, label: "Text" },
                    { type: InputType.JSON, iconInfo: { name: "Object", type: "Common" } as const, label: "JSON (structured data)" },
                ],
            },
            {
                category: "Selection Inputs",
                items: [
                    { type: InputType.Checkbox, iconInfo: { name: "ListCheck", type: "Text" } as const, label: "Checkbox (Select multiple from list)" },
                    { type: InputType.Radio, iconInfo: { name: "ListBullet", type: "Text" } as const, label: "Radio (Select one from list)" },
                    { type: InputType.Selector, iconInfo: { name: "List", type: "Text" } as const, label: "Selector (Dynamic dropdown)" },
                    { type: InputType.Switch, iconInfo: { name: "Switch", type: "Common" } as const, label: "Switch (Toggle on/off or true/false)" },
                ],
            },
            {
                category: "Link Inputs",
                items: [
                    { type: InputType.LinkUrl, iconInfo: { name: "Link", type: "Common" } as const, label: "Link external URL" },
                    { type: InputType.LinkItem, iconInfo: { name: "Vrooli", type: "Common" } as const, label: "Link Vrooli object" },
                ],
            },
            {
                category: "Numeric Inputs",
                items: [
                    { type: InputType.IntegerInput, iconInfo: { name: "Number", type: "Common" } as const, label: "Number" },
                    { type: InputType.Slider, iconInfo: { name: "Slider", type: "Common" } as const, label: "Slider (Select a number from a range)" },
                ],
            },
            {
                category: "File Inputs",
                items: [
                    { type: InputType.Dropzone, iconInfo: { name: "Upload", type: "Common" } as const, label: "Dropzone (file upload)" },
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
                items: FORM_HEADER_SIZE_OPTIONS,
            }, {
                category: "Page Elements",
                items: [
                    { type: FormStructureType.Divider, iconInfo: { name: "Minus", type: "Common" } as const, label: "Divider" },
                ],
            }, {
                category: "Informational",
                items: [
                    { type: FormStructureType.Tip, iconInfo: { name: "Help", type: "Common" } as const, label: "Tip" },
                    { type: FormStructureType.Image, iconInfo: { name: "Image", type: "Common" } as const, label: "Image (URL)" },
                    { type: FormStructureType.Video, iconInfo: { name: "Play", type: "Common" } as const, label: "Video (URL)" },
                    { type: FormStructureType.QrCode, iconInfo: { name: "QrCode", type: "Common" } as const, label: "QR Code" },
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
        const createFormInputData: CreateFormInputProps = {
            fieldName: `field-${randomString(FIELD_NAME_RANDOM_LENGTH)}`,
            label: `Input #${selectedElementIndex !== null ? selectedElementIndex + 1 : schema.elements.length + 1}`,
            ...data,
        };
        const newElement = createFormInput(createFormInputData);
        if (!newElement) {
            console.error("Failed to create form input", data);
            return;
        }
        handleAddElement(newElement);
        closeInputPopover();
    }, [handleAddElement, closeInputPopover, selectedElementIndex, schema.elements.length]);

    const handleUpdateInput = useCallback(function handleUpdateInputCallback(index: number, data: Partial<FormInputType>) {
        const existingElement = schema.elements[index];
        if (!existingElement || !("fieldName" in existingElement)) {
            console.error("Element at index is not a FormInputType", index, existingElement);
            return;
        }
        const element = mergeDeep(data, existingElement);
        const newElements: FormElement[] = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)];
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
            label: data.label ?? "",
            tag,
            ...data,
        });
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const handleAddStructure = useCallback(function handleAddStructureCallback({ type }: { type: "Divider" | "Image" | "QrCode" | "Tip" | "Video" }) {
        switch (type) {
            case "Divider":
                handleAddElement<FormDividerType>({
                    type: FormStructureType.Divider,
                    label: "",
                });
                break;
            case "Image":
                handleAddElement<FormImageType>({
                    type: FormStructureType.Image,
                    label: "",
                    url: "",
                });
                break;
            case "QrCode":
                handleAddElement<FormQrCodeType>({
                    type: FormStructureType.QrCode,
                    label: "",
                    url: "",
                });
                break;
            case "Tip":
                handleAddElement<FormTipType>({
                    type: FormStructureType.Tip,
                    label: "",
                });
                break;
            case "Video":
                handleAddElement<FormVideoType>({
                    type: FormStructureType.Video,
                    label: "",
                    url: "",
                });
                break;
            default:
                console.error("Invalid structure type", type);
        }
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const handleUpdateStructure = useCallback(function handleUpdateStructureCallback(index: number, data: Partial<FormInformationalType>) {
        const existingElement = schema.elements[index];
        if (!existingElement) {
            console.error("Element at index not found", index);
            return;
        }
        const element: FormElement = {
            ...existingElement,
            ...data,
        };
        const newElements = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)];
        onSchemaChange({
            ...schema,
            containers: normalizeFormContainers(schema),
            elements: newElements,
        });
    }, [onSchemaChange, schema]);

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

        function onFormStructureUpdate(data: Partial<FormInformationalType>) {
            if (!isSelected) return;
            handleUpdateStructure(index, data);
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
                    role="button"
                    tabIndex={0}
                    aria-label={`Form element ${index + 1}`}
                >
                    {element.type === FormStructureType.Header && (
                        <FormHeader
                            element={element as FormHeaderType}
                            isEditing={isSelected}
                            onUpdate={onFormStructureUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Divider && (
                        <FormDivider
                            isEditing={isSelected}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Image && (
                        <FormImage
                            element={element as FormImageType}
                            isEditing={isSelected}
                            onUpdate={onFormStructureUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.QrCode && (
                        <FormQrCode
                            element={element as FormQrCodeType}
                            isEditing={isSelected}
                            onUpdate={onFormStructureUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Tip && (
                        <FormTip
                            element={element as FormTipType}
                            isEditing={isSelected}
                            onUpdate={onFormStructureUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Video && (
                        <FormVideo
                            element={element as FormVideoType}
                            isEditing={isSelected}
                            onUpdate={onFormStructureUpdate}
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
                        <IconCommon
                            decorative
                            fill={palette.secondary.contrastText}
                            name="Drag"
                            size={24}
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
        const displayedInputIconInfo = firstInputItem?.iconInfo ?? { name: "CaseSensitive", type: "Text" };
        function inputOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstInputItem) {
                handleAddInput({ type: firstInputItem.type });
            } else {
                handleInputPopoverOpen(event);
            }
        }

        const numStructureItems = structureItems.reduce((acc, { items }) => acc + items.length, 0);
        const firstStructureItem = numStructureItems === 1 ? structureItems[0].items[0] : null;
        const displayedStructureIconInfo = firstStructureItem?.iconInfo ?? { name: "Header", type: "Text" };
        function structureOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstStructureItem) {
                if (firstStructureItem.tag) {
                    handleAddHeader({ tag: firstStructureItem.tag });
                } else {
                    handleAddStructure({ type: firstStructureItem.type as "Divider" | "Image" | "QrCode" | "Tip" | "Video" });
                }
            } else {
                handleStructurePopoverOpen(event);
            }
        }

        return (
            <ToolbarBox formElementsCount={schema.elements.length}>
                {numInputItems > 0 && <>
                    {isMobile ? <IconButton onClick={inputOnClick} variant="transparent">
                        <Icon
                            decorative
                            info={displayedInputIconInfo}
                            size={24}
                        />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={inputOnClick}>
                        {numInputItems === 1 && firstInputItem ? `Add ${firstInputItem.label.toLowerCase()}` : "Add input"}
                    </Typography>}
                </>}
                {numStructureItems > 0 && <>
                    {isMobile ? <IconButton onClick={structureOnClick} variant="transparent">
                        <Icon
                            decorative
                            info={displayedStructureIconInfo}
                            size={24}
                        />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={structureOnClick}>
                        {numStructureItems === 1 && firstStructureItem ? `Add ${firstStructureItem.label.toLowerCase()}` : "Add structure"}
                    </Typography>}
                </>}
            </ToolbarBox>
        );
    }, [inputItems, structureItems, schema.elements.length, isMobile, handleAddInput, handleInputPopoverOpen, handleAddHeader, handleAddStructure, handleStructurePopoverOpen]);

    const initialValues = useMemo(function initialValuesMemo() {
        return FormBuilder.generateInitialValues(schema.elements, fieldNamePrefix);
    }, [fieldNamePrefix, schema.elements]);

    return (
        <div data-testid="form-build-view">
            <Popover
                open={isInputPopoverOpen}
                anchorEl={inputAnchorEl}
                onClose={closeInputPopover}
                anchorOrigin={popoverAnchorOrigin}
                data-testid="input-popover"
            >
                <List disablePadding>
                    {inputItems.map(({ category, items }) => (
                        <Fragment key={category}>
                            <ListSubheader>{category}</ListSubheader>
                            {items.map((item) => (
                                <PopoverListItem
                                    key={`${item.type}-${item.label}`}
                                    iconInfo={item.iconInfo}
                                    label={item.label}
                                    type={item.type as PopoverListItemProps["type"]}
                                    onAddHeader={handleAddHeader}
                                    onAddInput={handleAddInput}
                                    onAddStructure={handleAddStructure}
                                />
                            ))}
                            <Divider />
                        </Fragment>
                    ))}
                </List>
            </Popover>
            <Popover
                open={isStructurePopoverOpen}
                anchorEl={structureAnchorEl}
                onClose={closeStructurePopover}
                anchorOrigin={popoverAnchorOrigin}
                data-testid="structure-popover"
            >
                <List disablePadding>
                    {structureItems.map(({ category, items }) => (
                        <Fragment key={category}>
                            <ListSubheader>{category}</ListSubheader>
                            {items.map((item) => (
                                <PopoverListItem
                                    key={`${item.type}-${item.tag || item.label}`}
                                    iconInfo={item.iconInfo}
                                    label={item.label}
                                    tag={item.tag}
                                    type={item.type}
                                    onAddHeader={handleAddHeader}
                                    onAddInput={handleAddInput}
                                    onAddStructure={handleAddStructure}
                                />
                            ))}
                            <Divider />
                        </Fragment>
                    ))}
                </List>
            </Popover>
            {schema.elements.length === 0 && <FormHelperText data-testid="empty-form-message" variant="body1">Use the options below to populate the form.</FormHelperText>}
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
                                    <Box ref={providedDrop.innerRef} {...providedDrop.droppableProps} data-testid="form-elements-container"> {/* Droppable area for form elements */}
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
});
