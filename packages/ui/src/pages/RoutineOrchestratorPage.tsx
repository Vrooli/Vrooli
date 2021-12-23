import { makeStyles } from '@mui/styles';
import { Theme, Typography } from '@mui/material';
import { NODE_TYPES, ORCHESTRATION_DATA } from '@local/shared';
import { ZoomI } from 'components';

//TEMP
const data: ORCHESTRATION_DATA = {
    title: 'Validate business idea',
    description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
    // Unordered list of top-level nodes
    // Page renders nodes starting at the start node
    nodes: [
        // Start node
        {
            id: '1',
            type: NODE_TYPES.Start,
            // ID of previous node
            previous: null,
            // ID of next node
            next: '2',
            // Additional data specific to the node type
            data: null
        },
        // Routine List node
        {
            id: '2',
            type: NODE_TYPES.RoutineList,
            previous: '1',
            next: '3',
            data: {
                title: 'Provide Basic Info',
                description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                isOrdered: false,
                routines: [
                    {
                        id: '1',
                        title: 'Provide Basic Info',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '3',
            type: NODE_TYPES.RoutineList,
            previous: '2',
            next: '4',
            data: {
                title: 'Knowledge management',
                description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                isOrdered: false,
                routines: [
                    {
                        id: '2',
                        title: 'Create task list',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                    },
                    {
                        id: '3',
                        title: 'Generate forecasts',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                    },
                    {
                        id: '4',
                        title: 'List objectives',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                    },
                    {
                        id: '5',
                        title: 'Create business model',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                    }
                ]
            }
        },
        // Decision node
        {
            id: '4',
            type: NODE_TYPES.Decision,
            previous: '3',
            // Decision nodes have no definitive next node
            // Instead, next node is determined by the decision
            next: null,
            data: {
                title: 'Worth pursuing?',
                description: null,
                decisions: [
                    {
                        id: '1',
                        title: 'Yes',
                        next: '5',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '2',
                        title: 'No',
                        next: '6',
                        when: []
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '5',
            type: NODE_TYPES.RoutineList,
            previous: '4',
            next: '7',
            data: {
                title: 'todo',
                description: null,
                isOrdered: false,
                routines: []
            },
        },
        // End node
        {
            id: '6',
            type: NODE_TYPES.End,
            previous: '4',
            next: null,
            data: {
                title: 'asdfas',
                description: 'afda',
                wasSuccessful: true,
            }
        },
        // Decision node
        {
            id: '7',
            type: NODE_TYPES.Decision,
            previous: '4',
            next: null,
            data: {
                title: 'Try again?',
                description: null,
                decisions: [
                    {
                        id: '3',
                        title: 'Yes',
                        next: '8',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '4',
                        title: 'No',
                        next: '9',
                        when: []
                    }
                ]
            }
        },
        // Redirect node
        {
            id: '8',
            type: NODE_TYPES.Redirect,
            previous: '7',
            next: '1',
            data: null,
        },
        // End node
        {
            id: '9',
            type: NODE_TYPES.End,
            previous: '7',
            next: null,
            data: {
                title: 'asdf',
                description: 'asdf',
                wasSuccessful: false,
            }
        },
    ]
}

const useStyles = makeStyles((theme: Theme) => ({
    root: {},
    title: {
        textAlign: 'center',
    },
}));

export const RoutineOrchestratorPage = () => {
    const classes = useStyles();
    console.log('GOT HERE')

    return (
        <div id="page" className={classes.root}>
            asdf
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            asdffffff
            {/* <ZoomI width={1000} height={1000}/> */}
            ytryutrtetyeuyt
        </div>
    )
};