var N=r=>{throw TypeError(r)};var E=(r,t,e)=>t.has(r)||N("Cannot "+e);var o=(r,t,e)=>(E(r,t,"read from private field"),e?e.call(r):t.get(r)),x=(r,t,e)=>t.has(r)?N("Cannot add the same private member more than once"):t instanceof WeakSet?t.add(r):t.set(r,e),O=(r,t,e,s)=>(E(r,t,"write to private field"),s?s.call(r,e):t.set(r,e),e),S=(r,t,e)=>(E(r,t,"access private method"),e);import{a9 as k,aa as A,ab as I,ac as K,ad as R,d as j,r as v,ae as U,af as $,a3 as q,j as L,ag as P}from"./main-DiOknWoR.js";var b,m,a,l,d,w,M,C,V=(C=class extends k{constructor(t,e){super();x(this,d);x(this,b);x(this,m);x(this,a);x(this,l);O(this,b,t),this.setOptions(e),this.bindMethods(),S(this,d,w).call(this)}bindMethods(){this.mutate=this.mutate.bind(this),this.reset=this.reset.bind(this)}setOptions(t){var s;const e=this.options;this.options=o(this,b).defaultMutationOptions(t),A(this.options,e)||o(this,b).getMutationCache().notify({type:"observerOptionsUpdated",mutation:o(this,a),observer:this}),e!=null&&e.mutationKey&&this.options.mutationKey&&I(e.mutationKey)!==I(this.options.mutationKey)?this.reset():((s=o(this,a))==null?void 0:s.state.status)==="pending"&&o(this,a).setOptions(this.options)}onUnsubscribe(){var t;this.hasListeners()||(t=o(this,a))==null||t.removeObserver(this)}onMutationUpdate(t){S(this,d,w).call(this),S(this,d,M).call(this,t)}getCurrentResult(){return o(this,m)}reset(){var t;(t=o(this,a))==null||t.removeObserver(this),O(this,a,void 0),S(this,d,w).call(this),S(this,d,M).call(this)}mutate(t,e){var s;return O(this,l,e),(s=o(this,a))==null||s.removeObserver(this),O(this,a,o(this,b).getMutationCache().build(o(this,b),this.options)),o(this,a).addObserver(this),o(this,a).execute(t)}},b=new WeakMap,m=new WeakMap,a=new WeakMap,l=new WeakMap,d=new WeakSet,w=function(){var e;const t=((e=o(this,a))==null?void 0:e.state)??K();O(this,m,{...t,isPending:t.status==="pending",isSuccess:t.status==="success",isError:t.status==="error",isIdle:t.status==="idle",mutate:this.mutate,reset:this.reset})},M=function(t){R.batch(()=>{var e,s,i,n,u,c,h,g;if(o(this,l)&&this.hasListeners()){const p=o(this,m).variables,y=o(this,m).context;(t==null?void 0:t.type)==="success"?((s=(e=o(this,l)).onSuccess)==null||s.call(e,t.data,p,y),(n=(i=o(this,l)).onSettled)==null||n.call(i,t.data,null,p,y)):(t==null?void 0:t.type)==="error"&&((c=(u=o(this,l)).onError)==null||c.call(u,t.error,p,y),(g=(h=o(this,l)).onSettled)==null||g.call(h,void 0,t.error,p,y))}this.listeners.forEach(p=>{p(o(this,m))})})},C);function Y(r,t){const e=j(),[s]=v.useState(()=>new V(e,r));v.useEffect(()=>{s.setOptions(r)},[s,r]);const i=v.useSyncExternalStore(v.useCallback(u=>s.subscribe(R.batchCalls(u)),[s]),()=>s.getCurrentResult(),()=>s.getCurrentResult()),n=v.useCallback((u,c)=>{s.mutate(u,c).catch(U)},[s]);if(i.error&&$(s.options.throwOnError,[i.error]))throw i.error;return{...i,mutate:n,mutateAsync:i.mutate}}const z={foreground:"var(--color-foreground)"},_={controlPadding:"var(--control-padding)"},f=(...r)=>r.filter(Boolean).join(" "),D={interaction:"var(--dur-quick) var(--ease-standard)",enter:"var(--dur-moderate) var(--ease-enter)",exit:"var(--dur-quick) var(--ease-exit)",spring:"var(--dur-moderate) var(--spring-subtle)"},B=(r,t="interaction")=>(Array.isArray(r)?r:[r]).map(s=>`${s} ${D[t]}`).join(", ");f("bg-[var(--color-primary)]","text-[var(--color-primary-foreground)]","border-transparent"),f("bg-[var(--color-surface)]","text-[var(--color-foreground)]","border-[var(--color-border)]"),f("bg-transparent","text-[var(--color-foreground)]","border-transparent"),f("bg-[var(--color-danger)]","text-[var(--color-primary-foreground)]","border-transparent");f("min-h-[var(--control-size-sm)]","px-[var(--space-sm)]","text-[var(--text-label)]"),f("min-h-[var(--control-size-md)]",`px-[${_.controlPadding}]`,`text-[${z.foreground}]`),f("min-h-[var(--control-size-lg)]","px-[var(--space-md)]","text-[var(--text-label)]");const F={raised:"shadow-[var(--elev-raised)]"};function Q(r){return typeof r=="function"}function G({value:r,defaultValue:t,onChange:e}){const[s,i]=v.useState(t),n=r!==void 0,u=n?r:s,c=v.useCallback(h=>{const g=Q(h)?h(u):h;n||i(g),e==null||e(g)},[n,u,e]);return[u,c]}const H=`
[data-rcl-input] {
  box-sizing: border-box;
  width: 100%;
  min-height: var(--tap-target-min);
  border: var(--border-hairline) solid var(--color-border);
  border-radius: var(--radius-control);
  background: var(--color-field);
  box-shadow: var(--elev-raised);
  color: ${z.foreground};
  padding-inline: var(--space-sm);
  font: inherit;
  transition: ${B(["border-color","box-shadow","background"])};
}
[data-rcl-input]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted); }
[data-rcl-input]:hover:not(:disabled) { border-color: var(--color-primary); }
[data-rcl-input]:focus-visible { border-color: var(--color-focus); outline: var(--border-strong) solid color-mix(in srgb, var(--color-focus) 30%, transparent); outline-offset: var(--space-3xs); }
[data-rcl-input]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }
@media (prefers-reduced-motion: reduce) { [data-rcl-input] { transition: none; } }

/* The "mobile-safe font sizing" this component has always advertised, finally
   implemented. iOS zooms the whole viewport when a text control smaller than
   16px receives focus, which on a fixed-position layout is a one-way trip.
   Deliberately at the component's own weight, so a consumer that states a size
   still wins: the promise is that the *default* is safe, not that the library
   overrules a product decision. '1em' resolves against the parent, so this is
   a floor and never an enlargement. */
@media (pointer: coarse) {
  [data-rcl-input] { font-size: max(16px, 1em); }
}
`,J=f(F.raised),Z=v.forwardRef(function({className:t,type:e,value:s,defaultValue:i,onChange:n,"data-testid":u,...c},h){q("react-component-library:Input","1.3.6",H);const[g,p]=G({value:s,defaultValue:i??""}),y=T=>{p(T.currentTarget.value),n==null||n(T)};return L.jsx("input",{ref:h,type:e,value:g,"data-testid":u??"forms.input","data-rcl-input":"true",className:P(J,t),onChange:y,...c})});export{Z as I,Y as u};
