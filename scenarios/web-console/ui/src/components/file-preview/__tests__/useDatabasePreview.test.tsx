import { act, renderHook } from "@testing-library/react";
import { afterEach,beforeEach,describe,expect,it,vi } from "vitest";
import { useDatabasePreview } from "../useDatabasePreview";
import type { DatabaseQuery } from "../databaseData";
class PreviewWorker {
 static instances:PreviewWorker[]=[];
 onmessage:((event:MessageEvent)=>void)|null=null;onerror:(()=>void)|null=null;
 terminate=vi.fn();postMessage=vi.fn();constructor(){PreviewWorker.instances.push(this);}
}
const query:DatabaseQuery={table:'tasks',page:0,filter:'',filterColumn:'',sortColumn:'',descending:false};
function message(data:object,index=0){act(()=>{PreviewWorker.instances[index]?.onmessage?.(new MessageEvent('message',{data}));});}
beforeEach(()=>{vi.useFakeTimers();PreviewWorker.instances=[];vi.stubGlobal('Worker',PreviewWorker);});
afterEach(()=>{vi.useRealTimers();vi.unstubAllGlobals();});
describe('database preview lifecycle',()=>{
 it('rejects oversized input and unavailable engines before starting a worker',()=>{const {result,rerender}=renderHook(({size})=>useDatabasePreview('/blob',size,0),{initialProps:{size:40000000}});expect(result.current.error).toBe('limit');expect(PreviewWorker.instances).toHaveLength(0);vi.stubGlobal('Worker',undefined);rerender({size:100});expect(result.current.error).toBe('unavailable');});
 it('requests the snapshot representation and terminates its worker on replacement and close',()=>{const {rerender,unmount}=renderHook(({href})=>useDatabasePreview(href,100,0),{initialProps:{href:'/blob?existing=yes'}});expect(JSON.stringify(PreviewWorker.instances[0]?.postMessage.mock.calls)).toContain('representation=sqlite-snapshot');rerender({href:'/new'});expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalledOnce();unmount();expect(PreviewWorker.instances[1]?.terminate).toHaveBeenCalledOnce();});
 it('ignores out-of-order page replies and preserves the last schema while new data loads',()=>{const {result}=renderHook(()=>useDatabasePreview('/blob',100,0));message({type:'ready',info:{tables:['tasks'],snapshotTime:'2026-10-18T12:00:00Z',bytes:100}});act(()=>{result.current.browse(query);result.current.browse({...query,page:1});});message({type:'page',id:1,page:{query,rows:[['old']]}});expect(result.current.page).toBeNull();message({type:'page',id:2,page:{query:{...query,page:1},rows:[],schema:{columns:[]},hasMore:false,page:1}});expect(result.current.page?.page).toBe(1);act(()=>{result.current.browse({...query,page:2});});expect(result.current.page?.page).toBe(1);message({type:'error',id:3,error:'timeout'});expect(result.current.queryError).toBe('timeout');expect(result.current.error).toBeNull();});
 it('bounds download, initialization and page operations',()=>{const {result,rerender}=renderHook(({attempt})=>useDatabasePreview('/blob',100,attempt),{initialProps:{attempt:0}});act(()=>{vi.advanceTimersByTime(45000);});expect(result.current.error).toBe('fetch');rerender({attempt:1});message({type:'loading'},1);act(()=>{vi.advanceTimersByTime(15000);});expect(result.current.error).toBe('timeout');rerender({attempt:2});message({type:'ready',info:{tables:['tasks']}},2);act(()=>{result.current.browse(query);vi.advanceTimersByTime(5000);});expect(result.current.error).toBe('timeout');expect(PreviewWorker.instances[2]?.terminate).toHaveBeenCalled();});
 it('ignores late events after teardown and supports a fresh retry',()=>{const {result,rerender}=renderHook(({attempt})=>useDatabasePreview('/blob',100,attempt),{initialProps:{attempt:0}});const late=PreviewWorker.instances[0]?.onmessage;message({type:'error',error:'stale'});expect(result.current.error).toBe('stale');act(()=>{late?.(new MessageEvent('message',{data:{type:'ready',info:{tables:['late']}}}));});expect(result.current.info).toBeNull();rerender({attempt:1});expect(result.current.error).toBeNull();expect(PreviewWorker.instances).toHaveLength(2);});
});
