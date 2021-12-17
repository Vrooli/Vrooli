import { PrismaSelect } from "@paljs/plugins";
import { Resource, ResourceInput } from "schema/types";
import { BaseModel } from "./base";

export const applyMap = {
    ORGANIZATION: 'organizationResources',
    PROJECT: 'projectResources',
    ROUTINE_CONTEXTUAL: 'routineResourcesContextual',
    ROUTINE_DONATION: 'routineResourcesDonation',
    ROUTINE_EXTERNAL: 'routineResourcesExternal',
    USER: 'userResources'
}

export class ResourceModel extends BaseModel<ResourceInput, Resource> {
    
    constructor(prisma: any) {
        super(prisma, 'resource');
    }

    /**
     * Creates a resource object
     * @param data 
     * @param info 
     * @returns 
     */
    async create(data: ResourceInput, info: any) {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Create base object
        const resource = await this.prisma.resouce.create({ data: resourceData });
        // Create join object
        await this.applyToObject(resource, createdFor, forId);
        // Return query
        return await this.prisma.resource.findOne({ where: { id: resource.id }, ...(new PrismaSelect(info).value) });
    }

    /**
     * Applies a resource object to an actor, project, organization, or routine
     * @param resource 
     * @returns
     */
    async applyToObject(resource: any, createdFor: keyof typeof applyMap, forId: string) {
        // Maps routine apply types to the correct prisma join tables
        const applyMap = {
            ORGANIZATION: 'organizationResources',
            PROJECT: 'projectResources',
            ROUTINE_CONTEXTUAL: 'routineResourcesContextual',
            ROUTINE_DONATION: 'routineResourcesDonation',
            ROUTINE_EXTERNAL: 'routineResourcesExternal',
            USER: 'userResources'
        }
        return await this.prisma[applyMap[createdFor]].create({ data: {
            forId,
            resourceId: resource.id
        }})
    }
    
    async update(data: ResourceInput, info: any): Promise<any> {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Check if resource needs to be associated with another object instead
        //TODO
        // Update base object and return query
        return await this.prisma.resource.update({
            where: { id: data.id },
            data: resourceData,
            ...(new PrismaSelect(info).value)
        });
    }
}