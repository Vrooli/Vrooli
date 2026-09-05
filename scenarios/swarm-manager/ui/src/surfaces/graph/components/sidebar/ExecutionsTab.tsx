/** Execution collection surface. Domain content remains in ExecutionSummaryCard. */
import { memo, useEffect, useMemo } from "react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useExecutionStore } from "../../../../stores";
import { executionService } from "../../../../services";
import { buildExecutionNodeId } from "../../lib/node-id-parser";
import { ExecutionSummaryCard } from "../../../../components/execution/execution-summary-card";
import { matchesSearch } from "./useSidebarSearch";
import type { ExecutionRecord } from "../../../../types";
import type { ExecutionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { CollectionList as BaseCollectionList, type CollectionListProps } from "@vrooli/react-component-library/CollectionList/1.0.0";

function CollectionList<T>(props: CollectionListProps<T>) { return <BaseCollectionList {...props} virtualize />; }

interface ExecutionsTabProps { searchQuery:string; filters:ExecutionFilters; sort:SortConfig; onItemClick:(nodeId:string)=>void; onClearSearch?:()=>void; selectionMode?:boolean; selectedIds?:Set<string>; onToggleSelection?:(id:string)=>void; onVisibleIdsChange?:(ids:string[])=>void }
const executionSelectionId=(item:ExecutionRecord)=>`execution:${item.executionId}`;
function ExecutionsTabImpl({searchQuery,filters,sort,onItemClick,onClearSearch,selectionMode=false,selectedIds=new Set<string>(),onToggleSelection,onVisibleIdsChange}:ExecutionsTabProps){
 const items=useExecutionStore(s=>s.items);const sorted=useMemo(()=>[...items].filter(item=>(filters.statuses.length===0||filters.statuses.includes(item.status))&&(filters.modes.length===0||filters.modes.includes(item.mode))).filter(item=>!searchQuery||matchesSearch(searchQuery,item.backlogName,item.status,item.mode)).sort((a,b)=>{const dir=sort.direction==="asc"?1:-1;return sort.field==="alphabetical"?a.backlogName.localeCompare(b.backlogName)*dir:sort.field==="status"?a.status.localeCompare(b.status)*dir:(new Date(b.createdAt).getTime()-new Date(a.createdAt).getTime())*dir}),[items,filters.modes,filters.statuses,searchQuery,sort.direction,sort.field]);
 useEffect(()=>{onVisibleIdsChange?.(sorted.map(executionSelectionId))},[onVisibleIdsChange,sorted]);
 if(sorted.length===0)return <SidebarEmptyState icon={SIDEBAR_TAB_ICONS.executions} title={searchQuery||filters.statuses.length||filters.modes.length?"No executions match your filters.":"No executions yet."} hint="Runs and reviews appear here as agents start work." query={searchQuery} onClearSearch={onClearSearch}/>;
 const sync=onToggleSelection?(keys:string[])=>{const next=new Set(keys);selectedIds.forEach(id=>{if(!next.has(id))onToggleSelection(id)});keys.forEach(id=>{if(!selectedIds.has(id))onToggleSelection(id)})}:undefined;
 return <CollectionList items={sorted} getKey={executionSelectionId} label="Executions" selection={{mode:selectionMode?"multi":"none",enterOn:["shortcut"],selected:[...selectedIds],onChange:sync}} actions={[{id:"open",label:"Open",onSelect:([item])=>{if(item)onItemClick(buildExecutionNodeId(item.executionId))}},{id:"cancel",label:"Cancel",bulk:true,hidden:item=>!(["pending","running","in_progress"].includes(item.status)),onSelect:async(rows)=>{for(const item of rows)await executionService.cancel(item.executionId)}},{id:"retry",label:"Retry",bulk:true,onSelect:async(rows)=>{for(const item of rows)await executionService.retry(item.executionId)}}]} renderItem={item=><ExecutionSummaryCard item={item} onOpen={()=>onItemClick(buildExecutionNodeId(item.executionId))}/>} className="space-y-1.5"/>;
}
export const ExecutionsTab=memo(ExecutionsTabImpl);
