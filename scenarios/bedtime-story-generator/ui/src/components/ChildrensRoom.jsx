import React, { useMemo, useRef, useState, useEffect } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import {
  Float,
  OrbitControls,
  Sparkles,
  ContactShadows,
  RoundedBox,
  Sky,
  useCursor
} from '@react-three/drei';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';
import { Vector3, CanvasTexture, RepeatWrapping, SRGBColorSpace } from 'three';

const SCENE_PRESETS = {
  day: {
    background: '#c6e6ff',
    ambient: 0.9,
    key: 0.95,
    fill: 0.35,
    rim: 0.25,
    palette: {
      floor: '#fff6f0',
      wallFront: '#f0f6ff',
      wallSide: '#f3e7ff',
      trim: '#ffd6a5',
      bedFrame: '#f6a74b',
      duvet: '#ffeef6',
      pillow: '#ffffff',
      rug: '#a8d8ff',
      bookshelf: '#ffe2c6',
      books: ['#ff9aa2', '#8ec5ff', '#ffe066', '#bdb2ff'],
      toy: '#ffc75f',
      canopy: '#ffcd94',
      exterior: '#89cff0'
    },
    sparkles: { count: 40, speed: 0.4, opacity: 0.25 }
  },
  evening: {
    background: '#ffbb9a',
    ambient: 0.7,
    key: 0.8,
    fill: 0.4,
    rim: 0.3,
    palette: {
      floor: '#fbe9dd',
      wallFront: '#ffd9c8',
      wallSide: '#f9c5c2',
      trim: '#f7a072',
      bedFrame: '#e97a62',
      duvet: '#ffd6f2',
      pillow: '#fff2f7',
      rug: '#f6c1ff',
      bookshelf: '#ffb490',
      books: ['#ffef9a', '#9ad6ff', '#ff9aa2', '#d0b4ff'],
      toy: '#ffc15d',
      canopy: '#ffb47d',
      exterior: '#ffad8c'
    },
    sparkles: { count: 70, speed: 0.6, opacity: 0.35 }
  },
  night: {
    background: '#1a1d3f',
    ambient: 0.4,
    key: 0.55,
    fill: 0.25,
    rim: 0.45,
    palette: {
      floor: '#1f2145',
      wallFront: '#212654',
      wallSide: '#1b203f',
      trim: '#4f54a6',
      bedFrame: '#384c92',
      duvet: '#2d3a78',
      pillow: '#4f5fcc',
      rug: '#28346a',
      bookshelf: '#2e366d',
      books: ['#8894ff', '#ffafcc', '#ffe066', '#a0c4ff'],
      toy: '#ffd166',
      canopy: '#7d87ff',
      exterior: '#1f2d65'
    },
    sparkles: { count: 120, speed: 0.9, opacity: 0.5 }
  }
};

const DEBUG_FOCUS = {
  bed: {
    position: new Vector3(-0.4, 2.6, 4.4),
    target: new Vector3(-1.4, -0.5, -1.2)
  },
  bookshelf: {
    position: new Vector3(3.2, 2.8, 2.4),
    target: new Vector3(3, -0.4, -1.7)
  },
  nightstand: {
    position: new Vector3(1.6, 2.3, 3.1),
    target: new Vector3(1.4, -0.4, -0.6)
  },
  window: {
    position: new Vector3(0.3, 2.6, 1.1),
    target: new Vector3(0, 1.4, -3.5)
  },
  mobile: {
    position: new Vector3(-3.4, 2.8, 3.2),
    target: new Vector3(-2.6, 2.2, 1.2)
  },
  toys: {
    position: new Vector3(2.3, 1.8, 4.4),
    target: new Vector3(2, -0.8, 1.6)
  },
  arch: {
    position: new Vector3(0.4, 2.1, 5.2),
    target: new Vector3(0.2, -0.9, 2.6)
  }
};

