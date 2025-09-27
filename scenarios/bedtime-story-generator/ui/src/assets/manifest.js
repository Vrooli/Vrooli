const base = "/experience";

export const ROOM_ASSET_MANIFEST = {
  studio: {
    baked: {
      day: `${base}/bakedDay.jpg`,
      evening: `${base}/bakedNeutral.jpg`,
      night: `${base}/bakedNight.jpg`,
      neutral: `${base}/bakedNeutral.jpg`,
      lightMap: `${base}/lightMap.jpg`,
    },
    models: {
      room: `${base}/roomModel.glb`,
      pcScreen: `${base}/pcScreenModel.glb`,
      macScreen: `${base}/macScreenModel.glb`,
      coffeeSteam: `${base}/coffeeSteamModel.glb`,
      loupedeck: `${base}/loupedeckButtonsModel.glb`,
      googleLeds: `${base}/googleHomeLedsModel.glb`,
      topChair: `${base}/topChairModel.glb`,
      elgato: `${base}/elgatoLightModel.glb`,
    },
    textures: {
      googleLedMask: `${base}/googleHomeLedMask.png`,
      threeLogo: `${base}/threejsJourneyLogo.png`,
    },
    screens: {
      pc: `${base}/videoPortfolio.mp4`,
      mac: `${base}/videoStream.mp4`,
    },
  },
  bedroom: {
    models: {
      environment: `${base}/bedroom/bedroomScene.glb`,
    },
  },
};

export const SHARED_ASSETS = {
  dracoDecoder: `${base}/draco/`,
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
