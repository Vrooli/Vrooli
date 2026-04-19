import { Suspense, type ReactNode } from "react";
import { Canvas } from "@react-three/fiber";

interface SceneCanvasProps {
  scene: ReactNode;
  fallback?: ReactNode;
}

const defaultFallback = (
  <div className="cc-loading" data-testid="scene-fallback">
    Loading scene…
  </div>
);

/**
 * Small wrapper around React Three Fiber's <Canvas> that provides a shared
 * default camera, DPR clamp, and Suspense boundary for lazy scenes.
 */
export function SceneCanvas({ scene, fallback }: SceneCanvasProps) {
  const fallbackNode = fallback ?? defaultFallback;
  return (
    <div className="cc-canvas-wrap" data-testid="scene-canvas">
      <Suspense fallback={fallbackNode}>
        <Canvas dpr={[1, 2]} camera={{ position: [0, 0, 5], fov: 60 }}>
          {scene}
        </Canvas>
      </Suspense>
    </div>
  );
}
