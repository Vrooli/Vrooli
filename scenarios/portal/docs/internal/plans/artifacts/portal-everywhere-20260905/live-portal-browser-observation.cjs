// Uses the existing browser test dependency; writes metadata only, no pixels or credentials.
const fs = require('node:fs');
const { createRequire } = require('node:module');
const path = require('node:path');
const root = path.resolve(__dirname, '../../../../../../..');
const { chromium } = createRequire(path.join(root, 'scenarios/browser-automation-studio/playwright-driver/package.json'))('rebrowser-playwright');
(async () => {
 const identity = JSON.parse(fs.readFileSync(process.env.PORTAL_TEST_IDENTITY, 'utf8'));
 const browser = await chromium.launch({headless:true});
 let stage = 'navigate';
 try {
  const page = await browser.newPage({viewport:{width:1440,height:1000}});
  page.setDefaultTimeout(15000);
  await page.goto(process.env.PORTAL_TEST_URL || 'http://127.0.0.1:24965');
  stage = 'select desktop';
  const option = page.locator('#surface-selection option').filter({hasText:'Desktop session 2'});
  await option.waitFor({state:'attached'});
  await page.selectOption('#surface-selection',await option.getAttribute('value'));
  stage = 'login';
  await page.getByLabel('Email',{exact:true}).fill(identity.email);
  await page.getByLabel('Password',{exact:true}).fill(identity.password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  stage = 'open observation';
  await page.getByRole('button',{name:'Open observation',exact:true}).click();
  const image = page.locator('section img[src^="blob:"]');
  await image.waitFor();
  await image.evaluate(img => img.decode());
  const dimensions = await image.evaluate(img=>({width:img.naturalWidth,height:img.naturalHeight}));
  if(dimensions.width!==1920||dimensions.height!==1080)throw new Error('unexpected dimensions');
  const selectionDisabled = await page.locator('#surface-selection').isDisabled();
  const imageDisabled = await page.locator('section button:has(img[src^="blob:"])').isDisabled();
  if(!selectionDisabled||!imageDisabled)throw new Error('observation guards missing');
  const persisted = await page.evaluate(()=>[...Object.values(localStorage),...Object.values(sessionStorage)]);
  if(persisted.some(v=>v.includes(identity.password)||v.includes('accessToken')||v.includes('refreshToken')||v.includes('eyJ')))throw new Error('credential persistence');
  stage = 'stop';
  await page.getByRole('button',{name:'Stop session',exact:true}).click();
  await image.waitFor({state:'detached'});
  stage = 'logout';
  await page.getByRole('button',{name:'Sign out',exact:true}).click();
  await page.getByRole('button',{name:'Sign in',exact:true}).waitFor();
  const evidence = {browser:'installed rebrowser-playwright Chromium',viewport:{width:1440,height:1000},login:true,observation:dimensions,selection_disabled_while_active:selectionDisabled,observation_input_disabled:imageDisabled,no_persisted_credentials:true,stop:true,sign_out:true,timestamp:new Date().toISOString()};
  fs.writeFileSync(path.join(__dirname,'live-portal-browser-observation.json'),JSON.stringify(evidence,null,2)+'\n');
  console.log(JSON.stringify(evidence));
 } catch (error) { throw new Error(stage+': '+error.name); }
 finally { await browser.close(); }
})().catch(error=>{console.error(error.message);process.exitCode=1;});
