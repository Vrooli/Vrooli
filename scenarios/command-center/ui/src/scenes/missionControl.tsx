import { useEffect, useMemo, useRef, useState } from "react";
import { useFrame } from "@react-three/fiber";
import { PointMaterial, Points } from "@react-three/drei";
import type { Mesh, Points as ThreePoints } from "three";

const FALLBACK_ACCENT = "#00d4ff";
const STARFIELD_COUNT = 3000;
const STARFIELD_RADIUS = 18;
const SATELLITE_COUNT = 4;

/**
 * Deterministic PRNG so the starfield doesn't flicker between re-mounts
 * (e.g. HMR / StrictMode). Seeded mulberry32.
 */
function createRng(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function generateStarfield(count: number, radius: number): Float32Array {
  const arr = new Float32Array(count * 3);
  const rng = createRng(0xc0ffee);
  for (let i = 0; i < count; i++) {
    // Distribute across a spherical shell with some radial jitter.
    const theta = rng() * Math.PI * 2;
    const phi = Math.acos(2 * rng() - 1);
    const r = radius * (0.85 + rng() * 0.3);
    const x = r * Math.sin(phi) * Math.cos(theta);
    const y = r * Math.sin(phi) * Math.sin(theta);
    const z = r * Math.cos(phi);
    arr[i * 3 + 0] = x;
    arr[i * 3 + 1] = y;
    arr[i * 3 + 2] = z;
  }
  return arr;
}

interface SatelliteSpec {
  radius: number;
  speed: number;
  tilt: number;
  size: number;
  phase: number;
}

function generateSatellites(count: number): SatelliteSpec[] {
  const rng = createRng(0xbadf00d);
  const specs: SatelliteSpec[] = [];
  for (let i = 0; i < count; i++) {
    specs.push({
      radius: 2.2 + rng() * 1.8,
      speed: 0.15 + rng() * 0.35,
      tilt: (rng() - 0.5) * 0.8,
      size: 0.12 + rng() * 0.09,
      phase: rng() * Math.PI * 2,
    });
  }
  return specs;
}

interface SatelliteProps {
  spec: SatelliteSpec;
  color: string;
}

function Satellite({ spec, color }: SatelliteProps) {
  const ref = useRef<Mesh>(null);
  useFrame(({ clock }) => {
    const mesh = ref.current;
    if (!mesh) return;
    const t = clock.getElapsedTime() * spec.speed + spec.phase;
    mesh.position.set(
      Math.cos(t) * spec.radius,
      Math.sin(t) * spec.radius * Math.sin(spec.tilt),
      Math.sin(t) * spec.radius,
    );
    mesh.rotation.y += 0.01;
  });
  return (
    <mesh ref={ref}>
      <sphereGeometry args={[spec.size, 16, 16]} />
      <meshStandardMaterial color={color} emissive={color} emissiveIntensity={0.45} />
    </mesh>
  );
}

function MissionControlScene() {
  const [accent, setAccent] = useState<string>(FALLBACK_ACCENT);
  const starsRef = useRef<ThreePoints>(null);

  useEffect(() => {
    const value = getComputedStyle(document.documentElement)
      .getPropertyValue("--cc-accent")
      .trim();
    if (value.length > 0) {
      setAccent(value);
    }
  }, []);

  const starPositions = useMemo(
    () => generateStarfield(STARFIELD_COUNT, STARFIELD_RADIUS),
    [],
  );
  const satelliteSpecs = useMemo(() => generateSatellites(SATELLITE_COUNT), []);

  useFrame((_, delta) => {
    const points = starsRef.current;
    if (!points) return;
    points.rotation.y += delta * 0.01;
  });

  return (
    <>
      <ambientLight intensity={0.25} />
      <directionalLight position={[4, 3, 5]} intensity={0.8} />
      <Points ref={starsRef} positions={starPositions} stride={3}>
        <PointMaterial
          transparent
          color={accent}
          size={0.04}
          sizeAttenuation
          depthWrite={false}
        />
      </Points>
      {satelliteSpecs.map((spec, i) => (
        <Satellite key={i} spec={spec} color={accent} />
      ))}
    </>
  );
}

export default MissionControlScene;
