import { randomBytes } from 'node:crypto'
import * as http from 'node:http'
import { StringDecoder } from 'node:string_decoder'
import { pipeline, Transform } from 'node:stream'
import { createBrotliDecompress, createUnzip } from 'node:zlib'
import type { Response } from 'express'

export const SANDBOX_POLICY = 'sandbox allow-scripts allow-forms allow-popups allow-downloads'

/** Inject a base URL and routing bootstrap without buffering the HTML body. */
export function createBaseTagStream(basePath: string, port: number, capability: string, nonce?: string): Transform {
  const decoder = new StringDecoder('utf8')
  let pending = ''
  let injected = false
  const inject = (html: string, flush = false): string => {
    if (injected) return html
    const match = /<head(?:\s[^>]*)?>/i.exec(html)
    if (match) {
      const at = match.index + match[0].length
      injected = true
      return `${html.slice(0, at)}<base href="${basePath}/">${createProxyBootstrap(basePath, port, capability, nonce)}${html.slice(at)}`
    }
    if (flush || pending.length > 64 * 1024) {
      injected = true
      const htmlTag = /<html(?:\s[^>]*)?>/i.exec(html)
      if (htmlTag) {
        const at = htmlTag.index + htmlTag[0].length
        return `${html.slice(0, at)}<base href="${basePath}/">${createProxyBootstrap(basePath, port, capability, nonce)}${html.slice(at)}`
      }
      const doctype = /<!doctype[^>]*>/i.exec(html)
      if (doctype) {
        const at = doctype.index + doctype[0].length
        return `${html.slice(0, at)}<base href="${basePath}/">${createProxyBootstrap(basePath, port, capability, nonce)}${html.slice(at)}`
      }
      return `<base href="${basePath}/">${createProxyBootstrap(basePath, port, capability, nonce)}${html}`
    }
    return ''
  }
  return new Transform({
    transform(chunk, _encoding, callback) {
      pending += decoder.write(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
      const output = inject(pending)
      if (output) {
        pending = ''
        this.push(output)
      }
      callback()
    },
    flush(callback) {
      pending += decoder.end()
      const output = inject(pending, true)
      if (output) this.push(output)
      callback()
    },
  })
}

function createProxyBootstrap(basePath: string, port: number, capability: string, nonce?: string): string {
  const config = JSON.stringify({ basePath, port, capability }).replace(/</g, '\\u003c')
  const code = [
    '(()=>{',
    `const c=${config};`,
    // Accessing Navigator.serviceWorker can throw for opaque-origin documents
    // (including this intentional sandbox). Leave normal origins untouched;
    // for the sandbox, expose the unavailable API as a rejected registration
    // so common feature-detection code does not abort the app at startup.
    'try{void navigator.serviceWorker}catch{try{const error=()=>Promise.reject(new DOMException("Service workers are unavailable in the isolated local preview","NotSupportedError"));Object.defineProperty(navigator,"serviceWorker",{configurable:true,value:{register:error,getRegistration:()=>Promise.resolve(undefined),getRegistrations:()=>Promise.resolve([]),addEventListener:()=>{},removeEventListener:()=>{},dispatchEvent:()=>false}})}catch{}}',
    'const parse=(v)=>{try{return new URL(String(v),document.baseURI)}catch{return null}};',
    'const loopback=(u)=>{if(!u)return false;const h=u.hostname.toLowerCase().replace(/^\\[|\\]$/g,"");const p=Number(u.port||(u.protocol==="https:"||u.protocol==="wss:"?443:80));return (["localhost","localhost.","127.0.0.1","0.0.0.0","::1"].includes(h)||/^127(?:\\.\\d{1,3}){3}$/.test(h))&&p===c.port};',
    'const mappedPath=(u)=>c.basePath+u.pathname+u.search+u.hash;',
    'const proxyRequest=(v)=>{const u=parse(v);return !!u&&u.pathname.startsWith(c.basePath+"/")};',
    'const httpUrl=(v)=>{const raw=String(v);const u=parse(raw);if(!u)return raw;if(loopback(u)&&u.protocol==="http:")return mappedPath(u);const page=new URL(document.baseURI);if(u.origin===page.origin&&!proxyRequest(u.href))return mappedPath(u);return raw};',
    // Some SPAs construct absolute same-origin resource URLs (origin + /path)
    // at runtime. They bypass the injected <base> tag and escape the proxy
    // route unless URL-bearing DOM setters receive the same mapping as fetch.
    'const originalSetAttribute=Element.prototype.setAttribute;',
    'Element.prototype.setAttribute=function(name,value){const key=String(name).toLowerCase();if(["src","href","action","formaction","poster","data","manifest"].includes(key))value=httpUrl(value);return originalSetAttribute.call(this,name,value)};',
    'const patchUrlProperty=(ctor,name)=>{try{const d=Object.getOwnPropertyDescriptor(ctor.prototype,name);if(!d||typeof d.set!=="function")return;Object.defineProperty(ctor.prototype,name,{...d,set(value){d.set.call(this,httpUrl(value))}})}catch{}};',
    'for(const [ctor,name] of [[window.HTMLImageElement,"src"],[window.HTMLScriptElement,"src"],[window.HTMLLinkElement,"href"],[window.HTMLAnchorElement,"href"],[window.HTMLFormElement,"action"],[window.HTMLIFrameElement,"src"],[window.HTMLSourceElement,"src"],[window.HTMLVideoElement,"poster"],[window.HTMLInputElement,"formAction"],[window.HTMLButtonElement,"formAction"],[window.HTMLObjectElement,"data"]])if(ctor)patchUrlProperty(ctor,name);',
    'const originalFetch=window.fetch.bind(window);',
    'window.fetch=(input,init)=>{const raw=input instanceof Request?input.url:input;const mapped=httpUrl(raw);if(!proxyRequest(mapped))return originalFetch(input,init);const options={...(init||{}),credentials:"include"};return input instanceof Request?originalFetch(new Request(mapped,input),options):originalFetch(mapped,options)};',
    'const xhrOpen=XMLHttpRequest.prototype.open;',
    'XMLHttpRequest.prototype.open=function(method,url,...args){const mapped=httpUrl(url);this.__vrootProxy=proxyRequest(mapped);return xhrOpen.call(this,method,mapped,...args)};',
    'const xhrSend=XMLHttpRequest.prototype.send;',
    'XMLHttpRequest.prototype.send=function(...args){if(this.__vrootProxy)this.withCredentials=true;return xhrSend.apply(this,args)};',
    'const NativeWebSocket=window.WebSocket;',
    'window.WebSocket=function(url,protocols){let raw=String(url);let u=parse(raw);const scheme=location.protocol==="https:"?"wss:":"ws:";if(raw.startsWith("/")){raw=scheme+"//"+location.host+c.basePath+raw}else if(u&&(u.protocol==="ws:"||u.protocol==="wss:")&&u.host===location.host&&!proxyRequest(u.href)){raw=scheme+"//"+location.host+mappedPath(u)}else if(loopback(u)&&(u.protocol==="ws:"||u.protocol==="http:")){raw=scheme+"//"+location.host+mappedPath(u)}return protocols===undefined?new NativeWebSocket(raw):new NativeWebSocket(raw,protocols)};',
    'window.WebSocket.prototype=NativeWebSocket.prototype;',
    'for(const k of ["CONNECTING","OPEN","CLOSING","CLOSED"])window.WebSocket[k]=NativeWebSocket[k];',
    '})();',
  ].join('\n')
  const nonceAttribute = nonce ? ` nonce="${nonce.replace(/["&<>]/g, '')}"` : ''
  return `<script${nonceAttribute}>${code}</script>`
}

export function createRootPathRewriteStream(basePath: string): Transform {
  const decoder = new StringDecoder('utf8')
  const tail = 2048
  let pending = ''
  return new Transform({
    transform(chunk, _encoding, callback) {
      pending += decoder.write(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
      if (pending.length > tail) {
        const boundary = pending.length - tail
        this.push(rootPathRewrite(pending.slice(0, boundary), basePath))
        pending = pending.slice(boundary)
      }
      callback()
    },
    flush(callback) {
      pending += decoder.end()
      if (pending) this.push(rootPathRewrite(pending, basePath))
      callback()
    },
  })
}

function rootPathRewrite(text: string, basePath: string): string {
  return text
    .replace(/(\b(?:src|href|action|poster|data|formaction|manifest)\s*=\s*["'])\/(?!\/)/gi, `$1${basePath}/`)
    .replace(/(url\(\s*["']?)\/(?!\/)/gi, `$1${basePath}/`)
    .replace(/(@import\s+["'])\/(?!\/)/gi, `$1${basePath}/`)
}

export function withBootstrapNonce(policy: string, nonce: string): string {
  const directives = policy.split(';').map((part) => part.trim()).filter(Boolean)
  const scriptDirective = directives.find((part) => /^script-src(?:\s|$)/i.test(part))
  const scriptElementDirective = directives.find((part) => /^script-src-elem(?:\s|$)/i.test(part))
  if (scriptElementDirective) {
    const updated = `${scriptElementDirective} 'nonce-${nonce}'`
    directives[directives.indexOf(scriptElementDirective)] = updated
  }
  if (scriptDirective) {
    const updated = `${scriptDirective} 'nonce-${nonce}'`
    directives[directives.indexOf(scriptDirective)] = updated
    return directives.join('; ')
  }
  if (scriptElementDirective) return directives.join('; ')
  const defaultDirective = directives.find((part) => /^default-src(?:\s|$)/i.test(part))
  if (!defaultDirective) return policy
  const scriptSources = defaultDirective.replace(/^default-src/i, 'script-src')
  directives.push(`${scriptSources} 'nonce-${nonce}'`)
  return directives.join('; ')
}

export function createBootstrapNonce(): string {
  return randomBytes(18).toString('base64')
}

export function streamHtml(res: Response, upstream: http.IncomingMessage, basePath: string, port: number, capability: string, nonce?: string): void {
  res.removeHeader('content-length')
  res.removeHeader('content-encoding')
  res.removeHeader('etag')
  const encoding = String(upstream.headers['content-encoding'] || '').trim().toLowerCase()
  const transforms: Transform[] = []
  if (encoding === 'gzip' || encoding === 'deflate') transforms.push(createUnzip())
  else if (encoding === 'br') transforms.push(createBrotliDecompress())
  transforms.push(createRootPathRewriteStream(basePath), createBaseTagStream(basePath, port, capability, nonce))
  pipeline([upstream, ...transforms, res], (error) => { if (error && !res.destroyed) res.destroy(error) })
}

export function streamCss(res: Response, upstream: http.IncomingMessage, basePath: string): void {
  res.removeHeader('content-length')
  res.removeHeader('content-encoding')
  res.removeHeader('etag')
  const encoding = String(upstream.headers['content-encoding'] || '').trim().toLowerCase()
  if (encoding && !['identity', 'gzip', 'deflate', 'br'].includes(encoding)) {
    upstream.destroy()
    res.status(502).end('Unsupported content encoding for proxied CSS')
    return
  }
  const transforms: Transform[] = []
  if (encoding === 'gzip' || encoding === 'deflate') transforms.push(createUnzip())
  else if (encoding === 'br') transforms.push(createBrotliDecompress())
  transforms.push(createRootPathRewriteStream(basePath))
  pipeline([upstream, ...transforms, res], (error) => { if (error && !res.destroyed) res.destroy(error) })
}
