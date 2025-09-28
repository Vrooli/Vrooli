/**
 * PhysicsSimulation - Interactive physics for bedroom objects
 * Implements soft-body physics for pillows, rigid-body for toys
 */

import * as THREE from 'three';

export default class PhysicsSimulation {
    constructor(experience) {
        this.experience = experience;
        this.scene = experience.scene;
        this.world = experience.world;
        
        this.enabled = false;
        this.gravity = new THREE.Vector3(0, -9.8, 0);
        this.time = 0;
        this.deltaTime = 0;
        
        this.interactiveObjects = [];
        this.constraints = [];
        this.springs = [];
        
        this.mouse = new THREE.Vector2();
        this.raycaster = new THREE.Raycaster();
        this.selectedObject = null;
        
        this.init();
    }
    
    init() {
        // Create physics-enabled objects
        this.createFloatingBooks();
        this.createBouncingToys();
        this.createSwayingCurtains();
        this.createMagicalOrbs();
        
        // Setup mouse interaction
        this.setupInteraction();
    }
    
    createFloatingBooks() {
        const bookGroup = new THREE.Group();
        bookGroup.name = 'FloatingBooks';
        
        for (let i = 0; i < 5; i++) {
            const book = this.createBook();
            book.position.set(
                Math.random() * 2 - 1,
                2 + Math.random(),
                Math.random() * 2 - 1
            );
            
            // Add physics properties
            book.userData.physics = {
                velocity: new THREE.Vector3(0, 0, 0),
                angularVelocity: new THREE.Vector3(
                    (Math.random() - 0.5) * 0.01,
                    (Math.random() - 0.5) * 0.01,
                    (Math.random() - 0.5) * 0.01
                ),
                mass: 0.5,
                floating: true,
                baseY: book.position.y,
                floatAmplitude: 0.1,
                floatSpeed: Math.random() * 2 + 1,
                phase: Math.random() * Math.PI * 2
            };
            
            this.interactiveObjects.push(book);
            bookGroup.add(book);
        }
        
        if (this.world?.roomGroups?.bedroom) {
            this.world.roomGroups.bedroom.add(bookGroup);
        } else {
            this.scene.add(bookGroup);
        }
    }
    
    createBook() {
        const bookGeometry = new THREE.BoxGeometry(0.15, 0.2, 0.03);
        const bookColors = [0x8b4513, 0x2e4e3e, 0x4a5d8e, 0x8e4a5d, 0x5d8e4a];
        const bookMaterial = new THREE.MeshStandardMaterial({
            color: bookColors[Math.floor(Math.random() * bookColors.length)],
            roughness: 0.8
        });
        
        const book = new THREE.Mesh(bookGeometry, bookMaterial);
        book.castShadow = true;
        book.receiveShadow = true;
        
        return book;
    }
    
    createBouncingToys() {
        const toyGroup = new THREE.Group();
        toyGroup.name = 'BouncingToys';
        
        // Create different toy types
        const toyTypes = [
            { geometry: new THREE.SphereGeometry(0.08), color: 0xff6b6b },
            { geometry: new THREE.BoxGeometry(0.12, 0.12, 0.12), color: 0x4ecdc4 },
            { geometry: new THREE.TetrahedronGeometry(0.1), color: 0xffe66d },
            { geometry: new THREE.OctahedronGeometry(0.09), color: 0xa8e6cf }
        ];
        
        for (let i = 0; i < 8; i++) {
            const toyType = toyTypes[i % toyTypes.length];
            const toy = new THREE.Mesh(
                toyType.geometry,
                new THREE.MeshStandardMaterial({
                    color: toyType.color,
                    roughness: 0.3,
                    metalness: 0.2
                })
            );
            
            toy.position.set(
                Math.random() * 3 - 1.5,
                0.5 + Math.random() * 0.5,
                Math.random() * 3 - 1.5
            );
            
            toy.userData.physics = {
                velocity: new THREE.Vector3(
                    (Math.random() - 0.5) * 2,
                    Math.random() * 3,
                    (Math.random() - 0.5) * 2
                ),
                angularVelocity: new THREE.Vector3(
                    Math.random() * 0.1,
                    Math.random() * 0.1,
                    Math.random() * 0.1
                ),
                mass: 0.2,
                restitution: 0.8,
                friction: 0.3,
                damping: 0.98
            };
            
            toy.castShadow = true;
            toy.receiveShadow = true;
            
            this.interactiveObjects.push(toy);
            toyGroup.add(toy);
        }
        
        if (this.world?.roomGroups?.bedroom) {
            this.world.roomGroups.bedroom.add(toyGroup);
        } else {
            this.scene.add(toyGroup);
        }
    }
    
