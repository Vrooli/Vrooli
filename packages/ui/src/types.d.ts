// Defines common props
import { findHandles_findHandles } from 'graphql/generated/findHandles';
import { forYouPage_forYouPage_activeRuns, forYouPage_forYouPage_recentlyStarred, forYouPage_forYouPage_recentlyViewed } from 'graphql/generated/forYouPage';
import { homePage_homePage_organizations, homePage_homePage_projects, homePage_homePage_routines, homePage_homePage_users } from 'graphql/generated/homePage';
import { organization_organization } from 'graphql/generated/organization';
import { profile_profile, profile_profile_emails, profile_profile_resourceLists, profile_profile_wallets } from 'graphql/generated/profile';
import { project_project } from 'graphql/generated/project';
import { reportCreate_reportCreate } from 'graphql/generated/reportCreate';
import { resource_resource } from 'graphql/generated/resource';
import { routine_routine, routine_routine_inputs, routine_routine_nodeLinks, routine_routine_nodes, routine_routine_nodes_data_NodeEnd, routine_routine_nodes_data_NodeLoop, routine_routine_nodes_data_NodeRoutineList, routine_routine_nodes_data_NodeRoutineList_routines, routine_routine_outputs, routine_routine_runs } from 'graphql/generated/routine';
import { standard_standard } from 'graphql/generated/standard';
import { tag_tag } from 'graphql/generated/tag';
import { user_user } from 'graphql/generated/user';
import { RoutineStepType } from 'utils';
import { FetchResult } from "@apollo/client";

// Top-level props that can be passed into any routed component
export type SessionChecked = boolean;
export type Session = {
    id?: string | null;
    languages?: string[] | null;
    roles?: string[] | null;
    theme?: string;
}
export interface CommonProps {
    session: Session;
    sessionChecked: SessionChecked;
}

// Rename auto-generated query objects
export type Comment = any; //TODO
export type Email = profile_profile_emails;
export type Handle = findHandles_findHandles;
export type ListOrganization = homePage_homePage_organizations;
export type ListProject = homePage_homePage_projects;
export type ListRoutine = homePage_homePage_routines;
export type ListRun = forYouPage_forYouPage_activeRuns
export type ListStandard = homePage_homePage_standards;
export type ListStar = forYouPage_forYouPage_recentlyStarred;
export type ListUser = homePage_homePage_users;
export type ListView = forYouPage_forYouPage_recentlyViewed;
export type Node = routine_routine_nodes;
export type NodeDataEnd = routine_routine_nodes_data_NodeEnd;
export type NodeDataLoop = routine_routine_nodes_data_NodeLoop;
export type NodeDataRoutineList = routine_routine_nodes_data_NodeRoutineList;
export type NodeDataRoutineListItem = routine_routine_nodes_data_NodeRoutineList_routines;
export type NodeLink = routine_routine_nodeLinks;
export type Organization = organization_organization;
export type Profile = profile_profile;
export type Project = project_project;
export type Report = reportCreate_reportCreate;
export type Resource = resource_resource;
export type ResourceList = profile_profile_resourceLists
export type Routine = routine_routine;
export type Run = routine_routine_runs[0];
export type RunStep = Run['steps'][0];
export type RoutineInput = routine_routine_inputs;
export type RoutineInputList = RoutineInput[];
export type RoutineOutput = routine_routine_outputs;
export type RoutineOutputList = RoutineOutput[];
export type Standard = standard_standard;
export type Tag = tag_tag;
export type User = user_user;
export type Wallet = profile_profile_wallets;

/**
 * Wrapper for removing __typename from any object. Useful when creating 
 * new objects, rather than using queried data.
 */
export type NoTypename<T> = T extends { __typename: string } ? Omit<T, '__typename'> : T;
/**
 * Wrapper for removing __typename, and making id optional. Usefule when 
 * creating new objects, rather than using queried data.
 */
export type NewObject<T> = T extends { __typename: string } ? Omit<T, '__typename' | 'id'> : T;


// Routine-related props
export interface BaseStep {
    title: string, // Title from node
    description: string | null, // Description from node
}
export interface DecisionStep extends BaseStep {
    type: RoutineStepType.Decision,
    links: NodeLink[]
}
export interface SubroutineStep extends BaseStep {
    type: RoutineStepType.Subroutine,
    index: number,
    routine: Routine
}
export interface RoutineListStep extends BaseStep {
    nodeId: string,
    isOrdered: boolean,
    type: RoutineStepType.RoutineList,
    steps: RoutineStep[],
}
export type RoutineStep = DecisionStep | SubroutineStep | RoutineListStep

export interface AutocompleteOption {
    __typename: string;
    id: string;
    label: string | null;
    stars?: number;
    [key: string]: any;
}

// Enable Nami integration
declare global {
    interface Window { cardano: any; }
}
window.cardano = window.cardano || {};

// Apollo GraphQL
export type ApolloResponse = FetchResult<any, Record<string, any>, Record<string, any>>;
export type ApolloError = {
    message?: string;
    graphQLErrors?: {
        message: string;
        extensions?: {
            code: string;
        };
    }[];
}

// Miscellaneous types
export type SetLocation = (to: Path, options?: { replace?: boolean }) => void;