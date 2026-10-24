# External installs

## 2026-10-05 — Agent Manager CLI managed projection refresh

- Tool and location: managed `agent-manager` CLI at `/home/matthalloran8/.vrooli/bin/agent-manager`.
- Reason: native reconciliation and assignment for original orchestrator run `2b16a068-2bc1-4a41-8629-54c4992c93c6` and existing keyboard task `40a2dcc0-58c1-4fee-9af2-85f90bc04cc9`; the CLI refreshed itself from its owning scenario sources while serving the task command.
- Authorization: owner-authorized operation inside active revision `rcl-rehabilitation-20260930-1`; managed CLI projection refresh only, not an external dependency install or service start.
- Cleanup: keep the managed projection; its lifecycle is owned by Agent Manager.

## 2026-10-01 — Browser Automation Studio CLI projection refresh

- Tool and size: `browser-automation-studio` CLI at `/home/matthalloran8/.vrooli/bin/browser-automation-studio` (36 MiB).
- Reason: orchestrator acceptance read of the retained desktop/mobile E1 workflow executions detected an outdated CLI projection; its owner refreshed the command from current BAS sources during the read.
- Authorization: orchestrator action within the already operator-authorized E1 acceptance work; no package/dependency install and no external process started.
- Cleanup: keep the managed CLI projection in place; removal is owned by the BAS CLI lifecycle and could break other consumers.

## 2026-10-07 — Browser Automation Studio CLI projection refresh

- Tool and location: managed `browser-automation-studio` CLI at `/home/matthalloran8/.vrooli/bin/browser-automation-studio`.
- Reason: read-only discovery of BAS-supported failure drills after the live replacement UI artifact preparation failure.
- Authorization: orchestrator operation inside the active RCL revision; invoking the owner CLI refreshed its projection from current BAS sources (fingerprint `2c4d7bb435bf9d4bde3bf1ea1948fe74d964da4ab8c95c72af0a6f0c85bf9b18`). No dependency install or service lifecycle action.
- Cleanup: keep the managed CLI projection; do not retry just to avoid this refresh.

## 2026-10-07 — Agent Manager CLI projection refresh

