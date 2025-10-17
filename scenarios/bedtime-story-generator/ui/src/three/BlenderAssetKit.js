/**
 * BlenderAssetKit - Generates starter Blender scenes for immersive scenarios
 * Exports scene configurations and templates that can be imported into Blender
 */

import * as THREE from 'three';
import { GLTFExporter } from 'three/addons/exporters/GLTFExporter.js';

export class BlenderAssetKit {
    constructor() {
        this.exporter = new GLTFExporter();
        this.templates = {
            bedroom: this.createBedroomTemplate(),
            classroom: this.createClassroomTemplate(),
            office: this.createOfficeTemplate(),
            outdoor: this.createOutdoorTemplate()
        };
    }

    /**
     * Creates a bedroom template scene
     */
    createBedroomTemplate() {
        const scene = new THREE.Scene();
        
        // Room structure
        const roomGeometry = new THREE.BoxGeometry(10, 3, 10);
        const roomMaterial = new THREE.MeshBasicMaterial({ 
            color: 0xf5f5f5,
            side: THREE.BackSide 
        });
        const room = new THREE.Mesh(roomGeometry, roomMaterial);
        room.name = 'Room';
        scene.add(room);

        // Floor
        const floorGeometry = new THREE.PlaneGeometry(10, 10);
        const floorMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x8b7355,
            roughness: 0.8 
        });
        const floor = new THREE.Mesh(floorGeometry, floorMaterial);
        floor.rotation.x = -Math.PI / 2;
        floor.position.y = -1.5;
        floor.name = 'Floor';
        scene.add(floor);

        // Bed placeholder
        const bedGroup = new THREE.Group();
        bedGroup.name = 'Bed';
        
        const bedFrame = new THREE.Mesh(
            new THREE.BoxGeometry(2, 0.5, 3),
            new THREE.MeshStandardMaterial({ color: 0x654321 })
        );
        bedFrame.position.y = -1.25;
        bedFrame.name = 'BedFrame';
        
        const mattress = new THREE.Mesh(
            new THREE.BoxGeometry(1.8, 0.3, 2.8),
            new THREE.MeshStandardMaterial({ color: 0xffffff })
        );
        mattress.position.y = -0.85;
        mattress.name = 'Mattress';
        
        bedGroup.add(bedFrame, mattress);
        bedGroup.position.set(0, 0, -3);
        scene.add(bedGroup);

        // Window placeholder
        const windowFrame = new THREE.Mesh(
            new THREE.BoxGeometry(2, 2, 0.1),
            new THREE.MeshStandardMaterial({ color: 0x333333 })
        );
        windowFrame.position.set(-4.95, 0.5, 0);
        windowFrame.name = 'Window';
        scene.add(windowFrame);

        // Bookshelf placeholder
        const bookshelfGroup = new THREE.Group();
        bookshelfGroup.name = 'Bookshelf';
        
        const shelfFrame = new THREE.Mesh(
            new THREE.BoxGeometry(1.5, 2, 0.3),
            new THREE.MeshStandardMaterial({ color: 0x8b4513 })
        );
        
        for (let i = 0; i < 3; i++) {
            const shelf = new THREE.Mesh(
                new THREE.BoxGeometry(1.4, 0.05, 0.25),
                new THREE.MeshStandardMaterial({ color: 0x8b4513 })
            );
            shelf.position.y = -0.5 + i * 0.6;
            shelf.name = `Shelf_${i}`;
            bookshelfGroup.add(shelf);
        }
        
        bookshelfGroup.add(shelfFrame);
        bookshelfGroup.position.set(3, 0, -3);
        scene.add(bookshelfGroup);

        // Lighting setup
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.3);
        ambientLight.name = 'AmbientLight';
        scene.add(ambientLight);

        const mainLight = new THREE.DirectionalLight(0xffffff, 0.7);
        mainLight.position.set(5, 5, 5);
        mainLight.castShadow = true;
        mainLight.name = 'MainLight';
        scene.add(mainLight);

        // Camera marker
        const cameraMarker = new THREE.Mesh(
            new THREE.SphereGeometry(0.1),
            new THREE.MeshBasicMaterial({ color: 0xff0000 })
        );
        cameraMarker.position.set(3, 0, 3);
        cameraMarker.name = 'CameraPosition';
        scene.add(cameraMarker);

        return scene;
    }

    /**
     * Creates a classroom template scene
     */
    createClassroomTemplate() {
        const scene = new THREE.Scene();
        
        // Room
        const roomGeometry = new THREE.BoxGeometry(12, 3.5, 8);
        const roomMaterial = new THREE.MeshBasicMaterial({ 
            color: 0xf0f0f0,
            side: THREE.BackSide 
        });
        const room = new THREE.Mesh(roomGeometry, roomMaterial);
        room.name = 'Classroom';
        scene.add(room);

        // Desks arrangement
        for (let row = 0; row < 3; row++) {
            for (let col = 0; col < 4; col++) {
                const deskGroup = new THREE.Group();
                deskGroup.name = `Desk_${row}_${col}`;
                
                const deskTop = new THREE.Mesh(
                    new THREE.BoxGeometry(0.8, 0.05, 0.6),
                    new THREE.MeshStandardMaterial({ color: 0x8b7355 })
                );
                deskTop.position.y = 0;
                
                const deskLeg = new THREE.Mesh(
                    new THREE.BoxGeometry(0.05, 0.7, 0.05),
                    new THREE.MeshStandardMaterial({ color: 0x666666 })
                );
                
                deskGroup.add(deskTop);
                for (let i = 0; i < 4; i++) {
                    const leg = deskLeg.clone();
                    leg.position.set(
                        (i % 2) * 0.7 - 0.35,
                        -0.35,
                        Math.floor(i / 2) * 0.5 - 0.25
                    );
                    deskGroup.add(leg);
                }
                
                deskGroup.position.set(
                    col * 2 - 3,
                    -1.2,
                    row * 2 - 2
                );
                scene.add(deskGroup);
            }
        }

        // Blackboard
        const blackboard = new THREE.Mesh(
            new THREE.BoxGeometry(4, 2, 0.1),
            new THREE.MeshStandardMaterial({ color: 0x1a1a1a })
        );
        blackboard.position.set(0, 0.5, -3.9);
        blackboard.name = 'Blackboard';
        scene.add(blackboard);

        // Teacher desk
        const teacherDesk = new THREE.Mesh(
            new THREE.BoxGeometry(1.5, 0.8, 0.8),
            new THREE.MeshStandardMaterial({ color: 0x654321 })
        );
        teacherDesk.position.set(0, -1, -3);
        teacherDesk.name = 'TeacherDesk';
        scene.add(teacherDesk);

        // Lighting
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.4);
        scene.add(ambientLight);

        const fluorescent1 = new THREE.RectAreaLight(0xffffff, 5, 3, 0.5);
        fluorescent1.position.set(-2, 1.7, 0);
        fluorescent1.rotation.x = -Math.PI / 2;
        fluorescent1.name = 'FluorescentLight1';
        scene.add(fluorescent1);

        const fluorescent2 = fluorescent1.clone();
        fluorescent2.position.x = 2;
        fluorescent2.name = 'FluorescentLight2';
        scene.add(fluorescent2);

        return scene;
    }

    /**
     * Creates an office template scene
     */
    createOfficeTemplate() {
        const scene = new THREE.Scene();
        
        // Office space
        const officeGeometry = new THREE.BoxGeometry(15, 3, 10);
        const officeMaterial = new THREE.MeshBasicMaterial({ 
            color: 0xf8f8f8,
            side: THREE.BackSide 
        });
        const office = new THREE.Mesh(officeGeometry, officeMaterial);
        office.name = 'Office';
        scene.add(office);

        // Workstations
        for (let i = 0; i < 6; i++) {
            const workstation = new THREE.Group();
            workstation.name = `Workstation_${i}`;
            
            // Desk
            const desk = new THREE.Mesh(
                new THREE.BoxGeometry(1.2, 0.05, 0.8),
                new THREE.MeshStandardMaterial({ color: 0x696969 })
            );
            desk.position.y = 0;
            
            // Monitor placeholder
            const monitor = new THREE.Mesh(
                new THREE.BoxGeometry(0.6, 0.4, 0.05),
                new THREE.MeshStandardMaterial({ color: 0x000000 })
            );
            monitor.position.set(0, 0.25, 0);
            
            // Chair placeholder
            const chair = new THREE.Mesh(
                new THREE.BoxGeometry(0.5, 0.5, 0.5),
                new THREE.MeshStandardMaterial({ color: 0x2b2b2b })
            );
            chair.position.set(0, -0.5, 0.5);
            
            workstation.add(desk, monitor, chair);
            workstation.position.set(
                (i % 3) * 3 - 3,
                -1.2,
                Math.floor(i / 3) * 3 - 1.5
            );
            scene.add(workstation);
        }

        // Meeting table
        const meetingTable = new THREE.Mesh(
            new THREE.BoxGeometry(3, 0.05, 1.5),
            new THREE.MeshStandardMaterial({ color: 0x4a4a4a })
        );
        meetingTable.position.set(-5, -1, 0);
        meetingTable.name = 'MeetingTable';
        scene.add(meetingTable);

        // Whiteboard
        const whiteboard = new THREE.Mesh(
            new THREE.BoxGeometry(2.5, 1.5, 0.05),
            new THREE.MeshStandardMaterial({ color: 0xffffff })
        );
        whiteboard.position.set(0, 0.5, -4.95);
        whiteboard.name = 'Whiteboard';
        scene.add(whiteboard);

        // Office lighting
        const officeAmbient = new THREE.AmbientLight(0xffffff, 0.5);
        scene.add(officeAmbient);

        for (let i = 0; i < 3; i++) {
            const ceilingLight = new THREE.PointLight(0xffffff, 0.5);
            ceilingLight.position.set(i * 4 - 4, 1.4, 0);
            ceilingLight.name = `CeilingLight_${i}`;
            scene.add(ceilingLight);
        }

        return scene;
    }

    /**
     * Creates an outdoor template scene
     */
    createOutdoorTemplate() {
        const scene = new THREE.Scene();
        
        // Ground
        const groundGeometry = new THREE.PlaneGeometry(50, 50);
        const groundMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x3a5f3a,
            roughness: 0.9 
        });
        const ground = new THREE.Mesh(groundGeometry, groundMaterial);
        ground.rotation.x = -Math.PI / 2;
        ground.name = 'Ground';
        scene.add(ground);

        // Trees
        for (let i = 0; i < 10; i++) {
            const tree = new THREE.Group();
            tree.name = `Tree_${i}`;
            
            // Trunk
            const trunk = new THREE.Mesh(
                new THREE.CylinderGeometry(0.3, 0.4, 3),
                new THREE.MeshStandardMaterial({ color: 0x4a3c28 })
            );
            trunk.position.y = 1.5;
            
            // Foliage
            const foliage = new THREE.Mesh(
                new THREE.ConeGeometry(2, 4, 8),
                new THREE.MeshStandardMaterial({ color: 0x2d5016 })
            );
            foliage.position.y = 4;
            
            tree.add(trunk, foliage);
            tree.position.set(
                Math.random() * 40 - 20,
                0,
                Math.random() * 40 - 20
            );
            tree.rotation.y = Math.random() * Math.PI * 2;
            scene.add(tree);
        }

        // Rocks
        for (let i = 0; i < 15; i++) {
            const rock = new THREE.Mesh(
                new THREE.SphereGeometry(Math.random() * 0.5 + 0.2, 6, 5),
                new THREE.MeshStandardMaterial({ 
                    color: 0x808080,
                    roughness: 1 
                })
            );
            rock.position.set(
                Math.random() * 30 - 15,
                Math.random() * 0.2,
                Math.random() * 30 - 15
            );
            rock.scale.y = Math.random() * 0.5 + 0.5;
            rock.name = `Rock_${i}`;
            scene.add(rock);
        }

        // Sky dome
        const skyGeometry = new THREE.SphereGeometry(45, 32, 15);
        const skyMaterial = new THREE.MeshBasicMaterial({ 
            color: 0x87ceeb,
            side: THREE.BackSide 
        });
        const sky = new THREE.Mesh(skyGeometry, skyMaterial);
        sky.name = 'Sky';
        scene.add(sky);

        // Sun light
        const sunLight = new THREE.DirectionalLight(0xffffff, 1);
        sunLight.position.set(10, 20, 10);
        sunLight.castShadow = true;
        sunLight.shadow.camera.left = -25;
        sunLight.shadow.camera.right = 25;
        sunLight.shadow.camera.top = 25;
        sunLight.shadow.camera.bottom = -25;
        sunLight.name = 'SunLight';
        scene.add(sunLight);

        // Ambient light
        const ambientLight = new THREE.AmbientLight(0x404040, 0.5);
        scene.add(ambientLight);

        return scene;
    }

    /**
     * Exports a template scene to GLTF format
     */
    async exportTemplate(templateName, options = {}) {
        const scene = this.templates[templateName];
        if (!scene) {
            throw new Error(`Template '${templateName}' not found`);
        }

        return new Promise((resolve, reject) => {
            this.exporter.parse(
                scene,
                (result) => {
                    if (result instanceof ArrayBuffer) {
                        // Binary GLB format
                        resolve(new Blob([result], { type: 'model/gltf-binary' }));
                    } else {
                        // JSON GLTF format
                        const json = JSON.stringify(result, null, 2);
                        resolve(new Blob([json], { type: 'model/gltf+json' }));
                    }
                },
                (error) => reject(error),
                options
            );
        });
    }

    /**
     * Generates a Python script for Blender to create the scene
     */
    generateBlenderScript(templateName) {
        const template = this.templates[templateName];
        if (!template) {
            throw new Error(`Template '${templateName}' not found`);
        }

        let script = `# Blender Python Script for ${templateName} template
# Generated by Bedtime Story Generator Asset Kit
# Usage: Open Blender, switch to Scripting tab, paste this script and run

import bpy
import mathutils

# Clear existing mesh objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)

# Create new collection for the scene
collection = bpy.data.collections.new("${templateName}_template")
bpy.context.scene.collection.children.link(collection)

`;

        // Add objects based on template
        template.traverse((child) => {
            if (child.isMesh) {
                script += this.generateBlenderObject(child);
            } else if (child.isLight) {
                script += this.generateBlenderLight(child);
            } else if (child.isGroup) {
                script += `# Group: ${child.name}\n`;
            }
        });

        script += `
# Set viewport shading to solid
for area in bpy.context.screen.areas:
    if area.type == 'VIEW_3D':
        for space in area.spaces:
            if space.type == 'VIEW_3D':
                space.shading.type = 'SOLID'
                
print("${templateName} template created successfully!")
`;

        return script;
    }

    /**
     * Generates Blender Python code for a mesh object
     */
    generateBlenderObject(mesh) {
        const geometry = mesh.geometry;
        const material = mesh.material;
        const position = mesh.position;
        const rotation = mesh.rotation;
        const scale = mesh.scale;

        let script = `# Object: ${mesh.name}\n`;

        if (geometry.type === 'BoxGeometry') {
            const params = geometry.parameters;
            script += `bpy.ops.mesh.primitive_cube_add(
    size=2,
    location=(${position.x}, ${position.z}, ${position.y})
)
obj = bpy.context.active_object
obj.name = "${mesh.name}"
obj.scale = (${params.width/2}, ${params.depth/2}, ${params.height/2})
`;
        } else if (geometry.type === 'PlaneGeometry') {
            const params = geometry.parameters;
            script += `bpy.ops.mesh.primitive_plane_add(
    size=1,
    location=(${position.x}, ${position.z}, ${position.y})
)
obj = bpy.context.active_object
obj.name = "${mesh.name}"
obj.scale = (${params.width}, ${params.height}, 1)
obj.rotation_euler = (${rotation.x}, ${rotation.z}, ${rotation.y})
`;
        } else if (geometry.type === 'SphereGeometry') {
            const params = geometry.parameters;
            script += `bpy.ops.mesh.primitive_uv_sphere_add(
    radius=${params.radius || 1},
    location=(${position.x}, ${position.z}, ${position.y})
)
obj = bpy.context.active_object
obj.name = "${mesh.name}"
`;
        } else if (geometry.type === 'CylinderGeometry') {
            const params = geometry.parameters;
            script += `bpy.ops.mesh.primitive_cylinder_add(
    radius=${params.radiusTop || 1},
    depth=${params.height || 2},
    location=(${position.x}, ${position.z}, ${position.y})
)
obj = bpy.context.active_object
obj.name = "${mesh.name}"
`;
        } else if (geometry.type === 'ConeGeometry') {
            const params = geometry.parameters;
            script += `bpy.ops.mesh.primitive_cone_add(
    radius1=${params.radius || 1},
    depth=${params.height || 2},
    location=(${position.x}, ${position.z}, ${position.y})
)
obj = bpy.context.active_object
obj.name = "${mesh.name}"
`;
        }

        // Add material
        if (material && material.color) {
            const color = material.color;
            script += `
# Create and assign material
mat = bpy.data.materials.new(name="${mesh.name}_Material")
mat.use_nodes = True
mat.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (${color.r}, ${color.g}, ${color.b}, 1.0)
obj.data.materials.append(mat)
`;
        }

        script += `collection.objects.link(obj)
bpy.context.scene.collection.objects.unlink(obj)

`;

        return script;
    }

    /**
     * Generates Blender Python code for a light
     */
    generateBlenderLight(light) {
        let script = `# Light: ${light.name}\n`;
        
        if (light.isDirectionalLight) {
            script += `bpy.ops.object.light_add(
    type='SUN',
    location=(${light.position.x}, ${light.position.z}, ${light.position.y})
)
light = bpy.context.active_object
light.name = "${light.name}"
light.data.energy = ${light.intensity * 10}
`;
        } else if (light.isPointLight) {
            script += `bpy.ops.object.light_add(
    type='POINT',
    location=(${light.position.x}, ${light.position.z}, ${light.position.y})
)
light = bpy.context.active_object
light.name = "${light.name}"
light.data.energy = ${light.intensity * 100}
`;
        } else if (light.isAmbientLight) {
            script += `# Ambient light - setting world environment
bpy.context.scene.world.node_tree.nodes["Background"].inputs[0].default_value = (${light.color.r}, ${light.color.g}, ${light.color.b}, 1.0)
bpy.context.scene.world.node_tree.nodes["Background"].inputs[1].default_value = ${light.intensity}
`;
        }

        if (!light.isAmbientLight) {
            script += `collection.objects.link(light)
bpy.context.scene.collection.objects.unlink(light)

`;
        }

        return script;
    }

    /**
     * Downloads the exported file
     */
    downloadFile(blob, filename) {
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        link.click();
        URL.revokeObjectURL(url);
    }

    /**
     * Exports all templates at once
     */
    async exportAllTemplates() {
        const exports = {};
        
        for (const templateName in this.templates) {
            try {
                // Export as GLB
                const glbBlob = await this.exportTemplate(templateName, { binary: true });
                
                // Generate Blender script
                const script = this.generateBlenderScript(templateName);
                const scriptBlob = new Blob([script], { type: 'text/x-python' });
                
                exports[templateName] = {
                    glb: glbBlob,
                    script: scriptBlob,
                    metadata: {
                        name: templateName,
                        objectCount: this.countObjects(this.templates[templateName]),
                        lightCount: this.countLights(this.templates[templateName]),
                        exportDate: new Date().toISOString()
                    }
                };
            } catch (error) {
                console.error(`Failed to export ${templateName}:`, error);
            }
        }
        
        return exports;
    }

    /**
     * Counts objects in a scene
     */
    countObjects(scene) {
        let count = 0;
        scene.traverse((child) => {
            if (child.isMesh) count++;
        });
        return count;
    }

    /**
     * Counts lights in a scene
     */
    countLights(scene) {
        let count = 0;
        scene.traverse((child) => {
            if (child.isLight) count++;
        });
        return count;
    }

    /**
     * Creates a ZIP file containing all exports
     */
    async createAssetPackage() {
        const { default: JSZip } = await import('jszip');
        const zip = new JSZip();
        
        const exports = await this.exportAllTemplates();
        
        for (const [templateName, data] of Object.entries(exports)) {
            const folder = zip.folder(templateName);
            
            // Add GLB file
            folder.file(`${templateName}.glb`, data.glb);
            
            // Add Blender script
            folder.file(`${templateName}_blender.py`, data.script);
            
            // Add metadata
            folder.file('metadata.json', JSON.stringify(data.metadata, null, 2));
        }
        
        // Add README
        const readme = `# Bedtime Story Generator - Asset Kit

This asset kit contains starter templates for creating immersive 3D scenarios.

## Templates Included
- bedroom: A cozy bedroom scene with bed, bookshelf, and window
- classroom: A classroom setup with desks, blackboard, and lighting
- office: Modern office space with workstations and meeting area
- outdoor: Natural outdoor environment with trees and rocks

## Usage

### Using GLB Files
1. Import the .glb file into your 3D software (Blender, Unity, Unreal, etc.)
2. The scene hierarchy and materials are preserved
3. Modify and enhance as needed

### Using Blender Scripts
1. Open Blender
2. Switch to the Scripting tab
3. Open the .py file or paste its contents
4. Click "Run Script"
5. The template will be created in your scene

## Structure
Each template folder contains:
- {template}.glb - The 3D scene in GLB format
- {template}_blender.py - Python script to recreate in Blender
- metadata.json - Information about the template

## Tips
- Templates use simple geometry for easy modification
- Materials are basic colors - enhance with textures
- Lighting is set up for good visibility
- Scale and proportions are realistic

Generated by Bedtime Story Generator v1.0
`;
        
        zip.file('README.md', readme);
        
        return zip.generateAsync({ type: 'blob' });
    }
}

// Export singleton instance
export default new BlenderAssetKit();