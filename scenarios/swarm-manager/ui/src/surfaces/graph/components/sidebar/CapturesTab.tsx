/** Capture collection surface. Domain content remains in CaptureCard. */
import { memo, useEffect, useMemo } from "react";
import { Plus } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useCaptureStore } from "../../../../stores";
import { CaptureCard } from "../../../../components/capture/capture-card";
import { captureService } from "../../../../services/capture-service";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";
import { captureDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { Button } from "../../../../components/ui/button";
import { CollectionList as BaseCollectionList, type CollectionListProps } from "@vrooli/react-component-library/CollectionList/1.0.0";

function CollectionList<T>(props: CollectionListProps<T>) { return <BaseCollectionList {...props} virtualize />; }

interface CapturesTabProps { searchQuery:string; filters:CaptureFilters; sort:SortConfig; onItemClick:(nodeId:string)=>void; onClearSearch?:()=>void; selectionMode?:boolean; selectedIds?:Set<string>; onToggleSelection?:(id:string)=>void; onVisibleIdsChange?:(ids:string[])=>void; onCreateCapture?:()=>void }
const captureSelectionId=(capture:Capture)=>`capture:${capture.id}`;
function CapturesTabImpl({searchQuery,filters,sort,onClearSearch,selectionMode=false,selectedIds=new Set<string>(),onToggleSelection,onVisibleIdsChange,onCreateCapture}:CapturesTabProps){
 const navigate=useNavigate();const captures=useCaptureStore(s=>s.captures);
 const sorted=useMemo(()=>[...captures].filter(c=>filters.statuses.length===0||filters.statuses.includes(c.status)).filter(c=>!searchQuery||matchesSearch(searchQuery,c.text)).sort((a,b)=>{const dir=sort.direction==="asc"?1:-1;return sort.field==="alphabetical"?a.text.localeCompare(b.text)*dir:sort.field==="status"?a.status.localeCompare(b.status)*dir:(new Date(b.created).getTime()-new Date(a.created).getTime())*dir}),[captures,filters.statuses,searchQuery,sort.direction,sort.field]);
 useEffect(()=>{onVisibleIdsChange?.(sorted.map(captureSelectionId))},[onVisibleIdsChange,sorted]);
 if(sorted.length===0)return <SidebarEmptyState icon={SIDEBAR_TAB_ICONS.captures} title={searchQuery||filters.statuses.length?"No captures match your filters.":"No captures yet."} hint="Quick thoughts and observations land here before classification." query={searchQuery} onClearSearch={onClearSearch} action={onCreateCapture?<Button type="button" size="sm" data-testid="captures-tab-create-capture" onClick={onCreateCapture}><Plus className="mr-1.5 h-3.5 w-3.5"/>Quick capture</Button>:undefined}/>;
 const sync=onToggleSelection?(keys:string[])=>{const next=new Set(keys);selectedIds.forEach(id=>{if(!next.has(id))onToggleSelection(id)});keys.forEach(id=>{if(!selectedIds.has(id))onToggleSelection(id)})}:undefined;
 return <CollectionList items={sorted} getKey={captureSelectionId} label="Captures" selection={{mode:selectionMode?"multi":"none",enterOn:["shortcut"],selected:[...selectedIds],onChange:sync}} actions={[{id:"open",label:"Open",onSelect:([capture])=>{if(capture)navigate(captureDetailPath(capture.id))}},{id:"classify",label:"Classify",bulk:true,onSelect:async(rows)=>{for(const capture of rows)await captureService.classify(capture.id)}},{id:"delete",label:"Delete",tone:"destructive",bulk:true,onSelect:async(rows)=>{for(const capture of rows)await captureService.remove(capture.id)}}]} renderItem={capture=><CaptureCard capture={capture} onClick={()=>navigate(captureDetailPath(capture.id))} className="min-w-0 flex-1 border-0 bg-transparent p-0"/>} className="space-y-1.5"/>;
}
export const CapturesTab=memo(CapturesTabImpl);
