/* c8 ignore start */
/**
 * Tests for Project API fixtures
 * 
 * These tests verify that the project fixtures are properly structured and provide
 * the expected functionality for testing project-related operations.
 */

import { ResourceType } from "../../../../api/types.js";
import { projectAPIFixtures, projectFixtures } from "./projectAPIFixtures.js";

describe("Project API Fixtures", () => {
    describe("Core fixture sets", () => {
        test("minimal project fixtures", () => {
            const { minimal } = projectFixtures;
            
            // Test create input
            expect(minimal.create.resourceType).toBe(ResourceType.Project);
            expect(minimal.create.isPrivate).toBe(false);
            expect(minimal.create.versionsCreate).toHaveLength(1);
            expect(minimal.create.versionsCreate[0].versionLabel).toBe("1.0.0");
            expect(minimal.create.versionsCreate[0].translationsCreate).toHaveLength(1);
            expect(minimal.create.versionsCreate[0].translationsCreate[0].name).toBe("Minimal Project");
            
            // Test update input
            expect(minimal.update.id).toBeDefined();
            expect(minimal.update.versionsUpdate).toBeDefined();
            
            // Test find result
            expect(minimal.find.__typename).toBe("Resource");
            expect(minimal.find.resourceType).toBe(ResourceType.Project);
            expect(minimal.find.versions).toHaveLength(1);
        });
        
        test("complete project fixtures", () => {
            const { complete } = projectFixtures;
            
            // Test create input with tags and translations
            expect(complete.create.resourceType).toBe(ResourceType.Project);
            expect(complete.create.tagsCreate).toHaveLength(2);
            expect(complete.create.tagsCreate[0].tag).toBe("open-source");
            expect(complete.create.versionsCreate[0].translationsCreate).toHaveLength(2);
            
            // Test multilingual support
            const translations = complete.create.versionsCreate[0].translationsCreate;
            const enTranslation = translations.find(t => t.language === "en");
            const esTranslation = translations.find(t => t.language === "es");
            
            expect(enTranslation).toBeDefined();
            expect(esTranslation).toBeDefined();
            expect(enTranslation?.name).toBe("Complete Educational Project");
            expect(esTranslation?.name).toBe("Proyecto Educativo Completo");
        });
        
        test("invalid fixtures", () => {
            const { invalid } = projectFixtures;
            
            // Test missing required fields
            expect(invalid.missingRequired.create.id).toBeUndefined();
            expect(invalid.missingRequired.update.id).toBeUndefined();
            
            // Test invalid types
            expect(typeof invalid.invalidTypes.create.id).toBe("number");
            expect(invalid.invalidTypes.create.resourceType).toBe("InvalidType");
        });
        
        test("edge cases", () => {
            const { edgeCases } = projectFixtures;
            
            // Test minimal valid
            expect(edgeCases.minimalValid.create.resourceType).toBe(ResourceType.Project);
            expect(edgeCases.minimalValid.create.versionsCreate[0].translationsCreate[0].name).toBe("A");
            
            // Test maximal valid
            expect(edgeCases.maximalValid.create.versionsCreate[0].translationsCreate).toHaveLength(3);
            expect(edgeCases.maximalValid.create.tagsCreate).toHaveLength(3);
        });
    });
    
    describe("Factory methods", () => {
        test("createFactory generates valid projects", () => {
            const project = projectAPIFixtures.createFactory();
            
            expect(project.resourceType).toBe(ResourceType.Project);
            expect(project.isPrivate).toBeDefined();
            expect(project.versionsCreate).toBeDefined();
            expect(project.versionsCreate.length).toBeGreaterThan(0);
        });
        
        test("createFactory with overrides", () => {
            const project = projectAPIFixtures.createFactory({
                isPrivate: true,
                permissions: JSON.stringify({ test: true }),
            });
            
            expect(project.isPrivate).toBe(true);
            expect(project.permissions).toBe(JSON.stringify({ test: true }));
        });
        
        test("updateFactory generates valid updates", () => {
            const testId = "test-project-id";
            const update = projectAPIFixtures.updateFactory(testId);
            
            expect(update.id).toBe(testId);
        });
    });
    
    describe("Project-specific helper methods", () => {
        test("createPersonalProject", () => {
            const userId = "user-123";
            const projectName = "My Personal Project";
            
            const project = projectAPIFixtures.createPersonalProject(userId, projectName);
            
            expect(project.ownedByUserConnect).toBe(userId);
            expect(project.isPrivate).toBe(false);
            expect(project.versionsCreate[0].translationsCreate[0].name).toBe(projectName);
        });
        
        test("createTeamProject", () => {
            const teamId = "team-456";
            const projectName = "Team Project";
            
            const project = projectAPIFixtures.createTeamProject(teamId, projectName);
            
            expect(project.ownedByTeamConnect).toBe(teamId);
            expect(project.permissions).toBeDefined();
            expect(project.versionsCreate[0].translationsCreate[0].name).toBe(projectName);
        });
        
        test("createPrivateProject", () => {
            const userId = "user-789";
            const projectName = "Secret Project";
            
            const project = projectAPIFixtures.createPrivateProject(userId, projectName);
            
            expect(project.ownedByUserConnect).toBe(userId);
            expect(project.isPrivate).toBe(true);
            expect(project.versionsCreate[0].isPrivate).toBe(true);
            expect(project.versionsCreate[0].translationsCreate[0].name).toBe(projectName);
        });
        
        test("addProjectVersion", () => {
            const projectId = "project-123";
            const versionLabel = "2.0.0";
            const versionName = "Major Update";
            
            const update = projectAPIFixtures.addProjectVersion(projectId, versionLabel, versionName);
            
            expect(update.id).toBe(projectId);
            expect(update.versionsCreate).toHaveLength(1);
            expect(update.versionsCreate[0].versionLabel).toBe(versionLabel);
            expect(update.versionsCreate[0].translationsCreate[0].name).toBe(versionName);
        });
        
        test("markProjectComplete", () => {
            const projectId = "project-456";
            const versionId = "version-789";
            
            const update = projectAPIFixtures.markProjectComplete(projectId, versionId);
            
            expect(update.id).toBe(projectId);
            expect(update.versionsUpdate).toHaveLength(1);
            expect(update.versionsUpdate[0].id).toBe(versionId);
            expect(update.versionsUpdate[0].isComplete).toBe(true);
        });
        
        test("transferProjectOwnership - to user", () => {
            const projectId = "project-111";
            const newUserId = "new-user-222";
            
            const update = projectAPIFixtures.transferProjectOwnership(projectId, newUserId, false);
            
            expect(update.id).toBe(projectId);
            expect(update.ownedByUserConnect).toBe(newUserId);
            expect(update.ownedByTeamDisconnect).toBe(true);
        });
        
        test("transferProjectOwnership - to team", () => {
            const projectId = "project-333";
            const newTeamId = "new-team-444";
            
            const update = projectAPIFixtures.transferProjectOwnership(projectId, newTeamId, true);
            
            expect(update.id).toBe(projectId);
            expect(update.ownedByTeamConnect).toBe(newTeamId);
            expect(update.ownedByUserDisconnect).toBe(true);
        });
        
        test("addProjectTags", () => {
            const projectId = "project-555";
            const tagNames = ["featured", "community", "beginner"];
            
            const update = projectAPIFixtures.addProjectTags(projectId, tagNames);
            
            expect(update.id).toBe(projectId);
            expect(update.tagsCreate).toHaveLength(3);
            expect(update.tagsCreate.map(t => t.tag)).toEqual(tagNames);
            expect(update.tagsCreate[0].id).toBeDefined();
        });
        
        test("createProjectWithConfig", () => {
            const projectName = "Open Source Project";
            const configType = "openSourceProject";
            
            const project = projectAPIFixtures.createProjectWithConfig(projectName, configType);
            
            expect(project.versionsCreate[0].translationsCreate[0].name).toBe(projectName);
            expect(project.versionsCreate[0].config).toBeDefined();
        });
    });
    
    describe("Data consistency", () => {
        test("all fixtures have consistent IDs", () => {
            const { minimal, complete } = projectFixtures;
            
            // IDs should be valid snowflake IDs
            expect(minimal.create.id).toMatch(/^\d+$/);
            expect(complete.create.id).toMatch(/^\d+$/);
            expect(minimal.find.id).toBe(minimal.create.id);
            expect(complete.find.id).toBe(complete.create.id);
        });
        
        test("version relationships are consistent", () => {
            const { minimal } = projectFixtures;
            
            const createVersionId = minimal.create.versionsCreate[0].id;
            const findVersionId = minimal.find.versions[0].id;
            
            expect(findVersionId).toBe(createVersionId);
        });
        
        test("translation consistency", () => {
            const { complete } = projectFixtures;
            
            const createTranslations = complete.create.versionsCreate[0].translationsCreate;
            const findTranslations = complete.find.versions[0].translations;
            
            expect(findTranslations).toHaveLength(createTranslations.length);
            
            createTranslations.forEach((createTrans, index) => {
                const findTrans = findTranslations[index];
                expect(findTrans.id).toBe(createTrans.id);
                expect(findTrans.language).toBe(createTrans.language);
                expect(findTrans.name).toBe(createTrans.name);
            });
        });
    });
});

/* c8 ignore stop */