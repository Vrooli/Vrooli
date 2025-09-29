/**
 * PostProcessing - Advanced visual effects for the 3D scene
 * Implements bloom, depth of field, and color grading
 */

import * as THREE from 'three';
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';
import { ShaderPass } from 'three/addons/postprocessing/ShaderPass.js';
import { FXAAShader } from 'three/addons/shaders/FXAAShader.js';

export default class PostProcessing {
    constructor(experience) {
        this.experience = experience;
        this.renderer = experience.renderer.instance;
        this.camera = experience.camera.instance;
        this.scene = experience.scene;
        this.sizes = experience.sizes;
        
        this.enabled = true;
        this.composer = null;
        
        this.params = {
            bloom: {
                enabled: true,
                strength: 0.3,
                radius: 0.4,
                threshold: 0.85
            },
            colorGrading: {
                enabled: true,
                contrast: 1.1,
                saturation: 1.2,
                brightness: 1.0
            },
            vignette: {
                enabled: true,
                darkness: 0.3,
                offset: 1.0
            },
            dof: {
                enabled: false,
                focus: 5.0,
                aperture: 0.025,
                maxblur: 0.01
            }
        };
        
        this.init();
    }
    
    init() {
        const width = this.sizes.width;
        const height = this.sizes.height;
        const pixelRatio = this.sizes.pixelRatio;

        const renderTarget = new THREE.WebGLRenderTarget(
            width,
            height,
            {
                minFilter: THREE.LinearFilter,
                magFilter: THREE.LinearFilter,
                format: THREE.RGBAFormat,
                colorSpace: THREE.SRGBColorSpace
            }
        );

        this.composer = new EffectComposer(this.renderer, renderTarget);
        this.composer.setPixelRatio(pixelRatio);
        
        // Render pass
        const renderPass = new RenderPass(this.scene, this.camera);
        this.composer.addPass(renderPass);
        
        // Bloom pass
        this.bloomPass = new UnrealBloomPass(
            new THREE.Vector2(width, height),
            this.params.bloom.strength,
            this.params.bloom.radius,
            this.params.bloom.threshold
        );
        this.composer.addPass(this.bloomPass);
        
        // Color grading and vignette
        this.colorGradingPass = new ShaderPass(this.createColorGradingShader());
        this.composer.addPass(this.colorGradingPass);
        
        // FXAA anti-aliasing
        const fxaaPass = new ShaderPass(FXAAShader);
        fxaaPass.uniforms['resolution'].value.set(
            1 / (width * pixelRatio),
            1 / (height * pixelRatio)
        );
        this.composer.addPass(fxaaPass);
        
        this.updateParams();
    }
    
    createColorGradingShader() {
        return {
            uniforms: {
                tDiffuse: { value: null },
                contrast: { value: 1.0 },
                saturation: { value: 1.0 },
                brightness: { value: 1.0 },
                vignetteEnabled: { value: 1.0 },
                vignetteDarkness: { value: 0.3 },
                vignetteOffset: { value: 1.0 }
            },
            vertexShader: `
                varying vec2 vUv;
                void main() {
                    vUv = uv;
                    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
                }
            `,
            fragmentShader: `
                uniform sampler2D tDiffuse;
                uniform float contrast;
                uniform float saturation;
                uniform float brightness;
                uniform float vignetteEnabled;
                uniform float vignetteDarkness;
                uniform float vignetteOffset;
                
                varying vec2 vUv;
                
                vec3 adjustContrast(vec3 color, float contrast) {
                    return (color - 0.5) * contrast + 0.5;
                }
                
                vec3 adjustSaturation(vec3 color, float saturation) {
                    float gray = dot(color, vec3(0.299, 0.587, 0.114));
                    return mix(vec3(gray), color, saturation);
                }
                
                float vignette(vec2 uv, float darkness, float offset) {
                    vec2 center = vec2(0.5);
                    float dist = distance(uv, center);
                    return 1.0 - darkness * smoothstep(offset - 0.5, offset, dist);
                }
                
                void main() {
                    vec4 texel = texture2D(tDiffuse, vUv);
                    vec3 color = texel.rgb;
                    
                    // Apply color grading
                    color = adjustContrast(color, contrast);
                    color = adjustSaturation(color, saturation);
                    color *= brightness;
                    
                    // Apply vignette
                    if (vignetteEnabled > 0.5) {
                        float vig = vignette(vUv, vignetteDarkness, vignetteOffset);
                        color *= vig;
                    }
                    
                    gl_FragColor = vec4(color, texel.a);
                }
            `
        };
    }
    
