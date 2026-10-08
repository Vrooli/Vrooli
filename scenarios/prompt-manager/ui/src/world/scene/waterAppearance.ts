import { Color, ShaderMaterial } from 'three'
import type { WaterVisualTuning } from '../config'

export function createWaterMaterial(settings: WaterVisualTuning, wobble: boolean): ShaderMaterial {
  return new ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: {
      uTime: { value: 0 }, uWobble: { value: wobble ? 1 : 0 }, uColor: { value: new Color(settings.color) },
      uWaveFrequencyX: { value: settings.waveFrequencyX }, uWaveFrequencyZ: { value: settings.waveFrequencyZ },
      uWaveSpeed: { value: settings.waveSpeed }, uCrossWaveSpeed: { value: settings.crossWaveSpeed },
      uWaveAmplitude: { value: settings.waveAmplitude }, uShoreFadeWidth: { value: settings.shoreFadeWidth },
      uShoreBrightness: { value: settings.shoreBrightness }, uShoreOpacity: { value: settings.shoreOpacity },
      uDeepOpacity: { value: settings.deepOpacity },
    },
    vertexShader: `
      attribute float shore;
      uniform float uTime;
      uniform float uWobble;
      uniform float uWaveFrequencyX;
      uniform float uWaveFrequencyZ;
      uniform float uWaveSpeed;
      uniform float uCrossWaveSpeed;
      uniform float uWaveAmplitude;
      varying float vShore;
      void main() {
        vec3 p = position;
        p.y += sin(p.x * uWaveFrequencyX + uTime * uWaveSpeed) * cos(p.z * uWaveFrequencyZ + uTime * uCrossWaveSpeed) * uWaveAmplitude * uWobble;
        vShore = shore;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
      }
    `,
    fragmentShader: `
      uniform vec3 uColor;
      uniform float uShoreFadeWidth;
      uniform float uShoreBrightness;
      uniform float uShoreOpacity;
      uniform float uDeepOpacity;
      varying float vShore;
      void main() {
        float edge = smoothstep(0.0, uShoreFadeWidth, vShore);
        gl_FragColor = vec4(uColor * mix(uShoreBrightness, 1.0, edge), mix(uShoreOpacity, uDeepOpacity, edge));
      }
    `,
  })
}
