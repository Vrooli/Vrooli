import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { PrismaClient } from "@prisma/client";
import { 
    RoutineDbFactory,
    RoutineVersionDbFactory,
    ResourceDbFactory,
    ResourceVersionDbFactory,
    ResourceVersionRelationDbFactory,
    UserDbFactory,
    TeamDbFactory,
    createRoutineDbFactory,
    createRoutineVersionDbFactory,
    createResourceDbFactory,
    createResourceVersionDbFactory,
    createResourceVersionRelationDbFactory,
    createUserDbFactory,
    createTeamDbFactory,
} from "./index.js";
import { routineConfigFixtures } from "@vrooli/shared/__test/fixtures/config";
import { ResourceType } from "@vrooli/shared";

describe("Core Business Objects Part 2 - Database Fixture Factories", () => {
    let prisma: PrismaClient;
    let userFactory: UserDbFactory;
    let teamFactory: TeamDbFactory;
    let routineFactory: RoutineDbFactory;
    let routineVersionFactory: RoutineVersionDbFactory;
    let resourceFactory: ResourceDbFactory;
    let resourceVersionFactory: ResourceVersionDbFactory;
    let resourceVersionRelationFactory: ResourceVersionRelationDbFactory;

    let testUserId: string;
    let testTeamId: string;

    beforeEach(async () => {
        prisma = new PrismaClient({
            datasources: {
                db: {
                    url: process.env.TEST_DATABASE_URL || "postgresql://test:test@localhost:5432/test_db",
                },
            },
        });

        await prisma.$connect();

        // Initialize factories
        userFactory = createUserDbFactory(prisma);
        teamFactory = createTeamDbFactory(prisma);
        routineFactory = createRoutineDbFactory(prisma);
        routineVersionFactory = createRoutineVersionDbFactory(prisma);
        resourceFactory = createResourceDbFactory(prisma);
        resourceVersionFactory = createResourceVersionDbFactory(prisma);
        resourceVersionRelationFactory = createResourceVersionRelationDbFactory(prisma);

        // Create test user and team
        const testUser = await userFactory.createMinimal({
            name: "Test User",
            handle: "testuser",
        });
        testUserId = testUser.id;

        const testTeam = await teamFactory.createWithOwnerAndMembers(testUserId);
        testTeamId = testTeam.id;
    });

    afterEach(async () => {
        // Clean up all created records
        await routineFactory.cleanupAll();
        await resourceFactory.cleanupAll();
        await teamFactory.cleanupAll();
        await userFactory.cleanupAll();

        await prisma.$disconnect();
    });

    describe("RoutineDbFactory", () => {
        test("should create minimal routine", async () => {
            const routine = await routineFactory.createMinimal({
                ownedByUser: { connect: { id: testUserId } },
            });

            expect(routine.id).toBeDefined();
            expect(routine.publicId).toBeDefined();
            expect(routine.isPrivate).toBe(false);
            expect(routine.ownedByUserId).toBe(testUserId);
        });

        test("should create complete routine with versions", async () => {
            const routine = await routineFactory.createComplete({
                ownedByUser: { connect: { id: testUserId } },
            });

            expect(routine.id).toBeDefined();
            expect(routine.versions).toBeDefined();
            
            // Fetch with include to verify versions
            const routineWithVersions = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: { versions: { include: { translations: true } } },
            });

            expect(routineWithVersions?.versions).toHaveLength(1);
            expect(routineWithVersions?.versions[0].isLatest).toBe(true);
        });

        test("should create simple task routine", async () => {
            const routine = await routineFactory.createSimpleTask(testUserId);

            expect(routine.id).toBeDefined();
            expect(routine.ownedByUserId).toBe(testUserId);

            const routineWithData = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: { 
                    versions: { include: { translations: true } },
                    tags: true,
                },
            });

            expect(routineWithData?.tags).toHaveLength(2); // ["task", "simple"]
            expect(routineWithData?.versions[0].config).toEqual(routineConfigFixtures.action.simple);
        });

        test("should create complex workflow routine", async () => {
            const routine = await routineFactory.createComplexWorkflow(testUserId);

            const routineWithData = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: { 
                    versions: { include: { translations: true } },
                    tags: true,
                },
            });

            expect(routineWithData?.tags).toHaveLength(3); // ["workflow", "complex", "data-processing"]
            expect(routineWithData?.versions[0].config).toEqual(routineConfigFixtures.multiStep.complexWorkflow);
        });

        test("should create automated routine", async () => {
            const routine = await routineFactory.createAutomatedRoutine(testTeamId);

            expect(routine.ownedByTeamId).toBe(testTeamId);

            const routineWithData = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: { 
                    versions: { include: { translations: true } },
                    tags: true,
                },
            });

            expect(routineWithData?.tags.some(tag => tag.tag === "automation")).toBe(true);
        });

        test("should create routine with version history", async () => {
            const routine = await routineFactory.createWithVersionHistory(testUserId);

            const routineWithVersions = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: { versions: { orderBy: { versionLabel: 'asc' } } },
            });

            expect(routineWithVersions?.versions).toHaveLength(3);
            expect(routineWithVersions?.versions.filter(v => v.isLatest)).toHaveLength(1);
            expect(routineWithVersions?.versions.find(v => v.isLatest)?.versionLabel).toBe("2.0.0");
        });

        test("should validate constraints", async () => {
            const routine = await routineFactory.createMinimal({
                ownedByUser: { connect: { id: testUserId } },
            });

            const validation = await routineFactory.verifyConstraints(routine.id);
            expect(validation.valid).toBe(true);
        });
    });

    describe("RoutineVersionDbFactory", () => {
        let testRoutineId: string;

        beforeEach(async () => {
            // Create a routine without versions for testing
            const routine = await prisma.routine.create({
                data: routineFactory.getMinimalData({
                    ownedByUser: { connect: { id: testUserId } },
                }),
            });
            testRoutineId = routine.id;
        });

        test("should create simple action version", async () => {
            const version = await routineVersionFactory.createSimpleAction(testRoutineId);

            expect(version.routineId).toBe(testRoutineId);
            expect(version.config).toEqual(routineConfigFixtures.action.simple);
            expect(version.complexity).toBe(1);
        });

        test("should create text generation version", async () => {
            const version = await routineVersionFactory.createTextGeneration(testRoutineId);

            expect(version.config).toEqual(routineConfigFixtures.generate.withComplexPrompt);
            expect(version.complexity).toBe(3);
        });

        test("should create multi-step workflow version", async () => {
            const version = await routineVersionFactory.createMultiStepWorkflow(testRoutineId);

            const versionWithData = await prisma.routineVersion.findUnique({
                where: { id: version.id },
                include: { 
                    resourceLists: { 
                        include: { 
                            resources: { 
                                include: { 
                                    resourceVersion: { include: { translations: true } } 
                                } 
                            } 
                        } 
                    } 
                },
            });

            expect(versionWithData?.resourceLists).toHaveLength(2); // Learn and Research
            expect(version.config).toEqual(routineConfigFixtures.multiStep.complexWorkflow);
            expect(version.complexity).toBe(8);
        });

        test("should create draft version", async () => {
            const version = await routineVersionFactory.createDraftVersion(testRoutineId);

            expect(version.isComplete).toBe(false);
            expect(version.versionLabel).toBe("0.1.0-draft");
        });

        test("should validate version constraints", async () => {
            const version = await routineVersionFactory.createSimpleAction(testRoutineId);

            const validation = await routineVersionFactory.verifyConstraints(version.id);
            expect(validation.valid).toBe(true);
        });
    });

    describe("ResourceDbFactory", () => {
        test("should create documentation resource", async () => {
            const resource = await resourceFactory.createDocumentation(testUserId);

            expect(resource.resourceType).toBe(ResourceType.Documentation);
            expect(resource.ownedByUserId).toBe(testUserId);

            const resourceWithData = await prisma.resource.findUnique({
                where: { id: resource.id },
                include: { 
                    versions: { include: { translations: true } },
                    tags: true,
                },
            });

            expect(resourceWithData?.tags.some(tag => tag.tag === "documentation")).toBe(true);
            expect(resourceWithData?.versions[0].link).toBe("https://docs.example.com/api");
        });

        test("should create video tutorial resource", async () => {
            const resource = await resourceFactory.createVideoTutorial(testUserId);

            expect(resource.resourceType).toBe(ResourceType.Tutorial);

            const resourceWithData = await prisma.resource.findUnique({
                where: { id: resource.id },
                include: { 
                    versions: { include: { translations: true } },
                    tags: true,
                },
            });

            expect(resourceWithData?.tags.some(tag => tag.tag === "video")).toBe(true);
        });

        test("should create external tool resource", async () => {
            const resource = await resourceFactory.createExternalTool(testTeamId);

            expect(resource.resourceType).toBe(ResourceType.ExternalTool);
            expect(resource.ownedByTeamId).toBe(testTeamId);
            expect(resource.isInternal).toBe(false);
        });

        test("should create resource with version history", async () => {
            const resource = await resourceFactory.createWithVersionHistory(testUserId);

            const resourceWithVersions = await prisma.resource.findUnique({
                where: { id: resource.id },
                include: { versions: { orderBy: { versionLabel: 'asc' } } },
            });

            expect(resourceWithVersions?.versions).toHaveLength(3);
            expect(resourceWithVersions?.versions.filter(v => v.isLatest)).toHaveLength(1);
            expect(resourceWithVersions?.versions.find(v => v.isLatest)?.versionLabel).toBe("2.0.0");
        });

        test("should create private internal resource", async () => {
            const resource = await resourceFactory.createPrivateInternal(testTeamId);

            expect(resource.isPrivate).toBe(true);
            expect(resource.isInternal).toBe(true);
            expect(resource.ownedByTeamId).toBe(testTeamId);
        });
    });

    describe("ResourceVersionDbFactory", () => {
        let testResourceId: string;

        beforeEach(async () => {
            // Create a resource without versions for testing
            const resource = await prisma.resource.create({
                data: resourceFactory.getMinimalData({
                    resourceType: ResourceType.Documentation,
                    ownedByUser: { connect: { id: testUserId } },
                }),
            });
            testResourceId = resource.id;
        });

        test("should create documentation version", async () => {
            const version = await resourceVersionFactory.createDocumentationVersion(testResourceId);

            expect(version.rootId).toBe(testResourceId);
            expect(version.link).toBe("https://docs.example.com/api/v1");
            expect(version.complexity).toBe(3);
        });

        test("should create tutorial version with relations", async () => {
            // First create a routine version to relate to
            const routine = await routineFactory.createSimpleTask(testUserId);
            const routineVersion = routine.versions?.[0];

            expect(routineVersion).toBeDefined();

            const version = await resourceVersionFactory.createTutorialVersionWithRelations(
                testResourceId,
                [{ id: routineVersion!.id, type: "RoutineVersion" }]
            );

            const versionWithData = await prisma.resourceVersion.findUnique({
                where: { id: version.id },
                include: { relations: true },
            });

            expect(versionWithData?.relations).toHaveLength(1);
            expect(versionWithData?.relations[0].routineVersionId).toBe(routineVersion!.id);
        });

        test("should create code repository version", async () => {
            const version = await resourceVersionFactory.createCodeRepositoryVersion(testResourceId);

            expect(version.link).toBe("https://github.com/example/sample-code");
            expect(version.versionLabel).toBe("2.1.0");
            expect(version.complexity).toBe(4);
        });

        test("should create draft version", async () => {
            const version = await resourceVersionFactory.createDraftVersion(testResourceId);

            expect(version.isComplete).toBe(false);
            expect(version.versionLabel).toBe("0.1.0-draft");
        });

        test("should create version with full metadata", async () => {
            const version = await resourceVersionFactory.createWithFullMetadata(testResourceId);

            const versionWithData = await prisma.resourceVersion.findUnique({
                where: { id: version.id },
                include: { translations: true },
            });

            expect(versionWithData?.translations).toHaveLength(2); // English and Spanish
            expect(version.complexity).toBe(7);
        });
    });

    describe("ResourceVersionRelationDbFactory", () => {
        let testResourceVersionId: string;
        let testRoutineVersionId: string;

        beforeEach(async () => {
            // Create test resource version
            const resource = await resourceFactory.createDocumentation(testUserId);
            testResourceVersionId = resource.versions![0].id;

            // Create test routine version
            const routine = await routineFactory.createSimpleTask(testUserId);
            testRoutineVersionId = routine.versions![0].id;
        });

        test("should create resource to routine relation", async () => {
            const relation = await resourceVersionRelationFactory.createResourceToRoutine(
                testResourceVersionId,
                testRoutineVersionId,
                0
            );

            expect(relation.resourceVersionId).toBe(testResourceVersionId);
            expect(relation.routineVersionId).toBe(testRoutineVersionId);
            expect(relation.index).toBe(0);
        });

        test("should create multiple relations", async () => {
            // Create another routine
            const routine2 = await routineFactory.createComplexWorkflow(testUserId);
            const routineVersion2Id = routine2.versions![0].id;

            const relations = await resourceVersionRelationFactory.createMultipleRelations(
                testResourceVersionId,
                [
                    { id: testRoutineVersionId, type: "RoutineVersion" },
                    { id: routineVersion2Id, type: "RoutineVersion" },
                ]
            );

            expect(relations).toHaveLength(2);
            expect(relations[0].index).toBe(0);
            expect(relations[1].index).toBe(1);
        });

        test("should create ordered relations", async () => {
            // Create multiple routines
            const routine2 = await routineFactory.createComplexWorkflow(testUserId);
            const routine3 = await routineFactory.createAutomatedRoutine(testTeamId);

            const routineIds = [
                testRoutineVersionId,
                routine2.versions![0].id,
                routine3.versions![0].id,
            ];

            const relations = await resourceVersionRelationFactory.createOrderedRelations(
                testResourceVersionId,
                routineIds,
                "RoutineVersion"
            );

            expect(relations).toHaveLength(3);
            relations.forEach((relation, index) => {
                expect(relation.index).toBe(index);
                expect(relation.routineVersionId).toBe(routineIds[index]);
            });
        });

        test("should validate relation constraints", async () => {
            const relation = await resourceVersionRelationFactory.createResourceToRoutine(
                testResourceVersionId,
                testRoutineVersionId
            );

            const validation = await resourceVersionRelationFactory.verifyConstraints(relation.id);
            expect(validation.valid).toBe(true);
        });

        test("should get relations by resource version", async () => {
            // Create multiple relations
            const routine2 = await routineFactory.createComplexWorkflow(testUserId);
            
            await resourceVersionRelationFactory.createResourceToRoutine(testResourceVersionId, testRoutineVersionId, 0);
            await resourceVersionRelationFactory.createResourceToRoutine(testResourceVersionId, routine2.versions![0].id, 1);

            const relations = await resourceVersionRelationFactory.getRelationsByResourceVersion(testResourceVersionId);

            expect(relations).toHaveLength(2);
            expect(relations[0].index).toBe(0);
            expect(relations[1].index).toBe(1);
        });
    });

    describe("Integration Tests", () => {
        test("should create complete workflow with all related objects", async () => {
            // 1. Create a routine with complex workflow
            const routine = await routineFactory.createComplexWorkflow(testUserId);
            const routineVersionId = routine.versions![0].id;

            // 2. Create documentation resource for the routine
            const docResource = await resourceFactory.createDocumentation(testUserId);
            const docVersionId = docResource.versions![0].id;

            // 3. Create tutorial resource
            const tutorialResource = await resourceFactory.createVideoTutorial(testUserId);
            const tutorialVersionId = tutorialResource.versions![0].id;

            // 4. Link resources to the routine
            await resourceVersionRelationFactory.createResourceToRoutine(docVersionId, routineVersionId, 0);
            await resourceVersionRelationFactory.createResourceToRoutine(tutorialVersionId, routineVersionId, 1);

            // 5. Verify the complete setup
            const routineWithData = await prisma.routine.findUnique({
                where: { id: routine.id },
                include: {
                    versions: {
                        include: {
                            resourceLists: {
                                include: {
                                    resources: {
                                        include: {
                                            resourceVersion: {
                                                include: {
                                                    relations: {
                                                        include: {
                                                            resourceVersion: {
                                                                include: {
                                                                    translations: true,
                                                                },
                                                            },
                                                        },
                                                    },
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                    tags: true,
                },
            });

            expect(routineWithData).toBeDefined();
            expect(routineWithData?.tags.some(tag => tag.tag === "workflow")).toBe(true);
            expect(routineWithData?.versions[0].resourceLists).toHaveLength(2); // Learn and Research

            // Verify resource relations
            const docRelations = await resourceVersionRelationFactory.getRelationsByResourceVersion(docVersionId);
            const tutorialRelations = await resourceVersionRelationFactory.getRelationsByResourceVersion(tutorialVersionId);

            expect(docRelations).toHaveLength(1);
            expect(tutorialRelations).toHaveLength(1);
            expect(docRelations[0].routineVersionId).toBe(routineVersionId);
            expect(tutorialRelations[0].routineVersionId).toBe(routineVersionId);
        });

        test("should handle resource attachment to multiple objects", async () => {
            // Create multiple routines
            const routine1 = await routineFactory.createSimpleTask(testUserId);
            const routine2 = await routineFactory.createComplexWorkflow(testUserId);

            // Create a shared resource (e.g., common documentation)
            const sharedResource = await resourceFactory.createDocumentation(testUserId);
            const sharedVersionId = sharedResource.versions![0].id;

            // Attach the resource to both routines
            await resourceVersionRelationFactory.createResourceToRoutine(
                sharedVersionId, 
                routine1.versions![0].id, 
                0
            );
            await resourceVersionRelationFactory.createResourceToRoutine(
                sharedVersionId, 
                routine2.versions![0].id, 
                0
            );

            // Verify relations
            const relations = await resourceVersionRelationFactory.getRelationsByResourceVersion(sharedVersionId);
            expect(relations).toHaveLength(2);

            // Verify each routine can access the shared resource
            const routine1Relations = relations.filter(r => r.routineVersionId === routine1.versions![0].id);
            const routine2Relations = relations.filter(r => r.routineVersionId === routine2.versions![0].id);

            expect(routine1Relations).toHaveLength(1);
            expect(routine2Relations).toHaveLength(1);
        });
    });
});