const LightRig = ({ palette, config }) => (
  <>
    <ambientLight intensity={config.ambient} color={palette.trim} />
    <directionalLight
      castShadow
      position={[5.2, 6, 5]}
      intensity={config.key}
      color={palette.trim}
      shadow-mapSize-width={1024}
      shadow-mapSize-height={1024}
      shadow-camera-left={-8}
      shadow-camera-right={8}
      shadow-camera-top={8}
      shadow-camera-bottom={-8}
      shadow-camera-near={1}
      shadow-camera-far={20}
    />
    <directionalLight
      position={[-4, 4, -3]}
      intensity={config.fill}
      color={palette.exterior}
    />
    <pointLight
      position={[1.2, 2.5, 0.8]}
      intensity={1.4}
      distance={5.4}
      decay={2}
      color={palette.canopy}
    />
    <pointLight
      position={[-2.4, 3, -2.2]}
      intensity={config.rim}
      distance={7}
      color={palette.exterior}
    />
  </>
);

const Wallpaper = ({ palette, timeOfDay, wallTexture, textureKey }) => (
  <group position={[0, 1.9, -4.2]}>
    <RoundedBox args={[12, 6, 0.4]} radius={0.25} smoothness={2}>
      <meshStandardMaterial
        key={`wall-front-${textureKey}`}
        color={palette.wallFront}
        roughness={0.85}
        map={wallTexture || null}
      />
    </RoundedBox>
    <group position={[0, 0, 0.25]}>
      <Float rotationIntensity={0.1} floatIntensity={0.2} speed={0.6}>
        <mesh position={[0, 1.2, 0]}>
          <torusGeometry args={[1.2, 0.15, 24, 48]} />
          <meshStandardMaterial color={palette.trim} roughness={0.5} />
        </mesh>
      </Float>
      <mesh position={[0, -0.4, 0]}>
        <planeGeometry args={[8.5, 3.5]} />
        <meshStandardMaterial color={timeOfDay === 'night' ? '#1f2752' : '#ffffff'} opacity={0.18} transparent />
      </mesh>
      <group position={[0, -1.2, 0]}>
        {[-2.5, 0, 2.5].map((x) => (
          <mesh key={x} position={[x, 0, 0]}>
            <sphereGeometry args={[0.6, 32, 32]} />
            <meshStandardMaterial color={palette.exterior} opacity={0.25} transparent />
          </mesh>
        ))}
      </group>
    </group>
  </group>
);

const SideWall = ({ palette, wallTexture, textureKey }) => (
  <RoundedBox args={[12, 6, 0.4]} radius={0.25} smoothness={2} position={[-5.8, 1.9, 0]} rotation={[0, Math.PI / 2, 0]}>
    <meshStandardMaterial
      key={`wall-side-${textureKey}`}
      color={palette.wallSide}
      roughness={0.9}
      map={wallTexture || null}
      opacity={wallTexture ? 0.95 : 1}
      transparent={!!wallTexture}
    />
  </RoundedBox>
);

const Floor = ({ palette, floorTexture, textureKey }) => (
  <group>
    <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -1.6, 0]} receiveShadow>
      <planeGeometry args={[20, 20]} />
      <meshStandardMaterial
        key={`floor-${textureKey}`}
        color={palette.floor}
        roughness={0.4}
        map={floorTexture || null}
      />
    </mesh>
    <RoundedBox args={[12.5, 0.2, 12.5]} radius={0.3} smoothness={2} position={[0, -1.49, 0]}>
      <meshStandardMaterial color={palette.floor} roughness={0.6} />
    </RoundedBox>
  </group>
);