    createSwayingCurtains() {
        const curtainGroup = new THREE.Group();
        curtainGroup.name = 'SwayingCurtains';
        
        const curtainWidth = 1.5;
        const curtainHeight = 2;
        const segments = 10;
        
        const curtainGeometry = new THREE.PlaneGeometry(
            curtainWidth,
            curtainHeight,
            segments,
            segments * 2
        );
        
        const curtainMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b8eb8,
            roughness: 0.7,
            side: THREE.DoubleSide,
            transparent: true,
            opacity: 0.8
        });
        
        const curtain = new THREE.Mesh(curtainGeometry, curtainMaterial);
        curtain.position.set(-3.5, 1, 0);
        
        // Store original vertices for wave animation
        const vertices = curtainGeometry.attributes.position.array;
        curtain.userData.originalVertices = new Float32Array(vertices);
        curtain.userData.physics = {
            waveAmplitude: 0.05,
            waveSpeed: 2,
            windStrength: 0.02,
            time: 0
        };
        
        this.interactiveObjects.push(curtain);
        curtainGroup.add(curtain);
        
        if (this.world?.roomGroups?.bedroom) {
            this.world.roomGroups.bedroom.add(curtainGroup);
        } else {
            this.scene.add(curtainGroup);
        }
    }
    
    createMagicalOrbs() {
        const orbGroup = new THREE.Group();
        orbGroup.name = 'MagicalOrbs';
        
        for (let i = 0; i < 3; i++) {
            const orb = new THREE.Group();
            
            // Core sphere
            const coreGeometry = new THREE.SphereGeometry(0.1, 16, 16);
            const coreMaterial = new THREE.MeshStandardMaterial({
                color: new THREE.Color().setHSL(i / 3, 1, 0.5),
                emissive: new THREE.Color().setHSL(i / 3, 1, 0.3),
                emissiveIntensity: 2,
                roughness: 0.1,
                metalness: 0.8
            });
            const core = new THREE.Mesh(coreGeometry, coreMaterial);
            orb.add(core);
            
            // Glow sphere
            const glowGeometry = new THREE.SphereGeometry(0.15, 16, 16);
            const glowMaterial = new THREE.MeshBasicMaterial({
                color: new THREE.Color().setHSL(i / 3, 1, 0.7),
                transparent: true,
                opacity: 0.3
            });
            const glow = new THREE.Mesh(glowGeometry, glowMaterial);
            orb.add(glow);
            
            // Position in space
            orb.position.set(
                Math.cos(i * Math.PI * 2 / 3) * 2,
                1.5,
                Math.sin(i * Math.PI * 2 / 3) * 2
            );
            
            orb.userData.physics = {
                velocity: new THREE.Vector3(0, 0, 0),
                mass: 0.1,
                attraction: [],
                orbitRadius: 2,
                orbitSpeed: 0.5,
                orbitPhase: i * Math.PI * 2 / 3,
                pulsePhase: Math.random() * Math.PI * 2,
                attractionForce: 0.001
            };
            
            this.interactiveObjects.push(orb);
            orbGroup.add(orb);
        }
        
        // Create attraction between orbs
        for (let i = 0; i < 3; i++) {
            for (let j = i + 1; j < 3; j++) {
                this.springs.push({
                    objectA: orbGroup.children[i],
                    objectB: orbGroup.children[j],
                    restLength: 3,
                    strength: 0.01,
                    damping: 0.95
                });
            }
        }
        
        if (this.world?.roomGroups?.bedroom) {
            this.world.roomGroups.bedroom.add(orbGroup);
        } else {
            this.scene.add(orbGroup);
        }
    }
    
    setupInteraction() {
        window.addEventListener('mousemove', (event) => {
            this.mouse.x = (event.clientX / window.innerWidth) * 2 - 1;
            this.mouse.y = -(event.clientY / window.innerHeight) * 2 + 1;
        });
        
        window.addEventListener('mousedown', () => {
            this.checkIntersection();
        });
        
        window.addEventListener('mouseup', () => {
            this.selectedObject = null;
        });
    }
    
    checkIntersection() {
        if (!this.experience.camera) return;
        
        this.raycaster.setFromCamera(this.mouse, this.experience.camera.instance);
        const intersects = this.raycaster.intersectObjects(this.interactiveObjects, true);
        
        if (intersects.length > 0) {
            this.selectedObject = intersects[0].object;
            
            // Apply impulse to selected object
            if (this.selectedObject.userData.physics) {
                const physics = this.selectedObject.userData.physics;
                physics.velocity.add(new THREE.Vector3(
                    (Math.random() - 0.5) * 5,
                    Math.random() * 5 + 2,
                    (Math.random() - 0.5) * 5
                ));
            }
        }
    }
    
    update(deltaTime) {
        if (!this.enabled) return;
        
        this.deltaTime = Math.min(deltaTime, 0.1); // Cap delta time
        this.time += this.deltaTime;
        
        // Update each physics object
        this.interactiveObjects.forEach(object => {
            if (!object.userData.physics) return;
            
            const physics = object.userData.physics;
            
            // Floating books
            if (physics.floating) {
                const floatY = physics.baseY + 
                    Math.sin(this.time * physics.floatSpeed + physics.phase) * 
                    physics.floatAmplitude;
                object.position.y = floatY;
                
                // Gentle rotation
                object.rotation.x += physics.angularVelocity.x;
                object.rotation.y += physics.angularVelocity.y;
                object.rotation.z += physics.angularVelocity.z;
            }
            
            // Bouncing toys
            else if (physics.restitution) {
                // Apply gravity
                physics.velocity.add(
                    this.gravity.clone().multiplyScalar(this.deltaTime * physics.mass)
                );
                
                // Update position
                object.position.add(
                    physics.velocity.clone().multiplyScalar(this.deltaTime)
                );
                
                // Ground collision
                if (object.position.y < 0.1) {
                    object.position.y = 0.1;
                    physics.velocity.y *= -physics.restitution;
                    physics.velocity.multiplyScalar(physics.damping);
                }
                
                // Wall collisions
                const bounds = 3;
                if (Math.abs(object.position.x) > bounds) {
                    object.position.x = Math.sign(object.position.x) * bounds;
                    physics.velocity.x *= -physics.restitution;
                }
                if (Math.abs(object.position.z) > bounds) {
                    object.position.z = Math.sign(object.position.z) * bounds;
                    physics.velocity.z *= -physics.restitution;
                }
                
                // Apply rotation
                object.rotation.x += physics.angularVelocity.x;
                object.rotation.y += physics.angularVelocity.y;
                object.rotation.z += physics.angularVelocity.z;
            }
            
            // Magical orbs
            else if (physics.orbitRadius) {
                // Orbital motion
                const angle = this.time * physics.orbitSpeed + physics.orbitPhase;
                const targetX = Math.cos(angle) * physics.orbitRadius;
                const targetZ = Math.sin(angle) * physics.orbitRadius;
                
                // Smooth movement towards orbit position
                object.position.x += (targetX - object.position.x) * 0.05;
                object.position.z += (targetZ - object.position.z) * 0.05;
                
                // Vertical floating
                object.position.y = 1.5 + Math.sin(this.time * 2 + physics.orbitPhase) * 0.2;
                
                // Pulsing glow
                if (object.children[1]) {
                    const pulse = 0.3 + Math.sin(this.time * 3 + physics.pulsePhase) * 0.1;
                    object.children[1].material.opacity = pulse;
                }
            }
            
            // Swaying curtains
            else if (object.userData.originalVertices) {
                const vertices = object.geometry.attributes.position.array;
                const original = object.userData.originalVertices;
                const phys = object.userData.physics;
                
                for (let i = 0; i < vertices.length; i += 3) {
                    const x = original[i];
                    const y = original[i + 1];
                    const z = original[i + 2];
                    
                    // Wave effect based on height
                    const waveOffset = Math.sin(
                        this.time * phys.waveSpeed + y * 2
                    ) * phys.waveAmplitude * (1 - y / 2);
                    
                    // Wind effect
                    const windOffset = Math.sin(
                        this.time * 0.5 + x
                    ) * phys.windStrength;
                    
                    vertices[i] = x;
                    vertices[i + 1] = y;
                    vertices[i + 2] = z + waveOffset + windOffset;
                }
                
                object.geometry.attributes.position.needsUpdate = true;
                object.geometry.computeVertexNormals();
            }
        });
        
        // Update springs (orb connections)
        this.springs.forEach(spring => {
            const delta = spring.objectB.position.clone()
                .sub(spring.objectA.position);
            const distance = delta.length();
            const force = (distance - spring.restLength) * spring.strength;
            
            delta.normalize().multiplyScalar(force);
            
            if (spring.objectA.userData.physics) {
                spring.objectA.userData.physics.velocity.add(delta);
                spring.objectA.userData.physics.velocity.multiplyScalar(spring.damping);
            }
            if (spring.objectB.userData.physics) {
                spring.objectB.userData.physics.velocity.sub(delta);
                spring.objectB.userData.physics.velocity.multiplyScalar(spring.damping);
            }
        });
    }
    
    enable() {
        this.enabled = true;
    }
    
    disable() {
        this.enabled = false;
    }
    
    toggle() {
        this.enabled = !this.enabled;
        return this.enabled;
    }
    
    dispose() {
        window.removeEventListener('mousemove', this.handleMouseMove);
        window.removeEventListener('mousedown', this.handleMouseDown);
        window.removeEventListener('mouseup', this.handleMouseUp);
        
        this.interactiveObjects.forEach(object => {
            if (object.geometry) object.geometry.dispose();
            if (object.material) object.material.dispose();
        });
        
        this.interactiveObjects = [];
        this.springs = [];
        this.constraints = [];
    }
}