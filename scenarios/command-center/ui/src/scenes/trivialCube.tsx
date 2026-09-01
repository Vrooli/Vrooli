import { useEffect, useRef, useState } from "react";
import { useFrame } from "@react-three/fiber";
import type { Mesh } from "three";

const FALLBACK_COLOR = "#00d4ff";

/**
 * Lazy-loaded default-export scene used by the placeholder dashboards.
 * Renders a slowly rotating cube coloured with the active theme's accent.
 */
function TrivialCube() {
  const meshRef = useRef<Mesh>(null);
  const [color, setColor] = useState<string>(FALLBACK_COLOR);

  useEffect(() => {
    // CSS variables are on the <html> element via <ThemeProvider>.
    const accent = getComputedStyle(document.documentElement)
      .getPropertyValue("--color-primary")
      .trim();
    if (accent.length > 0) {
      setColor(accent);
    }
  }, []);

  useFrame((_state, delta) => {
    const mesh = meshRef.current;
    if (!mesh) return;
    mesh.rotation.x += delta * 0.4;
    mesh.rotation.y += delta * 0.6;
  });

  return (
    <>
      <ambientLight intensity={0.35} />
      <directionalLight position={[5, 5, 5]} intensity={0.9} />
      <mesh ref={meshRef}>
        <boxGeometry args={[1.4, 1.4, 1.4]} />
        <meshStandardMaterial color={color} />
      </mesh>
    </>
  );
}

export default TrivialCube;