- Tool and location: managed `agent-manager` CLI at `/home/matthalloran8/.vrooli/bin/agent-manager`.
- Reason: owner `run park` operations refreshed the command from current Agent Manager sources before successfully parking original run `2b16a068-2bc1-4a41-8629-54c4992c93c6`.
- Authorization: orchestrator operation inside the active RCL revision; managed CLI projection refresh only. Additional fingerprint observed during the latest park: `e6aaba3280e97fff4f26bb21132377240735b7caf4fe3fee1410569e90db4a4d`. No dependency install or service lifecycle action.
- Cleanup: keep the managed CLI projection; its lifecycle is owned by Agent Manager.
- Later park refresh (2026-10-08 15:50Z): Agent Manager rebuilt its managed projection to source fingerprint `ea4529bf3f77c6fe52f054015ad8a221b62c389e468dad38782de799200d9ef7`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 16:05Z): Agent Manager rebuilt its managed projection to source fingerprint `a1ce32c70cb6870b105616e39be5074c232d0869940c0f5483a693b3c5b3147b`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 16:21Z): Agent Manager rebuilt its managed projection to source fingerprint `470a0b79c6ae4a9bcbac57598c2df306c1717e1d13e1ded5eef5c171013e34a5`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 16:36Z): Agent Manager rebuilt its managed projection to source fingerprint `c3f4a0adf33ae0512497ec861e4bb17f6b95b92adab22efa661e107f8de754a3`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 16:51Z): Agent Manager rebuilt its managed projection to source fingerprint `c78013325d640f83133b5c82d7f34a802f4841af97a0990b2b2954c6cf537ccf`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 17:06Z): Agent Manager rebuilt its managed projection to source fingerprint `c94966bc57a0172c962fa504e866117424a25e1eaa0e0bc6d08d0e90e470acbb`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 17:37Z): Agent Manager rebuilt its managed projection to source fingerprint `45deb946c6a5d01e4dd4da0a67c83b69e40527ed384fe1138adf7060285f13cc`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 17:52Z): Agent Manager rebuilt its managed projection to source fingerprint `e9843773ab3f2fb7bdecbef60f8b684f07f7d0626bcc53feadc7dc18b968c2d8`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 18:07Z): Agent Manager rebuilt its managed projection to source fingerprint `cf3957d549e93b10105c8f0ce3268f1f73d990dea35da49b60a6a02dd6701580`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 18:22Z): Agent Manager rebuilt its managed projection to source fingerprint `470fc422c46419ae2e1a75881eb295c00be906c65caf9d33b6bde50a39924d2f`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-08 18:40Z): Agent Manager rebuilt its managed projection to source fingerprint `1cb4df4bf80372c44ee2d6615a4ca6131b5aef653a4d6e2fcbb77fe48e47f82e`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-09 05:46Z): Agent Manager rebuilt its managed projection to source fingerprint `b124b1b727f7b3ffb86695d51467726df9b25c1b0081f6b9bd9b040c5938c517`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-09 06:01Z): Agent Manager rebuilt its managed projection to source fingerprint `882dafc917429e66130c1ff8e9485bcc8c382865ae07f05d961de64ba49c78a6`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-09 16:37Z): Agent Manager rebuilt its managed projection to source fingerprint `cec6dcd99347aca7d153a7e1013daeadcfd8873e6990aa600de607853d22449a`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 01:46Z): Agent Manager rebuilt its managed projection to source fingerprint `b46b6c3372309f5541ee31993ab499216be6fa1f0ef3d920e55fe455f6f14875`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 02:01Z): Agent Manager rebuilt its managed projection to source fingerprint `01ee635d4c7b3f40c1846030d3480bb8a7b478d883b8ca42e2f182ca812202f1`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 02:16Z): Agent Manager rebuilt its managed projection to source fingerprint `8ac93a71ea3da176e8353ec9537ad55ab170373f5cb47b7897adbfc9ff88e9d7`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 02:47Z): Agent Manager rebuilt its managed projection to source fingerprint `52ea9a1621a12058a0a692bb805d5404ef57572e8ce4e3817a69f5968279fac1`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 03:02Z): Agent Manager rebuilt its managed projection to source fingerprint `677dc3d8cfe7fed0cb544a1603f06d35590774cff965f902580e0108d20519e8`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 03:17Z): Agent Manager rebuilt its managed projection to source fingerprint `930f33b564d405f14cd8a35c08766eeece2c59e559e461792e941232ee616d11`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 03:32Z): Agent Manager rebuilt its managed projection to source fingerprint `4cec1ffa7f7a8beccd1635ba23f35916edcfee2632a1cb2734c4f58a33046c89`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 03:48Z): Agent Manager rebuilt its managed projection to source fingerprint `062a91950d9a1a78127242f6e613e6b54195f39396dd56a427b8cb0ee57e9f74`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 04:03Z): Agent Manager rebuilt its managed projection to source fingerprint `65b7cf83ef38e8a82d7187a0656ed9d3caeefe5fc40ffed7cdffd5b6538bc2c8`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-11 14:57Z): Agent Manager rebuilt its managed projection to source fingerprint `c90595fbf53fdba282be4131866aa2f5de5b1b6d39c194b545490cd47603f9cb`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 00:37Z): Agent Manager rebuilt its managed projection to source fingerprint `4c4e6da66f179f7eb9849265c78d16149e4cdd7936c75a4ffe5fbdcbb549118f`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 01:08Z): Agent Manager rebuilt its managed projection to source fingerprint `97a1768cfd6e743067125ec027a64b3fa3b2462c55ddcfe29cd8e839996d8ec0`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 01:23Z): Agent Manager rebuilt its managed projection to source fingerprint `33dff902276f14d83c6753e376d6b1d1f103e79e8c3627ae1da8214422f2bf21`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 01:38Z): Agent Manager rebuilt its managed projection to source fingerprint `1b3354ffffa6e4d465a741530baf1dbde6a13f34b475c223ab918e55977830cd`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 01:54Z): Agent Manager rebuilt its managed projection to source fingerprint `11ca91541fd07db3b1832d439490487c64809c1a6d079155602c858a57dbbe18`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 02:09Z): Agent Manager rebuilt its managed projection to source fingerprint `1f9230d4ef0bde4bbe467abf66052dc2a4198f5b57d7fe30193338096b1aed98`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 02:54Z): Agent Manager rebuilt its managed projection to source fingerprint `35971fb56cf0be76c0c564e313843e48497870c448827faf7189286d8ac5a7dc`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 04:10Z): Agent Manager rebuilt its managed projection to source fingerprint `5311b516a6ca6aa8b6716bce1ee04f5a6c0049316ec726d075f9dce49460c1f4`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 12:46Z): Agent Manager rebuilt its managed projection to source fingerprint `f7e870ce6ac1c5dc9fc29381ca0bc9e1262c3b22091d8038cbaf23779275ccf9`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 13:01Z): Agent Manager rebuilt its managed projection to source fingerprint `c431f6e0aeabf331cb935b0414c9883bc3275497c51ab57655f39e4a985ec9a`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-12 17:05Z): Agent Manager rebuilt its managed projection to source fingerprint `c6a8136cbf381f56294051dcb58fe0e38fec9f98789c7e7c40b9d46c0261a830`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-13 17:40Z): Agent Manager rebuilt its managed projection to source fingerprint `81539a6324b97f6525a62c6c32807d4db07843a86a8ab8ff6df3a3506f022582`; no dependency install or service lifecycle action.
- Later park refresh (2026-10-13 23:14Z): Agent Manager rebuilt its managed projection to source fingerprint `f97e2e6fed049fc84ae18296b1b4f0b17c575cfc2cbf58ac11b8dde62650afe4`; no dependency install or service lifecycle action.
