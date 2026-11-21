import type { MessageInitShape } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import {
  DownloadService,
  DownloadAppSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/download_pb';
import type {
  DownloadApp,
  DownloadAsset,
  DownloadStorefront,
} from '@vrooli/proto-types/landing-page-react-vite/v1/download_pb';

import { transport } from './client';

const downloadClient = createClient(DownloadService, transport);

export type DownloadAppInput = MessageInitShape<typeof DownloadAppSchema>;

/** Authorizes a download for an app/platform and returns the release asset. */
export async function requestDownload(
  appKey: string,
  platform: string,
): Promise<DownloadAsset | undefined> {
  const resp = await downloadClient.authorizeDownload({ app: appKey, platform });
  return resp.asset;
}

/** Lists all configured download apps (admin). */
export async function listDownloadAppsAdmin(): Promise<DownloadApp[]> {
  const resp = await downloadClient.listDownloadApps({});
  return resp.apps;
}

/** Creates a new download app (admin). */
export async function createDownloadAppAdmin(app: DownloadAppInput): Promise<DownloadApp | undefined> {
  const resp = await downloadClient.createDownloadApp({ app });
  return resp.app;
}

/** Creates or replaces a download app by key (admin). */
export async function saveDownloadAppAdmin(
  appKey: string,
  app: DownloadAppInput,
): Promise<DownloadApp | undefined> {
  const resp = await downloadClient.saveDownloadApp({ appKey, app });
  return resp.app;
}

export type { DownloadApp, DownloadAsset, DownloadStorefront };
