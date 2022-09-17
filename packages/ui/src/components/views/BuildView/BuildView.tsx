import { Box, IconButton, Palette, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, SubroutineInfoDialog, SubroutineSelectOrCreateDialog, AddAfterLinkDialog, AddBeforeLinkDialog, EditableLabel, UnlinkedNodesDialog, BuildInfoDialog, HelpButton, userFromSession } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation } from '@apollo/client';
import { routineCreateMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { deleteArrayIndex, BuildAction, BuildRunState, Status, updateArray, getTranslation, getUserLanguages, parseSearchParams, shapeRoutineUpdate, shapeRoutineCreate, NodeShape, NodeLinkShape, PubSub, getRoutineStatus, initializeRoutine, addSearchParams, keepSearchParams, TagShape, RoutineTranslationShape } from 'utils';
import { Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, ResourceList, Routine, Run } from 'types';
import { useLocation } from '@shared/route';
import { APP_LINKS, ResourceListUsedFor } from '@shared/consts';
import { isEqual } from '@shared/utils';
import { NodeType } from 'graphql/generated/globalTypes';
import { ObjectAction } from 'components/dialogs/types';
import { BuildViewProps } from '../types';
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { StatusMessageArray } from 'components/buttons/types';
import { StatusButton } from 'components/buttons';
import { routineUpdate, routineUpdateVariables } from 'graphql/generated/routineUpdate';
import { routineCreate, routineCreateVariables } from 'graphql/generated/routineCreate';
import { MoveNodeMenu as MoveNodeDialog } from 'components/graphs/NodeGraph/MoveNodeDialog/MoveNodeDialog';
import { AddLinkIcon, CloseIcon, CompressIcon, EditIcon, RedoIcon, UndoIcon } from '@shared/icons';
import { requiredErrorMessage, routineUpdateForm as validationSchema, title as titleValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { RelationshipsObject } from 'components/inputs/types';

const helpText =
    `## What am I looking at?
This is the **Build** page. Here you can create and edit multi-step routines.

## What is a routine?
A *routine* is simply a process for completing a task, which takes inputs, performs some action, and outputs some results. By connecting multiple routines together, you can perform arbitrarily complex tasks.

All valid multi-step routines have a *start* node and at least one *end* node. Each node inbetween stores a list of subroutines, which can be optional or required.

When a user runs the routine, they traverse the routine graph from left to right. Each subroutine is rendered as a page, with components such as TextFields for each input. Where the graph splits, users are given a choice of which subroutine to go to next.

## How do I build a multi-step routine?
If you are starting from scratch, you will see a *start* node, a *routine list* node, and an *end* node.

You can press the routine list node to toggle it open/closed. The *open* stats allows you to select existing subroutines from Vrooli, or create a new one.

Each link connecting nodes has a circle. Pressing this circle opens a popup menu with options to insert a node, split the graph, or delete the link.

You also have the option to *unlink* nodes. These are stored on the top status bar - along with the status indicator, a button to clean up the graph, a button to add a new link, this help button, and an info button that sets overall routine information.

At the bottom of the screen, there is a slider to control the scale of the graph, and buttons to create/update and cancel the routine.
`

const commonButtonProps = (palette: Palette) => ({
    background: palette.secondary.main,
    marginRight: 1,
    transition: 'brightness 0.2s ease-in-out',
    '&:hover': {
        filter: `brightness(120%)`,
        background: palette.secondary.main,
    },
})

/**
 * Generates a new link object, but doesn't add it to the routine
 * @param fromId - The ID of the node the link is coming from
 * @param toId - The ID of the node the link is going to
 * @returns The new link object
 */
const generateNewLink = (fromId: string, toId: string): NodeLinkShape => ({
    __typename: 'NodeLink',
    id: uuid(),
    fromId,
    toId,
})

export const BuildView = ({
    handleClose,
    loading,
    onChange,
    routine,
    session,
    zIndex,
}: BuildViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const id: string = useMemo(() => routine?.id ?? '', [routine]);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    /**
     * On page load, check if editingd
     */
    useEffect(() => {
        const searchParams = parseSearchParams(window.location.search);
        const routineId = id.length > 0 ? id : window.location.pathname.split('/').pop();
        // Editing if specified in search params, or id not set (new routine)
        if (searchParams.edit || !routineId || !uuidValidate(routineId)) {
            setIsEditing(true);
        }
    }, [id]);

    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    // Routine mutators
    const [routineCreate] = useMutation<routineCreate, routineCreateVariables>(routineCreateMutation);
    const [routineUpdate] = useMutation<routineUpdate, routineUpdateVariables>(routineUpdateMutation);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<StatusMessageArray>({ status: Status.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const canEdit = useMemo<boolean>(() => routine?.permissionsRoutine?.canEdit === true, [routine?.permissionsRoutine?.canEdit]);

    // Stores previous routine states for undo/redo
    const [changeStack, setChangeStack] = useState<Routine[]>([]);
    const [changeStackIndex, setChangeStackIndex] = useState<number>(0);
    const clearChangeStack = useCallback((routine: Routine | null) => {
        setChangeStack(routine ? [routine] : []);
        setChangeStackIndex(routine ? 0 : -1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setChangedRoutine(routine);
    }, [setChangeStack, setChangeStackIndex]);
    /**
     * Moves back one in the change stack
     */
    const undo = useCallback(() => {
        if (changeStackIndex > 0) {
            setChangeStackIndex(changeStackIndex - 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            setChangedRoutine(changeStack[changeStackIndex - 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutine]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.length > 0, [changeStackIndex, changeStack]);
    /**
     * Moves forward one in the change stack
     */
    const redo = useCallback(() => {
        if (changeStackIndex < changeStack.length - 1) {
            setChangeStackIndex(changeStackIndex + 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            setChangedRoutine(changeStack[changeStackIndex + 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutine]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.length - 1 && changeStack.length > 0, [changeStackIndex, changeStack]);
    /**
     * Adds, to change stack, and removes anything from the change stack after the current index
     */
    const addToChangeStack = useCallback((changedRoutine: Routine) => {
        const newChangeStack = [...changeStack];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(changedRoutine);
        setChangeStack(newChangeStack);
        setChangeStackIndex(newChangeStack.length - 1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setChangedRoutine(changedRoutine);
    }, [changeStack, changeStackIndex, setChangeStack, setChangeStackIndex, setChangedRoutine]);

    // Handle undo and redo keys
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // CTRL + Y or CTRL + SHIFT + Z = redo
            if (e.ctrlKey && (e.key === 'y' || e.key === 'Z')) { redo() }
            // CTRL + Z = undo
            else if (e.ctrlKey && e.key === 'z') { undo() }
        };
        // Attach the event listener
        document.addEventListener('keydown', handleKeyDown);
        // Remove the event listener
        return () => { document.removeEventListener('keydown', handleKeyDown) };
    }, [redo, undo]);

    useEffect(() => {
        clearChangeStack(routine);
        // Update language
        if (routine) {
            const userLanguages = getUserLanguages(session);
            const routineLanguages = routine?.translations?.map(t => t.language)?.filter(l => typeof l === 'string' && l.length > 1) ?? [];
            // Find the first language in the user's languages that is also in the routine's languages
            const lang = userLanguages.find(l => routineLanguages.includes(l));
            if (lang) setLanguage(lang);
            else if (routineLanguages.length > 0) setLanguage(routineLanguages[0]);
            else setLanguage(userLanguages[0]);
        }
    }, [clearChangeStack, routine, session]);

    // Handle translations
    type Translation = RoutineTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        setRelationships({
            ...relationships,
            ...newRelationshipsObject,
        });
    }, [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({ 
            isComplete: routine?.isComplete ?? false,
            isPrivate: routine?.isPrivate ?? false,
            owner: routine?.owner ?? null,
            parent: null,
            // parent: routine?.parent ?? null, TODO
            project: null, //TODO
        });
        setResourceList(routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(routine?.tags ?? []);
        setTranslations(routine?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            instructions: t.instructions ?? '',
            title: t.title ?? '',
        })) ?? []);
    }, [routine]);

    // Formik validation
    const formik = useFormik({
        initialValues: {
            description: getTranslation(routine, 'description', [language]) ?? '',
            instructions: getTranslation(routine, 'instructions', [language]) ?? '',
            isInternal: routine?.isInternal ?? false,
            title: getTranslation(routine, 'title', [language]) ?? '',
            version: routine?.version ?? '1.0.0',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: routine?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            // If routine is new, create it
            if (!uuidValidate(id)) {
                if (!changedRoutine) {
                    PubSub.get().publishSnack({ message: 'Cannot update: Invalid routine data', severity: 'error' });
                    return;
                }
                mutationWrapper({
                    mutation: routineCreate,
                    input: shapeRoutineCreate({
                        ...changedRoutine,
                        id: uuid(),
                        isInternal: values.isInternal,
                        isComplete: relationships.isComplete,
                        isPrivate: relationships.isPrivate,
                        owner: relationships.owner,
                        parent: relationships.parent as Routine | null | undefined,
                        version: values.version,
                        resourceLists: [resourceList],
                        tags: tags,
                        translations: allTranslations,
                    }),
                    successMessage: () => 'Routine created.',
                    onSuccess: ({ data }) => {
                        onChange(data.routineCreate);
                        setLocation(`${APP_LINKS.Routine}/${data.routineCreate.id}?build=true`)
                    },
                })
            }
            // Otherwise, update it
            else {
                // Don't update if no changes
                if (!changedRoutine || isEqual(routine, changedRoutine)) {
                    PubSub.get().publishSnack({ message: 'No changes detected', severity: 'error' });
                    return;
                }
                if (!routine || !changedRoutine.id) {
                    PubSub.get().publishSnack({ message: 'Cannot update: Invalid routine data', severity: 'error' });
                    return;
                }
                mutationWrapper({
                    mutation: routineUpdate,
                    input: shapeRoutineUpdate(routine, {
                        ...changedRoutine,
                        isInternal: values.isInternal,
                        isComplete: relationships.isComplete,
                        isPrivate: relationships.isPrivate,
                        owner: relationships.owner,
                        parent: relationships.parent as Routine | null | undefined,
                        version: values.version,
                        resourceLists: [resourceList],
                        tags: tags,
                        translations: allTranslations,
                    }),
                    successMessage: () => 'Routine updated.',
                    onSuccess: ({ data }) => {
                        onChange(data.routineUpdate);
                        keepSearchParams(setLocation, ['build']);
                        setIsEditing(false);
                    },
                })
            }
        },
    });

    /**
     * Calculates:
     * - 2D array of positioned nodes data (to represent columns and rows)
     * - 1D array of unpositioned nodes data
     * - dictionary of positioned node IDs to their data
     * Also sets the status of the routine (valid/invalid/incomplete)
     */
    const { columns, nodesOffGraph, nodesById } = useMemo(() => {
        if (!changedRoutine) return { columns: [], nodesOffGraph: [], nodesById: {} };
        const { messages, nodesById, nodesOnGraph, nodesOffGraph, status } = getRoutineStatus(changedRoutine);
        // Check for critical errors
        if (messages.includes('No node or link data found')) {
            // Create new routine data
            const initialized = initializeRoutine(language);
            // Set empty nodes and links
            setChangedRoutine({
                ...changedRoutine,
                nodes: initialized.nodes,
                nodeLinks: initialized.nodeLinks,
            });
            return { columns: [], nodesOffGraph: [], nodesById: {} };
        }
        if (messages.includes('Ran into error determining node positions')) {
            // Remove all node positions and links
            setChangedRoutine({
                ...changedRoutine,
                nodes: changedRoutine.nodes.map(n => ({ ...n, columnIndex: null, rowIndex: null })),
                nodeLinks: [],
            })
            return { columns: [], nodesOffGraph: changedRoutine.nodes, nodesById: {} };
        }
        // Update status
        setStatus({ status, messages });
        // Remove any links which reference unlinked nodes
        const goodLinks = changedRoutine.nodeLinks.filter(link => !nodesOffGraph.some(node => node.id === link.fromId || node.id === link.toId));
        // If routine was mutated, update the routine
        const finalNodes = [...nodesOnGraph, ...nodesOffGraph]
        const haveNodesChanged = !isEqual(finalNodes, changedRoutine.nodes);
        const haveLinksChanged = !isEqual(goodLinks, changedRoutine.nodeLinks);
        if (haveNodesChanged || haveLinksChanged) {
            setChangedRoutine({
                ...changedRoutine,
                nodes: finalNodes,
                nodeLinks: goodLinks,
            })
        }
        // Create 2D node data array, ordered by column. Each column is ordered by row index
        const columns: Node[][] = [];
        // Loop through positioned nodes
        for (const node of nodesOnGraph) {
            // Skips nodes without a columnIndex or rowIndex
            if ((node.columnIndex === null || node.columnIndex === undefined) || (node.rowIndex === null || node.rowIndex === undefined)) continue;
            // Add new column(s) if necessary
            while (columns.length <= node.columnIndex) {
                columns.push([]);
            }
            // Add node to column
            columns[node.columnIndex].push(node);
        }
        // Now sort each column by row index
        for (const column of columns) {
            column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
        }
        // Add one empty column to the end, so nodes, can be dragged to the end of the graph
        columns.push([]);
        // Return
        return { columns, nodesOffGraph, nodesById };
    }, [changedRoutine, language]);

    // Add subroutine dialog
    const [addSubroutineNode, setAddSubroutineNode] = useState<string | null>(null);
    const closeAddSubroutineDialog = useCallback(() => { setAddSubroutineNode(null); }, []);
    // "Add after" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addAfterLinkNode, setAddAfterLinkNode] = useState<string | null>(null);
    const closeAddAfterLinkDialog = useCallback(() => { setAddAfterLinkNode(null); }, []);
    // "Add before" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addBeforeLinkNode, setAddBeforeLinkNode] = useState<string | null>(null);
    const closeAddBeforeLinkDialog = useCallback(() => { setAddBeforeLinkNode(null); }, []);

    // Subroutine info drawer
    const [openedSubroutine, setOpenedSubroutine] = useState<{ node: NodeDataRoutineList, routineItemId: string } | null>(null);
    const handleSubroutineOpen = useCallback((nodeId: string, subroutineId: string) => {
        const node = nodesById[nodeId];
        if (node) setOpenedSubroutine({ node: (node.data as NodeDataRoutineList), routineItemId: subroutineId });
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => {
        setOpenedSubroutine(null);
    }, []);

    const [isLinkDialogOpen, setIsLinkDialogOpen] = useState(false);
    const [linkDialogFrom, setLinkDialogFrom] = useState<Node | null>(null);
    const [linkDialogTo, setLinkDialogTo] = useState<Node | null>(null);
    const openLinkDialog = useCallback(() => setIsLinkDialogOpen(true), []);
    const handleLinkDialogClose = useCallback((link?: NodeLink) => {
        setLinkDialogFrom(null);
        setLinkDialogTo(null);
        setIsLinkDialogOpen(false);
        if (!changedRoutine) return;
        // If no link data, return
        if (!link) return;
        // Upsert link
        const newLinks = [...changedRoutine.nodeLinks];
        const existingLinkIndex = newLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
        if (existingLinkIndex >= 0) {
            newLinks[existingLinkIndex] = { ...link } as NodeLink;
        } else {
            newLinks.push(link as NodeLink);
        }
        addToChangeStack({
            ...changedRoutine,
            nodeLinks: newLinks,
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Deletes a link, without deleting any nodes.
     */
    const handleLinkDelete = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        addToChangeStack({
            ...changedRoutine,
            nodeLinks: changedRoutine.nodeLinks.filter(l => l.id !== link.id),
        });
    }, [addToChangeStack, changedRoutine]);

    const handleScaleChange = (newScale: number) => {
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setScale(newScale)
    };
    const handleScaleDelta = useCallback((delta: number) => {
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setScale(s => Math.max(0.25, Math.min(1, s + delta)));
    }, []);

    const handleRunDelete = useCallback((run: Run) => {
        if (!changedRoutine) return;
        // Doesn't affect change stack
        setChangedRoutine({
            ...changedRoutine,
            runs: changedRoutine.runs.filter(r => r.id !== run.id),
        });
    }, [changedRoutine]);

    const handleRunAdd = useCallback((run: Run) => {
        if (!changedRoutine) return;
        // Doesn't affect change stack
        setChangedRoutine({
            ...changedRoutine,
            runs: [run, ...changedRoutine.runs],
        });
    }, [changedRoutine]);

    const startEditing = useCallback(() => {
        addSearchParams(setLocation, { edit: true });
        setIsEditing(true);
    }, [setLocation]);

    /**
     * On page leave, check if routine has changed. 
     * If so, prompt user to save changes
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (isEditing && changeStack.length > 1) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [changeStack.length, isEditing, routine]);

    const updateRoutineTitle = useCallback((title: string) => {
        if (!changedRoutine) return;
        const newTranslations = [...changedRoutine.translations.map(t => {
            if (t.language === language) {
                return { ...t, title }
            }
            return { ...t }
        })];
        addToChangeStack({
            ...changedRoutine,
            translations: newTranslations
        });
    }, [addToChangeStack, changedRoutine, language]);

    const revertChanges = useCallback(() => {
        // Helper function to revert changes
        const revert = () => {
            // If updating routine, revert to original routine
            if (id) {
                clearChangeStack(routine);
                setIsEditing(false);
            }
            // If adding new routine, go back
            else window.history.back();
        }
        // Confirm if changes have been made
        if (changeStack.length > 1) {
            PubSub.get().publishAlertDialog({
                message: 'There are unsaved changes. Are you sure you would like to cancel?',
                buttons: [
                    {
                        text: 'Yes', onClick: () => { revert(); }
                    },
                    {
                        text: "No", onClick: () => { }
                    },
                ]
            });
        } else {
            revert();
        }
    }, [changeStack.length, clearChangeStack, id, routine])

    /**
     * If closing with unsaved changes, prompt user to save
     */
    const onClose = useCallback(() => {
        if (isEditing) {
            revertChanges();
        } else {
            keepSearchParams(setLocation, []);
            if (!uuidValidate(id)) window.history.back();
            else handleClose();
        }
    }, [handleClose, id, isEditing, setLocation, revertChanges]);

    /**
     * Calculates the new set of links for an routine when a node is 
     * either deleted or unlinked. In certain cases, the new links can be 
     * calculated automatically.
     * @param nodeId - The ID of the node which is being deleted or unlinked
     * @param currLinks - The current set of links
     * @returns The new set of links
     */
    const calculateLinksAfterNodeRemove = useCallback((nodeId: string): NodeLink[] => {
        if (!changedRoutine) return [];
        const deletingLinks = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId || l.toId === nodeId);
        const newLinks: NodeLinkShape[] = [];
        // Find all "from" and "to" nodes in the deleting links
        const fromNodeIds = deletingLinks.map(l => l.fromId).filter(id => id !== nodeId);
        const toNodeIds = deletingLinks.map(l => l.toId).filter(id => id !== nodeId);
        // If there is only one "from" node, create a link between it and every "to" node
        if (fromNodeIds.length === 1) {
            toNodeIds.forEach(toId => { newLinks.push(generateNewLink(fromNodeIds[0], toId)) });
        }
        // If there is only one "to" node, create a link between it and every "from" node
        else if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push(generateNewLink(fromId, toNodeIds[0])) });
        }
        // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // Delete old links
        let keptLinks = changedRoutine.nodeLinks.filter(l => !deletingLinks.includes(l));
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, [changedRoutine]);

    /**
     * Finds the closest node position available to the given position
     * @param column - The preferred column
     * @param row - The preferred row
     * @returns a node position in the same column, with the first available row starting at the given row
     */
    const closestOpenPosition = useCallback((
        column: number | null,
        row: number | null
    ): { columnIndex: number, rowIndex: number } => {
        if (column === null || row === null) return { columnIndex: -1, rowIndex: -1 };
        const columnNodes = changedRoutine?.nodes?.filter(n => n.columnIndex === column) ?? [];
        let rowIndex: number = row;
        // eslint-disable-next-line no-loop-func
        while (columnNodes.some(n => n.rowIndex !== null && n.rowIndex === rowIndex) && rowIndex <= 100) {
            rowIndex++;
        }
        if (rowIndex > 100) return { columnIndex: -1, rowIndex: -1 };
        return { columnIndex: column, rowIndex };
    }, [changedRoutine?.nodes]);

    /**
     * Generates a new routine list node object, but doesn't add it to the routine
     * @param column Suggested column for the node
     * @param row Suggested row for the node
     */
    const createRoutineListNode = useCallback((column: number | null, row: number | null) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNode: Omit<NodeShape, 'routineId'> = {
            __typename: 'Node',
            id: uuid(),
            type: NodeType.RoutineList,
            rowIndex,
            columnIndex,
            data: {
                id: uuid(),
                __typename: 'NodeRoutineList',
                isOrdered: false,
                isOptional: false,
                routines: [],
            },
            // Generate unique placeholder title
            translations: [{
                __typename: 'NodeTranslation',
                id: uuid(),
                language,
                title: `Node ${(changedRoutine?.nodes?.length ?? 0) - 1}`,
                description: '',
            }],
        }
        return newNode;
    }, [closestOpenPosition, language, changedRoutine?.nodes?.length]);

    /**
     * Generates new end node object, but doesn't add it to the routine
     * @param column Suggested column for the node
     * @param row Suggested row for the node
     */
    const createEndNode = useCallback((column: number | null, row: number | null) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNode: Omit<NodeShape, 'routineId'> = {
            __typename: 'Node',
            id: uuid(),
            type: NodeType.End,
            rowIndex,
            columnIndex,
            data: {
                id: uuid(),
                wasSuccessful: true,
            },
            translations: []
        }
        return newNode;
    }, [closestOpenPosition]);

    /**
     * Creates a link between two nodes which already exist in the linked routine. 
     * This assumes that the link is valid.
     */
    const handleLinkCreate = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        addToChangeStack({
            ...changedRoutine,
            nodeLinks: [...changedRoutine.nodeLinks, link]
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Updates an existing link between two nodes
     */
    const handleLinkUpdate = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.id === link.id);
        if (linkIndex === -1) return;
        addToChangeStack({
            ...changedRoutine,
            nodeLinks: updateArray(changedRoutine.nodeLinks, linkIndex, link),
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Deletes a node, and all links connected to it. 
     * Also attemps to create new links to replace the deleted links.
     */
    const handleNodeDelete = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const linksList = calculateLinksAfterNodeRemove(nodeId);
        addToChangeStack({
            ...changedRoutine,
            nodes: deleteArrayIndex(changedRoutine.nodes, nodeIndex),
            nodeLinks: linksList,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutine]);

    /**
     * Deletes a subroutine from a node
     */
    const handleSubroutineDelete = useCallback((nodeId: string, subroutineId: string) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const node = changedRoutine.nodes[nodeIndex];
        const subroutineIndex = (node.data as NodeDataRoutineList).routines.findIndex((item: NodeDataRoutineListItem) => item.id === subroutineId);
        if (subroutineIndex === -1) return;
        const newRoutineList = deleteArrayIndex((node.data as NodeDataRoutineList).routines, subroutineIndex);
        addToChangeStack({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...node,
                data: {
                    ...node.data,
                    routines: newRoutineList,
                }
            }),
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Drops or unlinks a node
     */
    const handleNodeDrop = useCallback((nodeId: string, columnIndex: number | null, rowIndex: number | null) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        // If columnIndex and rowIndex null, then it is being unlinked
        if (columnIndex === null && rowIndex === null) {
            const linksList = calculateLinksAfterNodeRemove(nodeId);
            addToChangeStack({
                ...changedRoutine,
                nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                    ...changedRoutine.nodes[nodeIndex],
                    rowIndex: null,
                    columnIndex: null,
                }),
                nodeLinks: linksList,
            });
            return;
        }
        // If one or the other is null, then there must be an error
        if (columnIndex === null || rowIndex === null) {
            PubSub.get().publishSnack({ message: 'Error: Invalid drop location.', severity: 'errror' });
            return;
        }
        // Otherwise, is a drop
        let updatedNodes = [...changedRoutine.nodes];
        // If dropped into the first column, then shift everything that's not the start node to the right
        if (columnIndex === 0) {
            updatedNodes = updatedNodes.map(n => {
                if (n.rowIndex === null || n.columnIndex === null || n.columnIndex === 0) return n;
                return {
                    ...n,
                    columnIndex: n.columnIndex + 1,
                }
            });
            // Update dropped node
            updatedNodes = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                columnIndex: 1,
                rowIndex,
            });
        }
        // If dropped into the same column the node started in, either shift or swap
        else if (columnIndex === changedRoutine.nodes[nodeIndex].columnIndex) {
            // Find and order nodes in the same column, which are above (or at the same position as) the dropped node
            const nodesAbove = changedRoutine.nodes.filter(n =>
                n.columnIndex === columnIndex &&
                n.rowIndex !== null &&
                n.rowIndex <= rowIndex
            ).sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            // If no nodes above, then shift everything in the column down by 1
            if (nodesAbove.length === 0) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex) return n;
                    return {
                        ...n,
                        rowIndex: n.rowIndex + 1,
                    }
                });
            }
            // Otherwise, swap with the last node in the above list
            else {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex) return n;
                    if (n.id === nodeId) return {
                        ...n,
                        rowIndex: nodesAbove[nodesAbove.length - 1].rowIndex,
                    }
                    if (n.rowIndex === nodesAbove[nodesAbove.length - 1].rowIndex) return {
                        ...n,
                        rowIndex: changedRoutine.nodes[nodeIndex].rowIndex,
                    }
                    return n;
                });
            }
        }
        // Otherwise, treat as a normal drop
        else {
            // If dropped into an existing column, shift rows in dropped column that are below the dropped node
            if (changedRoutine.nodes.some(n => n.columnIndex === columnIndex)) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex === columnIndex && n.rowIndex !== null && n.rowIndex >= rowIndex) {
                        return { ...n, rowIndex: n.rowIndex + 1 }
                    }
                    return n;
                });
            }
            // If the column the node was from is now empty, then shift all columns after it.
            const originalColumnIndex = changedRoutine.nodes[nodeIndex].columnIndex;
            const isRemovingColumn = originalColumnIndex !== null && changedRoutine.nodes.filter(n => n.columnIndex === originalColumnIndex).length === 1;
            if (isRemovingColumn) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex !== null && n.columnIndex > originalColumnIndex) {
                        return { ...n, columnIndex: n.columnIndex - 1 }
                    }
                    return n;
                });
            }
            updatedNodes = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                columnIndex: (isRemovingColumn && originalColumnIndex < columnIndex) ?
                    columnIndex - 1 :
                    columnIndex,
                rowIndex,
            })
        }
        // Update the routine
        addToChangeStack({
            ...changedRoutine,
            nodes: updatedNodes,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutine]);

    // Move node dialog for context menu (mainly for accessibility)
    const [moveNode, setMoveNode] = useState<Node | null>(null);
    const closeMoveNodeDialog = useCallback((newPosition?: { columnIndex: number, rowIndex: number }) => {
        if (newPosition && moveNode) {
            handleNodeDrop(moveNode.id, newPosition.columnIndex, newPosition.rowIndex);
        }
        setMoveNode(null);
    }, [handleNodeDrop, moveNode]);

    /**
     * Updates a node's data
     */
    const handleNodeUpdate = useCallback((node: Node) => {
        console.log('handlenodeupdate starttt', node)
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === node.id);
        if (nodeIndex === -1) return;
        addToChangeStack({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, node),
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Inserts a new routine list node along an edge
     */
    const handleNodeInsert = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        // Find link index
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
        // Delete link
        const linksList = deleteArrayIndex(changedRoutine.nodeLinks, linkIndex);
        // Find "to" node. New node will be placed in its row and column
        const toNode = changedRoutine.nodes.find(n => n.id === link.toId);
        if (!toNode) {
            PubSub.get().publishSnack({ message: 'Error occurred.', severity: 'Error' });
            return;
        }
        const { columnIndex, rowIndex } = toNode;
        // Move every node starting from the "to" node to the right by one
        const nodesList = changedRoutine.nodes.map(n => {
            if (n.columnIndex !== null && n.columnIndex !== undefined && n.columnIndex >= (columnIndex ?? 0)) {
                return { ...n, columnIndex: n.columnIndex + 1 };
            }
            return n;
        });
        // Create new routine list node
        const newNode: Omit<NodeShape, 'routineId'> = createRoutineListNode(columnIndex, rowIndex);
        // Find every node 
        // Create two new links
        const newLinks: NodeLinkShape[] = [
            generateNewLink(link.fromId, newNode.id),
            generateNewLink(newNode.id, link.toId),
        ];
        // Insert new node and links
        const newRoutine = {
            ...changedRoutine,
            nodes: [...nodesList, newNode as any],
            nodeLinks: [...linksList, ...newLinks as any],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutine, createRoutineListNode]);

    /**
     * Inserts a new routine list node, with its own branch
     */
    const handleBranchInsert = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        // Find "to" node. New node will be placed in its column
        const toNode = changedRoutine.nodes.find(n => n.id === link.toId);
        if (!toNode) {
            PubSub.get().publishSnack({ message: 'Error occurred.', severity: 'Error' });
            return;
        }
        // Find the largest row index in the column. New node will be placed in the next row
        const maxRowIndex = changedRoutine.nodes.filter(n => n.columnIndex === toNode.columnIndex).map(n => n.rowIndex).reduce((a, b) => Math.max(a ?? 0, b ?? 0), 0);
        const newNode: Omit<NodeShape, 'routineId'> = createRoutineListNode(toNode.columnIndex, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Since this is a new branch, we also need to add an end node after the new node
        const newEndNode: Omit<NodeShape, 'routineId'> = createEndNode((toNode.columnIndex ?? 0) + 1, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Create new link, going from the "from" node to the new node
        const newLink: NodeLinkShape = generateNewLink(link.fromId, newNode.id);
        // Create new link, going from the new node to the end node
        const newEndLink: NodeLinkShape = generateNewLink(newNode.id, newEndNode.id);
        // Insert new nodes and links
        const newRoutine = {
            ...changedRoutine,
            nodes: [...changedRoutine.nodes, newNode as any, newEndNode as any],
            nodeLinks: [...changedRoutine.nodeLinks, newLink as any, newEndLink as any],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutine, createEndNode, createRoutineListNode]);

    /**
     * Adds a subroutine routine list
     */
    const handleSubroutineAdd = useCallback((nodeId: string, routine: Routine) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        let routineItem: NodeDataRoutineListItem = {
            id: uuid(),
            index: routineList.routines.length,
            isOptional: true,
            routine,
        } as any
        if (routineList.isOrdered) routineItem.index = routineList.routines.length
        addToChangeStack({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                data: {
                    ...routineList,
                    routines: [...routineList.routines, routineItem],
                }
            }),
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Reoders a subroutine in a routine list item
     * @param nodeId The node id of the routine list item
     * @param oldIndex The old index of the subroutine
     * @param newIndex The new index of the subroutine
     */
    const handleSubroutineReorder = useCallback((nodeId: string, oldIndex: number, newIndex: number) => {
        // Find routines being swapped
        if (!changedRoutine) return;
        // Node containing routine list data with ID nodeId
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.data?.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        const routines = [...routineList.routines];
        // Find subroutines matching old and new index
        const aIndex = routines.findIndex(r => r.index === oldIndex);
        const bIndex = routines.findIndex(r => r.index === newIndex);
        if (aIndex === -1 || bIndex === -1) return;
        // Swap the routine indexes
        routines[aIndex] = { ...routines[aIndex], index: newIndex };
        routines[bIndex] = { ...routines[bIndex], index: oldIndex };
        // Update the routine list
        addToChangeStack({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                data: {
                    ...routineList,
                    routines,
                }
            }),
        });
    }, [addToChangeStack, changedRoutine]);

    /**
     * Add a new end node AFTER a node
     */
    const handleAddEndAfter = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        // Find links where this node is the "from" node
        const links = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId);
        // If multiple links, open a dialog to select which one to add after
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        // If only one link, add after that link
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            const node = changedRoutine.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, 'routineId'> = createEndNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink: NodeLinkShape = generateNewLink(nodeId, newNode.id);
            addToChangeStack({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutine, createEndNode, handleNodeInsert]);

    /**
     * Add a new routine list AFTER a node
     */
    const handleAddListAfter = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        // Find links where this node is the "from" node
        const links = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId);
        // If multiple links, open a dialog to select which one to add after
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        // If only one link, add after that link
        else if (links.length === 1) {
            handleNodeInsert(links[0]);
        }
        // If no links, create link and node
        else {
            const node = changedRoutine.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, 'routineId'> = createRoutineListNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink: NodeLinkShape = generateNewLink(nodeId, newNode.id);
            addToChangeStack({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutine, createRoutineListNode, handleNodeInsert]);

    /**
     * Add a new routine list BEFORE a node
     */
    const handleAddListBefore = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        // Find links where this node is the "to" node
        const links = changedRoutine.nodeLinks.filter(l => l.toId === nodeId);
        // If multiple links, open a dialog to select which one to add before
        if (links.length > 1) {
            setAddBeforeLinkNode(nodeId);
            return;
        }
        // If only one link, add before that link
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            const node = changedRoutine.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, 'routineId'> = createRoutineListNode((node.columnIndex ?? 1) - 1, (node.rowIndex ?? 0));
            const newLink: NodeLinkShape = generateNewLink(newNode.id, nodeId);
            addToChangeStack({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutine, createRoutineListNode, handleNodeInsert]);

    /**
     * Updates the current selected subroutine
     */
    const handleSubroutineUpdate = useCallback((updatedSubroutine: NodeDataRoutineListItem) => {
        if (!changedRoutine) return;
        // Update routine
        addToChangeStack({
            ...changedRoutine,
            nodes: changedRoutine.nodes.map((n: Node) => {
                if (n.type === NodeType.RoutineList && (n.data as NodeDataRoutineList).routines.some(r => r.id === updatedSubroutine.id)) {
                    return {
                        ...n,
                        data: {
                            ...n.data,
                            routines: (n.data as NodeDataRoutineList).routines.map(r => {
                                if (r.id === updatedSubroutine.id) {
                                    return {
                                        ...r,
                                        ...updatedSubroutine,
                                        routine: {
                                            ...r.routine,
                                            ...updatedSubroutine.routine,
                                        }
                                    };
                                }
                                return r;
                            }),
                        },
                    };
                }
                return n;
            }),
        } as any);
        // Close dialog
        closeRoutineInfo();
    }, [addToChangeStack, changedRoutine, closeRoutineInfo]);

    /**
     * Navigates to a subroutine's build page. Fist checks if there are unsaved changes
     */
    const handleSubroutineViewFull = useCallback(() => {
        if (!openedSubroutine) return;
        if (!isEqual(routine, changedRoutine)) {
            PubSub.get().publishSnack({ message: 'You have unsaved changes. Please save or discard them before navigating to another routine.' });
            return;
        }
        // TODO - buildview should have its own buildview, to recursively open subroutines
        //setLocation(`${APP_LINKS.Build}/${selectedSubroutine.id}`);
    }, [changedRoutine, openedSubroutine, routine]);

    const handleAction = useCallback((action: BuildAction, nodeId: string, subroutineId?: string) => {
        const node = changedRoutine?.nodes?.find(n => n.id === nodeId);
        switch (action) {
            case BuildAction.AddIncomingLink:
                setLinkDialogTo(node ?? null);
                setIsLinkDialogOpen(true);
                break;
            case BuildAction.AddOutgoingLink:
                setLinkDialogFrom(node ?? null);
                setIsLinkDialogOpen(true);
                break;
            case BuildAction.AddSubroutine:
                setAddSubroutineNode(nodeId);
                break;
            case BuildAction.DeleteNode:
                handleNodeDelete(nodeId);
                break;
            case BuildAction.DeleteSubroutine:
                handleSubroutineDelete(nodeId, subroutineId ?? '');
                break;
            case BuildAction.EditSubroutine:
                handleSubroutineOpen(nodeId, subroutineId ?? '');
                break;
            case BuildAction.OpenSubroutine:
                handleSubroutineOpen(nodeId, subroutineId ?? '');
                break;
            case BuildAction.UnlinkNode:
                handleNodeDrop(nodeId, null, null);
                break;
            case BuildAction.AddEndAfterNode:
                handleAddEndAfter(nodeId);
                break;
            case BuildAction.AddListAfterNode:
                handleAddListAfter(nodeId);
                break;
            case BuildAction.AddListBeforeNode:
                handleAddListBefore(nodeId);
                break;
            case BuildAction.MoveNode:
                if (node) setMoveNode(node);
                break;
        }
    }, [changedRoutine?.nodes, handleNodeDelete, handleSubroutineDelete, handleSubroutineOpen, handleNodeDrop, handleAddEndAfter, handleAddListAfter, handleAddListBefore]);

    const handleRoutineAction = useCallback((action: ObjectAction, data: any) => {
        switch (action) {
            case ObjectAction.Copy:
                setLocation(`${APP_LINKS.Routine}/${data.copy.routine.id}`);
                break;
            case ObjectAction.Delete:
                setLocation(APP_LINKS.Home);
                break;
            case ObjectAction.Fork:
                setLocation(`${APP_LINKS.Routine}/${data.fork.routine.id}`);
                break;
            case ObjectAction.Report:
                //TODO
                break;
            case ObjectAction.Share:
                //TODO
                break;
            case ObjectAction.Star:
            case ObjectAction.StarUndo:
                if (data.star.success) {
                    onChange({
                        ...routine,
                        isStarred: action === ObjectAction.Star,
                    } as any)
                }
                break;
            case ObjectAction.Stats:
                //TODO
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                if (data.vote.success) {
                    onChange({
                        ...routine,
                        isUpvoted: action === ObjectAction.VoteUp,
                    } as any)
                }
                break;
        }
    }, [setLocation, onChange, routine]);

    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    /**
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path. 
     * Also removes links that don't have both a valid fromId and toId.
     */
    const cleanUpGraph = useCallback(() => {
        if (!changedRoutine) return;
        const resultRoutine = JSON.parse(JSON.stringify(changedRoutine));
        // Loop through the columns, and remove gaps in rowIndex
        for (const column of columns) {
            // Sort nodes in column by rowIndex
            const sortedNodes = column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            // If the nodes don't go from 0 to n without any gaps
            if (sortedNodes.length > 0 && sortedNodes.some((n, i) => (n.rowIndex ?? 0) !== i)) {
                // Update nodes in resultRoutine with new rowIndexes
                const newNodes = sortedNodes.map((n, i) => ({
                    ...n,
                    rowIndex: i,
                }));
                // Replace nodes in resultRoutine
                resultRoutine.nodes = resultRoutine.nodes.map(oldNode => {
                    const newNode = newNodes.find(nn => nn.id === oldNode.id);
                    if (newNode) {
                        return newNode;
                    }
                    return oldNode;
                });
            }
        }
        // Find every node that does not have a link leaving it, which is also 
        // not an end node
        for (const node of resultRoutine.nodes) {
            // If not an end node
            if (node.type !== NodeType.End) {
                // Check if any links have a "fromId" matching this node's ID
                const leavingLinks = resultRoutine.nodeLinks.filter(link => link.fromId === node.id);
                // If there are no leaving links, create a new link and end node
                if (leavingLinks.length === 0) {
                    // Generate node ID
                    const newEndNodeId = uuid();
                    // Calculate rowIndex and columnIndex
                    // Column is 1 after current column
                    const columnIndex: number = (node.columnIndex ?? 0) + 1;
                    // Node is 1 after last rowIndex in column
                    const rowIndex = (columnIndex >= 0 && columnIndex < columns.length) ? columns[columnIndex].length : 0;
                    const newLink: NodeLinkShape = generateNewLink(node.id, newEndNodeId);
                    const newEndNode: Omit<NodeShape, 'routineId'> = {
                        __typename: 'Node',
                        id: newEndNodeId,
                        type: NodeType.End,
                        rowIndex,
                        columnIndex,
                        data: {
                            wasSuccessful: false,
                        } as any,
                        translations: [],
                    }
                    // Add link and end node to resultRoutine
                    resultRoutine.nodeLinks.push(newLink as any);
                    resultRoutine.nodes.push(newEndNode as any);
                }
            }
        }
        // Remove links that don't have both a valid fromId and toId
        resultRoutine.nodeLinks = resultRoutine.nodeLinks.filter(link => {
            const fromNode = resultRoutine.nodes.find(n => n.id === link.fromId);
            const toNode = resultRoutine.nodes.find(n => n.id === link.toId);
            return Boolean(fromNode && toNode);
        });
        // Update changedRoutine with resultRoutine
        addToChangeStack(resultRoutine);
    }, [addToChangeStack, changedRoutine, columns]);

    const editActions = useMemo(() => {
        if (!isEditing) return null;
        return (<Stack direction="row" spacing={1} sx={{
            zIndex: 2,
            height: '48px',
            background: 'transparent',
            color: palette.primary.contrastText,
            justifyContent: 'center',
            alignItems: 'center',
            paddingTop: '8px',
        }}>
            {(canUndo || canRedo) && <Tooltip title={canUndo ? 'Undo' : ''}>
                <IconButton
                    id="undo-button"
                    disabled={!canUndo}
                    onClick={undo}
                    aria-label="Undo"
                    sx={commonButtonProps(palette)}
                >
                    <UndoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                </IconButton>
            </Tooltip>}
            {(canUndo || canRedo) && <Tooltip title={canRedo ? 'Redo' : ''}>
                <IconButton
                    id="redo-button"
                    disabled={!canRedo}
                    onClick={redo}
                    aria-label="Redo"
                    sx={commonButtonProps(palette)}
                >
                    <RedoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                </IconButton>
            </Tooltip>}
            <Tooltip title='Clean up graph'>
                <IconButton
                    id="clean-graph-button"
                    onClick={cleanUpGraph}
                    aria-label='Clean up graph'
                    sx={commonButtonProps(palette)}
                >
                    <CompressIcon id="clean-up-button-icon" fill={palette.secondary.contrastText} />
                </IconButton>
            </Tooltip>
            <Tooltip title='Add new link'>
                <IconButton
                    id="add-link-button"
                    onClick={openLinkDialog}
                    aria-label='Add link'
                    sx={commonButtonProps(palette)}
                >
                    <AddLinkIcon id="add-link-button-icon" fill={palette.secondary.contrastText} />
                </IconButton>
            </Tooltip>
        </Stack>)
    }, [canRedo, canUndo, cleanUpGraph, isEditing, openLinkDialog, palette, redo, undo]);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Popup for adding new subroutines */}
            {addSubroutineNode && <SubroutineSelectOrCreateDialog
                handleAdd={handleSubroutineAdd}
                handleClose={closeAddSubroutineDialog}
                isOpen={Boolean(addSubroutineNode)}
                nodeId={addSubroutineNode}
                routineId={routine?.id ?? ''}
                session={session}
                zIndex={zIndex + 3}
            />}
            {/* Popup for editing existing multi-step subroutines */}
            {/* TODO */}
            {/* Popup for "Add after" dialog */}
            {addAfterLinkNode && <AddAfterLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddAfterLinkDialog}
                isOpen={Boolean(addAfterLinkNode)}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
                nodeId={addAfterLinkNode}
                session={session}
                zIndex={zIndex + 3}
            />}
            {/* Popup for "Add before" dialog */}
            {addBeforeLinkNode && <AddBeforeLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddBeforeLinkDialog}
                isOpen={Boolean(addBeforeLinkNode)}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
                nodeId={addBeforeLinkNode}
                session={session}
                zIndex={zIndex + 3}
            />}
            {/* Popup for creating new links */}
            {changedRoutine ? <LinkDialog
                handleClose={handleLinkDialogClose}
                handleDelete={handleLinkDelete}
                isAdd={true}
                isOpen={isLinkDialogOpen}
                language={language}
                link={undefined}
                nodeFrom={linkDialogFrom}
                nodeTo={linkDialogTo}
                routine={changedRoutine}
                zIndex={zIndex + 3}
            // partial={ }
            /> : null}
            {/* Popup for moving nodes */}
            {moveNode && <MoveNodeDialog
                handleClose={closeMoveNodeDialog}
                isOpen={Boolean(moveNode)}
                language={language}
                node={moveNode}
                routine={changedRoutine}
                zIndex={zIndex + 3}
            />}
            {/* Displays routine information when you click on a routine list item*/}
            <SubroutineInfoDialog
                data={openedSubroutine}
                defaultLanguage={language}
                isEditing={isEditing}
                handleUpdate={handleSubroutineUpdate}
                handleReorder={handleSubroutineReorder}
                handleViewFull={handleSubroutineViewFull}
                open={Boolean(openedSubroutine)}
                session={session}
                onClose={closeRoutineInfo}
                zIndex={zIndex + 3}
            />
            {/* Display top navbars */}
            {/* First contains close icon and title */}
            <Stack
                id="routine-title-and-language"
                direction="row"
                sx={{
                    zIndex: 2,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    height: '64px',
                }}>
                {/* Title */}
                <EditableLabel
                    canEdit={isEditing}
                    handleUpdate={updateRoutineTitle}
                    placeholder={loading ? 'Loading...' : 'Enter title...'}
                    renderLabel={(t) => (
                        <Typography
                            component="h2"
                            variant="h5"
                            textAlign="center"
                            sx={{
                                fontSize: { xs: '1em', sm: '1.25em', md: '1.5em' },
                            }}
                        >{t ?? (loading ? 'Loading...' : 'Enter title')}</Typography>
                    )}
                    text={getTranslation(changedRoutine, 'title', [language], false) ?? ''}
                    sxs={{
                        stack: { marginLeft: 'auto' }
                    }}
                    validationSchema={titleValidation.required(requiredErrorMessage)}
                />
                {/* Close Icon */}
                <IconButton
                    edge="start"
                    aria-label="close"
                    onClick={onClose}
                    color="inherit"
                    sx={{
                        marginLeft: 'auto',
                        marginRight: 1,
                        marginTop: 'auto',
                        marginBottom: 'auto',
                    }}
                >
                    <CloseIcon width='32px' height='32px' />
                </IconButton>
            </Stack>
            {/* Second contains additional info and icons */}
            <Stack
                id="build-routine-information-bar"
                direction="row"
                spacing={2}
                width="100%"
                justifyContent="space-between"
                sx={{
                    zIndex: 2,
                    height: '48px',
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    // ...smallHorizontalScrollbar(palette) TODO this is needed for mobile, but it breaks Unlinked dialog's overflow-y: visble. Internet solution is to use popover, but this would mess up dragging nodes into container
                }}
            >
                <StatusButton status={status.status} messages={status.messages} sx={{
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    marginLeft: 2,
                    marginRight: 1,
                }} />
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    <UnlinkedNodesDialog
                        handleNodeDelete={handleNodeDelete}
                        handleToggleOpen={toggleUnlinkedNodes}
                        language={language}
                        nodes={nodesOffGraph}
                        open={isUnlinkedNodesOpen}
                        zIndex={zIndex + 3}
                    />
                    {/* Edit button */}
                    {canEdit && !isEditing ? (
                        <IconButton aria-label="confirm-title-change" onClick={startEditing} >
                            <EditIcon fill={palette.secondary.light} />
                        </IconButton>
                    ) : null}
                    {/* Help button */}
                    <HelpButton markdown={helpText} sx={{ fill: palette.secondary.light }} sxRoot={{ margin: "auto", marginRight: 1 }} />
                    {/* Display overall routine description, insturctions, etc. */}
                    <BuildInfoDialog
                        formik={formik}
                        handleAction={handleRoutineAction}
                        handleLanguageChange={setLanguage}
                        handleRelationshipsChange={onRelationshipsChange}
                        handleResourcesUpdate={handleResourcesUpdate}
                        handleTagsUpdate={handleTagsUpdate}
                        handleTranslationDelete={deleteTranslation}
                        handleTranslationUpdate={updateTranslation}
                        handleUpdate={(updated: Routine) => { addToChangeStack(updated); }}
                        isEditing={isEditing}
                        language={language}
                        loading={loading}
                        relationships={relationships}
                        routine={changedRoutine}
                        session={session}
                        sxs={{ icon: { fill: palette.secondary.light }, iconButton: { marginRight: 1 } }}
                        tags={tags}
                        translations={translations}
                        zIndex={zIndex + 1}
                    />
                </Box>
            </Stack>
            {/* Third displays above graph, only when editing */}
            {editActions}
            {/* Displays main routine's information and some buttons */}
            <Box sx={{
                background: palette.background.default,
                bottom: '0',
                display: 'flex',
                flexDirection: 'column',
                position: 'fixed',
                width: '100%',
            }}>
                <NodeGraph
                    columns={columns}
                    handleAction={handleAction}
                    handleBranchInsert={handleBranchInsert}
                    handleLinkCreate={handleLinkCreate}
                    handleLinkUpdate={handleLinkUpdate}
                    handleLinkDelete={handleLinkDelete}
                    handleNodeInsert={handleNodeInsert}
                    handleNodeUpdate={handleNodeUpdate}
                    handleNodeDrop={handleNodeDrop}
                    handleScaleChange={handleScaleDelta}
                    isEditing={isEditing}
                    labelVisible={true}
                    language={language}
                    links={changedRoutine?.nodeLinks ?? []}
                    nodesById={nodesById}
                    scale={scale}
                    zIndex={zIndex}
                />
                <BuildBottomContainer
                    canCancelMutate={!loading}
                    canSubmitMutate={!loading && !isEqual(routine, changedRoutine)}
                    errors={{
                        ...(formik.errors ?? {}),
                        'graph': status.status !== Status.Valid ? status.messages : null,
                        'unchanged': isEqual(routine, changedRoutine) ? 'No changes made' : null,
                    }}
                    handleCancel={revertChanges}
                    handleSubmit={() => { formik.submitForm() }}
                    handleScaleChange={handleScaleChange}
                    handleRunDelete={handleRunDelete}
                    handleRunAdd={handleRunAdd}
                    hasNext={false}
                    hasPrevious={false}
                    isAdding={!uuidValidate(id)}
                    isEditing={isEditing}
                    loading={loading}
                    scale={scale}
                    session={session}
                    sliderColor={palette.secondary.light}
                    routine={routine}
                    runState={BuildRunState.Stopped}
                    zIndex={zIndex}
                />
            </Box>
        </Box>
    )
};