const Bed = ({ palette, highlighted }) => (
  <group position={[-1.8, -1, -1.2]}>
    <RoundedBox args={[4.4, 0.4, 2.4]} radius={0.3} smoothness={3} castShadow receiveShadow>
      <meshStandardMaterial
        color={palette.bedFrame}
        roughness={0.35}
        emissive={highlighted ? palette.canopy : '#000000'}
        emissiveIntensity={highlighted ? 0.5 : 0}
      />
    </RoundedBox>
    <RoundedBox args={[4, 0.6, 2]} radius={0.25} smoothness={3} position={[0, 0.45, 0]} castShadow receiveShadow>
      <meshStandardMaterial
        color={palette.duvet}
        roughness={0.4}
        emissive={highlighted ? palette.duvet : '#000000'}
        emissiveIntensity={highlighted ? 0.4 : 0}
      />
    </RoundedBox>
    <RoundedBox args={[0.35, 1.8, 2.2]} radius={0.2} smoothness={3} position={[-2, 1, 0]} castShadow>
      <meshStandardMaterial color={palette.trim} roughness={0.3} />
    </RoundedBox>
    <RoundedBox args={[1.1, 0.35, 0.7]} radius={0.18} smoothness={3} position={[1.2, 0.8, -0.6]} castShadow>
      <meshStandardMaterial
        color={palette.pillow}
        roughness={0.35}
        emissive={highlighted ? '#ffffff' : '#000000'}
        emissiveIntensity={highlighted ? 0.4 : 0}
      />
    </RoundedBox>
    <RoundedBox args={[1.1, 0.35, 0.7]} radius={0.18} smoothness={3} position={[1.2, 0.8, 0.6]} castShadow>
      <meshStandardMaterial
        color={palette.pillow}
        roughness={0.35}
        emissive={highlighted ? '#ffffff' : '#000000'}
        emissiveIntensity={highlighted ? 0.4 : 0}
      />
    </RoundedBox>
  </group>
);

const Rug = ({ palette }) => (
  <Float floatIntensity={0.15} rotationIntensity={0.05} speed={0.6}>
    <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0.8, -1.45, 1.8]} receiveShadow>
      <circleGeometry args={[2, 48]} />
      <meshStandardMaterial color={palette.rug} roughness={0.6} />
    </mesh>
  </Float>
);

const useGeneratedTexture = (type, palette, builder) => {
  const texture = useMemo(() => {
    if (typeof document === 'undefined' || !type || type === 'smooth' || type === 'solid') {
      return null;
    }
    const built = builder(type, palette);
    if (built) {
      built.colorSpace = SRGBColorSpace;
      built.needsUpdate = true;
    }
    return built;
  }, [type, palette, builder]);

  useEffect(() => () => texture?.dispose(), [texture]);

  return texture;
};

const buildFloorTexture = (type, palette) => {
  const canvas = document.createElement('canvas');
  const size = 512;
  canvas.width = canvas.height = size;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = palette.floor;
  ctx.fillRect(0, 0, size, size);

  if (type === 'soft-stripes') {
    ctx.globalAlpha = 0.45;
    const stripeHeight = size / 10;
    ctx.fillStyle = palette.trim;
    for (let y = 0; y < size; y += stripeHeight * 2) {
      ctx.fillRect(0, y, size, stripeHeight);
    }
  } else if (type === 'storybook-checker') {
    ctx.globalAlpha = 0.35;
    const cell = size / 8;
    const accent = palette.books[1] || palette.trim;
    for (let x = 0; x < size; x += cell) {
      for (let y = 0; y < size; y += cell) {
        if ((x / cell + y / cell) % 2 === 0) {
          ctx.fillStyle = accent;
          ctx.fillRect(x, y, cell, cell);
        }
      }
    }
  } else if (type === 'speckled-sparkle') {
    ctx.fillStyle = palette.canopy;
    ctx.globalAlpha = 0.25;
    for (let i = 0; i < 120; i += 1) {
      const radius = Math.random() * 6 + 2;
      ctx.beginPath();
      ctx.arc(Math.random() * size, Math.random() * size, radius, 0, Math.PI * 2);
      ctx.fill();
    }
  }

  const texture = new CanvasTexture(canvas);
  texture.wrapS = RepeatWrapping;
  texture.wrapT = RepeatWrapping;
  const repeat = type === 'speckled-sparkle' ? 2 : 3;
  texture.repeat.set(repeat, repeat);
  texture.anisotropy = 4;
  texture.needsUpdate = true;
  return texture;
};

