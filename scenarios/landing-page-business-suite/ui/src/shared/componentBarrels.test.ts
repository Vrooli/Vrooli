import { describe, expect, it } from 'vitest';

import * as coupons from '../surfaces/admin-portal/components/coupons';
import * as storageWizard from '../surfaces/admin-portal/components/storage-wizard';
import * as userAuth from '../surfaces/user-auth';

// Barrel files are supported public module boundaries for route composition.
// Importing every named export catches broken re-exports before a lazy route
// reaches them and exercises the otherwise unreferenced module branches.
describe('component barrel contracts', () => {
  it('exports every coupon administration component', () => {
    expect(coupons).toMatchObject({
      CouponCard: expect.any(Function),
      CreateCouponModal: expect.any(Function),
      EditCouponModal: expect.any(Function),
      ImportCouponModal: expect.any(Function),
    });
  });

  it('exports every storage wizard step', () => {
    expect(storageWizard).toMatchObject({
      StorageWizard: expect.any(Function),
      StepProvider: expect.any(Function),
      StepConfiguration: expect.any(Function),
      StepCredentials: expect.any(Function),
      StepVerify: expect.any(Function),
      HelpModal: expect.any(Function),
    });
  });

  it('exports both user authentication route components', () => {
    expect(userAuth).toMatchObject({
      UserLogin: expect.any(Function),
      VerifyMagicLink: expect.any(Function),
    });
  });
});
