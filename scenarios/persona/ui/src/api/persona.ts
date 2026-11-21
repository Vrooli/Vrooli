import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import {
  CreatePersonaRequestSchema,
  IdentifierSchema,
  LegalBasisSchema,
  PersonaKind,
  PersonasService,
} from "@vrooli/proto-types/persona/v1/personas/personas_pb";
import { HandoffsService } from "@vrooli/proto-types/persona/v1/handoffs/handoffs_pb";
import { JournalService } from "@vrooli/proto-types/persona/v1/journal/journal_pb";
import { ChannelsService } from "@vrooli/proto-types/persona/v1/channels/channels_pb";

import { transport } from "./client";

const personas = createClient(PersonasService, transport);
const handoffs = createClient(HandoffsService, transport);
const journal = createClient(JournalService, transport);
const channels = createClient(ChannelsService, transport);

export type PersonaCreateInput = {
  kind: PersonaKind;
  subjectId: string;
  subjectName: string;
  basisType: string;
  displayName: string;
  identifierType: string;
  identifierValue: string;
};

export async function listPersonas(includeArchived = false) {
  const response = await personas.listPersonas({ limit: 100, includeArchived });
  return response.personas;
}

export async function getPersona(personaId: string) {
  const response = await personas.getPersona({ personaId });
  return response.persona;
}

export async function createPersona(input: PersonaCreateInput) {
  const response = await personas.createPersona(
    create(CreatePersonaRequestSchema, {
      kind: input.kind,
      legalBasis: create(LegalBasisSchema, {
        subjectId: input.subjectId,
        subjectName: input.subjectName,
        basisType: input.basisType,
      }),
      displayName: input.displayName,
      identifiers: [create(IdentifierSchema, { type: input.identifierType, value: input.identifierValue })],
    }),
  );
  return response.persona;
}

export async function checkPersonaHealth(personaId: string) {
  const response = await personas.checkHealth({ personaId });
  return response.findings;
}

export async function listHandoffs(personaId: string) {
  const response = await handoffs.listHandoffs({ personaId, limit: 100 });
  return response.handoffs;
}

export async function listChannels(personaId: string) {
  const response = await channels.listChannels({ personaId });
  return response.channels;
}

export async function retrieveCode(personaId: string, channelId: string, purpose: string) {
  const response = await channels.retrieveCode({ personaId, channelId, purpose });
  return response;
}

export async function getHandoff(handoffId: string) {
  const response = await handoffs.getHandoff({ handoffId });
  return response.handoff;
}

export async function completeHandoff(handoffId: string) {
  const response = await handoffs.completeHandoff({ handoffId, completedBy: "operator" });
  return response.handoff;
}

export async function listJournal(personaId: string) {
  const response = await journal.list({ personaId, limit: 200 });
  return response.entries;
}

export async function listAllHandoffs(personaRecords: readonly { id: string }[]) {
  const batches = await Promise.all(personaRecords.map((persona) => listHandoffs(persona.id)));
  return batches.flat();
}

export async function listAllJournal(personaRecords: readonly { id: string }[]) {
  const batches = await Promise.all(personaRecords.map((persona) => listJournal(persona.id)));
  return batches.flat();
}
