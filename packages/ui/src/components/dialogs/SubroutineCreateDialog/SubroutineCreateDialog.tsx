import { TranslationKeyCommon } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { ArrowLeftIcon, RoutineIcon } from "icons";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SvgComponent } from "types";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { SubroutineCreateDialogProps } from "../types";

type SubroutineType = "Api" | "Code" | "Data" | "Generate" | "Prompt" | "SmartContract" | "WebContent";
type SubroutineInfo = {
    objectType: SubroutineType;
    description: TranslationKeyCommon,
    Icon: SvgComponent,
    id: string,
}

const subroutineTypes: SubroutineInfo[] = [
    {
        objectType: "Prompt",
        description: "SubroutineDescriptionPrompt",
        Icon: RoutineIcon,
        id: "select-prompt-card",
    },
    {
        objectType: "Data",
        description: "SubroutineDescriptionData",
        Icon: RoutineIcon,
        id: "select-data-card",
    },
    {
        objectType: "Generate",
        description: "SubroutineDescriptionGenerate",
        Icon: RoutineIcon,
        id: "select-generate-card",
    },
    {
        objectType: "Api",
        description: "SubroutineDescriptionApi",
        Icon: RoutineIcon,
        id: "select-api-card",
    },
    {
        objectType: "SmartContract",
        description: "SubroutineDescriptionSmartContract",
        Icon: RoutineIcon,
        id: "select-smart-contract-card",
    },
    {
        objectType: "WebContent",
        description: "SubroutineDescriptionWebContent",
        Icon: RoutineIcon,
        id: "select-web-content-card",
    },
    {
        objectType: "Code",
        description: "SubroutineDescriptionCode",
        Icon: RoutineIcon,
        id: "select-code-card",
    },
];

export const SubroutineCreateDialog = ({
    isOpen,
    onClose,
}: SubroutineCreateDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = "dialog";

    const [selectedType, setSelectedType] = useState<SubroutineType | null>(null);
    const [page, setPage] = useState<"select" | "create">("select");
    const restart = () => {
        setPage("select");
        setSelectedType(null);
    };

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
                    aria-label="Back"
                    onClick={restart}
                    sx={{
                        width: "48px",
                        height: "48px",
                        marginLeft: 1,
                        marginRight: 1,
                        cursor: "pointer",
                    }}
                >
                    <ArrowLeftIcon fill={palette.primary.contrastText} width="100%" height="100%" />
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
                    {subroutineTypes.map(({ objectType, description, Icon, id }, index) => (
                        <TIDCard
                            buttonText={t("Create")}
                            description={description}
                            Icon={Icon}
                            id={id}
                            key={index}
                            onClick={() => setSelectedType(objectType)}
                            title={t(objectType, { count: 1, defaultValue: objectType })}
                        />
                    ))}
                </CardGrid>
            </Box>}
        </LargeDialog>
    );
};
