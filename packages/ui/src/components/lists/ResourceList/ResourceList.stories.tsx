import { Box, Button, Stack, Typography } from "@mui/material";
import type { Meta } from "@storybook/react";
import { LINKS, ResourceUsedFor, type Resource, type ResourceList as ResourceListType } from "@vrooli/shared";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { ResourceList } from "./ResourceList.js";

const meta = {
    title: "Components/Lists/ResourceList",
    component: ResourceList,
    parameters: {
        docs: {
            description: {
                component: "A list component for displaying and managing resources with horizontal and vertical layouts.",
            },
        },
    },
} satisfies Meta<typeof ResourceList>;

export default meta;

// Mock data for the story
const baseMockResources: Resource[] = [
    {
        __typename: "Resource",
        id: "1",
        link: "https://github.com/example/repo1",
        usedFor: ResourceUsedFor.Related,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "ResourceTranslation",
            id: "1-en",
            language: "en",
            name: "Example Repository",
            description: "A sample repository for demonstration",
        }],
        list: {
            __typename: "ResourceList",
            id: "mock-list",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            resources: [],
            translations: [],
            listFor: { __typename: "RoutineVersion", id: "mock-routine" },
        },
    },
    {
        __typename: "Resource",
        id: "2",
        link: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
        usedFor: ResourceUsedFor.Learning,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "ResourceTranslation",
            id: "2-en",
            language: "en",
            name: "Tutorial Video",
            description: "A helpful video tutorial",
        }],
        list: {
            __typename: "ResourceList",
            id: "mock-list",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            resources: [],
            translations: [],
            listFor: { __typename: "RoutineVersion", id: "mock-routine" },
        },
    },
    {
        __typename: "Resource",
        id: "3",
        link: "https://reddit.com/r/programming",
        usedFor: ResourceUsedFor.Social,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "ResourceTranslation",
            id: "3-en",
            language: "en",
            name: "Programming Community",
            description: "Reddit programming community",
        }],
        list: {
            __typename: "ResourceList",
            id: "mock-list",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            resources: [],
            translations: [],
            listFor: { __typename: "RoutineVersion", id: "mock-routine" },
        },
    },
    {
        __typename: "Resource",
        id: "4",
        link: `http://localhost:3000${LINKS.User}/123`,
        usedFor: ResourceUsedFor.Context,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "ResourceTranslation",
            id: "4-en",
            language: "en",
            name: "Example User",
            description: "A Vrooli user profile",
        }],
        list: {
            __typename: "ResourceList",
            id: "mock-list",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            resources: [],
            translations: [],
            listFor: { __typename: "RoutineVersion", id: "mock-routine" },
        },
    },
    {
        __typename: "Resource",
        id: "5",
        link: `http://localhost:3000${LINKS.Team}/456`,
        usedFor: ResourceUsedFor.Context,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "ResourceTranslation",
            id: "5-en",
            language: "en",
            name: "Example Team",
            description: "A Vrooli team page",
        }],
        list: {
            __typename: "ResourceList",
            id: "mock-list",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            resources: [],
            translations: [],
            listFor: { __typename: "RoutineVersion", id: "mock-routine" },
        },
    },
];

// Create extended list by repeating the base resources 5 times with different IDs
const extendedMockResources: Resource[] = Array.from({ length: 5 }).flatMap((_, i) =>
    baseMockResources.map((resource, j) => ({
        ...resource,
        id: `${i * baseMockResources.length + j + 1}`,
        translations: [{
            ...resource.translations[0],
            id: `${i * baseMockResources.length + j + 1}-en`,
            name: `${resource.translations[0].name} ${i + 1}`,
        }],
    })),
);

const mockList: ResourceListType = {
    __typename: "ResourceList",
    id: "mock-list",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    resources: baseMockResources,
    translations: [],
    listFor: { __typename: "RoutineVersion", id: "mock-routine" },
};

const controlsContainerStyle = {
    marginBottom: 4,
    padding: 2,
    border: 1,
    borderColor: "divider",
    borderRadius: 1,
} as const;

const controlsRowStyle = {
    display: "flex",
    gap: 2,
    alignItems: "center",
    marginBottom: 2,
} as const;

const controlLabelStyle = {
    minWidth: 120,
} as const;

/**
 * Interactive story showcasing ResourceList with configurable options
 */
export function Interactive() {
    const [horizontal, setHorizontal] = useState(true);
    const [canUpdate, setCanUpdate] = useState(true);
    const [loading, setLoading] = useState(false);
    const [listState, setListState] = useState<"short" | "long" | "empty">("short");
    const [list, setList] = useState<ResourceListType>(mockList);

    function handleUpdate(newList: ResourceListType) {
        setList(newList);
    }

    function handleHorizontalClick() {
        setHorizontal(true);
    }
    function handleVerticalClick() {
        setHorizontal(false);
    }
    function handleCanUpdateClick() {
        setCanUpdate(!canUpdate);
    }
    function handleLoadingClick() {
        setLoading(!loading);
    }
    function handleListLengthClick() {
        // Cycle through states: short -> long -> empty -> short
        const nextState = listState === "short" ? "long" : listState === "long" ? "empty" : "short";
        setListState(nextState);
        setList({
            ...list,
            resources: nextState === "long" ? extendedMockResources :
                nextState === "short" ? baseMockResources :
                    [],
        });
    }

    return (
        <PageContainer>
            <ScrollBox>
                {/* Controls */}
                <Box sx={controlsContainerStyle}>
                    <Typography variant="h6" gutterBottom>Controls</Typography>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Layout:</Typography>
                        <Stack direction="row" spacing={1}>
                            <Button
                                variant={horizontal ? "contained" : "outlined"}
                                onClick={handleHorizontalClick}
                            >
                                Horizontal
                            </Button>
                            <Button
                                variant={!horizontal ? "contained" : "outlined"}
                                onClick={handleVerticalClick}
                            >
                                Vertical
                            </Button>
                        </Stack>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>List Length:</Typography>
                        <Button
                            variant="contained"
                            onClick={handleListLengthClick}
                            color={listState === "empty" ? "error" : "primary"}
                        >
                            {listState === "long" ? "Long List" :
                                listState === "short" ? "Short List" :
                                    "Empty List"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Can Update:</Typography>
                        <Button
                            variant={canUpdate ? "contained" : "outlined"}
                            onClick={handleCanUpdateClick}
                        >
                            {canUpdate ? "Enabled" : "Disabled"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Loading:</Typography>
                        <Button
                            variant={loading ? "contained" : "outlined"}
                            onClick={handleLoadingClick}
                        >
                            {loading ? "Loading" : "Loaded"}
                        </Button>
                    </Box>
                </Box>

                {/* ResourceList */}
                <ResourceList
                    horizontal={horizontal}
                    canUpdate={canUpdate}
                    loading={loading}
                    list={{
                        ...list,
                        resources: loading ? [] : list.resources,
                    }}
                    handleUpdate={handleUpdate}
                    parent={{ __typename: "RoutineVersion", id: "mock-routine" }}
                />
            </ScrollBox>
        </PageContainer>
    );
}
Interactive.parameters = {
    docs: {
        description: {
            story: "An interactive demo of the ResourceList component with configurable layout, list length, update permissions, and loading states.",
        },
    },
}; 
