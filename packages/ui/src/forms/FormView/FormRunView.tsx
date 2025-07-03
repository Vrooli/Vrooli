import { Divider } from "../../components/layout/Divider.js";
import { Grid } from "../../components/layout/Grid.js";
import { Stack } from "../../components/layout/Stack.js";
import { FormStructureType, noop, type FormElement, type FormHeaderType, type FormImageType, type FormInputType, type FormQrCodeType, type FormRunViewProps, type FormTipType, type FormVideoType, type GridContainer } from "@vrooli/shared";
import { memo, useCallback, useMemo } from "react";
import { ContentCollapse } from "../../components/containers/ContentCollapse.js";
import { FormDivider } from "../../components/inputs/form/FormDivider.js";
import { FormHeader } from "../../components/inputs/form/FormHeader.js";
import { FormImage } from "../../components/inputs/form/FormImage.js";
import { FormInput } from "../../components/inputs/form/FormInput.js";
import { FormQrCode } from "../../components/inputs/form/FormQrCode.js";
import { FormTip } from "../../components/inputs/form/FormTip.js";
import { FormVideo } from "../../components/inputs/form/FormVideo.js";
import { FormErrorBoundary } from "../../forms/FormErrorBoundary/FormErrorBoundary.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { GeneratedGridItem } from "./GeneratedGridItem.js";
import { normalizeFormContainers } from "./FormView.utils.js";
import { ElementRunOuterBox, FormHelperText, formViewDividerStyle, sectionsStackStyle } from "./FormView.styles.js";

export const FormRunView = memo(function FormRunViewMemo({
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
                {element.type === FormStructureType.Image && (
                    <FormImage
                        element={element as FormImageType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
                    />
                )}
                {element.type === FormStructureType.QrCode && (
                    <FormQrCode
                        element={element as FormQrCodeType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
                    />
                )}
                {element.type === FormStructureType.Tip && (
                    <FormTip
                        element={element as FormTipType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
                    />
                )}
                {element.type === FormStructureType.Video && (
                    <FormVideo
                        element={element as FormVideoType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
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
                <Grid key={`form-section-container-${i}`} columns={gridItems.length} gap="sm">
                    {gridItems}
                </Grid>
            ) : (
                <Stack key={`form-section-container-${i}`} {...containerProps}>
                    {gridItems}
                </Stack>
            );

            // If a title is provided, the items are wrapped in a collapsible container
            if (currContainer?.title) {
                sections.push(
                    <ContentCollapse
                        key={`form-section-${i}`}
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
                    sections.push(<div key={`form-section-${i}`}>
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
        <div 
            id={fieldNamePrefix ? `${fieldNamePrefix}-${ELEMENT_IDS.FormRunView}` : ELEMENT_IDS.FormRunView}
            data-testid={fieldNamePrefix ? `${fieldNamePrefix}-form-run-view` : "form-run-view"}
        >
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
});