const buildWallTexture = (type, palette) => {
  const canvas = document.createElement('canvas');
  const size = 512;
  canvas.width = canvas.height = size;
  const ctx = canvas.getContext('2d');

  if (type === 'storybook-clouds') {
    ctx.fillStyle = palette.wallFront;
    ctx.fillRect(0, 0, size, size);
    ctx.fillStyle = '#ffffff';
    ctx.globalAlpha = 0.35;
    for (let i = 0; i < 5; i += 1) {
      const x = (i + 0.5) * (size / 5);
      const y = size / 3 + (i % 2) * 30;
      ctx.beginPath();
      ctx.ellipse(x, y, 90, 60, 0, 0, Math.PI * 2);
      ctx.fill();
    }
  } else if (type === 'starlight') {
    ctx.fillStyle = palette.wallFront;
    ctx.fillRect(0, 0, size, size);
    ctx.fillStyle = '#fff8d6';
    ctx.globalAlpha = 0.5;
    for (let i = 0; i < 160; i += 1) {
      const radius = Math.random() * 2 + 1;
      ctx.beginPath();
      ctx.arc(Math.random() * size, Math.random() * size, radius, 0, Math.PI * 2);
      ctx.fill();
    }
  } else if (type === 'sunset-gradient') {
    const gradient = ctx.createLinearGradient(0, 0, 0, size);
    gradient.addColorStop(0, palette.wallFront);
    gradient.addColorStop(0.5, palette.exterior);
    gradient.addColorStop(1, palette.trim);
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, size, size);
  }

  const texture = new CanvasTexture(canvas);
  texture.wrapS = RepeatWrapping;
  texture.wrapT = RepeatWrapping;
  texture.repeat.set(1.5, 1.5);
  texture.anisotropy = 4;
  texture.needsUpdate = true;
  return texture;
};

const BookshelfUnit = ({ palette, libraryOpen, onFocus, highlighted }) => {
  const [hovered, setHovered] = useState(false);
  useCursor(hovered && !libraryOpen, 'pointer');

  const handleFocus = (event) => {
    event.stopPropagation();
    onFocus();
  };

  return (
    <group
      position={[3, -0.4, -1.9]}
      scale={libraryOpen ? 1 : hovered ? 1.05 : 1}
      onClick={handleFocus}
      onPointerOver={(event) => {
        event.stopPropagation();
        if (!libraryOpen) {
          setHovered(true);
        }
      }}
      onPointerOut={() => setHovered(false)}
    >
      <RoundedBox args={[0.7, 3, 1.6]} radius={0.25} smoothness={2} castShadow>
        <meshStandardMaterial
          color={palette.bookshelf}
          roughness={0.55}
          emissive={highlighted ? palette.canopy : '#000000'}
          emissiveIntensity={highlighted ? 0.6 : 0}
        />
      </RoundedBox>
      {[0, 1, 2, 3].map((index) => (
        <RoundedBox key={index} args={[0.75, 0.08, 1.5]} radius={0.1} position={[0, -1 + index * 0.85, 0]}>
          <meshStandardMaterial color={palette.trim} roughness={0.3} />
        </RoundedBox>
      ))}
      {palette.books.map((color, index) => (
        <RoundedBox
          key={color + index}
          args={[0.12, 0.6 + (index % 3) * 0.15, 0.4]}
          radius={0.05}
          position={[-0.2, 0.1 + (index % 4) * 0.4, -0.55 + (index % 3) * 0.45]}
          rotation={[0, 0, Math.PI * 0.03 * (index % 2 === 0 ? 1 : -1)]}
          castShadow
        >
          <meshStandardMaterial
            color={color}
            roughness={0.35}
            emissive={highlighted ? '#ffffff' : '#000000'}
            emissiveIntensity={highlighted ? 0.4 : 0}
          />
        </RoundedBox>
      ))}

      <Float visible={!libraryOpen} floatIntensity={0.6} speed={1} position={[0, 1.7, 0]}>
        <mesh onClick={handleFocus}>
          <ringGeometry args={[0.45, 0.6, 42]} />
          <meshStandardMaterial color={palette.canopy} emissive={palette.canopy} emissiveIntensity={0.4} transparent opacity={0.85} />
        </mesh>
        <mesh position={[0, 0, -0.02]} onClick={handleFocus}>
          <circleGeometry args={[0.4, 32]} />
          <meshStandardMaterial color={palette.trim} opacity={0.32} transparent />
        </mesh>
      </Float>
    </group>
  );
};

