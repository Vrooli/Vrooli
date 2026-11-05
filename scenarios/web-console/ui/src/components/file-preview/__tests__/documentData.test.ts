import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { parseDocument } from '../parseDocument';
import { sanitizeDocument } from '../sanitizeDocument';
import { MAX_DOCUMENT_BYTES } from '../documentData';
const fixture = readFileSync('../fixtures/file-preview/operations-brief.docx');
function bytes() { return Uint8Array.from(fixture).buffer; }
// Stored ZIP fixtures exercise the real converter without another test dependency.
function storedZip(parts: Record<string, string | Uint8Array>): ArrayBuffer {
  const local: Buffer[] = [], central: Buffer[] = []; let offset = 0;
  for (const [name, value] of Object.entries(parts)) {
    const filename = Buffer.from(name), data = Buffer.from(value);
    let crc = 0xffffffff;
    for (const byte of data) { crc ^= byte; for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0); }
    crc = (crc ^ 0xffffffff) >>> 0;
    const header = Buffer.alloc(30); header.writeUInt32LE(0x04034b50); header.writeUInt16LE(20, 4); header.writeUInt32LE(crc, 14); header.writeUInt32LE(data.length, 18); header.writeUInt32LE(data.length, 22); header.writeUInt16LE(filename.length, 26);
    local.push(header, filename, data);
    const directory = Buffer.alloc(46); directory.writeUInt32LE(0x02014b50); directory.writeUInt16LE(20, 4); directory.writeUInt16LE(20, 6); directory.writeUInt32LE(crc, 16); directory.writeUInt32LE(data.length, 20); directory.writeUInt32LE(data.length, 24); directory.writeUInt16LE(filename.length, 28); directory.writeUInt32LE(offset, 42);
    central.push(directory, filename); offset += header.length + filename.length + data.length;
  }
  const directory = Buffer.concat(central), end = Buffer.alloc(22); end.writeUInt32LE(0x06054b50); end.writeUInt16LE(Object.keys(parts).length, 8); end.writeUInt16LE(Object.keys(parts).length, 10); end.writeUInt32LE(directory.length, 12); end.writeUInt32LE(offset, 16);
  return Uint8Array.from(Buffer.concat([...local, directory, end])).buffer;
}
function imageDocument(type: string, external = false) {
  return storedZip({
    '[Content_Types].xml': `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="png" ContentType="${type}"/><Default Extension="xml" ContentType="application/xml"/></Types>`,
    'word/document.xml': `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body><w:p><w:r><w:drawing><wp:inline><wp:docPr descr="Delivery chart"/><a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:${external ? 'link' : 'embed'}="image1"/></pic:blipFill></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p></w:body></w:document>`,
    'word/_rels/document.xml.rels': `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="image1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="${external ? 'https://tracker.invalid/image.png' : 'media/image.png'}"${external ? ' TargetMode="External"' : ''}/></Relationships>`,
    'word/media/image.png': new Uint8Array([137, 80, 78, 71]),
  });
}
describe('Word content conversion', () => {
  it('reads real Word headings, paragraphs and table cells in document order', async () => {
    const result = await parseDocument(bytes()); expect(result.error).toBeUndefined();
    const html = result.data?.html ?? ''; expect(html).toContain('<h1>Project overview</h1>'); expect(html).toContain('<h2>Our focus</h2>');
    expect(html).toContain('<table>'); expect(html).toContain('Maya Chen'); expect(html.indexOf('Project overview')).toBeLessThan(html.indexOf('Delivery milestones'));
  });
  it('embeds raster images with alt text and omits active image formats', async () => {
    const raster = await parseDocument(imageDocument('image/png')); expect(raster.data?.html).toContain('data:image/png;base64,'); expect(raster.data?.html).toContain('alt="Delivery chart"');
    const vector = await parseDocument(imageDocument('image/svg+xml')); expect(vector.data?.omittedImages).toBe(1); expect(vector.data?.html).not.toContain('data:image/svg');
  });
  it('never fetches linked images from Word relationships', async () => {
    const result = await parseDocument(imageDocument('image/png', true)); expect(result.data?.html).not.toContain('https://tracker.invalid'); expect(result.data?.warnings).toBe(true);
  });
  it('rejects invalid, encrypted, truncated and oversized input without displaying partial content', async () => {
    expect(await parseDocument(new TextEncoder().encode('<html>wrong format</html>').buffer)).toEqual({ error: 'invalid' });
    expect(await parseDocument(new Uint8Array([0xd0, 0xcf, 0x11, 0xe0]).buffer)).toEqual({ error: 'encrypted' });
    expect(await parseDocument(new Uint8Array([0x50, 0x4b, 3, 4]).buffer)).toEqual({ error: 'invalid' });
    expect(await parseDocument(new ArrayBuffer(MAX_DOCUMENT_BYTES + 1))).toEqual({ error: 'limit' });
  });
  it('rejects declared ZIP expansion and ZIP64 before inflating document XML', async () => {
    const archive = bytes(), view = new DataView(archive);
    let directory = 0; for (let i = 0; i < archive.byteLength - 46; i++) if (view.getUint32(i, true) === 0x02014b50) { directory = i; break; }
    view.setUint32(directory + 24, 129 * 1024 * 1024, true); expect(await parseDocument(archive)).toEqual({ error: 'limit' });
    const zip64 = bytes(), zipView = new DataView(zip64); let end = zip64.byteLength - 22;
    while (zipView.getUint32(end, true) !== 0x06054b50) end--;
    zipView.setUint16(end + 10, 65535, true); expect(await parseDocument(zip64)).toEqual({ error: 'limit' });
  });
});
describe('inert Word rendering boundary', () => {
  it('removes executable markup, authored styles, foreign elements, forms and remote image requests', () => {
    const value = sanitizeDocument('<h1 style="position:fixed" onclick="evil()">Title</h1><script>evil()</script><style>body{display:none}</style><iframe src="/private"></iframe><form><input autofocus></form><svg onload="evil()"><circle/></svg><img src="https://tracker.invalid/record" onerror="evil()"><img src="data:image/svg+xml;base64,PHN2Zz4=">', 'Document table');
    const node = document.createElement('div'); node.innerHTML = value;
    expect(node.querySelector('h1')?.textContent).toBe('Title'); expect(node.querySelector('script,style,iframe,form,input,svg,img,[style],[onclick],[onerror]')).toBeNull();
  });
  it('retains semantic content and bounded embedded raster images, with safe external links and local notes', () => {
    const value = sanitizeDocument('<p><strong>Bold</strong><em>Italic</em><a href="javascript:evil()">bad</a><a href="file:///etc/passwd">local</a><a href="https://example.com">web</a><a href="#docx-note">note</a></p><img alt="Chart" src="data:image/png;base64,AAAA"><table><tr><td colspan="2">Value</td></tr></table>', 'Document table');
    const node = document.createElement('div'); node.innerHTML = value;
    expect(node.querySelector('strong')?.textContent).toBe('Bold'); expect(node.querySelector('em')?.textContent).toBe('Italic');
    expect([...node.querySelectorAll('a')].map(a => a.getAttribute('href'))).toEqual([null, null, 'https://example.com', '#docx-note']);
    expect(node.querySelector('a[target]')).toHaveAttribute('rel', 'noopener noreferrer'); expect(node.querySelector('img')).toHaveAttribute('alt', 'Chart'); expect(node.querySelector('img')).toHaveAttribute('loading', 'lazy');
    expect(node.querySelector('.wc-document-table td')).toHaveAttribute('colspan', '2');
  });
});
