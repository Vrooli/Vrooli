// AUTO-GENERATED — do not edit by hand.
//
// Source : config/profile-catalog.json
// Codegen: GOWORK=off go run ./cmd/gen-gateway-catalog (from api/)
// Verify : GOWORK=off go run ./cmd/gen-gateway-catalog -check (from api/)
import { Profile } from "@vrooli/proto-types/ai-gateway/v1/shared/gateway_pb";
import { strings } from "./strings";

export const PROFILE_CATALOG = [
  { value: Profile.LOCAL_ONLY, key: "local-only", labelKey: strings.profiles.localOnly, descriptionKey: strings.profiles.localOnlyDesc },
  { value: Profile.LOCAL_FIRST, key: "local-first", labelKey: strings.profiles.localFirst, descriptionKey: strings.profiles.localFirstDesc },
  { value: Profile.REMOTE_ONLY, key: "remote-only", labelKey: strings.profiles.remoteOnly, descriptionKey: strings.profiles.remoteOnlyDesc },
  { value: Profile.QUALITY_FIRST, key: "quality-first", labelKey: strings.profiles.qualityFirst, descriptionKey: strings.profiles.qualityFirstDesc },
  { value: Profile.CHEAP_FIRST, key: "cheap-first", labelKey: strings.profiles.cheapFirst, descriptionKey: strings.profiles.cheapFirstDesc },
  { value: Profile.PRIVACY_SENSITIVE, key: "privacy-sensitive", labelKey: strings.profiles.privacySensitive, descriptionKey: strings.profiles.privacySensitiveDesc },
] as const;