const Nightstand = ({ palette, highlighted }) => (
  <group position={[1.4, -0.9, -0.6]}>
    <RoundedBox args={[1, 1, 1]} radius={0.2} smoothness={2} castShadow>
      <meshStandardMaterial
        color={palette.trim}
        roughness={0.4}
        emissive={highlighted ? palette.canopy : '#000000'}
        emissiveIntensity={highlighted ? 0.5 : 0}
      />
    </RoundedBox>
    <RoundedBox args={[1.1, 0.12, 1.1]} radius={0.1} position={[0, 0.55, 0]} castShadow>
      <meshStandardMaterial color={palette.bookshelf} roughness={0.35} />
    </RoundedBox>
    <Float floatIntensity={0.4} rotationIntensity={0.3} speed={1.2}>
      <group position={[0, 1.1, 0]}>
        <mesh castShadow>
          <coneGeometry args={[0.5, 0.9, 24]} />
          <meshStandardMaterial emissive={palette.canopy} emissiveIntensity={0.6} color={palette.canopy} />
        </mesh>
        <mesh position={[0, -0.4, 0]} castShadow>
          <cylinderGeometry args={[0.18, 0.18, 0.4, 24]} />
          <meshStandardMaterial color={palette.trim} roughness={0.3} />
        </mesh>
      </group>
    </Float>
  </group>
);

const WindowView = ({ palette, timeOfDay, highlighted }) => (
  <group position={[0, 1.6, -4.15]}>
    <RoundedBox args={[5, 3, 0.4]} radius={0.2} smoothness={2}>
      <meshStandardMaterial
        color={palette.trim}
        emissive={highlighted ? palette.trim : '#000000'}
        emissiveIntensity={highlighted ? 0.4 : 0}
      />
    </RoundedBox>
    <mesh position={[0, 0, 0.26]}>
      <planeGeometry args={[4.6, 2.6]} />
      <meshStandardMaterial
        color={palette.exterior}
        opacity={0.85}
        transparent
        emissive={highlighted ? palette.exterior : '#000000'}
        emissiveIntensity={highlighted ? 0.3 : 0}
      />
    </mesh>
    <group position={[0, 0.05, 0.28]}>
      {timeOfDay === 'day' ? (
        <>
          <mesh position={[1.6, 1, -0.01]}>
            <sphereGeometry args={[0.35, 32, 32]} />
            <meshStandardMaterial color="#fff2ae" emissive="#ffe29a" emissiveIntensity={highlighted ? 0.8 : 0.3} />
          </mesh>
          <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.95, 0]}>
            <planeGeometry args={[4.4, 1.1]} />
            <meshStandardMaterial color="#8fdda7" opacity={0.7} transparent />
          </mesh>
        </>
      ) : (
        <>
          <mesh position={[1.4, 0.8, 0]}>
            <sphereGeometry args={[0.45, 32, 32]} />
            <meshStandardMaterial color="#fdf2ad" emissive="#ffe58d" emissiveIntensity={highlighted ? 1 : 0.5} />
          </mesh>
          <group>
            {[...Array(15)].map((_, i) => (
              <mesh key={i} position={[Math.sin(i) * 1.8, Math.cos(i) * 0.8, 0]}>
                <sphereGeometry args={[0.08, 16, 16]} />
                <meshStandardMaterial color="#fff4d6" emissive="#fff4d6" emissiveIntensity={0.45} />
              </mesh>
            ))}
          </group>
        </>
      )}
    </group>
    <mesh position={[0, -1.65, 0.28]}>
      <planeGeometry args={[4.6, 0.5]} />
      <meshStandardMaterial color={palette.trim} />
    </mesh>
  </group>
);

