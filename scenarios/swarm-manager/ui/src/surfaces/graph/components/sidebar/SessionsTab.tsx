/** Agent-session collection surface. Domain content remains in SessionSummaryCard. */
import { memo, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { isActiveAgentSession, useAgentSessionStore } from "../../../../stores";
import { SessionSummaryCard } from "../../../../components/session/session-summary-card";
import { sessionDetailPath } from "../../../../app/routes/route-paths";
import { matchesSearch } from "./useSidebarSearch";
import { applySessionFilters, applySessionSort } from "./session-list-utils";
import type { AgentSession } from "../../../../types";
import type { SessionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { CollectionList as BaseCollectionList, type CollectionListProps } from "@vrooli/react-component-library/CollectionList/1.0.0";

function CollectionList<T>(props: CollectionListProps<T>) { return <BaseCollectionList {...props} virtualize />; }

interface SessionsTabProps { searchQuery:string; filters:SessionFilters; sort:SortConfig; onOpenSession?:(sessionId:string)=>void; onClearSearch?:()=>void; selectionMode?:boolean; selectedIds?:Set<string>; onToggleSelection?:(id:string)=>void; onVisibleIdsChange?:(ids:string[])=>void }
const sessionSelectionId=(session:AgentSession)=>`session:${session.id}`;
function SessionsTabImpl({searchQuery,filters,sort,onOpenSession,onClearSearch,selectionMode=false,selectedIds=new Set<string>(),onToggleSelection,onVisibleIdsChange}:SessionsTabProps){
 const sessions=useAgentSessionStore(s=>s.sessions);const sorted=useMemo(()=>applySessionSort(applySessionFilters(sessions,filters).filter(session=>!searchQuery||matchesSearch(searchQuery,session.title,session.kind,session.status,session.skillId,session.runId)),sort),[filters,searchQuery,sessions,sort]);const navigate=useNavigate();
 useEffect(()=>{onVisibleIdsChange?.(sorted.map(sessionSelectionId))},[onVisibleIdsChange,sorted]);
 const handleOpen=(id:string)=>{navigate(sessionDetailPath(id));onOpenSession?.(id)};
 if(sorted.length===0)return <SidebarEmptyState icon={SIDEBAR_TAB_ICONS.sessions} title={searchQuery||filters.statuses.length||filters.kinds.length?"No sessions match your filters.":"No agent sessions yet."} hint="Plan-work, operations, and authoring conversations show up here once started." query={searchQuery} onClearSearch={onClearSearch}/>;
 const sync=onToggleSelection?(keys:string[])=>{const next=new Set(keys);selectedIds.forEach(id=>{if(!next.has(id))onToggleSelection(id)});keys.forEach(id=>{if(!selectedIds.has(id))onToggleSelection(id)})}:undefined;
 return <CollectionList items={sorted} getKey={sessionSelectionId} label="Sessions" selection={{mode:selectionMode?"multi":"none",enterOn:["shortcut"],selected:[...selectedIds],onChange:sync}} actions={[{id:"open",label:"Open",onSelect:([session])=>{if(session)handleOpen(session.id)}},{id:"cancel",label:"Cancel",bulk:true,disabled:session=>isActiveAgentSession(session)?false:"Only active sessions can be canceled",onSelect:async(rows)=>{for(const session of rows)await useAgentSessionStore.getState().cancelSession(session.id)}},{id:"delete",label:"Delete",tone:"destructive",bulk:true,disabled:session=>isActiveAgentSession(session)?"Active sessions cannot be deleted":false,onSelect:async(rows)=>{for(const session of rows)await useAgentSessionStore.getState().deleteSession(session.id)}}]} renderItem={session=><SessionSummaryCard session={session} onOpen={handleOpen}/>} className="space-y-1.5"/>;
}
export const SessionsTab=memo(SessionsTabImpl);
