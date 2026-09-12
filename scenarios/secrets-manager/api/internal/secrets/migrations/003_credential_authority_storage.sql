-- Make the credential authority the only ordinary secret validation/storage method.
-- Existing installations may have been created with the former Vault-oriented checks.
ALTER TABLE secret_validations
    DROP CONSTRAINT IF EXISTS secret_validations_validation_method_check;
ALTER TABLE secret_validations
    ADD CONSTRAINT secret_validations_validation_method_check
    CHECK (validation_method IN ('credential_authority', 'env', 'file', 'api'));

ALTER TABLE secret_provisions
    DROP CONSTRAINT IF EXISTS secret_provisions_storage_method_check;
ALTER TABLE secret_provisions
    ADD CONSTRAINT secret_provisions_storage_method_check
    CHECK (storage_method IN ('credential-authority', 'env', 'file', 'cloud'));