const HangingMobile = ({ palette, timeOfDay, highlighted }) => (
  <Float floatIntensity={0.6} rotationIntensity={0.4} speed={1} position={[-2.6, 2.6, 1.2]}>
    <group>
      {[...Array(5)].map((_, index) => (
        <mesh key={index} position={[Math.sin(index) * 0.7, -index * 0.4, Math.cos(index) * 0.4]}>
          <sphereGeometry args={[0.2 + index * 0.04, 16, 16]} />
          <meshStandardMaterial
            color={timeOfDay === 'night' ? '#ffe066' : palette.books[index % palette.books.length]}
            emissive={highlighted ? '#fff8d6' : timeOfDay === 'night' ? '#ffe066' : '#000000'}
            emissiveIntensity={highlighted ? 0.7 : timeOfDay === 'night' ? 0.45 : 0}
          />
        </mesh>
      ))}
    </group>
  </Float>
);

const SoftToys = ({ palette, highlighted }) => (
  <group position={[2, -1.2, 1.6]}>
    <Float speed={1.2} rotationIntensity={0.2} floatIntensity={0.15}>
      <group>
        <mesh castShadow position={[0, 0.25, 0]}>
          <sphereGeometry args={[0.35, 32, 32]} />
          <meshStandardMaterial
            color={palette.toy}
            roughness={0.6}
            emissive={highlighted ? palette.toy : '#000000'}
            emissiveIntensity={highlighted ? 0.5 : 0}
          />
        </mesh>
        <mesh castShadow position={[-0.2, 0.15, 0.3]}>
          <sphereGeometry args={[0.18, 32, 32]} />
          <meshStandardMaterial
            color={palette.toy}
            roughness={0.6}
            emissive={highlighted ? palette.toy : '#000000'}
            emissiveIntensity={highlighted ? 0.4 : 0}
          />
        </mesh>
        <mesh castShadow position={[0.2, 0.15, 0.3]}>
          <sphereGeometry args={[0.18, 32, 32]} />
          <meshStandardMaterial
            color={palette.toy}
            roughness={0.6}
            emissive={highlighted ? palette.toy : '#000000'}
            emissiveIntensity={highlighted ? 0.4 : 0}
          />
        </mesh>
      </group>
    </Float>
  </group>
);

const DecorativeArch = ({ palette, highlighted }) => (
  <Float rotationIntensity={0.05} floatIntensity={0.1} speed={0.8}>
    <mesh position={[0.2, -1.35, 2.6]} rotation={[0, Math.PI / 2.6, 0]} castShadow>
      <torusGeometry args={[1.3, 0.35, 32, 64, Math.PI]} />
      <meshStandardMaterial
        color={palette.rug}
        roughness={0.5}
        emissive={highlighted ? palette.rug : '#000000'}
        emissiveIntensity={highlighted ? 0.6 : 0}
      />
    </mesh>
  </Float>
);

const SceneComposition = ({
  timeOfDay,
  config,
  libraryOpen,
  onLibraryActivate,
  floorTexture,
  wallTexture,
  floorTextureKey,
  wallTextureKey,
  debugSelection
}) => {
  const { palette, sparkles } = config;

  return (
    <group>
      <LightRig palette={palette} config={config} />
      {timeOfDay === 'day' && (
        <Sky sunPosition={[5, 5, -10]} turbidity={4} rayleigh={2.4} mieCoefficient={0.005} mieDirectionalG={0.8} />
      )}
      <Wallpaper palette={palette} timeOfDay={timeOfDay} wallTexture={wallTexture} textureKey={wallTextureKey} />
      <SideWall palette={palette} wallTexture={wallTexture} textureKey={wallTextureKey} />
      <Floor palette={palette} floorTexture={floorTexture} textureKey={floorTextureKey} />
      <WindowView palette={palette} timeOfDay={timeOfDay} highlighted={debugSelection === 'window'} />
      <Bed palette={palette} highlighted={debugSelection === 'bed'} />
      <Rug palette={palette} />
      <BookshelfUnit
        palette={palette}
        libraryOpen={libraryOpen}
        onFocus={onLibraryActivate}
        highlighted={debugSelection === 'bookshelf'}
      />
      <Nightstand palette={palette} highlighted={debugSelection === 'nightstand'} />
      <HangingMobile palette={palette} timeOfDay={timeOfDay} highlighted={debugSelection === 'mobile'} />
      <SoftToys palette={palette} highlighted={debugSelection === 'toys'} />
      <DecorativeArch palette={palette} highlighted={debugSelection === 'arch'} />
      <Sparkles
        count={sparkles.count}
        scale={[10, 5, 4]}
        size={3}
        speed={sparkles.speed}
        opacity={sparkles.opacity}
        color={timeOfDay === 'night' ? '#ffffff' : '#ffd6a5'}
      />
      <ContactShadows position={[0, -1.6, 0]} opacity={0.35} blur={2.8} far={10} />
    </group>
  );
};

