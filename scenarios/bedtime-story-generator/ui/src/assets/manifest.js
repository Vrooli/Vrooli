const resolveAsset = (path) => {
  const trimmed = path.startsWith("/") ? path.slice(1) : path;

  if (typeof document !== "undefined" && document.baseURI) {
    return new URL(trimmed, document.baseURI).pathname;
  }

  const base = import.meta.env?.BASE_URL ?? "/";
  const normalized = base.endsWith("/") ? base : `${base}/`;
  return `${normalized}${trimmed}`;
};

export const ROOM_ASSET_MANIFEST = {
  studio: {
    baked: {
      day: resolveAsset("experience/bakedDay.jpg"),
      evening: resolveAsset("experience/bakedNeutral.jpg"),
      night: resolveAsset("experience/bakedNight.jpg"),
      neutral: resolveAsset("experience/bakedNeutral.jpg"),
      lightMap: resolveAsset("experience/lightMap.jpg"),
    },
    models: {
      room: resolveAsset("experience/roomModel.glb"),
      pcScreen: resolveAsset("experience/pcScreenModel.glb"),
      macScreen: resolveAsset("experience/macScreenModel.glb"),
      coffeeSteam: resolveAsset("experience/coffeeSteamModel.glb"),
      loupedeck: resolveAsset("experience/loupedeckButtonsModel.glb"),
      googleLeds: resolveAsset("experience/googleHomeLedsModel.glb"),
      topChair: resolveAsset("experience/topChairModel.glb"),
      elgato: resolveAsset("experience/elgatoLightModel.glb"),
    },
    textures: {
      googleLedMask: resolveAsset("experience/googleHomeLedMask.png"),
      threeLogo: resolveAsset("experience/threejsJourneyLogo.png"),
    },
    screens: {
      pc: resolveAsset("experience/videoPortfolio.mp4"),
      mac: resolveAsset("experience/videoStream.mp4"),
    },
  },
  bedroom: {
    models: {
      environment: resolveAsset("experience/bedroom/bedroomScene.glb"),
    },
  },
};

export const SHARED_ASSETS = {
  dracoDecoder: `${resolveAsset("experience/draco/")}`,
};

export const LEGACY_ENVIRONMENT_MANIFEST = {
  ...ROOM_ASSET_MANIFEST.studio,
  dracoDecoder: SHARED_ASSETS.dracoDecoder,
};

export const SCREEN_SOURCES = {
  ...ROOM_ASSET_MANIFEST.studio.screens,
};

const assetManifest = {
  environment: LEGACY_ENVIRONMENT_MANIFEST,
  rooms: ROOM_ASSET_MANIFEST,
  shared: SHARED_ASSETS,
  screens: SCREEN_SOURCES,
};

export default assetManifest;
