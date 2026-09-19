import { createClient } from '@connectrpc/connect';
import {
  VisionNavigationService,
  type ListNavigatorsResponse,
  type StartNavigationResponse,
  type GetNavigationStatusResponse,
  type AbortNavigationResponse,
  type ResumeNavigationResponse,
  type NavigatorInfo,
} from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';

import { transport } from './client';

// Connect-Web client for the BAS VisionNavigationService.
//
// The legacy REST surface (/api/v1/ai-navigate/*) was removed in the
// proto+Connect-RPC migration. The corresponding playwright-driver callback
// (POST /api/v1/internal/ai-navigate/callback) remains REST as an
// intentional webhook receiver; see docs/internal/REST_EXCEPTIONS.md.
export const visionNavigationClient = createClient(VisionNavigationService, transport);

export type {
  ListNavigatorsResponse,
  StartNavigationResponse,
  GetNavigationStatusResponse,
  AbortNavigationResponse,
  ResumeNavigationResponse,
  NavigatorInfo,
};
