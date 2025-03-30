import { TranslationKeyCommon } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon, IconInfo } from "../../../icons/Icons.js";
import { CardGrid } from "../../lists/CardGrid/CardGrid.js";
import { TIDCard } from "../../lists/TIDCard/TIDCard.js";
import { TopBar } from "../../navigation/TopBar.js";
import { Title } from "../../text/Title.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { SubroutineCreateDialogProps } from "../types.js";

type SubroutineType = "Api" | "Code" | "Data" | "Generate" | "Prompt" | "SmartContract" | "WebContent";
type SubroutineInfo = {
    objectType: SubroutineType;
    description: TranslationKeyCommon,
    iconInfo: IconInfo,
    id: string,
}

const subroutineTypes: SubroutineInfo[] = [
    {
        objectType: "Prompt",
        description: "SubroutineDescriptionPrompt",
        iconInfo: { name: "Standard", type: "Common" },
        id: "select-prompt-card",
    },
    {
        objectType: "Data",
        description: "SubroutineDescriptionData",
        iconInfo: { name: "Article", type: "Common" },
        id: "select-data-card",
    },
    {
        objectType: "Generate",
        description: "SubroutineDescriptionGenerate",
        iconInfo: { name: "Magic", type: "Common" },
        id: "select-generate-card",
    },
    {
        objectType: "Api",
        description: "SubroutineDescriptionApi",
        iconInfo: { name: "Api", type: "Common" },
        id: "select-api-card",
    },
    {
        objectType: "SmartContract",
        description: "SubroutineDescriptionSmartContract",
        iconInfo: { name: "SmartContract", type: "Common" },
        id: "select-smart-contract-card",
    },
    {
        objectType: "WebContent",
        description: "SubroutineDescriptionWebContent",
        iconInfo: { name: "Website", type: "Common" },
        id: "select-web-content-card",
    },
    {
        objectType: "Code",
        description: "SubroutineDescriptionCode",
        iconInfo: { name: "Terminal", type: "Common" },
        id: "select-code-card",
    },
];

export function SubroutineCreateDialog({
    isOpen,
    onClose,
}: SubroutineCreateDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = "dialog";

    const [selectedType, setSelectedType] = useState<SubroutineType | null>(null);
    const [page, setPage] = useState<"select" | "create">("select");
    function restart() {
        setPage("select");
        setSelectedType(null);
    }

    const selectedForm = useMemo(() => {
        if (!selectedType) return null;
        // switch (type) {
        //     case 'Prompt':
        //         return <PromptForm />;
        //     case 'Data':
        //         return <DataForm />;
        //     // ... other cases
        //     default:
        //         return null;
        // }
        return "TODO";
    }, [selectedType]);

    return (
        <LargeDialog
            id="subroutine-select-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={""}
            sxs={{ paper: { width: "min(100%, 1200px)" } }}
        >
            <TopBar
                display={display}
                onClose={onClose}
                startComponent={selectedType ? <IconButton
                    aria-label={t("Back")}
                    onClick={restart}
                    sx={{
                        width: "48px",
                        height: "48px",
                        marginLeft: 1,
                        marginRight: 1,
                        cursor: "pointer",
                    }}
                >
                    <IconCommon
                        decorative
                        fill={palette.primary.contrastText}
                        name="ArrowLeft"
                    />
                </IconButton> : undefined}
                title={t("CreateSubroutine")}
            />
            {selectedForm}
            {!selectedForm && <Box p={2} display="flex" flexDirection="column" gap={2}>
                <Title
                    title={t("SelectSubroutineType")}
                    variant="subheader"
                />
                <CardGrid minWidth={300} disableMargin>
                    {subroutineTypes.map(({ objectType, description, iconInfo, id }, index) => {
                        function handleClick() {
                            setSelectedType(objectType);
                        }

                        return (
                            <TIDCard
                                buttonText={t("Create")}
                                description={description}
                                iconInfo={iconInfo}
                                id={id}
                                key={index}
                                onClick={handleClick}
                                title={t(objectType, { count: 1, defaultValue: objectType })}
                            />
                        );
                    })}
                </CardGrid>
            </Box>}
        </LargeDialog>
    );
}
