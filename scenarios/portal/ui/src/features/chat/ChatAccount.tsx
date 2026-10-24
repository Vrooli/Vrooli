/* eslint-disable react-refresh/only-export-components */
import {createContext,useContext,useMemo,useRef,type ReactNode} from "react";
import type {ChatAccess} from "../../api/chat";
const Context=createContext<ChatAccess|undefined>(undefined);
export const useChatAccess=()=>useContext(Context);
export function ChatAccountProvider({account,children}:{account?:{id:string;realm:string;token:string;expires:number};children:ReactNode}){
 const actor=account?JSON.stringify([account.realm,account.id]):"legacy";
 const controllerRef=useRef<{actor:string;controller:AbortController}>();
 if (!controllerRef.current || controllerRef.current.actor !== actor) {
  controllerRef.current?.controller.abort();
  controllerRef.current={actor,controller:new AbortController()};
 }
 const controller=controllerRef.current.controller;
 const access=useMemo<ChatAccess>(()=>({actor,signal:controller.signal,...(account?{account:{token:account.token,expires:account.expires}}:{})}),[actor,controller,account]);
 return <Context.Provider value={access}>{children}</Context.Provider>;
}
