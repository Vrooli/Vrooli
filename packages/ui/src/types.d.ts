// Defines common props
import { findHandles_findHandles } from 'graphql/generated/findHandles';
import { historyPage_historyPage_activeRuns, historyPage_historyPage_recentlyStarred, historyPage_historyPage_recentlyViewed } from 'graphql/generated/historyPage';
import { homePage_homePage_organizations, homePage_homePage_projects, homePage_homePage_routines, homePage_homePage_standards, homePage_homePage_users } from 'graphql/generated/homePage';
import { organization_organization, organization_organization_translations } from 'graphql/generated/organization';
import { profile_profile, profile_profile_translations, profile_profile_emails, profile_profile_resourceLists, profile_profile_resourceLists_translations, profile_profile_wallets, profile_profile_hiddenTags } from 'graphql/generated/profile';
import { project_project, project_project_translations } from 'graphql/generated/project';
import { reportCreate_reportCreate } from 'graphql/generated/reportCreate';
import { resource_resource, resource_resource_translations } from 'graphql/generated/resource';
import { routine_routine, routine_routine_inputs, routine_routine_inputs_translations, routine_routine_nodeLinks, routine_routine_nodeLinks_whens, routine_routine_nodeLinks_whens_translations, routine_routine_nodes, routine_routine_nodes_data_NodeEnd, routine_routine_nodes_data_NodeLoop, routine_routine_nodes_data_NodeRoutineList, routine_routine_nodes_data_NodeRoutineList_routines, routine_routine_nodes_data_NodeRoutineList_routines_translations, routine_routine_nodes_translations, routine_routine_outputs, routine_routine_outputs_translations, routine_routine_runs, routine_routine_translations } from 'graphql/generated/routine';
import { standard_standard, standard_standard_translations } from 'graphql/generated/standard';
import { tag_tag, tag_tag_translations } from 'graphql/generated/tag';
import { user_user } from 'graphql/generated/user';
import { RoutineStepType } from 'utils';
import { FetchResult } from "@apollo/client";
import { comment_comment } from 'graphql/generated/comment';
import { comments_comments_threads } from 'graphql/generated/comments';

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
export type Comment = comment_comment;
export type CommentThread = comments_comments_threads;
export type Email = profile_profile_emails;
export type Handle = findHandles_findHandles;
export type ListOrganization = homePage_homePage_organizations;
export type ListProject = homePage_homePage_projects;
export type ListRoutine = homePage_homePage_routines;
export type ListRun = historyPage_historyPage_activeRuns
export type ListStandard = homePage_homePage_standards;
export type ListStar = historyPage_historyPage_recentlyStarred;
export type ListUser = homePage_homePage_users;
export type ListView = historyPage_historyPage_recentlyViewed;
export type Node = routine_routine_nodes;
export type NodeTranslation = routine_routine_nodes_translations;
export type NodeDataEnd = routine_routine_nodes_data_NodeEnd;
export type NodeDataLoop = routine_routine_nodes_data_NodeLoop;
export type NodeDataRoutineList = routine_routine_nodes_data_NodeRoutineList;
export type NodeDataRoutineListItem = routine_routine_nodes_data_NodeRoutineList_routines;
export type NodeDataRoutineListItemTranslation = routine_routine_nodes_data_NodeRoutineList_routines_translations;
export type NodeLink = routine_routine_nodeLinks;
export type NodeLinkWhen = routine_routine_nodeLinks_whens;
export type NodeLinkWhenTranslation = routine_routine_nodeLinks_whens_translations;
export type Organization = organization_organization;
export type OrganizationTranslation = organization_organization_translations;
export type Profile = profile_profile;
export type ProfileTranslation = profile_profile_translations;
export type Project = project_project;
export type ProjectTranslation = project_project_translations;
export type Report = reportCreate_reportCreate;
export type Resource = resource_resource;
export type ResourceTranslation = resource_resource_translations;
export type ResourceList = profile_profile_resourceLists;
export type ResourceListTranslation = profile_profile_resourceLists_translations;
export type Routine = routine_routine;
export type RoutineTranslation = routine_routine_translations;
export type Run = routine_routine_runs[0];
export type RunStep = Run['steps'][0];
export type RoutineInput = routine_routine_inputs;
export type RoutineInputTranslation = routine_routine_inputs_translations;
export type RoutineInputList = RoutineInput[];
export type RoutineOutput = routine_routine_outputs;
export type RoutineOutputTranslation = routine_routine_outputs_translations;
export type RoutineOutputList = RoutineOutput[];
export type Standard = standard_standard;
export type StandardTranslation = standard_standard_translations;
export type Tag = tag_tag;
export type TagHidden = profile_profile_hiddenTags;
export type TagTranslation = tag_tag_translations;
export type User = user_user;
export type Wallet = profile_profile_wallets;

/**
 * Wrapper type for helping convert objects in the shape of a query result, 
 * to a create/update input object.
 */
export type ShapeWrapper<T> = Partial<Omit<T, '__typename' | 'createdAt' | 'created_at' | 'updatedAt' | 'updated_at' | 'completedAt' | 'completed_at' | 'commentsCount' | 'isUpvoted' | 'isStarred' | 'reportsCount' | 'role' | 'score' | 'stars'>> & { __typename?: string };

// Common query input groups
export type IsCompleteInput = {
    isComplete?: boolean;
    isCompleteExceptions: { id: string, relation: string }[];
}
export type IsInternalInput = {
    isInternal?: boolean;
    isInternalExceptions: { id: string, relation: string }[];
}

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