    updateParams() {
        if (!this.composer) return;
        
        // Update bloom
        this.bloomPass.enabled = this.params.bloom.enabled;
        this.bloomPass.strength = this.params.bloom.strength;
        this.bloomPass.radius = this.params.bloom.radius;
        this.bloomPass.threshold = this.params.bloom.threshold;
        
        // Update color grading
        const uniforms = this.colorGradingPass.uniforms;
        uniforms.contrast.value = this.params.colorGrading.contrast;
        uniforms.saturation.value = this.params.colorGrading.saturation;
        uniforms.brightness.value = this.params.colorGrading.brightness;
        uniforms.vignetteEnabled.value = this.params.vignette.enabled ? 1.0 : 0.0;
        uniforms.vignetteDarkness.value = this.params.vignette.darkness;
        uniforms.vignetteOffset.value = this.params.vignette.offset;
    }
    
    setTimeOfDay(timeOfDay) {
        // Adjust post-processing based on time of day
        switch(timeOfDay) {
            case 'day':
                this.params.bloom.strength = 0.2;
                this.params.colorGrading.contrast = 1.1;
                this.params.colorGrading.saturation = 1.1;
                this.params.vignette.darkness = 0.2;
                break;
            case 'evening':
                this.params.bloom.strength = 0.4;
                this.params.colorGrading.contrast = 1.2;
                this.params.colorGrading.saturation = 1.3;
                this.params.vignette.darkness = 0.4;
                break;
            case 'night':
                this.params.bloom.strength = 0.5;
                this.params.colorGrading.contrast = 1.3;
                this.params.colorGrading.saturation = 0.9;
                this.params.vignette.darkness = 0.5;
                break;
        }
        this.updateParams();
    }
    
    setStoryMood(mood) {
        // Adjust effects based on story mood
        switch(mood) {
            case 'adventure':
                this.params.colorGrading.saturation = 1.4;
                this.params.bloom.strength = 0.35;
                break;
            case 'magical':
                this.params.bloom.strength = 0.6;
                this.params.bloom.radius = 0.5;
                break;
            case 'calm':
                this.params.colorGrading.saturation = 0.9;
                this.params.bloom.strength = 0.2;
                break;
            case 'mysterious':
                this.params.colorGrading.contrast = 1.4;
                this.params.vignette.darkness = 0.6;
                break;
        }
        this.updateParams();
    }
    
    resize() {
        if (!this.composer) return;
        
        const { width, height, pixelRatio } = this.sizes;

        this.composer.setSize(width, height);
        this.composer.setPixelRatio(pixelRatio);

        // Update FXAA resolution
        const fxaaPass = this.composer.passes.find(pass => 
            pass.uniforms && pass.uniforms.resolution
        );
        if (fxaaPass) {
            fxaaPass.uniforms.resolution.value.set(
                1 / (width * pixelRatio),
                1 / (height * pixelRatio)
            );
        }
    }
    
    render() {
        if (this.enabled && this.composer) {
            this.composer.render();
        } else {
            this.renderer.render(this.scene, this.camera);
        }
    }
    
    toggle() {
        this.enabled = !this.enabled;
        return this.enabled;
    }
    
    setBloomStrength(value) {
        this.params.bloom.strength = value;
        this.updateParams();
    }
    
    setContrast(value) {
        this.params.colorGrading.contrast = value;
        this.updateParams();
    }
    
    setSaturation(value) {
        this.params.colorGrading.saturation = value;
        this.updateParams();
    }
    
    setBrightness(value) {
        this.params.colorGrading.brightness = value;
        this.updateParams();
    }
    
    setVignetteDarkness(value) {
        this.params.vignette.darkness = value;
        this.updateParams();
    }
    
    dispose() {
        if (this.composer) {
            this.composer.dispose();
            this.composer = null;
        }
    }
}
