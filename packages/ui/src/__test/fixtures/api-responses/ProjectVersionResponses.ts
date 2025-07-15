/**
 * ProjectVersion API Response Fixtures
 * 
 * This file extends ResourceVersion fixtures to provide project-specific mock data.
 * Projects in Vrooli are represented as Resources with specific properties.
 */
// AI_CHECK: TYPE_SAFETY=fixed-project-team-types | LAST: 2025-07-02 - Fixed Team properties, removed invalid 'directories' and 'suggestedNextByProject' fields

import { http, HttpResponse } from "msw";
import type { ResourceVersion, Team, User } from "@vrooli/shared";
import { ResourceVersionResponseFactory } from "./ResourceVersionResponses.js";

/**
 * ProjectVersion API response factory extending ResourceVersion
 */
export class ProjectVersionResponseFactory extends ResourceVersionResponseFactory {
    /**
     * Create mock project version data
     */
    createMockProjectVersion(overrides?: Partial<ResourceVersion>): ResourceVersion {
        const now = new Date().toISOString();
        const id = `proj_ver_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
        const projectId = `proj_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
        const userId = `user_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
        
        const defaultProjectVersion: ResourceVersion = {
            __typename: "ResourceVersion",
            id,
            createdAt: now,
            updatedAt: now,
            versionLabel: "1.0.0",
            versionNotes: "Initial version",
            isComplete: false,
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            completedAt: null,
            complexity: 5,
            translations: [{
                __typename: "ResourceVersionTranslation",
                id: `trans_${id}`,
                language: "en",
                name: "My Awesome Project",
                description: "A project to organize and track my work efficiently",
            }],
            translationsCount: 1,
            root: {
                __typename: "Project",
                id: projectId,
                createdAt: now,
                updatedAt: now,
                handle: `project-${Date.now()}`,
                isDeleted: false,
                isPrivate: false,
                permissions: "View",
                completedAt: null,
                score: 10,
                bookmarks: 0,
                views: 42,
                owner: {
                    __typename: "User",
                    id: userId,
                    createdAt: now,
                    updatedAt: now,
                    handle: "testuser",
                    name: "Test User",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    isPrivateApis: true,
                    isPrivateApisCreated: true,
                    isPrivateMemberships: true,
                    isPrivateProjects: false,
                    isPrivateProjectsCreated: false,
                    isPrivatePullRequests: true,
                    isPrivateQuestionsAnswered: true,
                    isPrivateQuestionsAsked: true,
                    isPrivateQuizzesCreated: true,
                    isPrivateRoles: true,
                    isPrivateRoutines: false,
                    isPrivateRoutinesCreated: false,
                    isPrivateStandards: false,
                    isPrivateStandardsCreated: false,
                    isPrivateTeamsCreated: true,
                    isPrivateBookmarks: true,
                    isPrivateVotes: true,
                    isPrivateResources: false,
                    isPrivateResourcesCreated: false,
                    bookmarkedBy: [],
                    bookmarks: 0,
                    profileImage: null,
                    bannerImage: null,
                    premium: null,
                    publicId: `user_public_${userId}`,
                    views: 123,
                    wallets: [],
                    translations: [],
                    translationsCount: 0,
                    you: {
                        __typename: "UserYou",
                        canDelete: false,
                        canReport: false,
                        canUpdate: false,
                        isBookmarked: false,
                        isViewed: false,
                    },
                },
                hasCompleteVersion: false,
                tags: [],
                tagsCount: 0,
                transfersCount: 0,
                versionsCount: 1,
                you: {
                    __typename: "ProjectYou",
                    canComment: true,
                    canCopy: true,
                    canDelete: true,
                    canUpdate: true,
                    canBookmark: true,
                    canReport: false,
                    canTransfer: true,
                    canReact: true,
                    isBookmarked: false,
                    reaction: null,
                },
            } as any,
            pullRequest: null,
            publicId: `pub_proj_ver_${id}`,
            comments: [],
            commentsCount: 0,
            forks: [],
            forksCount: 0,
            relatedVersions: [],
            reports: [],
            reportsCount: 0,
            timesCompleted: 0,
            timesStarted: 0,
            versionIndex: 1,
            you: {
                __typename: "ResourceVersionYou",
                canComment: true,
                canCopy: true,
                canDelete: true,
                canUpdate: true,
                canBookmark: true,
                canReport: false,
                canRead: true,
                canReact: true,
                canRun: true,
            },
            ...overrides,
        };
        
        return defaultProjectVersion;
    }
    
    /**
     * Create multiple mock project versions
     */
    createMockProjectVersions(count = 5): ResourceVersion[] {
        const projects: ResourceVersion[] = [];
        const projectNames = [
            "AI Agent Swarm",
            "Data Pipeline Automation",
            "Customer Support Bot",
            "Analytics Dashboard",
            "Content Management System",
            "E-commerce Platform",
            "Social Media Scheduler",
            "Inventory Management",
            "Task Automation Suite",
            "Knowledge Base System",
        ];
        
        const projectDescriptions = [
            "Coordinate multiple AI agents to solve complex problems collaboratively",
            "Automate data collection, processing, and visualization workflows",
            "Intelligent chatbot for handling customer inquiries and support tickets",
            "Real-time analytics and reporting dashboard for business metrics",
            "Flexible content management system with version control",
            "Complete e-commerce solution with payment processing",
            "Schedule and manage social media posts across multiple platforms",
            "Track inventory levels and automate reordering processes",
            "Automate repetitive tasks and workflows with custom routines",
            "Centralized knowledge repository with search and categorization",
        ];
        
        for (let i = 0; i < count; i++) {
            const project = this.createMockProjectVersion({
                translations: [{
                    __typename: "ResourceVersionTranslation",
                    id: `trans_${i}`,
                    language: "en",
                    name: projectNames[i % projectNames.length],
                    description: projectDescriptions[i % projectDescriptions.length],
                }],
                root: {
                    ...this.createMockProjectVersion().root,
                    handle: `project-${projectNames[i % projectNames.length].toLowerCase().replace(/\s+/g, "-")}`,
                    views: Math.floor(Math.random() * 1000),
                    score: Math.floor(Math.random() * 100),
                    bookmarks: Math.floor(Math.random() * 50),
                } as any,
            });
            projects.push(project);
        }
        
        return projects;
    }
    
    /**
     * Create mock team-owned project version
     */
    createMockTeamProjectVersion(teamName = "Dev Team"): ResourceVersion {
        const teamId = `team_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
        
        return this.createMockProjectVersion({
            root: {
                ...this.createMockProjectVersion().root,
                owner: {
                    __typename: "Team" as const,
                    id: teamId,
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    handle: teamName.toLowerCase().replace(/\s+/g, "-"),
                    translatedName: teamName,
                    isOpenToNewMembers: true,
                    isPrivate: false,
                    bookmarkedBy: [],
                    bookmarks: 0,
                    comments: [],
                    commentsCount: 0,
                    forks: [],
                    issues: [],
                    issuesCount: 0,
                    meetings: [],
                    meetingsCount: 0,
                    members: [],
                    membersCount: 0,
                    parent: null,
                    paymentHistory: [],
                    permissions: "{}",
                    premium: null,
                    publicId: `team_public_${teamId}`,
                    reports: [],
                    reportsCount: 0,
                    resources: [],
                    resourcesCount: 0,
                    stats: [],
                    tags: [],
                    transfersIncoming: [],
                    transfersOutgoing: [],
                    translations: [{
                        __typename: "TeamTranslation",
                        id: `team_trans_${teamId}`,
                        language: "en",
                        name: teamName,
                        bio: `${teamName} working on collaborative projects`,
                    }],
                    views: 0,
                    wallets: [],
                    you: {
                        __typename: "TeamYou",
                        canAddMembers: true,
                        canBookmark: true,
                        canDelete: false,
                        canReport: false,
                        canUpdate: true,
                        canRead: true,
                        isBookmarked: false,
                        isViewed: false,
                        yourMembership: null,
                    },
                } as any,
            } as any,
        });
    }
}

