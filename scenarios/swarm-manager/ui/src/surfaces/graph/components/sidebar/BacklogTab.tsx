/** Backlog collection surface; BacklogCard remains scenario-owned content. */
import { Profiler, memo, useCallback, useEffect, useMemo, useState, type KeyboardEvent, type MouseEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { onProfilerRender } from "../../../../lib/profiler";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useBacklogStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import { itemActionsFromNextAction } from "../../../../lib";
import { buildBacklogCompareFn, sortBacklogItems } from "../../../../lib/backlog-sort";
import { computeUnblockingMap } from "../../../../lib/dependency-sort";
import { filterSnoozed, snoozeKeyForBacklog } from "../../../../lib/snooze-utils";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import { BacklogCard } from "../../../../components/backlog/backlog-card";
import { RunSheet, type RunSheetTarget } from "../../../../components/backlog/run-sheet";
import { Button } from "../../../../components/ui/button";
import { CollectionList as BaseCollectionList, type CollectionListProps } from "@vrooli/react-component-library/CollectionList/1.0.0";
import { backlogService, autoFilerService } from "../../../../services";
import { backlogDetailPath } from "../../../../app/routes/route-paths";
import { nextActionDetailTab } from "../../../../lib/backlog-next-action";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { shouldOpenBacklogRow } from "./backlog-row-interaction";
import { useCommandPostItemActions, type StableItemCallbacks } from "../../../../hooks/useCommandPostItemActions";
import type { BacklogItem, PendingQuestion } from "../../../../types";
import type { BacklogNextAction } from "../../../../services/backlog";
import type { BacklogFilters, SortConfig } from "./types";
import type { AttentionReason } from "../../../../lib/attention";
import type { StepperCompletionResult } from "../../../../components/backlog/inline-question-stepper";

function CollectionList<T>(props: CollectionListProps<T>) { return <BaseCollectionList {...props} virtualize />; }

interface BacklogTabProps { searchQuery:string; filters:BacklogFilters; sort:SortConfig; onItemClick:(nodeId:string)=>void; onClearSearch?:()=>void; selectionMode?:boolean; selectedIds?:Set<string>; onToggleSelection?:(id:string)=>void; onVisibleIdsChange?:(ids:string[])=>void; onCreateBacklog?:()=>void; onCreateFromPlan?:()=>void }
const EMPTY_REASONS:AttentionReason[]=[];
const backlogSelectionId=(item:BacklogItem)=>`backlog:${item.kind}/${item.name}`;
const applyFilters=(items:BacklogItem[],filters:BacklogFilters)=>items.filter(item=>(filters.showArchived||item.archivedAt==null)&&(filters.statuses.length===0||filters.statuses.includes(item.status))&&(filters.kinds.length===0||filters.kinds.includes(item.kind))&&(filters.priorityMin===null||item.priority>=filters.priorityMin)&&(filters.priorityMax===null||item.priority<=filters.priorityMax));
function BacklogTabImpl({searchQuery,filters,sort,onItemClick,onClearSearch,selectionMode=false,selectedIds=new Set<string>(),onToggleSelection,onVisibleIdsChange,onCreateBacklog,onCreateFromPlan}:BacklogTabProps){
 const navigate=useNavigate();const items=useBacklogStore(s=>s.items);const fetchBacklog=useBacklogStore(s=>s.fetchBacklog);const snoozedKeys=useSnoozedKeys();const[runModalTarget,setRunModalTarget]=useState<RunSheetTarget>();const[pendingDismissKey,setPendingDismissKey]=useState<string|null>(null);
 const unblockingMap=useMemo(()=>computeUnblockingMap(items),[items]);const sorted=useMemo(()=>sortBacklogItems(filterSnoozed(applyFilters(items,filters).filter(item=>!searchQuery||matchesSearch(searchQuery,item.title,item.name,item.description,...(item.tags??[]))),item=>snoozeKeyForBacklog(item.kind,item.name),snoozedKeys),buildBacklogCompareFn(sort,unblockingMap),items),[filters,items,searchQuery,snoozedKeys,sort,unblockingMap]);
 useEffect(()=>{onVisibleIdsChange?.(sorted.map(backlogSelectionId))},[onVisibleIdsChange,sorted]);
 const handleSelect=useCallback((kind:string,name:string)=>navigate(backlogDetailPath(kind as BacklogItem["kind"],name)),[navigate]);const handleRun=useCallback((kind:BacklogItem["kind"],name:string,title?:string)=>setRunModalTarget({kind,name,title:title??""}),[]);
 const callbacks=useCommandPostItemActions({onSelectBacklog:handleSelect,onRunItem:handleRun});const{data:nextActions={}}=useQuery({queryKey:["backlog","next-actions",sorted.map(item=>`${item.kind}/${item.name}`)],queryFn:()=>backlogService.getNextActions(sorted.map(({kind,name})=>({kind,name}))),enabled:sorted.length>0});
 const dismiss=useCallback(async(item:BacklogItem)=>{const key=`${item.kind}/${item.name}`;setPendingDismissKey(key);try{await autoFilerService.dismissSuggestion(item.kind,item.name);await fetchBacklog({force:true})}finally{setPendingDismissKey(null)}},[fetchBacklog]);
 if(sorted.length===0)return <SidebarEmptyState icon={SIDEBAR_TAB_ICONS.backlog} title={searchQuery?"No backlog items match your filters.":"No backlog items yet."} hint="Capture an idea or chore to get started." query={searchQuery} onClearSearch={onClearSearch} action={(onCreateBacklog||onCreateFromPlan)?<div className="mt-1 flex gap-2"><>{onCreateBacklog?<Button size="sm" data-testid="backlog-tab-create-item" onClick={onCreateBacklog}><Plus className="mr-1.5 h-3.5 w-3.5"/>Create item</Button>:null}{onCreateFromPlan?<Button size="sm" variant="outline" onClick={onCreateFromPlan}>Create from plan</Button>:null}</></div>:undefined}/>;
 const sync=onToggleSelection?(keys:string[])=>{const next=new Set(keys);selectedIds.forEach(id=>{if(!next.has(id))onToggleSelection(id)});keys.forEach(id=>{if(!selectedIds.has(id))onToggleSelection(id)})}:undefined;
 return <><CollectionList items={sorted} getKey={backlogSelectionId} label="Backlog" virtualize selection={{mode:selectionMode?"multi":"none",enterOn:["shortcut"],selected:[...selectedIds],onChange:sync}} actions={[{id:"open",label:"Open",bulk:true,onSelect:([item])=>{if(item)onItemClick(`${item.kind}/${item.name}`)}},{id:"archive",label:"Archive",tone:"destructive",bulk:true,disabled:item=>item.archivedAt==null?false:"Already archived",onSelect:async(rows)=>{for(const item of rows)await backlogService.archiveItem(item.kind,item.name)}}]} renderItem={(item)=><BacklogRow item={item} nextAction={nextActions[`${item.kind}/${item.name}`]} callbacks={callbacks.getItemCallbacks(item)} attentionReasons={callbacks.attentionReasonsMap.get(`${item.kind}/${item.name}`)??EMPTY_REASONS} pendingQuestions={callbacks.pendingQuestionsMap.get(`${item.kind}/${item.name}`)} agentRunning={callbacks.activeRunKeys.has(`${item.kind}/${item.name}`)} isStepperCompleted={callbacks.completedSteppers.has(`${item.kind}/${item.name}`)} archivePending={callbacks.pendingArchiveKey===`${item.kind}/${item.name}`} dismissPending={pendingDismissKey===`${item.kind}/${item.name}`} statusChangePending={callbacks.pendingStatusKey===`${item.kind}/${item.name}`} runningLabel={callbacks.activeRunLabels.get(`${item.kind}/${item.name}`)} handleStepperCompleted={callbacks.handleStepperCompleted} onDismissSuggestion={()=>void dismiss(item)} onItemClick={onItemClick}/>} className="space-y-2"/><RunSheet isOpen={!!runModalTarget} onClose={()=>setRunModalTarget(undefined)} target={runModalTarget} onSuccess={()=>{setRunModalTarget(undefined);void fetchBacklog({force:true})}}/></>;
}
function BacklogRow({item,nextAction,attentionReasons,pendingQuestions,agentRunning,isStepperCompleted,callbacks,archivePending,dismissPending,statusChangePending,runningLabel,handleStepperCompleted,onDismissSuggestion,onItemClick}:{item:BacklogItem;nextAction?:BacklogNextAction;attentionReasons:AttentionReason[];pendingQuestions?:PendingQuestion[];agentRunning:boolean;isStepperCompleted:boolean;callbacks:StableItemCallbacks;archivePending:boolean;dismissPending:boolean;statusChangePending:boolean;runningLabel?:string;handleStepperCompleted:(key:string,item:BacklogItem,result:StepperCompletionResult)=>void;onDismissSuggestion:()=>void;onItemClick:(id:string)=>void}){
 const navigate=useNavigate();const key=`${item.kind}/${item.name}`;const nodeId=buildBacklogNodeId(item.kind,item.name);const itemActions=useMemo(()=>itemActionsFromNextAction(item,nextAction,{agentRunning}),[agentRunning,item,nextAction]);const handleNext=useCallback(()=>{if(nextAction?.id==="run")return callbacks.onRun();if(nextAction?.id==="archive")return callbacks.onArchive();if(nextAction?.id==="accept_suggestion")return callbacks.onStatusChange("backlog");if(nextAction?.id==="retry")return void backlogService.retry(item.kind,item.name).then(()=>fetchBacklogForRow());const tab=nextAction&&nextActionDetailTab(nextAction);navigate(backlogDetailPath(item.kind,item.name,tab?{tab}:undefined))},[callbacks,item.kind,item.name,navigate,nextAction]);const fetchBacklogForRow=()=>useBacklogStore.getState().fetchBacklog({force:true});
 const isInteractiveTarget = (target: EventTarget | null): boolean => {
   return !shouldOpenBacklogRow(target);
 };
 const handleClick = (event: MouseEvent<HTMLDivElement>) => {
   // BacklogCard contains its own action controls. The previous outer
   // <button> made the DOM invalid and caused browsers to re-parent those
   // controls, which swallowed clicks on the card itself. Keep the row a
   // non-button container and only open it when the click targets the row
   // content rather than one of its controls.
   if (shouldOpenBacklogRow(event.target)) onItemClick(nodeId);
 };
 const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
   if ((event.key === "Enter" || event.key === " ") && !isInteractiveTarget(event.target)) {
     event.preventDefault();
     onItemClick(nodeId);
   }
 };
 return <div role="button" tabIndex={0} onClick={handleClick} onKeyDown={handleKeyDown} className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 focus:outline-none focus:ring-2 focus:ring-cyan-400/50 hover:bg-slate-800/60" data-testid="sidebar-backlog-item" aria-label={`Open ${item.title}`}><BacklogCard item={item} nextAction={nextAction} itemActions={itemActions} attentionReasons={attentionReasons} pendingQuestions={pendingQuestions} isStepperCompleted={isStepperCompleted} onStepperCompleted={result=>handleStepperCompleted(key,item,result)} onRun={callbacks.onRun} onNextAction={handleNext} onArchive={callbacks.onArchive} onFollowUp={callbacks.onFollowUp} onAcceptSuggestion={()=>callbacks.onStatusChange("backlog")} onDismissSuggestion={onDismissSuggestion} onStatusChange={callbacks.onStatusChange} archivePending={archivePending} dismissPending={dismissPending} statusChangePending={statusChangePending} runningLabel={runningLabel}/></div>;
}
export const BacklogTab=memo(function BacklogTab(props:BacklogTabProps){return <Profiler id="BacklogTab" onRender={onProfilerRender}><BacklogTabImpl {...props}/></Profiler>});
