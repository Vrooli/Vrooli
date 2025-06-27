/**
 * ProjectVersion API Response Fixtures
 * 
 * This file extends ResourceVersion fixtures to provide project-specific mock data.
 * Projects in Vrooli are represented as Resources with specific properties.
 */

import type { ProjectVersion, Project, Team, User } from "@vrooli/shared";
import { ResourceVersionResponseFactory } from "./ResourceVersionResponses.js";

/**
 * ProjectVersion API response factory extending ResourceVersion
 */
export class ProjectVersionResponseFactory extends ResourceVersionResponseFactory {
    /**
     * Create mock project version data
     */
    createMockProjectVersion(overrides?: Partial<ProjectVersion>): ProjectVersion {
        const now = new Date().toISOString();
        const id = `proj_ver_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        const projectId = `proj_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        const userId = `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        
        const defaultProjectVersion: ProjectVersion = {
            __typename: "ProjectVersion",
            id,
            created_at: now,
            updated_at: now,
            versionLabel: "1.0.0",
            versionNotes: "Initial version",
            isComplete: false,
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            completedAt: null,
            complexity: 5,
            translations: [{
                __typename: "ProjectVersionTranslation",
                id: `trans_${id}`,
                language: "en",
                name: "My Awesome Project",
                description: "A project to organize and track my work efficiently",
            }],
            translationsCount: 1,
            directories: [],
            directoriesCount: 0,
            root: {
                __typename: "Project",
                id: projectId,
                created_at: now,
                updated_at: now,
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
                    created_at: now,
                    updated_at: now,
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
                } as User,
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
            } as Project,
            pullRequest: null,
            suggestedNextByProject: [],
            suggestedNextByProjectCount: 0,
            you: {
                __typename: "ProjectVersionYou",
                canComment: true,
                canCopy: true,
                canDelete: true,
                canUpdate: true,
                canBookmark: true,
                canReport: false,
                canRead: true,
                canReact: true,
            },
            ...overrides,
        };
        
        return defaultProjectVersion;
    }
    
    /**
     * Create multiple mock project versions
     */
    createMockProjectVersions(count: number = 5): ProjectVersion[] {
        const projects: ProjectVersion[] = [];
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
                    __typename: "ProjectVersionTranslation",
                    id: `trans_${i}`,
                    language: "en",
                    name: projectNames[i % projectNames.length],
                    description: projectDescriptions[i % projectDescriptions.length],
                }],
                root: {
                    ...this.createMockProjectVersion().root,
                    handle: `project-${projectNames[i % projectNames.length].toLowerCase().replace(/\s+/g, '-')}`,
                    views: Math.floor(Math.random() * 1000),
                    score: Math.floor(Math.random() * 100),
                    bookmarks: Math.floor(Math.random() * 50),
                } as Project,
            });
            projects.push(project);
        }
        
        return projects;
    }
    
    /**
     * Create mock team-owned project version
     */
    createMockTeamProjectVersion(teamName: string = "Dev Team"): ProjectVersion {
        const teamId = `team_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        
        return this.createMockProjectVersion({
            root: {
                ...this.createMockProjectVersion().root,
                owner: {
                    __typename: "Team",
                    id: teamId,
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    handle: teamName.toLowerCase().replace(/\s+/g, '-'),
                    name: teamName,
                    isOpenToNewMembers: true,
                    isPrivate: false,
                    translations: [{
                        __typename: "TeamTranslation",
                        id: `team_trans_${teamId}`,
                        language: "en",
                        name: teamName,
                        bio: `${teamName} working on collaborative projects`,
                    }],
                    you: {
                        __typename: "TeamYou",
                        canAddMembers: true,
                        canDelete: false,
                        canUpdate: true,
                        canBookmark: true,
                        canReport: false,
                        canRead: true,
                        isBookmarked: false,
                        isMember: true,
                        isOwner: false,
                    },
                } as Team,
            } as Project,
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
    createSuccessHandlers(): import("msw").RestHandler[] {
        const { rest } = require("msw");
        
        return [
            // Get user's projects
            http.post("*/api/projectVersion/findMany", (req: any, res: any, ctx: any) => {
                const projects = this.factory.createMockProjectVersions(10);
                return res(
                    ctx.status(200),
                    ctx.json({
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
                    }),
                );
            }),
            
            // Get single project version
            http.get("*/api/projectVersion/:id", (req: any, res: any, ctx: any) => {
                const { id } = req.params;
                const project = this.factory.createMockProjectVersion({ id });
                return res(
                    ctx.status(200),
                    ctx.json(this.factory.createSuccessResponse(project)),
                );
            }),
            
            // Create project version
            http.post("*/api/projectVersion", (req: any, res: any, ctx: any) => {
                const project = this.factory.createMockProjectVersion(req.body);
                return res(
                    ctx.status(201),
                    ctx.json(this.factory.createSuccessResponse(project)),
                );
            }),
            
            // Update project version
            http.put("*/api/projectVersion/:id", (req: any, res: any, ctx: any) => {
                const { id } = req.params;
                const project = this.factory.createMockProjectVersion({ 
                    ...req.body,
                    id,
                    updated_at: new Date().toISOString(),
                });
                return res(
                    ctx.status(200),
                    ctx.json(this.factory.createSuccessResponse(project)),
                );
            }),
        ];
    }
    
    /**
     * Create loading handlers with delay
     */
    createLoadingHandlers(delay: number = 1000): import("msw").RestHandler[] {
        const { rest } = require("msw");
        
        return [
            http.post("*/api/projectVersion/findMany", (req: any, res: any, ctx: any) => {
                return res(ctx.delay(delay), ctx.status(200), ctx.json({ edges: [], pageInfo: {} }));
            }),
        ];
    }
}

// Export instances
export const projectVersionResponseFactory = new ProjectVersionResponseFactory();
export const projectVersionMSWHandlers = new ProjectVersionMSWHandlers();