/**
 * MSW handlers for ProjectVersion endpoints
 */
export class ProjectVersionMSWHandlers {
    private factory: ProjectVersionResponseFactory;
    
    constructor() {
        this.factory = new ProjectVersionResponseFactory();
    }
    
    /**
     * Create success handlers for project version endpoints
     */
    createSuccessHandlers(): import("msw").RequestHandler[] {
        return [
            // Get user's projects
            http.post("*/api/projectVersion/findMany", async ({ request }) => {
                const projects = this.factory.createMockProjectVersions(10);
                return HttpResponse.json({
                    edges: projects.map(project => ({
                        cursor: project.id,
                        node: project,
                    })),
                    pageInfo: {
                        hasNextPage: false,
                        hasPreviousPage: false,
                        startCursor: projects[0]?.id,
                        endCursor: projects[projects.length - 1]?.id,
                    },
                }, { status: 200 });
            }),
            
            // Get single project version
            http.get("*/api/projectVersion/:id", ({ params }) => {
                const { id } = params;
                const project = this.factory.createMockProjectVersion({ id: id as string });
                return HttpResponse.json(this.factory.createSuccessResponse(project), { status: 200 });
            }),
            
            // Create project version
            http.post("*/api/projectVersion", async ({ request }) => {
                const body = await request.json();
                const project = this.factory.createMockProjectVersion(body as any);
                return HttpResponse.json(this.factory.createSuccessResponse(project), { status: 201 });
            }),
            
            // Update project version
            http.put("*/api/projectVersion/:id", async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as Partial<ResourceVersion>;
                const project = this.factory.createMockProjectVersion({ 
                    ...body,
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                });
                return HttpResponse.json(this.factory.createSuccessResponse(project), { status: 200 });
            }),
        ];
    }
    
    /**
     * Create loading handlers with delay
     */
    createLoadingHandlers(delay = 1000): import("msw").RequestHandler[] {
        return [
            http.post("*/api/projectVersion/findMany", async ({ request }) => {
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json({ edges: [], pageInfo: {} }, { status: 200 });
            }),
        ];
    }
}

// Export instances
export const projectVersionResponseFactory = new ProjectVersionResponseFactory();
export const projectVersionMSWHandlers = new ProjectVersionMSWHandlers();