const CameraRig = ({ libraryOpen, debugSelection }) => {
  const controls = useRef();
  const { camera } = useThree();

  const defaultPosition = useMemo(() => new Vector3(5.6, 3.9, 6.2), []);
  const defaultTarget = useMemo(() => new Vector3(0, 0, 0), []);
  const libraryPosition = useMemo(() => new Vector3(3.2, 2.8, 2.4), []);
  const libraryTarget = useMemo(() => new Vector3(3, -0.4, -1.7), []);

  useFrame(() => {
    if (!controls.current) return;
    const focus = debugSelection ? DEBUG_FOCUS[debugSelection] : null;
    const desiredPosition = focus?.position || (libraryOpen ? libraryPosition : defaultPosition);
    const desiredTarget = focus?.target || (libraryOpen ? libraryTarget : defaultTarget);

    camera.position.lerp(desiredPosition, focus ? 0.08 : 0.06);
    controls.current.target.lerp(desiredTarget, focus ? 0.12 : 0.08);
    controls.current.update();
  });

  return (
    <OrbitControls
      ref={controls}
      enablePan={false}
      enableZoom={false}
      enableRotate={!libraryOpen && !debugSelection}
      minPolarAngle={Math.PI / 4}
      maxPolarAngle={Math.PI / 2.5}
      minAzimuthAngle={-0.6}
      maxAzimuthAngle={0.8}
    />
  );
};

const ChildrensRoom = ({
  timeOfDay,
  theme,
  children,
  libraryOpen,
  onLibraryActivate,
  floorTextureType,
  wallTextureType,
  debugSelection
}) => {
  const sceneConfig = useMemo(() => SCENE_PRESETS[timeOfDay] || SCENE_PRESETS.day, [timeOfDay]);
  const floorTexture = useGeneratedTexture(floorTextureType, sceneConfig.palette, buildFloorTexture);
  const wallTexture = useGeneratedTexture(wallTextureType, sceneConfig.palette, buildWallTexture);

  return (
    <div className={`childrens-room ${theme.class}`}>
      <Canvas className="room-canvas" shadows camera={{ position: [5.6, 3.9, 6.2], fov: 42 }} dpr={[1, 2]}>
        <color attach="background" args={[sceneConfig.background]} />
        <SceneComposition
          timeOfDay={timeOfDay}
          config={sceneConfig}
          libraryOpen={libraryOpen}
          onLibraryActivate={onLibraryActivate}
          floorTexture={floorTexture}
          wallTexture={wallTexture}
          floorTextureKey={floorTextureType}
          wallTextureKey={wallTextureType}
          debugSelection={debugSelection}
        />
        <EffectComposer>
          <Bloom intensity={0.7} luminanceThreshold={0.2} luminanceSmoothing={0.6} />
          <Vignette eskil={false} offset={0.18} darkness={0.7} />
        </EffectComposer>
        <CameraRig libraryOpen={libraryOpen} debugSelection={debugSelection} />
      </Canvas>

      <div className="room-content">
        {children}
      </div>
    </div>
  );
};

export default ChildrensRoom;
